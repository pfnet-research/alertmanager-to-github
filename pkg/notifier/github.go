package notifier

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/google/go-github/v32/github"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/rs/zerolog/log"
	"net/url"
	"sort"
	"strings"
)

type GitHubNotifier struct {
	GitHubClient    *github.Client
	BodyTemplate    *template.Template
	TitleTemplate   *template.Template
	AlertIDTemplate *template.Template
	Labels          []string
}

func NewGitHub() (*GitHubNotifier, error) {
	return &GitHubNotifier{}, nil
}

func (n *GitHubNotifier) Notify(ctx context.Context, payload *types.WebhookPayload, queryParams url.Values) error {
	owner := queryParams.Get("owner")
	repo := queryParams.Get("repo")

	labels := n.Labels
	if l := queryParams.Get("labels"); l != "" {
		labels = strings.Split(l, ",")
	}

	alertID, err := n.getAlertID(payload)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`repo:%s/%s "%s"`, owner, repo, alertID)
	searchResult, _, err := n.GitHubClient.Search.Issues(ctx, query, &github.SearchOptions{
		TextMatch: true,
	})
	if err != nil {
		return err
	}

	var issue *github.Issue
	if searchResult.GetTotal() == 1 {
		issue = searchResult.Issues[0]
	} else if searchResult.GetTotal() > 1 {
		log.Warn().Interface("searchResultTotal", searchResult.GetTotal()).
			Str("groupKey", payload.GroupKey).Msg("too many search result")

		for _, i := range searchResult.Issues {
			if issue == nil || issue.GetCreatedAt().Before(i.GetCreatedAt()) {
				issue = i
			}
		}
	}

	body, err := n.BodyTemplate.Execute(payload)
	if err != nil {
		return err
	}
	body += fmt.Sprintf("\n---\n(DO NOT MODIFY: %s )\n", alertID)

	title, err := n.TitleTemplate.Execute(payload)
	if err != nil {
		return err
	}

	req := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}

	if issue == nil {
		issue, _, err = n.GitHubClient.Issues.Create(ctx, owner, repo, req)
		if err != nil {
			return err
		}
		log.Info().Msg("created an issue")
	} else {
		issue, _, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}
		log.Info().Msg("edited an issue")
	}

	var desiredState string
	switch payload.Status {
	case types.AlertStatusFiring:
		desiredState = "open"
	case types.AlertStatusResolved:
		desiredState = "closed"
	default:
		return fmt.Errorf("invalid alert status %s", payload.Status)
	}

	if desiredState != issue.GetState() {
		req = &github.IssueRequest{
			State: github.String(desiredState),
		}
		issue, _, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}

		log.Info().Str("state", desiredState).Msg("updated state of the issue")
	}

	if err := n.cleanupIssues(ctx, owner, repo, alertID); err != nil {
		return err
	}

	return nil
}

func (n *GitHubNotifier) cleanupIssues(ctx context.Context, owner, repo, alertID string) error {
	query := fmt.Sprintf(`repo:%s/%s "%s"`, owner, repo, alertID)
	searchResult, _, err := n.GitHubClient.Search.Issues(ctx, query, &github.SearchOptions{
		TextMatch: true,
	})
	if err != nil {
		return err
	}

	issues := searchResult.Issues
	if len(issues) <= 1 {
		return nil
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].GetCreatedAt().Before(issues[j].GetCreatedAt())
	})

	latestIssue := issues[len(issues)-1]
	oldIssues := issues[:len(issues)-1]
	for _, issue := range oldIssues {
		req := &github.IssueRequest{
			Body:  github.String(fmt.Sprintf("duplicated %s", latestIssue.GetHTMLURL())),
			State: github.String("closed"),
		}
		issue, _, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *GitHubNotifier) getAlertID(payload *types.WebhookPayload) (string, error) {
	id, err := n.AlertIDTemplate.Execute(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(id))), nil
}
