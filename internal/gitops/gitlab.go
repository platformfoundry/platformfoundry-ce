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

// GitLabProvider implements Provider for GitLab
type GitLabProvider struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// GitLabConfig contains GitLab configuration
type GitLabConfig struct {
	Token   string
	BaseURL string // defaults to https://gitlab.com/api/v4
}

// NewGitLabProvider creates a new GitLab provider
func NewGitLabProvider(cfg GitLabConfig) (*GitLabProvider, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gitlab.com/api/v4"
	}

	return &GitLabProvider{
		token:   cfg.Token,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the provider name
func (p *GitLabProvider) Name() string {
	return "gitlab"
}

// CreateRepository creates a new repository
func (p *GitLabProvider) CreateRepository(ctx context.Context, opts CreateRepoOpts) (*Repository, error) {
	payload := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Description,
		"visibility":  "private",
	}
	if !opts.Private {
		payload["visibility"] = "public"
	}
	if opts.AutoInit {
		payload["initialize_with_readme"] = true
	}
	if opts.DefaultBranch != "" {
		payload["default_branch"] = opts.DefaultBranch
	}

	resp, err := p.doRequest(ctx, "POST", "/projects", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	var project gitlabProject
	if err := json.Unmarshal(resp, &project); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertProject(&project), nil
}

// GetRepository gets a repository
func (p *GitLabProvider) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s", projectPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var project gitlabProject
	if err := json.Unmarshal(resp, &project); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertProject(&project), nil
}

// DeleteRepository deletes a repository
func (p *GitLabProvider) DeleteRepository(ctx context.Context, owner, repo string) error {
	projectPath := url.PathEscape(owner + "/" + repo)
	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s", projectPath), nil)
	return err
}

// GetPullRequest gets a merge request
func (p *GitLabProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/merge_requests/%d", projectPath, number), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	var mr gitlabMergeRequest
	if err := json.Unmarshal(resp, &mr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertMergeRequest(&mr), nil
}

// CreatePullRequest creates a merge request
func (p *GitLabProvider) CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePullRequestOpts) (*PullRequest, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := map[string]interface{}{
		"source_branch": opts.SourceBranch,
		"target_branch": opts.TargetBranch,
		"title":         opts.Title,
	}
	if opts.Body != "" {
		payload["description"] = opts.Body
	}
	if len(opts.Labels) > 0 {
		payload["labels"] = strings.Join(opts.Labels, ",")
	}
	if len(opts.Reviewers) > 0 {
		// GitLab uses reviewer_ids, would need to resolve usernames to IDs
		// For now, skip reviewers
	}
	if len(opts.Assignees) > 0 {
		// GitLab uses assignee_ids
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/projects/%s/merge_requests", projectPath), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create merge request: %w", err)
	}

	var mr gitlabMergeRequest
	if err := json.Unmarshal(resp, &mr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertMergeRequest(&mr), nil
}

