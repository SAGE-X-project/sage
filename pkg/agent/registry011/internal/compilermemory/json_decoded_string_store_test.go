package compilermemory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"
)

func decodedStoreRegion(t *testing.T, action context.Context) *Region {
	t.Helper()
	if action == nil {
		var cancel context.CancelFunc
		action, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		t.Cleanup(cancel)
	}
	return jsonTestRegion(t, action)
}

func TestRegionJSONDecodedStringStoreConsumesOwnedBytes(t *testing.T) {
	for index, raw := range []string{
		`""`, `"plain"`, `"<>&"`, `"\"\\\/\b\f\n\r\t"`,
		`"\u0061\u00af\uabcd"`, `"\ud83d\ude00"`, `"\ud800"`,
		`"\udfff"`, `"\ud800\u0061"`, `"\ud800\ud800"`,
		`"\u0000"`, `"\ufffd"`, `"한글😀"`,
	} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			var expected string
			if err := json.Unmarshal([]byte(raw), &expected); err != nil {
				t.Fatal(err)
			}
			source, document := jsonTestDocument(t, raw)
			destination := decodedStoreRegion(t, nil)
			prefix, err := destination.Store([]byte("prefix:"))
			if err != nil {
				t.Fatal(err)
			}
			span, err := document.Root().StoreDecodedString(destination)
			if err != nil || span.owner != destination || span.offset != 7 || span.length != len(expected) {
				t.Fatalf("stored span=%+v, error=%v", span, err)
			}
			if err := source.Close(); err != nil {
				t.Fatal(err)
			}
			suffix, err := destination.Store([]byte(":suffix"))
			if err != nil {
				t.Fatal(err)
			}
			if err := destination.Seal(); err != nil {
				t.Fatal(err)
			}
			for _, item := range []struct {
				span Span
				want string
			}{{prefix, "prefix:"}, {span, expected}, {suffix, ":suffix"}} {
				actual := make([]byte, len(item.want))
				if err := destination.Read(item.span, actual); err != nil || string(actual) != item.want {
					t.Fatalf("retained bytes=%q, error=%v; want=%q", actual, err, item.want)
				}
			}
		})
	}
}

