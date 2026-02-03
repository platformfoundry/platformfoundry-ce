package gitops

import (
	"context"
	"fmt"
	"time"
)

// Provider defines the interface for Git provider operations
type Provider interface {
	// Name returns the provider name
	Name() string

	// Repository operations
	CreateRepository(ctx context.Context, opts CreateRepoOpts) (*Repository, error)
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	DeleteRepository(ctx context.Context, owner, repo string) error

	// Pull Request operations
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	CreatePullRequest(ctx context.Context, owner, repo string, opts CreatePullRequestOpts) (*PullRequest, error)
	UpdatePullRequest(ctx context.Context, owner, repo string, number int, opts UpdatePullRequestOpts) (*PullRequest, error)
	MergePullRequest(ctx context.Context, owner, repo string, number int, opts MergeOpts) error
	ClosePullRequest(ctx context.Context, owner, repo string, number int) error
	ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestOpts) ([]*PullRequest, error)

	// File operations
	GetFileContents(ctx context.Context, owner, repo, path, ref string) ([]byte, error)
	CreateOrUpdateFile(ctx context.Context, owner, repo string, opts CreateFileOpts) (*FileCommit, error)
	DeleteFile(ctx context.Context, owner, repo string, opts DeleteFileOpts) error

	// Commit operations
	GetCommit(ctx context.Context, owner, repo, sha string) (*Commit, error)
	ListCommits(ctx context.Context, owner, repo string, opts ListCommitsOpts) ([]*Commit, error)
	CompareCommits(ctx context.Context, owner, repo, base, head string) (*CommitComparison, error)

	// Branch operations
	GetBranch(ctx context.Context, owner, repo, branch string) (*Branch, error)
	CreateBranch(ctx context.Context, owner, repo string, opts CreateBranchOpts) (*Branch, error)
	DeleteBranch(ctx context.Context, owner, repo, branch string) error
	ListBranches(ctx context.Context, owner, repo string) ([]*Branch, error)

	// Webhook operations
	CreateWebhook(ctx context.Context, owner, repo string, opts CreateWebhookOpts) (*Webhook, error)
	DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error
	ListWebhooks(ctx context.Context, owner, repo string) ([]*Webhook, error)

	// Comment operations
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error)
	ListComments(ctx context.Context, owner, repo string, number int) ([]*Comment, error)

	// Review operations
	CreateReview(ctx context.Context, owner, repo string, number int, opts CreateReviewOpts) (*Review, error)
	ListReviews(ctx context.Context, owner, repo string, number int) ([]*Review, error)
}

// Repository represents a Git repository
type Repository struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	FullName      string    `json:"fullName"`
	Description   string    `json:"description"`
	URL           string    `json:"url"`
	CloneURL      string    `json:"cloneUrl"`
	SSHURL        string    `json:"sshUrl"`
	DefaultBranch string    `json:"defaultBranch"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// PullRequest represents a pull request / merge request
type PullRequest struct {
	ID            int64      `json:"id"`
	Number        int        `json:"number"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	State         string     `json:"state"` // open, closed, merged
	URL           string     `json:"url"`
	SourceBranch  string     `json:"sourceBranch"`
	TargetBranch  string     `json:"targetBranch"`
	Author        string     `json:"author"`
	Assignees     []string   `json:"assignees,omitempty"`
	Reviewers     []string   `json:"reviewers,omitempty"`
	Labels        []string   `json:"labels,omitempty"`
	Draft         bool       `json:"draft"`
	Mergeable     bool       `json:"mergeable"`
	MergeableState string    `json:"mergeableState,omitempty"`
	Approvals     int        `json:"approvals"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	MergedAt      *time.Time `json:"mergedAt,omitempty"`
	MergedBy      string     `json:"mergedBy,omitempty"`
	HeadSHA       string     `json:"headSha"`
	BaseSHA       string     `json:"baseSha"`
}

// Commit represents a Git commit
type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Committer string    `json:"committer"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
	Parents   []string  `json:"parents,omitempty"`
}

// Branch represents a Git branch
type Branch struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

