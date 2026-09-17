package faults

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestHeadersAfterAppliesSetAndRemove(t *testing.T) {
	var injections []Injection
	ctx := newFaultContext()
	ctx.Emit = func(injection Injection) {
		injections = append(injections, injection)
	}

	fault := newHeadersFault(config.HeadersConfig{
		Probability: 1,
		Set: map[string]string{
			"Cache-Control": "max-age=3600",
			"X-New-Header":  "custom-value",
		},
		Remove: []string{"Content-Type", "X-Old-Header"},
	})

	response := &http.Response{
		Header: http.Header{
			"Content-Type":  []string{"application/json"},
			"Cache-Control": []string{"no-cache"},
			"X-Old-Header":  []string{"deprecated"},
			"X-Preserved":   []string{"stay"},
		},
	}

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}

	if got := response.Header.Get("Cache-Control"); got != "max-age=3600" {
		t.Errorf("Cache-Control = %q, want max-age=3600", got)
	}
	if got := response.Header.Get("X-New-Header"); got != "custom-value" {
		t.Errorf("X-New-Header = %q, want custom-value", got)
	}
	if got := response.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want empty (removed)", got)
	}
	if got := response.Header.Get("X-Old-Header"); got != "" {
		t.Errorf("X-Old-Header = %q, want empty (removed)", got)
	}
	if got := response.Header.Get("X-Preserved"); got != "stay" {
		t.Errorf("X-Preserved = %q, want stay", got)
	}

	wantInjections := []Injection{{Fault: "headers", Detail: "set Cache-Control, set X-New-Header, removed Content-Type, removed X-Old-Header"}}
	if !reflect.DeepEqual(injections, wantInjections) {
		t.Fatalf("injections = %+v, want %+v", injections, wantInjections)
	}
}

func TestHeadersProbabilityZeroDoesNotTrigger(t *testing.T) {
	var injections []Injection
	ctx := newFaultContext()
	ctx.Emit = func(injection Injection) {
		injections = append(injections, injection)
	}

	fault := newHeadersFault(config.HeadersConfig{
		Probability: 0,
		Set:         map[string]string{"X-Test": "new"},
		Remove:      []string{"Content-Type"},
	})

	response := &http.Response{
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}

	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := response.Header.Get("X-Test"); got != "" {
		t.Errorf("X-Test = %q, want empty", got)
	}
	if len(injections) != 0 {
		t.Errorf("injections = %+v, want none", injections)
	}
}

func TestHeadersBeforeIsNoOp(t *testing.T) {
	fault := newHeadersFault(config.HeadersConfig{Probability: 1, Set: map[string]string{"X-Test": "val"}})
	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() = (%#v, %v), want (nil, nil)", shortCircuit, err)
	}
}
