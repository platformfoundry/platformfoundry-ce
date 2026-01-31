package gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// GitHubPRClient implements PRClient for GitHub
type GitHubPRClient struct {
	token      string
	owner      string
	repo       string
	httpClient *http.Client
	baseURL    string
}

// GitHubConfig contains GitHub client configuration
type GitHubConfig struct {
	Token   string
	Owner   string
	Repo    string
	BaseURL string // For GitHub Enterprise, defaults to api.github.com
}

// NewGitHubPRClient creates a new GitHub PR client
func NewGitHubPRClient(cfg GitHubConfig) (*GitHubPRClient, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return nil, fmt.Errorf("GitHub owner and repo are required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	return &GitHubPRClient{
		token:      cfg.Token,
		owner:      cfg.Owner,
		repo:       cfg.Repo,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// CreatePR creates a new pull request
func (c *GitHubPRClient) CreatePR(ctx context.Context, opts CreatePROptions) (*types.PullRequestState, error) {
	payload := map[string]interface{}{
		"title": opts.Title,
		"body":  opts.Body,
		"head":  opts.SourceBranch,
		"base":  opts.TargetBranch,
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, c.owner, c.repo)
	resp, err := c.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	var prResp githubPRResponse
	if err := json.Unmarshal(resp, &prResp); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	pr := &types.PullRequestState{
		Number:    prResp.Number,
		Title:     prResp.Title,
		URL:       prResp.HTMLURL,
		State:     prResp.State,
		Branch:    opts.SourceBranch,
		CreatedAt: prResp.CreatedAt,
		UpdatedAt: prResp.UpdatedAt,
	}

	// Add labels if specified
	if len(opts.Labels) > 0 {
		if err := c.AddLabels(ctx, prResp.Number, opts.Labels); err != nil {
			fmt.Printf("Warning: failed to add labels: %v\n", err)
		}
	}

	// Add reviewers if specified
	if len(opts.Reviewers) > 0 {
		if err := c.AddReviewers(ctx, prResp.Number, opts.Reviewers); err != nil {
			fmt.Printf("Warning: failed to add reviewers: %v\n", err)
		}
	}

	return pr, nil
}

// GetPR retrieves a pull request by number
func (c *GitHubPRClient) GetPR(ctx context.Context, number int) (*types.PullRequestState, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, c.owner, c.repo, number)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var prResp githubPRResponse
	if err := json.Unmarshal(resp, &prResp); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	pr := &types.PullRequestState{
		Number:    prResp.Number,
		Title:     prResp.Title,
		URL:       prResp.HTMLURL,
		State:     prResp.State,
		Branch:    prResp.Head.Ref,
		CreatedAt: prResp.CreatedAt,
		UpdatedAt: prResp.UpdatedAt,
		MergedAt:  prResp.MergedAt,
	}

	// Get approvals count
	reviews, err := c.getReviews(ctx, number)
	if err == nil {
		approvals := 0
		for _, review := range reviews {
			if review.State == "APPROVED" {
				approvals++
			}
		}
		pr.Approvals = approvals
	}

	return pr, nil
}

// MergePR merges a pull request
func (c *GitHubPRClient) MergePR(ctx context.Context, number int) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", c.baseURL, c.owner, c.repo, number)
	payload := map[string]interface{}{
		"merge_method": "squash",
	}

	_, err := c.doRequest(ctx, "PUT", url, payload)
	if err != nil {
		return fmt.Errorf("failed to merge PR: %w", err)
	}

	return nil
}

// ListOpenPRs lists all open pull requests
func (c *GitHubPRClient) ListOpenPRs(ctx context.Context) ([]types.PullRequestState, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open", c.baseURL, c.owner, c.repo)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	var prResps []githubPRResponse
	if err := json.Unmarshal(resp, &prResps); err != nil {
		return nil, fmt.Errorf("failed to parse PRs response: %w", err)
	}

	prs := make([]types.PullRequestState, len(prResps))
	for i, prResp := range prResps {
		prs[i] = types.PullRequestState{
			Number:    prResp.Number,
			Title:     prResp.Title,
			URL:       prResp.HTMLURL,
			State:     prResp.State,
			Branch:    prResp.Head.Ref,
			CreatedAt: prResp.CreatedAt,
			UpdatedAt: prResp.UpdatedAt,
		}
	}

	return prs, nil
}

// AddReviewers adds reviewers to a pull request
func (c *GitHubPRClient) AddReviewers(ctx context.Context, number int, reviewers []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/requested_reviewers", c.baseURL, c.owner, c.repo, number)
	payload := map[string]interface{}{
		"reviewers": reviewers,
	}

	_, err := c.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to add reviewers: %w", err)
	}

	return nil
}

// AddLabels adds labels to a pull request
func (c *GitHubPRClient) AddLabels(ctx context.Context, number int, labels []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.baseURL, c.owner, c.repo, number)
	payload := map[string]interface{}{
		"labels": labels,
	}

	_, err := c.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	return nil
}

// getReviews gets reviews for a PR
func (c *GitHubPRClient) getReviews(ctx context.Context, number int) ([]githubReview, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", c.baseURL, c.owner, c.repo, number)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var reviews []githubReview
	if err := json.Unmarshal(resp, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
}

// doRequest performs an HTTP request to the GitHub API
func (c *GitHubPRClient) doRequest(ctx context.Context, method, url string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// GitHub API response types

type githubPRResponse struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Head      struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

type githubReview struct {
	State string `json:"state"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

// ParseGitHubURL parses a GitHub URL to extract owner and repo
func ParseGitHubURL(url string) (owner, repo string, err error) {
	// Handle HTTPS URLs
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	// Handle SSH URLs
	if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	return "", "", fmt.Errorf("unable to parse GitHub URL: %s", url)
}
