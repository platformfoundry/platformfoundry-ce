// Package services provides a dependency injection container for Platform Foundry.
package services

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/sdk"
)

// RegisterBuiltinServices registers the built-in core services
func RegisterBuiltinServices(container *DefaultContainer) error {
	// Register logger
	if err := container.Register(LoggerServiceRef, func(ctx context.Context, c Container) (interface{}, error) {
		return sdk.NewDefaultLogger(false), nil
	}); err != nil {
		return err
	}

	// Register config
	if err := container.Register(ConfigServiceRef, func(ctx context.Context, c Container) (interface{}, error) {
		return NewDefaultConfig(), nil
	}); err != nil {
		return err
	}

	// Register event bus
	if err := container.Register(EventBusServiceRef, func(ctx context.Context, c Container) (interface{}, error) {
		return NewDefaultEventBus(), nil
	}); err != nil {
		return err
	}

	// Register HTTP client
	if err := container.Register(HTTPClientServiceRef, func(ctx context.Context, c Container) (interface{}, error) {
		return NewDefaultHTTPClient(), nil
	}); err != nil {
		return err
	}

	// Register cache
	if err := container.Register(CacheServiceRef, func(ctx context.Context, c Container) (interface{}, error) {
		return NewDefaultCache(), nil
	}); err != nil {
		return err
	}

	return nil
}

// DefaultConfig is a simple in-memory configuration
type DefaultConfig struct {
	mu     sync.RWMutex
	values map[string]interface{}
}

// NewDefaultConfig creates a new default config
func NewDefaultConfig() *DefaultConfig {
	return &DefaultConfig{
		values: make(map[string]interface{}),
	}
}

func (c *DefaultConfig) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

func (c *DefaultConfig) GetString(key string) string {
	if v, ok := c.Get(key).(string); ok {
		return v
	}
	return ""
}

func (c *DefaultConfig) GetInt(key string) int {
	switch v := c.Get(key).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func (c *DefaultConfig) GetBool(key string) bool {
	if v, ok := c.Get(key).(bool); ok {
		return v
	}
	return false
}

func (c *DefaultConfig) GetDuration(key string) time.Duration {
	switch v := c.Get(key).(type) {
	case time.Duration:
		return v
	case string:
		d, _ := time.ParseDuration(v)
		return d
	case int64:
		return time.Duration(v)
	}
	return 0
}

func (c *DefaultConfig) GetStringSlice(key string) []string {
	if v, ok := c.Get(key).([]string); ok {
		return v
	}
	return nil
}

func (c *DefaultConfig) GetStringMap(key string) map[string]string {
	if v, ok := c.Get(key).(map[string]string); ok {
		return v
	}
	return nil
}

func (c *DefaultConfig) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

func (c *DefaultConfig) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.values[key]
	return ok
}

// DefaultEventBus is a simple in-memory event bus
type DefaultEventBus struct {
	mu           sync.RWMutex
	subscribers  []subscriber
	nextSubID    int
}

type subscriber struct {
	id      int
	filter  EventFilter
	handler EventHandler
}

// NewDefaultEventBus creates a new default event bus
func NewDefaultEventBus() *DefaultEventBus {
	return &DefaultEventBus{}
}

func (b *DefaultEventBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	subs := make([]subscriber, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subs {
		if b.matchesFilter(event, sub.filter) {
			if err := sub.handler(ctx, event); err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}

func (b *DefaultEventBus) Subscribe(ctx context.Context, filter EventFilter, handler EventHandler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextSubID++
	sub := subscriber{
		id:      b.nextSubID,
		filter:  filter,
		handler: handler,
	}
	b.subscribers = append(b.subscribers, sub)

	return &defaultSubscription{
		bus: b,
		id:  sub.id,
	}, nil
}

func (b *DefaultEventBus) matchesFilter(event Event, filter EventFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Sources) > 0 {
		found := false
		for _, s := range filter.Sources {
			if s == event.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (b *DefaultEventBus) unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		if sub.id == id {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			return
		}
	}
}

type defaultSubscription struct {
	bus *DefaultEventBus
	id  int
}

func (s *defaultSubscription) Unsubscribe() error {
	s.bus.unsubscribe(s.id)
	return nil
}

// DefaultHTTPClient wraps the standard HTTP client
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new default HTTP client
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *DefaultHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

func (c *DefaultHTTPClient) Post(ctx context.Context, url string, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.client.Do(req)
}

// DefaultCache is a simple in-memory cache
type DefaultCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewDefaultCache creates a new default cache
func NewDefaultCache() *DefaultCache {
	cache := &DefaultCache{
		items: make(map[string]cacheItem),
	}
	// Start cleanup goroutine
	go cache.cleanup()
	return cache
}

func (c *DefaultCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false, nil
	}

	if time.Now().After(item.expiresAt) {
		return nil, false, nil
	}

	return item.value, true, nil
}

func (c *DefaultCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *DefaultCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

func (c *DefaultCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
	return nil
}

func (c *DefaultCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
