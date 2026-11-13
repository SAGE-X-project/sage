//go:build darwin && registry_native_limits

package registry011

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry011/internal/compilermemory"
)

func nativeHistoryReadValue(t *testing.T, raw string, output reflect.Value, depth int) error {
	t.Helper()
	action, cancel := context.WithTimeout(context.Background(), time.Minute)
	cleanup, stopCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	defer stopCleanup()
	limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 64, MaximumNodes: len(raw)}
	capacity, err := compilermemory.JSONValueCapacity(len(raw), limits)
	if err != nil {
		return err
	}
	owner, err := compilermemory.Acquire(action, cleanup, capacity)
	if err != nil {
		if owner != nil {
			_ = owner.Close()
		}
		t.Fatal(err)
	}
	defer func() {
		if err := owner.Close(); err != nil {
			t.Error(err)
		}
	}()
	span, err := owner.Store([]byte(raw))
	if err != nil {
		return err
	}
	document, err := owner.ParseJSONAtValueDepth(span, limits)
	if err != nil {
		return err
	}
	return decodeNativeHistoryValue(document.Root(), output, depth)
}

func TestProviderHistoryPreparationNativeTypedValuesRemainExact(t *testing.T) {
	for _, tc := range []struct {
		name   string
		raw    string
		target any
		want   any
		valid  bool
	}{
		{"int8-maximum", "127", new(int8), int8(127), true},
		{"int8-overflow", "128", new(int8), nil, false},
		{"int64-minimum", "-9223372036854775808", new(int64), int64(math.MinInt64), true},
		{"int64-overflow", "9223372036854775808", new(int64), nil, false},
		{"uint8-maximum", "255", new(uint8), uint8(255), true},
		{"uint8-overflow", "256", new(uint8), nil, false},
		{"uint64-maximum", "18446744073709551615", new(uint64), uint64(math.MaxUint64), true},
		{"uint64-overflow", "18446744073709551616", new(uint64), nil, false},
		{"exact-above-float", "9007199254740993", new(uint64), uint64(9007199254740993), true},
		{"negative-unsigned", "-1", new(uint64), nil, false},
		{"negative-zero", "-0", new(int), nil, false},
		{"exponent", "1e0", new(uint64), nil, false},
		{"fraction", "1.0", new(int), nil, false},
		{"leading-zero", "01", new(int), nil, false},
		{"null-int", "null", new(int), nil, false},
		{"boolean", "true", new(bool), true, true},
		{"wrong-boolean", "0", new(bool), nil, false},
		{"string", "\"<>&\\u0000\\ud83d\\ude00\"", new(string), "<>&\x00\U0001f600", true},
		{"wrong-string", "0", new(string), nil, false},
		{"array", "[1,65535]", new([2]uint16), [2]uint16{1, 65535}, true},
		{"array-short", "[1]", new([2]uint16), nil, false},
		{"array-long", "[1,2,3]", new([2]uint16), nil, false},
		{"array-wrong-kind", "{}", new([2]uint16), nil, false},
		{"struct", "{\"X\":1}", new(struct{ X int }), struct{ X int }{1}, true},
		{"struct-case", "{\"x\":1}", new(struct{ X int }), nil, false},
		{"struct-unknown", "{\"X\":1,\"extra\":2}", new(struct{ X int }), nil, false},
		{"struct-wrong-kind", "[]", new(struct{ X int }), nil, false},
		{"unsupported", "{}", new(map[string]int), nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := reflect.ValueOf(tc.target).Elem()
			err := nativeHistoryReadValue(t, tc.raw, output, 0)
			if (err == nil) != tc.valid {
				t.Fatalf("typed acceptance changed: %v", err)
			}
			if tc.valid && !reflect.DeepEqual(output.Interface(), tc.want) {
				t.Fatalf("typed result=%#v, expected=%#v", output.Interface(), tc.want)
			}
		})
	}
	for _, invalid := range []reflect.Value{{}, reflect.Zero(reflect.TypeFor[int]())} {
		if err := nativeHistoryReadValue(t, "0", invalid, 0); err == nil {
			t.Fatal("invalid destination accepted")
		}
	}
	value := 0
	if err := nativeHistoryReadValue(t, "0", reflect.ValueOf(&value).Elem(), 65); err == nil {
		t.Fatal("native DTO depth guard lost")
	}
}