// UpdatePullRequest updates a merge request
func (p *GitLabProvider) UpdatePullRequest(ctx context.Context, owner, repo string, number int, opts UpdatePullRequestOpts) (*PullRequest, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := make(map[string]interface{})
	if opts.Title != nil {
		payload["title"] = *opts.Title
	}
	if opts.Body != nil {
		payload["description"] = *opts.Body
	}
	if opts.State != nil {
		// GitLab uses state_event: close, reopen
		if *opts.State == "closed" {
			payload["state_event"] = "close"
		} else if *opts.State == "open" {
			payload["state_event"] = "reopen"
		}
	}
	if len(opts.Labels) > 0 {
		payload["labels"] = strings.Join(opts.Labels, ",")
	}

	resp, err := p.doRequest(ctx, "PUT", fmt.Sprintf("/projects/%s/merge_requests/%d", projectPath, number), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to update merge request: %w", err)
	}

	var mr gitlabMergeRequest
	if err := json.Unmarshal(resp, &mr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertMergeRequest(&mr), nil
}

// MergePullRequest merges a merge request
func (p *GitLabProvider) MergePullRequest(ctx context.Context, owner, repo string, number int, opts MergeOpts) error {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := make(map[string]interface{})
	if opts.Method == "squash" {
		payload["squash"] = true
	}
	if opts.CommitMessage != "" {
		payload["merge_commit_message"] = opts.CommitMessage
	}
	if opts.SHA != "" {
		payload["sha"] = opts.SHA
	}
	if opts.DeleteBranch {
		payload["should_remove_source_branch"] = true
	}

	_, err := p.doRequest(ctx, "PUT", fmt.Sprintf("/projects/%s/merge_requests/%d/merge", projectPath, number), payload)
	return err
}

// ClosePullRequest closes a merge request
func (p *GitLabProvider) ClosePullRequest(ctx context.Context, owner, repo string, number int) error {
	state := "closed"
	_, err := p.UpdatePullRequest(ctx, owner, repo, number, UpdatePullRequestOpts{State: &state})
	return err
}

// ListPullRequests lists merge requests
func (p *GitLabProvider) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestOpts) ([]*PullRequest, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	query := url.Values{}
	if opts.State != "" && opts.State != "all" {
		query.Set("state", opts.State)
	}
	if opts.Sort != "" {
		query.Set("order_by", opts.Sort)
	}
	if opts.Order != "" {
		query.Set("sort", opts.Order)
	}
	if opts.Limit > 0 {
		query.Set("per_page", strconv.Itoa(opts.Limit))
	}
	if opts.Page > 0 {
		query.Set("page", strconv.Itoa(opts.Page))
	}

	path := fmt.Sprintf("/projects/%s/merge_requests", projectPath)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	var mrs []gitlabMergeRequest
	if err := json.Unmarshal(resp, &mrs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*PullRequest, len(mrs))
	for i := range mrs {
		result[i] = p.convertMergeRequest(&mrs[i])
	}

	return result, nil
}

// GetFileContents gets file contents
func (p *GitLabProvider) GetFileContents(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	filePath := url.PathEscape(path)

	query := url.Values{}
	if ref != "" {
		query.Set("ref", ref)
	}

	apiPath := fmt.Sprintf("/projects/%s/repository/files/%s", projectPath, filePath)
	if len(query) > 0 {
		apiPath += "?" + query.Encode()
	}

	resp, err := p.doRequest(ctx, "GET", apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(resp, &file); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if file.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(file.Content)
	}

	return []byte(file.Content), nil
}

