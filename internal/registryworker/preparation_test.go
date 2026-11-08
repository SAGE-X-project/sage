package registryworker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func preparationTestBinding() journalBinding {
	return journalBinding{strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)}
}

func preparationTestRecord() string {
	b := preparationTestBinding()
	return "sage-worker-preparation-v1 " + b.nonce + " " + b.input + " " + b.worker + " 1000000000 2000000000 3000000000 4000000000 512 1024\n"
}

func TestPreparationHeaderPreservesOriginalBoundaries(t *testing.T) {
	window := PreparationSnapshot{
		Origin: time.Unix(1, 0), ActionDeadline: time.Unix(2, 0),
		HardStopDeadline: time.Unix(3, 0), WholeDeadline: time.Unix(4, 0),
		MaximumInputBytes: 512, MaximumResultBytes: 1024,
	}
	if got := encodePreparationHeader(window, preparationTestBinding()); got != preparationTestRecord() {
		t.Fatalf("unexpected original-window record: %q", got)
	}
	got, err := parsePreparationHeader(preparationTestRecord(), preparationTestBinding())
	if err != nil || got != window {
		t.Fatalf("original window changed: got=%+v err=%v", got, err)
	}
}

func TestPreparationHeaderRejectsMalformedData(t *testing.T) {
	record := preparationTestRecord()
	cases := map[string]string{
		"wrong format": strings.Replace(record, preparationMagic, "other-preparation", 1),
		"wrong attempt": strings.Replace(record, strings.Repeat("a", 64), strings.Repeat("d", 64), 1),
		"wrong input": strings.Replace(record, strings.Repeat("b", 64), strings.Repeat("d", 64), 1),
		"wrong worker": strings.Replace(record, strings.Repeat("c", 64), strings.Repeat("d", 64), 1),
		"missing newline": strings.TrimSuffix(record, "\n"),
		"trailing record": record + "\n",
		"extra field": strings.TrimSuffix(record, "\n") + " 1\n",
		"missing field": strings.Replace(record, " 512 1024\n", " 512\n", 1),
		"zero origin": strings.Replace(record, "1000000000", "0", 1),
		"negative origin": strings.Replace(record, "1000000000", "-1", 1),
		"noncanonical origin": strings.Replace(record, "1000000000", "01000000000", 1),
		"signed origin": strings.Replace(record, "1000000000", "+1000000000", 1),
		"overflow": strings.Replace(record, "1000000000", "9223372036854775808", 1),
		"action before origin": strings.Replace(record, "2000000000", "999999999", 1),
		"hard stop before action": strings.Replace(record, "3000000000", "1999999999", 1),
		"whole ends at hard stop": strings.Replace(record, "4000000000", "3000000000", 1),
		"window too long": strings.Replace(record, "4000000000", "2678401000000001", 1),
		"zero input": strings.Replace(record, " 512 1024\n", " 0 1024\n", 1),
		"input overflow": strings.Replace(record, " 512 1024\n", " 67108865 1024\n", 1),
		"zero result": strings.Replace(record, " 512 1024\n", " 512 0\n", 1),
		"result overflow": strings.Replace(record, " 512 1024\n", " 512 67108865\n", 1),
		"oversize": strings.Repeat("x", preparationHeaderMaximum+1) + "\n",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parsePreparationHeader(raw, preparationTestBinding())
			if !errors.Is(err, ErrConfiguration) || got != (PreparationSnapshot{}) {
				t.Fatalf("malformed data accepted: %+v %v", got, err)
			}
		})
	}
	if _, err := parsePreparationHeader(record, journalBinding{}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("unbound record accepted: %v", err)
	}
}

func TestPreparationControlKeepsOriginalParentWindow(t *testing.T) {
	origin := time.Now()
	whole, cancelWhole := context.WithDeadline(t.Context(), origin.Add(time.Minute))
	action, cancelAction := context.WithDeadline(t.Context(), origin.Add(40*time.Second))
	hardStop, cancelHardStop := context.WithDeadline(t.Context(), origin.Add(50*time.Second))
	defer cancelWhole()
	defer cancelAction()
	defer cancelHardStop()
	o := &Owner{origin: origin, whole: whole, action: action, hardStop: hardStop,
		configuration: Configuration{Limits: Limits{InputBytes: 512, ResultBytes: 1024}}}
	o.self = o
	control, err := newPreparationControl(o, preparationTestBinding())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := control.close(); err != nil {
			t.Errorf("control cleanup: %v", err)
		}
	})
	reader := bufio.NewReader(control.read)
	raw, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := parsePreparationHeader(raw, preparationTestBinding())
	if err != nil || !snapshot.Origin.Equal(origin) || !snapshot.ActionDeadline.Equal(origin.Add(40*time.Second)) ||
		!snapshot.HardStopDeadline.Equal(origin.Add(50*time.Second)) || !snapshot.WholeDeadline.Equal(origin.Add(time.Minute)) {
		t.Fatalf("parent window replaced: %+v %v", snapshot, err)
	}
	cancelAction()
	end := make(chan error, 1)
	go func() {
		_, err := reader.ReadByte()
		end <- err
	}()
	wait, cancelWait := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancelWait()
	select {
	case err := <-end:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("control stream renewed or failed: %v", err)
		}
	case <-wait.Done():
		t.Fatal("original action cancellation did not close the control stream")
	}
}

func TestPreparationLifetimeCopiesAndJSONCannotRestoreControl(t *testing.T) {
	original := &PreparationLifetime{}
	original.self = original
	copyValue := reflect.New(reflect.TypeFor[PreparationLifetime]())
	copyValue.Elem().Set(reflect.ValueOf(original).Elem())
	copied := copyValue.Interface().(*PreparationLifetime)
	for _, l := range []*PreparationLifetime{nil, {}, copied} {
		if _, err := l.Snapshot(); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid snapshot accepted: %v", err)
		}
		if _, err := l.Context(); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid action accepted: %v", err)
		}
		if _, err := l.CleanupContext(); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid cleanup accepted: %v", err)
		}
		if _, err := l.ParentDone(); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid parent accepted: %v", err)
		}
		if err := l.Close(); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("invalid close accepted: %v", err)
		}
	}
	if err := json.Unmarshal([]byte(`{}`), new(PreparationLifetime)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("JSON restored a preparation lifetime: %v", err)
	}
}
