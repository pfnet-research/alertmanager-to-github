package notifier

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/pfnet-research/alertmanager-to-github/pkg/template"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

const (
	ownerLabelName = "atg_owner"
	repoLabelName  = "atg_repo"
)

var (
	rateLimit = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_api_rate_limit",
			Help: "The limit of API requests the client can make.",
		},
		// The GitHub API this rate limit applies to. e.g. "search" or "issues"
		[]string{"api"},
	)
	rateRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_api_rate_remaining",
			Help: "The remaining API requests the client can make until reset time.",
		},
		// The GitHub API this rate limit applies to. e.g. "search" or "issues"
		[]string{"api"},
	)
	rateResetTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_api_rate_reset",
			Help: "The time when the current rate limit will reset.",
		},
		// The GitHub API this rate limit applies to. e.g. "search" or "issues"
		[]string{"api"},
	)
	operationCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_api_requests_total",
			Help: "Number of API operations performed.",
		},
		// api: The GitHub API this rate limit applies to. e.g. "search" or "issues"
		// status: The status code of the response
		[]string{"api", "status"},
	)
)

type GitHubNotifier struct {
	GitHubClient            *github.Client
	BodyTemplate            *template.Template
	TitleTemplate           *template.Template
	AlertIDTemplate         *template.Template
	Labels                  []string
	AutoCloseResolvedIssues bool
}

func NewGitHub() (*GitHubNotifier, error) {
	return &GitHubNotifier{}, nil
}

func resolveRepository(payload *types.WebhookPayload, queryParams url.Values) (error, string, string) {
	owner := queryParams.Get("owner")
	repo := queryParams.Get("repo")

	if payload.CommonLabels[ownerLabelName] != "" {
		owner = payload.CommonLabels[ownerLabelName]
	}
	if payload.CommonLabels[repoLabelName] != "" {
		repo = payload.CommonLabels[repoLabelName]
	}
	if owner == "" {
		return fmt.Errorf("owner was not specified in either the webhook URL, or the alert labels"), "", ""
	}
	if repo == "" {
		return fmt.Errorf("repo was not specified in either the webhook URL, or the alert labels"), "", ""
	}
	return nil, owner, repo
}

func (n *GitHubNotifier) Notify(ctx context.Context, payload *types.WebhookPayload, queryParams url.Values) error {
	err, owner, repo := resolveRepository(payload, queryParams)
	if err != nil {
		return err
	}

	labels := n.Labels
	if l := queryParams.Get("labels"); l != "" {
		labels = strings.Split(l, ",")
	}

	alertID, err := n.getAlertID(payload)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`repo:%s/%s "%s"`, owner, repo, alertID)
	searchResult, response, err := n.GitHubClient.Search.Issues(ctx, query, &github.SearchOptions{
		TextMatch: true,
	})
	if err != nil {
		return err
	}

	updateGithubApiMetrics("search", response)
	if err = checkSearchResponse(response); err != nil {
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
	body += fmt.Sprintf("\n<!-- (UNIQUE ALERT ID, DO NOT MODIFY: %s ) -->\n", alertID)

	title, err := n.TitleTemplate.Execute(payload)
	if err != nil {
		return err
	}
	// prevent trailing newline characters in the title due to template formatting
	// newlines in titles prevent Github->Slack webhooks working with issues as of 2022-05-06
	title = strings.TrimSpace(title)

	req := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}

	if issue == nil {
		issue, response, err = n.GitHubClient.Issues.Create(ctx, owner, repo, req)
		if err != nil {
			return err
		}

		updateGithubApiMetrics("issues", response)
		log.Info().Msgf("created an issue: %s", issue.GetURL())
	} else {
		// we have to merge existing labels because Edit api replaces its  labels
		mergedLabels := []string{}
		labelSet := map[string]bool{}
		for _, l := range issue.Labels {
			name := *l.Name
			if !labelSet[name] {
				labelSet[name] = true
				mergedLabels = append(mergedLabels, name)
			}
		}
		for _, l := range labels {
			if !labelSet[l] {
				labelSet[l] = true
				mergedLabels = append(mergedLabels, l)
			}
		}
		req.Labels = &mergedLabels
		issue, _, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}

		updateGithubApiMetrics("issues", response)
		log.Info().Msgf("edited an issue: %s", issue.GetURL())
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

	currentState := issue.GetState()
	canUpdateState := desiredState != "closed" || n.AutoCloseResolvedIssues

	if desiredState != currentState && canUpdateState {
		req = &github.IssueRequest{
			State: github.String(desiredState),
		}
		issue, response, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}

		updateGithubApiMetrics("issues", response)
		log.Info().Str("state", desiredState).Msgf("updated state of the issue: %s", issue.GetURL())
	}

	if err := n.cleanupIssues(ctx, owner, repo, alertID); err != nil {
		return err
	}

	return nil
}

func (n *GitHubNotifier) cleanupIssues(ctx context.Context, owner, repo, alertID string) error {
	query := fmt.Sprintf(`repo:%s/%s "%s"`, owner, repo, alertID)
	searchResult, response, err := n.GitHubClient.Search.Issues(ctx, query, &github.SearchOptions{
		TextMatch: true,
	})
	if err != nil {
		return err
	}

	updateGithubApiMetrics("search", response)
	if err = checkSearchResponse(response); err != nil {
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
		issue, response, err = n.GitHubClient.Issues.Edit(ctx, owner, repo, issue.GetNumber(), req)
		if err != nil {
			return err
		}

		updateGithubApiMetrics("issues", response)
		log.Info().Msgf("closed an issue: %s", issue.GetURL())
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

func checkSearchResponse(response *github.Response) error {
	if response.StatusCode < 200 || 300 <= response.StatusCode {
		return fmt.Errorf("issue search returned %d", response.StatusCode)
	}
	return nil
}

func updateGithubApiMetrics(apiName string, resp *github.Response) {
	rateLimit.WithLabelValues(apiName).Set(float64(resp.Rate.Limit))
	rateRemaining.WithLabelValues(apiName).Set(float64(resp.Rate.Remaining))
	rateResetTime.WithLabelValues(apiName).Set(float64(resp.Rate.Reset.UTC().Unix()))
	operationCount.WithLabelValues(apiName, strconv.Itoa(resp.StatusCode)).Inc()
}