// CreateOrUpdateFile creates or updates a file
func (p *GitLabProvider) CreateOrUpdateFile(ctx context.Context, owner, repo string, opts CreateFileOpts) (*FileCommit, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	filePath := url.PathEscape(opts.Path)

	payload := map[string]interface{}{
		"branch":         opts.Branch,
		"content":        base64.StdEncoding.EncodeToString(opts.Content),
		"commit_message": opts.Message,
		"encoding":       "base64",
	}

	method := "POST"
	action := "create"
	if opts.SHA != "" {
		method = "PUT"
		action = "update"
	}

	resp, err := p.doRequest(ctx, method, fmt.Sprintf("/projects/%s/repository/files/%s", projectPath, filePath), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to %s file: %w", action, err)
	}

	var result struct {
		FilePath string `json:"file_path"`
		Branch   string `json:"branch"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &FileCommit{
		Path:    result.FilePath,
		Message: opts.Message,
	}, nil
}

// DeleteFile deletes a file
func (p *GitLabProvider) DeleteFile(ctx context.Context, owner, repo string, opts DeleteFileOpts) error {
	projectPath := url.PathEscape(owner + "/" + repo)
	filePath := url.PathEscape(opts.Path)

	payload := map[string]interface{}{
		"branch":         opts.Branch,
		"commit_message": opts.Message,
	}

	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s/repository/files/%s", projectPath, filePath), payload)
	return err
}

// GetCommit gets a commit
func (p *GitLabProvider) GetCommit(ctx context.Context, owner, repo, sha string) (*Commit, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/commits/%s", projectPath, sha), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	var commit gitlabCommit
	if err := json.Unmarshal(resp, &commit); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertCommit(&commit), nil
}

// ListCommits lists commits
func (p *GitLabProvider) ListCommits(ctx context.Context, owner, repo string, opts ListCommitsOpts) ([]*Commit, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	query := url.Values{}
	if opts.SHA != "" {
		query.Set("ref_name", opts.SHA)
	}
	if opts.Path != "" {
		query.Set("path", opts.Path)
	}
	if opts.Since != nil {
		query.Set("since", opts.Since.Format(time.RFC3339))
	}
	if opts.Until != nil {
		query.Set("until", opts.Until.Format(time.RFC3339))
	}
	if opts.Limit > 0 {
		query.Set("per_page", strconv.Itoa(opts.Limit))
	}

	path := fmt.Sprintf("/projects/%s/repository/commits", projectPath)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	var commits []gitlabCommit
	if err := json.Unmarshal(resp, &commits); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Commit, len(commits))
	for i := range commits {
		result[i] = p.convertCommit(&commits[i])
	}

	return result, nil
}

// CompareCommits compares two commits
func (p *GitLabProvider) CompareCommits(ctx context.Context, owner, repo, base, head string) (*CommitComparison, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/compare?from=%s&to=%s", projectPath, base, head), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to compare commits: %w", err)
	}

	var compare struct {
		Commits []gitlabCommit `json:"commits"`
		Diffs   []struct {
			NewPath string `json:"new_path"`
			OldPath string `json:"old_path"`
			Diff    string `json:"diff"`
		} `json:"diffs"`
	}
	if err := json.Unmarshal(resp, &compare); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	commits := make([]*Commit, len(compare.Commits))
	for i := range compare.Commits {
		commits[i] = p.convertCommit(&compare.Commits[i])
	}

	files := make([]*FileDiff, len(compare.Diffs))
	for i, d := range compare.Diffs {
		files[i] = &FileDiff{
			Filename: d.NewPath,
			Patch:    d.Diff,
		}
	}

	return &CommitComparison{
		Commits:      commits,
		Files:        files,
		TotalCommits: len(commits),
	}, nil
}

// GetBranch gets a branch
func (p *GitLabProvider) GetBranch(ctx context.Context, owner, repo, branch string) (*Branch, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	branchName := url.PathEscape(branch)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/branches/%s", projectPath, branchName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	var b gitlabBranch
	if err := json.Unmarshal(resp, &b); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertBranch(&b), nil
}

// CreateBranch creates a branch
func (p *GitLabProvider) CreateBranch(ctx context.Context, owner, repo string, opts CreateBranchOpts) (*Branch, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := map[string]interface{}{
		"branch": opts.Name,
		"ref":    opts.FromRef,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/projects/%s/repository/branches", projectPath), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	var b gitlabBranch
	if err := json.Unmarshal(resp, &b); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertBranch(&b), nil
}

// DeleteBranch deletes a branch
func (p *GitLabProvider) DeleteBranch(ctx context.Context, owner, repo, branch string) error {
	projectPath := url.PathEscape(owner + "/" + repo)
	branchName := url.PathEscape(branch)

	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s/repository/branches/%s", projectPath, branchName), nil)
	return err
}

// ListBranches lists branches
func (p *GitLabProvider) ListBranches(ctx context.Context, owner, repo string) ([]*Branch, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/branches", projectPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []gitlabBranch
	if err := json.Unmarshal(resp, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Branch, len(branches))
	for i := range branches {
		result[i] = p.convertBranch(&branches[i])
	}

	return result, nil
}

// CreateWebhook creates a webhook
func (p *GitLabProvider) CreateWebhook(ctx context.Context, owner, repo string, opts CreateWebhookOpts) (*Webhook, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := map[string]interface{}{
		"url": opts.URL,
	}
	if opts.Secret != "" {
		payload["token"] = opts.Secret
	}
	// Map events
	for _, event := range opts.Events {
		switch event {
		case "push":
			payload["push_events"] = true
		case "pull_request", "merge_request":
			payload["merge_requests_events"] = true
		case "issue":
			payload["issues_events"] = true
		}
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/projects/%s/hooks", projectPath), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	var hook gitlabWebhook
	if err := json.Unmarshal(resp, &hook); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertWebhook(&hook), nil
}

// DeleteWebhook deletes a webhook
func (p *GitLabProvider) DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error {
	projectPath := url.PathEscape(owner + "/" + repo)
	_, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s/hooks/%d", projectPath, webhookID), nil)
	return err
}

// ListWebhooks lists webhooks
func (p *GitLabProvider) ListWebhooks(ctx context.Context, owner, repo string) ([]*Webhook, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/hooks", projectPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	var hooks []gitlabWebhook
	if err := json.Unmarshal(resp, &hooks); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Webhook, len(hooks))
	for i := range hooks {
		result[i] = p.convertWebhook(&hooks[i])
	}

	return result, nil
}

// CreateComment creates a comment on a merge request
func (p *GitLabProvider) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	payload := map[string]interface{}{
		"body": body,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/projects/%s/merge_requests/%d/notes", projectPath, number), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	var note gitlabNote
	if err := json.Unmarshal(resp, &note); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.convertNote(&note), nil
}

// ListComments lists comments on a merge request
func (p *GitLabProvider) ListComments(ctx context.Context, owner, repo string, number int) ([]*Comment, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/merge_requests/%d/notes", projectPath, number), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	var notes []gitlabNote
	if err := json.Unmarshal(resp, &notes); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Comment, len(notes))
	for i := range notes {
		result[i] = p.convertNote(&notes[i])
	}

	return result, nil
}

// CreateReview creates a review (approval) on a merge request
func (p *GitLabProvider) CreateReview(ctx context.Context, owner, repo string, number int, opts CreateReviewOpts) (*Review, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	if opts.Event == "APPROVE" {
		_, err := p.doRequest(ctx, "POST", fmt.Sprintf("/projects/%s/merge_requests/%d/approve", projectPath, number), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to approve: %w", err)
		}

		return &Review{
			State:     "approved",
			CreatedAt: time.Now(),
		}, nil
	}

	// For other review types, just add a comment
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

// ListReviews lists reviews on a merge request
func (p *GitLabProvider) ListReviews(ctx context.Context, owner, repo string, number int) ([]*Review, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/merge_requests/%d/approvals", projectPath, number), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvals: %w", err)
	}

	var approvals struct {
		ApprovedBy []struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"approved_by"`
	}
	if err := json.Unmarshal(resp, &approvals); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]*Review, len(approvals.ApprovedBy))
	for i, a := range approvals.ApprovedBy {
		result[i] = &Review{
			State:  "approved",
			Author: a.User.Username,
		}
	}

	return result, nil
}

