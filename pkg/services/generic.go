// Package services provides generic type-safe helpers for the service container.
// These functions use Go 1.18+ generics to provide type-safe service retrieval.
package services

import (
	"context"
	"fmt"
	"reflect"
)

// Get retrieves a service from the container with type safety.
// This is a generic wrapper around Container.Get that performs type assertion.
//
// Example:
//
//	logger, err := services.Get[*log.Logger](ctx, container, LoggerServiceRef)
func Get[T any](ctx context.Context, container Container, ref ServiceRef) (T, error) {
	var zero T

	instance, err := container.Get(ctx, ref)
	if err != nil {
		return zero, err
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("service %s is not of expected type %T, got %T", ref.ID, zero, instance)
	}

	return typed, nil
}

// MustGet retrieves a service from the container with type safety, panicking on error.
// Use this only in initialization code where failures should be fatal.
//
// Example:
//
//	logger := services.MustGet[*log.Logger](ctx, container, LoggerServiceRef)
func MustGet[T any](ctx context.Context, container Container, ref ServiceRef) T {
	result, err := Get[T](ctx, container, ref)
	if err != nil {
		panic(err)
	}
	return result
}

// GetOrDefault retrieves a service from the container with type safety,
// returning the default value if the service is not found or type assertion fails.
//
// Example:
//
//	logger := services.GetOrDefault[*log.Logger](ctx, container, LoggerServiceRef, defaultLogger)
func GetOrDefault[T any](ctx context.Context, container Container, ref ServiceRef, defaultValue T) T {
	result, err := Get[T](ctx, container, ref)
	if err != nil {
		return defaultValue
	}
	return result
}

// Register is a generic helper for registering a typed service factory.
// The factory function returns the concrete type, which is automatically
// converted to interface{} for storage.
//
// Example:
//
//	err := services.Register[*log.Logger](container, LoggerServiceRef, func(ctx context.Context, c Container) (*log.Logger, error) {
//	    return log.Default(), nil
//	})
func Register[T any](container Container, ref ServiceRef, factory func(ctx context.Context, c Container) (T, error)) error {
	return container.Register(ref, func(ctx context.Context, c Container) (interface{}, error) {
		return factory(ctx, c)
	})
}

// RegisterSingleton is a generic helper for registering a typed singleton.
//
// Example:
//
//	err := services.RegisterSingleton[*log.Logger](container, LoggerServiceRef, logger)
func RegisterSingleton[T any](container Container, ref ServiceRef, instance T) error {
	return container.RegisterSingleton(ref, instance)
}

// TypedRef wraps a ServiceRef with type information for compile-time safety.
// This is useful when you want to associate a ref with its expected type.
type TypedRef[T any] struct {
	ServiceRef
}

// NewTypedRef creates a new typed service reference.
//
// Example:
//
//	var LoggerRef = services.NewTypedRef[*log.Logger]("logger")
func NewTypedRef[T any](id string) TypedRef[T] {
	var t T
	return TypedRef[T]{
		ServiceRef: ServiceRef{
			ID:   id,
			Type: reflect.TypeOf(&t).Elem(),
		},
	}
}

// Get retrieves the service using the typed reference.
func (r TypedRef[T]) Get(ctx context.Context, container Container) (T, error) {
	return Get[T](ctx, container, r.ServiceRef)
}

// MustGet retrieves the service using the typed reference, panicking on error.
func (r TypedRef[T]) MustGet(ctx context.Context, container Container) T {
	return MustGet[T](ctx, container, r.ServiceRef)
}

// GetOrDefault retrieves the service using the typed reference with a default.
func (r TypedRef[T]) GetOrDefault(ctx context.Context, container Container, defaultValue T) T {
	return GetOrDefault[T](ctx, container, r.ServiceRef, defaultValue)
}

// Result represents a result with a typed value and potential error.
// Useful for batch operations.
type Result[T any] struct {
	Value T
	Err   error
}

// GetMultiple retrieves multiple services of the same type.
//
// Example:
//
//	results := services.GetMultiple[Plugin](ctx, container, pluginRef1, pluginRef2)
func GetMultiple[T any](ctx context.Context, container Container, refs ...ServiceRef) []Result[T] {
	results := make([]Result[T], len(refs))
	for i, ref := range refs {
		value, err := Get[T](ctx, container, ref)
		results[i] = Result[T]{Value: value, Err: err}
	}
	return results
}

// Map applies a function to each element of a slice.
// Generic helper for common collection operations.
func Map[T, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

// Filter returns elements that satisfy the predicate.
func Filter[T any](items []T, fn func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

// Find returns the first element that satisfies the predicate.
func Find[T any](items []T, fn func(T) bool) (T, bool) {
	for _, item := range items {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Contains checks if an element exists in a slice.
func Contains[T comparable](items []T, target T) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// Keys returns all keys from a map.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values from a map.
func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// Ptr returns a pointer to the value.
func Ptr[T any](v T) *T {
	return &v
}

// Deref dereferences a pointer, returning the zero value if nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// DerefOr dereferences a pointer, returning the default if nil.
func DerefOr[T any](p *T, defaultValue T) T {
	if p == nil {
		return defaultValue
	}
	return *p
}
