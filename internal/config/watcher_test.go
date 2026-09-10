package config_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestWatcherHandlesAtomicSaveAndRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, statusConfig(503, "old-rule"))
	initial := loadValidConfig(t, path)
	var logs synchronizedBuffer
	applied := make(chan *config.Config, 4)
	watcher, err := config.NewWatcher(path, initial, func(candidate *config.Config) error {
		applied <- candidate
		return nil
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewWatcher() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = watcher.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()

	atomicReplace(t, path, statusConfig(418, "new-rule"))
	got := waitForConfig(t, applied)
	if got.Rules[0].Status.Code != 418 {
		t.Fatalf("reloaded status = %d, want 418", got.Rules[0].Status.Code)
	}
	waitForText(t, &logs, "rules_added=[new-rule]")
	if !strings.Contains(logs.String(), "rules_removed=[old-rule]") {
		t.Fatalf("reload log = %q, want removed rule", logs.String())
	}

	writeFile(t, path, "target: [invalid\n")
	waitForText(t, &logs, "config reload rejected:")
	writeFile(t, path, strings.Replace(statusConfig(500, "invalid"), "probability: 1", "probability: 2", 1))
	waitForText(t, &logs, "probability must be between 0 and 1")
	select {
	case unexpected := <-applied:
		t.Fatalf("invalid configuration was applied: %+v", unexpected)
	case <-time.After(300 * time.Millisecond):
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() error after cancellation: %v", err)
	}
}

func TestWatcherDebouncesWriteBursts(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, statusConfig(500, "fault"))
	initial := loadValidConfig(t, path)
	var applyCount atomic.Int64
	applied := make(chan *config.Config, 4)
	watcher, err := config.NewWatcher(path, initial, func(candidate *config.Config) error {
		applyCount.Add(1)
		applied <- candidate
		return nil
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewWatcher() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = watcher.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()

	for _, code := range []int{501, 502, 503} {
		writeFile(t, path, statusConfig(code, "fault"))
		time.Sleep(25 * time.Millisecond)
	}
	got := waitForConfig(t, applied)
	if got.Rules[0].Status.Code != 503 {
		t.Fatalf("debounced status = %d, want final value 503", got.Rules[0].Status.Code)
	}
	time.Sleep(300 * time.Millisecond)
	if got := applyCount.Load(); got != 1 {
		t.Fatalf("apply count = %d, want one debounced reload", got)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() error after cancellation: %v", err)
	}
}

func TestNewWatcherRejectsInvalidDependencies(t *testing.T) {
	t.Parallel()

	valid := &config.Config{Target: "http://localhost"}
	logger := log.New(io.Discard, "", 0)
	apply := func(*config.Config) error { return nil }
	for name, create := range map[string]func() (*config.Watcher, error){
		"empty path": func() (*config.Watcher, error) { return config.NewWatcher("", valid, apply, logger) },
		"nil config": func() (*config.Watcher, error) { return config.NewWatcher("config.yaml", nil, apply, logger) },
		"nil apply":  func() (*config.Watcher, error) { return config.NewWatcher("config.yaml", valid, nil, logger) },
		"nil logger": func() (*config.Watcher, error) { return config.NewWatcher("config.yaml", valid, apply, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			if watcher, err := create(); err == nil {
				_ = watcher.Close()
				t.Fatal("NewWatcher() error = nil")
			}
		})
	}
}

func loadValidConfig(t *testing.T, path string) *config.Config {
	t.Helper()
	configured, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := configured.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
	return configured
}

func waitForConfig(t *testing.T, configs <-chan *config.Config) *config.Config {
	t.Helper()
	select {
	case configured := <-configs:
		return configured
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for configuration reload")
		return nil
	}
}

func waitForText(t *testing.T, output *synchronizedBuffer, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(output.String(), expected) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("output %q does not contain %q", output.String(), expected)
}

func atomicReplace(t *testing.T, path, contents string) {
	t.Helper()
	temporary := path + ".new"
	backup := path + ".old"
	writeFile(t, temporary, contents)
	if err := os.Rename(path, backup); err != nil {
		t.Fatalf("rename existing config: %v", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		t.Fatalf("rename replacement config: %v", err)
	}
	if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove backup config: %v", err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func statusConfig(code int, name string) string {
	return "target: http://localhost:9000\n" +
		"rules:\n" +
		"  - name: " + name + "\n" +
		"    match: GET /api\n" +
		"    status:\n" +
		"      code: " + fmt.Sprint(code) + "\n" +
		"      probability: 1\n"
}

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *synchronizedBuffer) Write(contents []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(contents)
}

func (buffer *synchronizedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}