// HTTP helper

func (p *GitLabProvider) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
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

	req.Header.Set("PRIVATE-TOKEN", p.token)
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
		return nil, fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// GitLab API types

type gitlabProject struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	PathWithNamespace string    `json:"path_with_namespace"`
	Description       string    `json:"description"`
	WebURL            string    `json:"web_url"`
	HTTPURLToRepo     string    `json:"http_url_to_repo"`
	SSHURLToRepo      string    `json:"ssh_url_to_repo"`
	DefaultBranch     string    `json:"default_branch"`
	Visibility        string    `json:"visibility"`
	CreatedAt         time.Time `json:"created_at"`
	LastActivityAt    time.Time `json:"last_activity_at"`
}

type gitlabMergeRequest struct {
	ID           int64      `json:"id"`
	IID          int        `json:"iid"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	WebURL       string     `json:"web_url"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	Author       struct {
		Username string `json:"username"`
	} `json:"author"`
	Draft           bool       `json:"draft"`
	MergeStatus     string     `json:"merge_status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	MergedAt        *time.Time `json:"merged_at"`
	MergedBy        *struct {
		Username string `json:"username"`
	} `json:"merged_by"`
	SHA             string   `json:"sha"`
	Labels          []string `json:"labels"`
}

type gitlabCommit struct {
	ID             string    `json:"id"`
	ShortID        string    `json:"short_id"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	AuthorName     string    `json:"author_name"`
	CommitterName  string    `json:"committer_name"`
	WebURL         string    `json:"web_url"`
	CreatedAt      time.Time `json:"created_at"`
	ParentIDs      []string  `json:"parent_ids"`
}