// Webhook represents a repository webhook
type Webhook struct {
	ID        int64    `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}

// Comment represents a PR/MR comment
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Review represents a PR review
type Review struct {
	ID        int64     `json:"id"`
	State     string    `json:"state"` // approved, changes_requested, commented, pending
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

// FileCommit represents a file commit result
type FileCommit struct {
	SHA     string `json:"sha"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// CommitComparison represents a comparison between two commits
type CommitComparison struct {
	BaseCommit   *Commit   `json:"baseCommit"`
	HeadCommit   *Commit   `json:"headCommit"`
	Commits      []*Commit `json:"commits"`
	Files        []*FileDiff `json:"files"`
	TotalCommits int       `json:"totalCommits"`
	AheadBy      int       `json:"aheadBy"`
	BehindBy     int       `json:"behindBy"`
}

// FileDiff represents a file diff in a comparison
type FileDiff struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // added, removed, modified, renamed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Patch     string `json:"patch,omitempty"`
}

// Option structs

// CreateRepoOpts contains options for creating a repository
type CreateRepoOpts struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private"`
	AutoInit      bool   `json:"autoInit"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
}

// CreatePullRequestOpts contains options for creating a pull request
type CreatePullRequestOpts struct {
	Title        string   `json:"title"`
	Body         string   `json:"body,omitempty"`
	SourceBranch string   `json:"sourceBranch"`
	TargetBranch string   `json:"targetBranch"`
	Draft        bool     `json:"draft,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Reviewers    []string `json:"reviewers,omitempty"`
	Assignees    []string `json:"assignees,omitempty"`
}

// UpdatePullRequestOpts contains options for updating a pull request
type UpdatePullRequestOpts struct {
	Title     *string  `json:"title,omitempty"`
	Body      *string  `json:"body,omitempty"`
	State     *string  `json:"state,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
}

// ListPullRequestOpts contains options for listing pull requests
type ListPullRequestOpts struct {
	State  string `json:"state,omitempty"` // open, closed, all
	Sort   string `json:"sort,omitempty"`  // created, updated
	Order  string `json:"order,omitempty"` // asc, desc
	Limit  int    `json:"limit,omitempty"`
	Page   int    `json:"page,omitempty"`
}

// MergeOpts contains options for merging a pull request
type MergeOpts struct {
	Method        string `json:"method,omitempty"` // merge, squash, rebase
	CommitTitle   string `json:"commitTitle,omitempty"`
	CommitMessage string `json:"commitMessage,omitempty"`
	SHA           string `json:"sha,omitempty"` // expected head SHA for safety
	DeleteBranch  bool   `json:"deleteBranch,omitempty"`
}

// CreateFileOpts contains options for creating/updating a file
type CreateFileOpts struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
	Message string `json:"message"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"` // required for updates
}

// DeleteFileOpts contains options for deleting a file
type DeleteFileOpts struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha"`
}

// ListCommitsOpts contains options for listing commits
type ListCommitsOpts struct {
	SHA    string     `json:"sha,omitempty"`
	Path   string     `json:"path,omitempty"`
	Since  *time.Time `json:"since,omitempty"`
	Until  *time.Time `json:"until,omitempty"`
	Limit  int        `json:"limit,omitempty"`
	Page   int        `json:"page,omitempty"`
}

// CreateBranchOpts contains options for creating a branch
type CreateBranchOpts struct {
	Name   string `json:"name"`
	FromRef string `json:"fromRef"` // SHA or branch name
}

// CreateWebhookOpts contains options for creating a webhook
type CreateWebhookOpts struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Secret      string   `json:"secret,omitempty"`
	ContentType string   `json:"contentType,omitempty"` // json, form
	Active      bool     `json:"active"`
	InsecureSSL bool     `json:"insecureSsl,omitempty"`
}

// CreateReviewOpts contains options for creating a review
type CreateReviewOpts struct {
	Body   string `json:"body,omitempty"`
	Event  string `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	CommitSHA string `json:"commitSha,omitempty"`
}

// ProviderRegistry manages available Git providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get returns a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry is the default provider registry
var DefaultRegistry = NewProviderRegistry()

// RegisterProvider registers a provider in the default registry
func RegisterProvider(provider Provider) {
	DefaultRegistry.Register(provider)
}

// GetProvider returns a provider from the default registry
func GetProvider(name string) (Provider, error) {
	return DefaultRegistry.Get(name)
}
