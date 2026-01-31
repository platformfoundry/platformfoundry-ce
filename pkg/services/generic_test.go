package services

import (
	"context"
	"errors"
	"testing"
)

// Test service type
type testService struct {
	Name string
}

func TestGet(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	svc := &testService{Name: "test"}
	ref := ServiceRef{ID: "test-service"}

	container.RegisterSingleton(ref, svc)

	// Successful get
	result, err := Get[*testService](ctx, container, ref)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Expected name 'test', got %s", result.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := ServiceRef{ID: "nonexistent"}

	_, err := Get[*testService](ctx, container, ref)
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestGet_WrongType(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := ServiceRef{ID: "test-service"}
	container.RegisterSingleton(ref, "string value")

	_, err := Get[*testService](ctx, container, ref)
	if err == nil {
		t.Error("Expected error for wrong type")
	}
}

func TestMustGet(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	svc := &testService{Name: "test"}
	ref := ServiceRef{ID: "test-service"}
	container.RegisterSingleton(ref, svc)

	result := MustGet[*testService](ctx, container, ref)
	if result.Name != "test" {
		t.Errorf("Expected name 'test', got %s", result.Name)
	}
}

func TestMustGet_Panics(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()
	ref := ServiceRef{ID: "nonexistent"}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic")
		}
	}()

	MustGet[*testService](ctx, container, ref)
}

func TestGetOrDefault(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	defaultSvc := &testService{Name: "default"}
	ref := ServiceRef{ID: "nonexistent"}

	result := GetOrDefault[*testService](ctx, container, ref, defaultSvc)
	if result.Name != "default" {
		t.Errorf("Expected default service")
	}
}

func TestGetOrDefault_Found(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	svc := &testService{Name: "found"}
	defaultSvc := &testService{Name: "default"}
	ref := ServiceRef{ID: "test-service"}
	container.RegisterSingleton(ref, svc)

	result := GetOrDefault[*testService](ctx, container, ref, defaultSvc)
	if result.Name != "found" {
		t.Errorf("Expected found service, got %s", result.Name)
	}
}

func TestRegister(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := ServiceRef{ID: "test-service"}

	err := Register[*testService](container, ref, func(ctx context.Context, c Container) (*testService, error) {
		return &testService{Name: "created"}, nil
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	result, err := Get[*testService](ctx, container, ref)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Name != "created" {
		t.Errorf("Expected name 'created', got %s", result.Name)
	}
}

func TestRegister_FactoryError(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := ServiceRef{ID: "test-service"}

	err := Register[*testService](container, ref, func(ctx context.Context, c Container) (*testService, error) {
		return nil, errors.New("factory error")
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = Get[*testService](ctx, container, ref)
	if err == nil {
		t.Error("Expected error from factory")
	}
}

func TestRegisterSingleton(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	svc := &testService{Name: "singleton"}
	ref := ServiceRef{ID: "test-service"}

	err := RegisterSingleton[*testService](container, ref, svc)
	if err != nil {
		t.Fatalf("RegisterSingleton failed: %v", err)
	}

	result, err := Get[*testService](ctx, container, ref)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != svc {
		t.Error("Expected same instance")
	}
}

func TestTypedRef(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := NewTypedRef[*testService]("test-service")
	svc := &testService{Name: "typed"}
	container.RegisterSingleton(ref.ServiceRef, svc)

	result, err := ref.Get(ctx, container)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Name != "typed" {
		t.Errorf("Expected name 'typed', got %s", result.Name)
	}
}

func TestTypedRef_MustGet(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := NewTypedRef[*testService]("test-service")
	svc := &testService{Name: "typed"}
	container.RegisterSingleton(ref.ServiceRef, svc)

	result := ref.MustGet(ctx, container)
	if result.Name != "typed" {
		t.Errorf("Expected name 'typed', got %s", result.Name)
	}
}

func TestTypedRef_GetOrDefault(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	ref := NewTypedRef[*testService]("nonexistent")
	defaultSvc := &testService{Name: "default"}

	result := ref.GetOrDefault(ctx, container, defaultSvc)
	if result.Name != "default" {
		t.Errorf("Expected default service")
	}
}

func TestGetMultiple(t *testing.T) {
	container := NewContainer()
	ctx := context.Background()

	svc1 := &testService{Name: "one"}
	svc2 := &testService{Name: "two"}
	ref1 := ServiceRef{ID: "service-1"}
	ref2 := ServiceRef{ID: "service-2"}

	container.RegisterSingleton(ref1, svc1)
	container.RegisterSingleton(ref2, svc2)

	results := GetMultiple[*testService](ctx, container, ref1, ref2)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("Unexpected error: %v", results[0].Err)
	}
	if results[1].Err != nil {
		t.Errorf("Unexpected error: %v", results[1].Err)
	}
	if results[0].Value.Name != "one" {
		t.Errorf("Expected 'one', got %s", results[0].Value.Name)
	}
	if results[1].Value.Name != "two" {
		t.Errorf("Expected 'two', got %s", results[1].Value.Name)
	}
}

func TestMap(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	result := Map(input, func(i int) int { return i * 2 })

	expected := []int{2, 4, 6, 8, 10}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	result := Filter(input, func(i int) bool { return i%2 == 0 })

	expected := []int{2, 4}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %d at index %d, got %d", expected[i], i, v)
		}
	}
}

func TestFind(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}

	result, found := Find(input, func(i int) bool { return i == 3 })
	if !found {
		t.Error("Expected to find 3")
	}
	if result != 3 {
		t.Errorf("Expected 3, got %d", result)
	}

	_, found = Find(input, func(i int) bool { return i == 10 })
	if found {
		t.Error("Expected not to find 10")
	}
}

func TestContains(t *testing.T) {
	input := []string{"a", "b", "c"}

	if !Contains(input, "b") {
		t.Error("Expected to contain 'b'")
	}
	if Contains(input, "d") {
		t.Error("Expected not to contain 'd'")
	}
}

func TestKeys(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(input)

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check all keys are present
	for _, k := range []string{"a", "b", "c"} {
		if !Contains(keys, k) {
			t.Errorf("Expected key %s", k)
		}
	}
}

func TestValues(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2, "c": 3}
	values := Values(input)

	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	// Check all values are present
	for _, v := range []int{1, 2, 3} {
		if !Contains(values, v) {
			t.Errorf("Expected value %d", v)
		}
	}
}

func TestPtr(t *testing.T) {
	value := 42
	ptr := Ptr(value)

	if *ptr != 42 {
		t.Errorf("Expected 42, got %d", *ptr)
	}
}

func TestDeref(t *testing.T) {
	value := 42
	ptr := &value

	result := Deref(ptr)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	var nilPtr *int
	result = Deref(nilPtr)
	if result != 0 {
		t.Errorf("Expected 0 for nil, got %d", result)
	}
}

func TestDerefOr(t *testing.T) {
	value := 42
	ptr := &value

	result := DerefOr(ptr, 100)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	var nilPtr *int
	result = DerefOr(nilPtr, 100)
	if result != 100 {
		t.Errorf("Expected 100 for nil, got %d", result)
	}
}