type gitlabBranch struct {
	Name      string `json:"name"`
	Commit    struct {
		ID string `json:"id"`
	} `json:"commit"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

type gitlabWebhook struct {
	ID                    int64     `json:"id"`
	URL                   string    `json:"url"`
	PushEvents            bool      `json:"push_events"`
	MergeRequestsEvents   bool      `json:"merge_requests_events"`
	IssuesEvents          bool      `json:"issues_events"`
	CreatedAt             time.Time `json:"created_at"`
}

type gitlabNote struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    struct {
		Username string `json:"username"`
	} `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Conversion helpers

func (p *GitLabProvider) convertProject(proj *gitlabProject) *Repository {
	return &Repository{
		ID:            proj.ID,
		Name:          proj.Name,
		FullName:      proj.PathWithNamespace,
		Description:   proj.Description,
		URL:           proj.WebURL,
		CloneURL:      proj.HTTPURLToRepo,
		SSHURL:        proj.SSHURLToRepo,
		DefaultBranch: proj.DefaultBranch,
		Private:       proj.Visibility == "private",
		CreatedAt:     proj.CreatedAt,
		UpdatedAt:     proj.LastActivityAt,
	}
}

func (p *GitLabProvider) convertMergeRequest(mr *gitlabMergeRequest) *PullRequest {
	pr := &PullRequest{
		ID:           mr.ID,
		Number:       mr.IID,
		Title:        mr.Title,
		Body:         mr.Description,
		State:        mr.State,
		URL:          mr.WebURL,
		SourceBranch: mr.SourceBranch,
		TargetBranch: mr.TargetBranch,
		Author:       mr.Author.Username,
		Draft:        mr.Draft,
		Mergeable:    mr.MergeStatus == "can_be_merged",
		Labels:       mr.Labels,
		CreatedAt:    mr.CreatedAt,
		UpdatedAt:    mr.UpdatedAt,
		MergedAt:     mr.MergedAt,
		HeadSHA:      mr.SHA,
	}

	if mr.MergedBy != nil {
		pr.MergedBy = mr.MergedBy.Username
	}

	if mr.State == "merged" {
		pr.State = "merged"
	}

	return pr
}

func (p *GitLabProvider) convertCommit(c *gitlabCommit) *Commit {
	return &Commit{
		SHA:       c.ID,
		Message:   c.Message,
		Author:    c.AuthorName,
		Committer: c.CommitterName,
		URL:       c.WebURL,
		Timestamp: c.CreatedAt,
		Parents:   c.ParentIDs,
	}
}

func (p *GitLabProvider) convertBranch(b *gitlabBranch) *Branch {
	return &Branch{
		Name:      b.Name,
		SHA:       b.Commit.ID,
		Protected: b.Protected,
		Default:   b.Default,
	}
}

func (p *GitLabProvider) convertWebhook(h *gitlabWebhook) *Webhook {
	events := []string{}
	if h.PushEvents {
		events = append(events, "push")
	}
	if h.MergeRequestsEvents {
		events = append(events, "merge_request")
	}
	if h.IssuesEvents {
		events = append(events, "issue")
	}

	return &Webhook{
		ID:        h.ID,
		URL:       h.URL,
		Events:    events,
		Active:    true,
		CreatedAt: h.CreatedAt,
	}
}

func (p *GitLabProvider) convertNote(n *gitlabNote) *Comment {
	return &Comment{
		ID:        n.ID,
		Body:      n.Body,
		Author:    n.Author.Username,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}

func init() {
	// Register provider
	RegisterProvider(&GitLabProvider{})
}
