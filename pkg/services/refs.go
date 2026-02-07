// Package services provides a dependency injection container for Platform Foundry.
package services

import (
	"context"
	"net/http"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/contracts/pai"
	"github.com/platformfoundry/pf-ce/pkg/contracts/poi"
	"github.com/platformfoundry/pf-ce/pkg/contracts/pse"
	"github.com/platformfoundry/pf-ce/pkg/contracts/psi"
	"github.com/platformfoundry/pf-ce/pkg/sdk"
)

// Core service references that all plugins can depend on

// LoggerServiceRef is the reference to the logger service
var LoggerServiceRef = NewServiceRef[sdk.Logger]("core.logger")

// ConfigServiceRef is the reference to the config service
var ConfigServiceRef = NewServiceRef[Config]("core.config")

// StateServiceRef is the reference to the state backend service
var StateServiceRef = NewServiceRef[psi.StateBackend]("core.state")

// SecretsServiceRef is the reference to the secrets engine service
var SecretsServiceRef = NewServiceRef[pse.SecretsEngine]("core.secrets")

// AuthServiceRef is the reference to the auth service
var AuthServiceRef = NewServiceRef[pai.AuthMethod]("core.auth")

// MetricsServiceRef is the reference to the metrics service
var MetricsServiceRef = NewServiceRef[poi.MetricsCollector]("core.metrics")

// EventBusServiceRef is the reference to the event bus service
var EventBusServiceRef = NewServiceRef[EventBus]("core.eventbus")

// SchedulerServiceRef is the reference to the scheduler service
var SchedulerServiceRef = NewServiceRef[Scheduler]("core.scheduler")

// HTTPClientServiceRef is the reference to the HTTP client service
var HTTPClientServiceRef = NewServiceRef[HTTPClient]("core.http")

// CacheServiceRef is the reference to the cache service
var CacheServiceRef = NewServiceRef[Cache]("core.cache")

// Config provides configuration values
type Config interface {
	// Get returns a configuration value
	Get(key string) interface{}

	// GetString returns a string configuration value
	GetString(key string) string

	// GetInt returns an integer configuration value
	GetInt(key string) int

	// GetBool returns a boolean configuration value
	GetBool(key string) bool

	// GetDuration returns a duration configuration value
	GetDuration(key string) time.Duration

	// GetStringSlice returns a string slice configuration value
	GetStringSlice(key string) []string

	// GetStringMap returns a string map configuration value
	GetStringMap(key string) map[string]string

	// Set sets a configuration value
	Set(key string, value interface{})

	// Has checks if a key exists
	Has(key string) bool
}

// EventBus provides event publishing and subscribing
type EventBus interface {
	// Publish publishes an event
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes to events matching the filter
	Subscribe(ctx context.Context, filter EventFilter, handler EventHandler) (Subscription, error)
}

// Event represents an event in the system
type Event struct {
	// ID is the unique event identifier
	ID string

	// Type is the event type
	Type string

	// Source is where the event originated
	Source string

	// Time is when the event occurred
	Time time.Time

	// Data contains the event payload
	Data interface{}

	// Metadata contains event metadata
	Metadata map[string]string
}

// EventFilter filters events for subscriptions
type EventFilter struct {
	// Types filters by event type (empty means all)
	Types []string

	// Sources filters by event source (empty means all)
	Sources []string
}

// EventHandler handles events
type EventHandler func(ctx context.Context, event Event) error

// Subscription represents an event subscription
type Subscription interface {
	// Unsubscribe cancels the subscription
	Unsubscribe() error
}

// Scheduler provides task scheduling
type Scheduler interface {
	// Schedule schedules a task to run at a specific time
	Schedule(ctx context.Context, task ScheduledTask) (string, error)

	// ScheduleRecurring schedules a recurring task
	ScheduleRecurring(ctx context.Context, task RecurringTask) (string, error)

	// Cancel cancels a scheduled task
	Cancel(ctx context.Context, taskID string) error

	// List lists scheduled tasks
	List(ctx context.Context) ([]TaskInfo, error)
}

// ScheduledTask is a task scheduled to run once
type ScheduledTask struct {
	// Name is the task name
	Name string

	// RunAt is when to run the task
	RunAt time.Time

	// Handler is the task handler
	Handler func(ctx context.Context) error
}

// RecurringTask is a task that runs on a schedule
type RecurringTask struct {
	// Name is the task name
	Name string

	// Schedule is the cron schedule
	Schedule string

	// Handler is the task handler
	Handler func(ctx context.Context) error
}

// TaskInfo contains information about a scheduled task
type TaskInfo struct {
	// ID is the task identifier
	ID string

	// Name is the task name
	Name string

	// NextRun is when the task will next run
	NextRun time.Time

	// LastRun is when the task last ran
	LastRun *time.Time

	// Recurring indicates if this is a recurring task
	Recurring bool
}

// HTTPClient provides HTTP client functionality
type HTTPClient interface {
	// Do executes an HTTP request
	Do(req *http.Request) (*http.Response, error)

	// Get performs a GET request
	Get(ctx context.Context, url string) (*http.Response, error)

	// Post performs a POST request
	Post(ctx context.Context, url string, contentType string, body []byte) (*http.Response, error)
}

// Cache provides caching functionality
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) (interface{}, bool, error)

	// Set stores a value in the cache
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Clear clears all values from the cache
	Clear(ctx context.Context) error
}