func TestProviderHistoryPreparationNativeSchemaMetadataRemainsClosed(t *testing.T) {
	integer := reflect.TypeFor[int]()
	for _, field := range []reflect.StructField{
		{Name: "A", Type: integer, Anonymous: true},
	} {
		if _, _, err := nativeHistoryFieldName(field); err == nil {
			t.Fatal("anonymous field entered the exact schema")
		}
	}
	for _, field := range []reflect.StructField{
		{Name: "hidden", PkgPath: "fixture", Type: integer},
		{Name: "Ignored", Type: integer, Tag: reflect.StructTag("json:\"-\"")},
	} {
		if _, included, err := nativeHistoryFieldName(field); err != nil || included {
			t.Fatal("ignored metadata changed the required wire members")
		}
	}
	tag := reflect.StructTag("json:\"same\"")
	duplicate := reflect.StructOf([]reflect.StructField{{Name: "A", Type: integer, Tag: tag}, {Name: "B", Type: integer, Tag: tag}})
	if _, err := nativeHistorySchemaNodes(duplicate, 0); err == nil {
		t.Fatal("ambiguous reflected names accepted")
	}
	for _, schema := range []reflect.Type{
		reflect.TypeFor[[1]*int](), reflect.TypeFor[struct{ Unsupported map[string]int }](),
	} {
		if _, err := nativeHistorySchemaNodes(schema, 0); err == nil {
			t.Fatal("unsupported nested schema accepted")
		}
	}
}

type nativeHistoryChangedDeadline struct {
	context.Context
	deadline time.Time
}

func (c *nativeHistoryChangedDeadline) Deadline() (time.Time, bool) { return c.deadline, true }

func TestProviderHistoryPreparationNativeOriginalCleanupCannotBeExtended(t *testing.T) {
	raw, _, _ := nativeHistoryTestRaw(t)
	origin := time.Now()
	action, cancel := context.WithDeadline(context.Background(), origin.Add(time.Minute))
	defer cancel()
	baseCleanup, stopCleanup := context.WithDeadline(context.Background(), origin.Add(2*time.Minute))
	defer stopCleanup()
	cleanup := &nativeHistoryChangedDeadline{Context: baseCleanup, deadline: origin.Add(2 * time.Minute)}
	session, err := NewNativeHistoryPreparationSession(action, cleanup, origin, len(raw), 4096)
	if err != nil {
		t.Fatal(err)
	}
	cleanup.deadline = cleanup.deadline.Add(time.Minute)
	if err := session.CompileOwned(raw); !errors.Is(err, ErrHistoryPreparationCleanup) {
		t.Fatalf("changed original cleanup accepted: %v", err)
	}
	state := session.native.(*nativeHistoryPreparation)
	if state.root.region != nil || state.policy.region != nil || state.result.region != nil {
		t.Fatal("changed cleanup acquired new storage")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if bad, err := newNativeHistoryPreparationSession(action, baseCleanup, origin, 1, 1, nil); bad != nil || err == nil {
		t.Fatal("missing fixed acquisition implementation accepted")
	}
	if bad, err := NewNativeHistoryPreparationSession(action, baseCleanup, origin, 0, 1); bad != nil || err == nil {
		t.Fatal("native constructor ignored existing transport bounds")
	}
}

type nativeHistoryReadContext struct {
	context.Context
	mu     sync.Mutex
	calls  int
	failAt int
}

func (c *nativeHistoryReadContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.failAt != 0 && c.calls >= c.failAt {
		return context.Canceled
	}
	return c.Context.Err()
}

func TestProviderHistoryPreparationNativeStringAdapterDiscardsCanceledValues(t *testing.T) {
	for checks := 1; checks <= 40; checks++ {
		t.Run(strconv.Itoa(checks), func(t *testing.T) {
			action, cancel := context.WithTimeout(context.Background(), time.Minute)
			cleanup, stopCleanup := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			defer stopCleanup()
			ctx := &nativeHistoryReadContext{Context: action}
			raw := []byte("\"copy\"")
			limits := compilermemory.JSONValueDepthLimits{MaximumValueDepth: 0, MaximumNodes: 1}
			capacity, err := compilermemory.JSONValueCapacity(len(raw), limits)
			if err != nil {
				t.Fatal(err)
			}
			owner, err := compilermemory.Acquire(ctx, cleanup, capacity)
			if owner != nil {
				defer func() {
					if err := owner.Close(); err != nil {
						t.Error(err)
					}
				}()
			}
			if err != nil {
				t.Fatal(err)
			}
			span, err := owner.Store(raw)
			if err != nil {
				t.Fatal(err)
			}
			document, err := owner.ParseJSONAtValueDepth(span, limits)
			if err != nil {
				t.Fatal(err)
			}
			ctx.mu.Lock()
			ctx.failAt = ctx.calls + checks
			ctx.mu.Unlock()
			output := ""
			err = decodeNativeHistoryValue(document.Root(), reflect.ValueOf(&output).Elem(), 0)
			if err != nil {
				if !errors.Is(err, context.Canceled) || output != "" {
					t.Fatalf("canceled materialization leaked a value: %q, %v", output, err)
				}
			} else if output != "copy" {
				t.Fatalf("materialized value=%q", output)
			}
		})
	}
}
