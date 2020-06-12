package notifier

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/google/go-github/v32/github"
	"github.com/rs/zerolog/log"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"net/url"
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
	log.Info().Interface("query", queryParams).Msg("")

	ownerAndRepo := queryParams.Get("repo")
	if ownerAndRepo == "" {
		return errors.New("repo query parameter is required")
	}
	parts := strings.SplitN(ownerAndRepo, "/", 2)
	owner := parts[0]
	repo := parts[1]

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
		Labels: &n.Labels,
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

	return nil
}

func (n *GitHubNotifier) getAlertID(payload *types.WebhookPayload) (string, error) {
	id, err := n.AlertIDTemplate.Execute(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(id))), nil
}