func TestRegionJSONDecodedStringStoreRefusesBeforeSpending(t *testing.T) {
	_, document := jsonTestDocument(t, `"native"`)
	for _, name := range []string{"nil", "copied", "sealed", "closed", "capacity"} {
		t.Run(name, func(t *testing.T) {
			destination := decodedStoreRegion(t, nil)
			if _, err := destination.Store([]byte("prefix")); err != nil {
				t.Fatal(err)
			}
			target := destination
			switch name {
			case "nil":
				target = nil
			case "copied":
				target = &Region{self: destination}
			case "sealed":
				if err := destination.Seal(); err != nil {
					t.Fatal(err)
				}
			case "closed":
				if err := destination.Close(); err != nil {
					t.Fatal(err)
				}
			case "capacity":
				if _, err := destination.Store(make([]byte, destination.capacity-destination.used-5)); err != nil {
					t.Fatal(err)
				}
			}
			before := destination.used
			span, err := document.Root().StoreDecodedString(target)
			if err == nil || span != (Span{}) || destination.used != before {
				t.Fatalf("refusal span=%+v, error=%v, used=%d; before=%d", span, err, destination.used, before)
			}
			if name == "capacity" && !errors.Is(err, ErrRange) {
				t.Fatalf("capacity error=%v", err)
			}
		})
	}
	for _, raw := range []string{"0", "true", "null", "[]", "{}"} {
		t.Run(raw, func(t *testing.T) {
			_, document := jsonTestDocument(t, raw)
			destination := decodedStoreRegion(t, nil)
			span, err := document.Root().StoreDecodedString(destination)
			if !errors.Is(err, ErrJSONType) || span != (Span{}) || destination.used != 0 {
				t.Fatalf("wrong-kind span=%+v, error=%v, used=%d", span, err, destination.used)
			}
		})
	}
	source, document := jsonTestDocument(t, `"native"`)
	if span, err := document.Root().StoreDecodedString(source); !errors.Is(err, ErrConfiguration) || span != (Span{}) {
		t.Fatalf("same-owner span=%+v, error=%v", span, err)
	}
	destination := decodedStoreRegion(t, nil)
	if span, err := (JSONValue{}).StoreDecodedString(destination); !errors.Is(err, ErrConfiguration) || span != (Span{}) {
		t.Fatalf("zero-source span=%+v, error=%v", span, err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if span, err := document.Root().StoreDecodedString(destination); !errors.Is(err, ErrClosed) || span != (Span{}) || destination.used != 0 {
		t.Fatalf("closed-source span=%+v, error=%v", span, err)
	}
}

func TestRegionJSONDecodedStringStoreSpendsAndClearsOnCancellation(t *testing.T) {
	for _, side := range []string{"source", "destination"} {
		t.Run(side, func(t *testing.T) {
			var refusedBefore, refusedAfter, completed bool
			for checks := 1; checks <= 64; checks++ {
				t.Run(strconv.Itoa(checks), func(t *testing.T) {
					action, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					t.Cleanup(cancel)
					controlled := &integerReadContext{Context: action}
					var sourceAction, destinationAction context.Context = action, action
					if side == "source" {
						sourceAction = controlled
					} else {
						destinationAction = controlled
					}
					source := decodedStoreRegion(t, sourceAction)
					input, err := source.Store([]byte(`"copy"`))
					if err != nil {
						t.Fatal(err)
					}
					document, err := source.ParseJSON(input, JSONLimits{MaximumDepth: 2, MaximumNodes: 1})
					if err != nil {
						t.Fatal(err)
					}
					destination := decodedStoreRegion(t, destinationAction)
					if _, err := destination.Store([]byte("prefix")); err != nil {
						t.Fatal(err)
					}
					before := destination.used
					controlled.mu.Lock()
					controlled.failAt = controlled.calls + checks
					controlled.mu.Unlock()
					span, err := document.Root().StoreDecodedString(destination)
					if string(destination.data[:before]) != "prefix" {
						t.Fatal("copy changed previously committed bytes")
					}
					if err == nil {
						completed = true
						if span.owner != destination || span.offset != before || span.length != 4 || destination.used != before+4 || string(destination.data[before:before+4]) != "copy" {
							t.Fatalf("successful copy span=%+v, used=%d", span, destination.used)
						}
						return
					}
					if !errors.Is(err, context.Canceled) || span != (Span{}) {
						t.Fatalf("cancellation span=%+v, error=%v", span, err)
					}
					switch destination.used {
					case before:
						refusedBefore = true
					case before + 4:
						refusedAfter = true
						if !bytes.Equal(destination.data[before:before+4], []byte{0, 0, 0, 0}) {
							t.Fatal("failed entered copy retained partial bytes")
						}
					default:
						t.Fatalf("unexpected spent range: %d -> %d", before, destination.used)
					}
				})
			}
			if !refusedBefore || !refusedAfter || !completed {
				t.Fatalf("missing boundary: before=%v after=%v completed=%v", refusedBefore, refusedAfter, completed)
			}
		})
	}
}

func TestRegionJSONDecodedStringStoreOppositeDirectionsDoNotDeadlock(t *testing.T) {
	left, leftDocument := jsonTestDocument(t, `"left"`)
	right, rightDocument := jsonTestDocument(t, `"right"`)
	start := make(chan struct{})
	errorsOut := make(chan error, 2)
	go func() {
		<-start
		_, err := leftDocument.Root().StoreDecodedString(right)
		errorsOut <- err
	}()
	go func() {
		<-start
		_, err := rightDocument.Root().StoreDecodedString(left)
		errorsOut <- err
	}()
	close(start)
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for range 2 {
		select {
		case err := <-errorsOut:
			if !errors.Is(err, ErrConfiguration) {
				t.Fatalf("sealed destination error=%v", err)
			}
		case <-deadline.C:
			t.Fatal("opposite-direction copies did not finish")
		}
	}
}
