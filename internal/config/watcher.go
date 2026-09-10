package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const reloadDebounce = 200 * time.Millisecond

// ApplyFunc atomically makes a validated configuration active. Implementations
// may apply command-line overrides to the candidate before returning.
type ApplyFunc func(*Config) error

// Watcher reloads one configuration file while preserving the last applied
// configuration when parsing, validation, or application fails.
type Watcher struct {
	path       string
	active     *Config
	apply      ApplyFunc
	logger     *log.Logger
	filesystem *fsnotify.Watcher
	closeOnce  sync.Once
	closeErr   error
}

// NewWatcher watches the configuration's parent directory. Directory watches
// survive the rename-and-create save strategy used by many editors.
func NewWatcher(path string, active *Config, apply ApplyFunc, logger *log.Logger) (*Watcher, error) {
	if path == "" {
		return nil, errors.New("config path must not be empty")
	}
	if active == nil {
		return nil, errors.New("active configuration must not be nil")
	}
	if apply == nil {
		return nil, errors.New("config apply function must not be nil")
	}
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	directory, err := filepath.EvalSymlinks(filepath.Dir(absolutePath))
	if err != nil {
		return nil, fmt.Errorf("resolve config directory: %w", err)
	}
	absolutePath = filepath.Join(directory, filepath.Base(absolutePath))

	filesystem, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create config watcher: %w", err)
	}
	if err := filesystem.Add(directory); err != nil {
		_ = filesystem.Close()
		return nil, fmt.Errorf("watch config directory %q: %w", directory, err)
	}

	return &Watcher{
		path:       absolutePath,
		active:     active,
		apply:      apply,
		logger:     logger,
		filesystem: filesystem,
	}, nil
}

// Run processes changes until ctx is canceled. Run must be called at most once.
func (watcher *Watcher) Run(ctx context.Context) error {
	defer watcher.Close()

	var timer *time.Timer
	var timerChannel <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watcher.filesystem.Events:
			if !open {
				return errors.New("config watcher event channel closed")
			}
			if !watcher.relevant(event) {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(reloadDebounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(reloadDebounce)
			}
			timerChannel = timer.C
		case <-timerChannel:
			timerChannel = nil
			watcher.reload()
		case err, open := <-watcher.filesystem.Errors:
			if !open {
				return errors.New("config watcher error channel closed")
			}
			watcher.logger.Printf("config watcher error: %v", err)
		}
	}
}

// Close releases operating-system watch resources. It is safe to call more
// than once.
func (watcher *Watcher) Close() error {
	watcher.closeOnce.Do(func() {
		watcher.closeErr = watcher.filesystem.Close()
	})
	return watcher.closeErr
}

func (watcher *Watcher) relevant(event fsnotify.Event) bool {
	if filepath.Clean(event.Name) != watcher.path {
		return false
	}
	return event.Has(fsnotify.Write) || event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove)
}

func (watcher *Watcher) reload() {
	candidate, err := Load(watcher.path)
	if err == nil {
		err = candidate.Validate()
	}
	if err == nil {
		err = watcher.apply(candidate)
	}
	if err != nil {
		watcher.logger.Printf("config reload rejected: %v", err)
		return
	}

	summary := summarizeChanges(watcher.active, candidate)
	watcher.active = candidate
	watcher.logger.Printf(
		"config reloaded: rules_added=%v rules_removed=%v rules_modified=%v target_changed=%t seed_changed=%t cors_changed=%t",
		summary.added,
		summary.removed,
		summary.modified,
		summary.targetChanged,
		summary.seedChanged,
		summary.corsChanged,
	)
}
