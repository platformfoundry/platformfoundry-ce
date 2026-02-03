package gitops

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BitbucketProvider implements Provider for Bitbucket Cloud
type BitbucketProvider struct {
	username   string
	appPassword string
	baseURL    string
	httpClient *http.Client
}

// BitbucketConfig contains Bitbucket configuration
type BitbucketConfig struct {
	Username    string
	AppPassword string
	BaseURL     string // defaults to https://api.bitbucket.org/2.0
}

// NewBitbucketProvider creates a new Bitbucket provider
func NewBitbucketProvider(cfg BitbucketConfig) (*BitbucketProvider, error) {
	if cfg.Username == "" || cfg.AppPassword == "" {
		return nil, fmt.Errorf("Bitbucket username and app password are required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.bitbucket.org/2.0"
	}

	return &BitbucketProvider{
		username:    cfg.Username,
		appPassword: cfg.AppPassword,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *BitbucketProvider) Name() string {
	return "bitbucket"
}

// CreateRepository creates a new repository
func (p *BitbucketProvider) CreateRepository(ctx context.Context, opts CreateRepoOpts) (*Repository, error) {
	// Bitbucket uses workspace/repo format
	// For simplicity, use username as workspace
	payload := map[string]interface{}{
		"scm":         "git",
		"is_private":  opts.Private,
		"description": opts.Description,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s", p.username, opts.Name), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	var repo bitbucketRepository
	if err := json.Unmarshal(resp, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertRepository(&repo), nil
}

// GetRepository gets a repository
func (p *BitbucketProvider) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s", owner, repo), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var r bitbucketRepository
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertRepository(&r), nil
}

// DeleteRepository deletes a repository
func (p *BitbucketProvider) DeleteRepository(ctx context.Context, owner, repo string) error {
	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/repositories/%s/%s", owner, repo), nil)
	return err
}

// GetPullRequest gets a pull request
func (p *BitbucketProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", owner, repo, number), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	var pr bitbucketPullRequest
	if err := json.Unmarshal(resp, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertPullRequest(&pr), nil
}

// CreatePullRequest creates a pull request
func (p *BitbucketProvider) CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePullRequestOpts) (*PullRequest, error) {
	payload := map[string]interface{}{
		"title": opts.Title,
		"source": map[string]interface{}{
			"branch": map[string]string{
				"name": opts.SourceBranch,
			},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{
				"name": opts.TargetBranch,
			},
		},
	}
	if opts.Body != "" {
		payload["description"] = opts.Body
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests", owner, repo), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	var pr bitbucketPullRequest
	if err := json.Unmarshal(resp, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertPullRequest(&pr), nil
}

// UpdatePullRequest updates a pull request
func (p *BitbucketProvider) UpdatePullRequest(ctx context.Context, owner, repo string, number int, opts UpdatePullRequestOpts) (*PullRequest, error) {
	payload := make(map[string]interface{})
	if opts.Title != nil {
		payload["title"] = *opts.Title
	}
	if opts.Body != nil {
		payload["description"] = *opts.Body
	}

	resp, err := p.doRequest(ctx, "PUT", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", owner, repo, number), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	var pr bitbucketPullRequest
	if err := json.Unmarshal(resp, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertPullRequest(&pr), nil
}

// MergePullRequest merges a pull request
func (p *BitbucketProvider) MergePullRequest(ctx context.Context, owner, repo string, number int, opts MergeOpts) error {
	payload := make(map[string]interface{})
	if opts.Method != "" {
		payload["merge_strategy"] = opts.Method
	}
	if opts.CommitMessage != "" {
		payload["message"] = opts.CommitMessage
	}
	if opts.DeleteBranch {
		payload["close_source_branch"] = true
	}

	_, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", owner, repo, number), payload)
	return err
}

// ClosePullRequest closes a pull request
func (p *BitbucketProvider) ClosePullRequest(ctx context.Context, owner, repo string, number int) error {
	_, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline", owner, repo, number), nil)
	return err
}

// ListPullRequests lists pull requests
func (p *BitbucketProvider) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestOpts) ([]*PullRequest, error) {
	query := url.Values{}
	if opts.State != "" && opts.State != "all" {
		query.Set("state", strings.ToUpper(opts.State))
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", owner, repo)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	var response struct {
		Values []bitbucketPullRequest `json:"values"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*PullRequest, len(response.Values))
	for i := range response.Values {
		result[i] = p.convertPullRequest(&response.Values[i])
	}

	return result, nil
}

// GetFileContents gets file contents
func (p *BitbucketProvider) GetFileContents(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	apiPath := fmt.Sprintf("/repositories/%s/%s/src/%s/%s", owner, repo, ref, path)
	if ref == "" {
		apiPath = fmt.Sprintf("/repositories/%s/%s/src/main/%s", owner, repo, path)
	}

	resp, err := p.doRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return resp, nil
}

// CreateOrUpdateFile creates or updates a file
func (p *BitbucketProvider) CreateOrUpdateFile(ctx context.Context, owner, repo string, opts CreateFileOpts) (*FileCommit, error) {
	// Bitbucket uses multipart form for file uploads
	// Simplified implementation using the src endpoint
	apiPath := fmt.Sprintf("/repositories/%s/%s/src", owner, repo)

	payload := map[string]interface{}{
		opts.Path: base64.StdEncoding.EncodeToString(opts.Content),
		"message": opts.Message,
		"branch":  opts.Branch,
	}

	resp, err := p.doRequest(ctx, "POST", apiPath, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update file: %w", err)
	}

	// Bitbucket returns limited info
	_ = resp
	return &FileCommit{
		Path:    opts.Path,
		Message: opts.Message,
	}, nil
}

// DeleteFile deletes a file
func (p *BitbucketProvider) DeleteFile(ctx context.Context, owner, repo string, opts DeleteFileOpts) error {
	// Bitbucket doesn't have a direct delete endpoint
	// Would need to use commits API
	return fmt.Errorf("delete file not implemented for Bitbucket")
}

// GetCommit gets a commit
func (p *BitbucketProvider) GetCommit(ctx context.Context, owner, repo, sha string) (*Commit, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/commit/%s", owner, repo, sha), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	var commit bitbucketCommit
	if err := json.Unmarshal(resp, &commit); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertCommit(&commit), nil
}

// ListCommits lists commits
func (p *BitbucketProvider) ListCommits(ctx context.Context, owner, repo string, opts ListCommitsOpts) ([]*Commit, error) {
	query := url.Values{}
	if opts.Path != "" {
		query.Set("path", opts.Path)
	}

	path := fmt.Sprintf("/repositories/%s/%s/commits", owner, repo)
	if opts.SHA != "" {
		path = fmt.Sprintf("/repositories/%s/%s/commits/%s", owner, repo, opts.SHA)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	var response struct {
		Values []bitbucketCommit `json:"values"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Commit, len(response.Values))
	for i := range response.Values {
		result[i] = p.convertCommit(&response.Values[i])
	}

	return result, nil
}

// CompareCommits compares two commits
func (p *BitbucketProvider) CompareCommits(ctx context.Context, owner, repo, base, head string) (*CommitComparison, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/diff/%s..%s", owner, repo, base, head), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits: %w", err)
	}

	// Bitbucket returns raw diff, parse it
	_ = resp
	return &CommitComparison{
		TotalCommits: 0,
	}, nil
}

// GetBranch gets a branch
func (p *BitbucketProvider) GetBranch(ctx context.Context, owner, repo, branch string) (*Branch, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", owner, repo, branch), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	var b bitbucketBranch
	if err := json.Unmarshal(resp, &b); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertBranch(&b), nil
}

// CreateBranch creates a branch
func (p *BitbucketProvider) CreateBranch(ctx context.Context, owner, repo string, opts CreateBranchOpts) (*Branch, error) {
	payload := map[string]interface{}{
		"name": opts.Name,
		"target": map[string]string{
			"hash": opts.FromRef,
		},
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/refs/branches", owner, repo), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	var b bitbucketBranch
	if err := json.Unmarshal(resp, &b); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertBranch(&b), nil
}

// DeleteBranch deletes a branch
func (p *BitbucketProvider) DeleteBranch(ctx context.Context, owner, repo, branch string) error {
	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", owner, repo, branch), nil)
	return err
}

// ListBranches lists branches
func (p *BitbucketProvider) ListBranches(ctx context.Context, owner, repo string) ([]*Branch, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/refs/branches", owner, repo), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var response struct {
		Values []bitbucketBranch `json:"values"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Branch, len(response.Values))
	for i := range response.Values {
		result[i] = p.convertBranch(&response.Values[i])
	}

	return result, nil
}

// CreateWebhook creates a webhook
func (p *BitbucketProvider) CreateWebhook(ctx context.Context, owner, repo string, opts CreateWebhookOpts) (*Webhook, error) {
	events := make([]string, len(opts.Events))
	for i, event := range opts.Events {
		switch event {
		case "push":
			events[i] = "repo:push"
		case "pull_request":
			events[i] = "pullrequest:created"
		default:
			events[i] = event
		}
	}

	payload := map[string]interface{}{
		"description": "PlatformFoundry webhook",
		"url":         opts.URL,
		"active":      opts.Active,
		"events":      events,
	}
	if opts.Secret != "" {
		payload["secret"] = opts.Secret
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/hooks", owner, repo), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	var hook bitbucketWebhook
	if err := json.Unmarshal(resp, &hook); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertWebhook(&hook), nil
}

// DeleteWebhook deletes a webhook
func (p *BitbucketProvider) DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error {
	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/repositories/%s/%s/hooks/%s", owner, repo, strconv.FormatInt(webhookID, 10)), nil)
	return err
}

// ListWebhooks lists webhooks
func (p *BitbucketProvider) ListWebhooks(ctx context.Context, owner, repo string) ([]*Webhook, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/hooks", owner, repo), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	var response struct {
		Values []bitbucketWebhook `json:"values"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Webhook, len(response.Values))
	for i := range response.Values {
		result[i] = p.convertWebhook(&response.Values[i])
	}

	return result, nil
}

// CreateComment creates a comment on a pull request
func (p *BitbucketProvider) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": body,
		},
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", owner, repo, number), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	var comment bitbucketComment
	if err := json.Unmarshal(resp, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertComment(&comment), nil
}

// ListComments lists comments on a pull request
func (p *BitbucketProvider) ListComments(ctx context.Context, owner, repo string, number int) ([]*Comment, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", owner, repo, number), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	var response struct {
		Values []bitbucketComment `json:"values"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Comment, len(response.Values))
	for i := range response.Values {
		result[i] = p.convertComment(&response.Values[i])
	}

	return result, nil
}

// CreateReview creates a review (approval) on a pull request
func (p *BitbucketProvider) CreateReview(ctx context.Context, owner, repo string, number int, opts CreateReviewOpts) (*Review, error) {
	if opts.Event == "APPROVE" {
		_, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", owner, repo, number), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to approve: %w", err)
		}

		return &Review{
			State:     "approved",
			CreatedAt: time.Now(),
		}, nil
	}

	// For other review types, add a comment
	comment, err := p.CreateComment(ctx, owner, repo, number, opts.Body)
	if err != nil {
		return nil, err
	}

	return &Review{
		ID:        comment.ID,
		State:     "commented",
		Body:      opts.Body,
		Author:    comment.Author,
		CreatedAt: comment.CreatedAt,
	}, nil
}

// ListReviews lists reviews on a pull request
func (p *BitbucketProvider) ListReviews(ctx context.Context, owner, repo string, number int) ([]*Review, error) {
	// Bitbucket doesn't have a direct reviews endpoint
	// Get PR details which include participants
	pr, err := p.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}

	// No direct approval list available from PR
	_ = pr
	return []*Review{}, nil
}

// HTTP helper

func (p *BitbucketProvider) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	// Basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(p.username + ":" + p.appPassword))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Bitbucket API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// Bitbucket API types

type bitbucketRepository struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
	MainBranch struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	IsPrivate bool      `json:"is_private"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type bitbucketPullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"destination"`
	Author struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	} `json:"author"`
	MergeCommit *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	ClosedBy *struct {
		DisplayName string `json:"display_name"`
	} `json:"closed_by"`
	CreatedOn time.Time  `json:"created_on"`
	UpdatedOn time.Time  `json:"updated_on"`
}

type bitbucketCommit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  struct {
		Raw  string `json:"raw"`
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
	} `json:"author"`
	Date    time.Time `json:"date"`
	Parents []struct {
		Hash string `json:"hash"`
	} `json:"parents"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

type bitbucketBranch struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

type bitbucketWebhook struct {
	UUID        string    `json:"uuid"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	Events      []string  `json:"events"`
	CreatedAt   time.Time `json:"created_at"`
}

type bitbucketComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	User struct {
		DisplayName string `json:"display_name"`
	} `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

// Conversion helpers

func (p *BitbucketProvider) convertRepository(r *bitbucketRepository) *Repository {
	var cloneURL, sshURL string
	for _, link := range r.Links.Clone {
		if link.Name == "https" {
			cloneURL = link.Href
		}
		if link.Name == "ssh" {
			sshURL = link.Href
		}
	}

	return &Repository{
		Name:          r.Name,
		FullName:      r.FullName,
		Description:   r.Description,
		URL:           r.Links.HTML.Href,
		CloneURL:      cloneURL,
		SSHURL:        sshURL,
		DefaultBranch: r.MainBranch.Name,
		Private:       r.IsPrivate,
		CreatedAt:     r.CreatedOn,
		UpdatedAt:     r.UpdatedOn,
	}
}

func (p *BitbucketProvider) convertPullRequest(pr *bitbucketPullRequest) *PullRequest {
	state := strings.ToLower(pr.State)
	if state == "merged" {
		state = "merged"
	} else if state == "declined" || state == "superseded" {
		state = "closed"
	} else {
		state = "open"
	}

	result := &PullRequest{
		ID:           int64(pr.ID),
		Number:       pr.ID,
		Title:        pr.Title,
		Body:         pr.Description,
		State:        state,
		URL:          pr.Links.HTML.Href,
		SourceBranch: pr.Source.Branch.Name,
		TargetBranch: pr.Destination.Branch.Name,
		Author:       pr.Author.Nickname,
		HeadSHA:      pr.Source.Commit.Hash,
		BaseSHA:      pr.Destination.Commit.Hash,
		CreatedAt:    pr.CreatedOn,
		UpdatedAt:    pr.UpdatedOn,
	}

	if pr.ClosedBy != nil {
		result.MergedBy = pr.ClosedBy.DisplayName
	}

	return result
}

func (p *BitbucketProvider) convertCommit(c *bitbucketCommit) *Commit {
	parents := make([]string, len(c.Parents))
	for i, parent := range c.Parents {
		parents[i] = parent.Hash
	}

	return &Commit{
		SHA:       c.Hash,
		Message:   c.Message,
		Author:    c.Author.User.DisplayName,
		URL:       c.Links.HTML.Href,
		Timestamp: c.Date,
		Parents:   parents,
	}
}

func (p *BitbucketProvider) convertBranch(b *bitbucketBranch) *Branch {
	return &Branch{
		Name: b.Name,
		SHA:  b.Target.Hash,
	}
}

func (p *BitbucketProvider) convertWebhook(h *bitbucketWebhook) *Webhook {
	// Parse UUID to int64 (simplified)
	var id int64
	if h.UUID != "" {
		// Use hash of UUID as ID
		for _, c := range h.UUID {
			id = id*31 + int64(c)
		}
		if id < 0 {
			id = -id
		}
	}

	return &Webhook{
		ID:        id,
		URL:       h.URL,
		Events:    h.Events,
		Active:    h.Active,
		CreatedAt: h.CreatedAt,
	}
}

func (p *BitbucketProvider) convertComment(c *bitbucketComment) *Comment {
	return &Comment{
		ID:        int64(c.ID),
		Body:      c.Content.Raw,
		Author:    c.User.DisplayName,
		CreatedAt: c.CreatedOn,
		UpdatedAt: c.UpdatedOn,
	}
}

func init() {
	// Register provider
	RegisterProvider(&BitbucketProvider{})
}
