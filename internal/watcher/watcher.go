package watcher

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventKind classifies what triggered a rebuild.
type EventKind int

const (
	KindContent  EventKind = iota // a .md file changed
	KindTemplate                  // a template file changed
	KindConfig                    // ssgo.json changed
)

// Event is emitted by the watcher after debouncing.
type Event struct {
	Kind EventKind
	Path string
}

// Watcher wraps fsnotify with debouncing.
type Watcher struct {
	fsw    *fsnotify.Watcher
	events chan Event
	done   chan struct{}
}

func New() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fsw:    fsw,
		events: make(chan Event, 8),
		done:   make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

// AddDir recursively adds dir and all its subdirectories.
func (w *Watcher) AddDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if info.IsDir() {
			return w.fsw.Add(path)
		}
		return nil
	})
}

// AddFile watches a single file.
func (w *Watcher) AddFile(path string) error {
	return w.fsw.Add(path)
}

// Events returns the debounced event channel.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Close shuts down the watcher.
func (w *Watcher) Close() {
	close(w.done)
	w.fsw.Close()
}

func (w *Watcher) loop() {
	const debounce = 300 * time.Millisecond
	var timer *time.Timer
	var pending *Event

	flush := func() {
		if pending != nil {
			select {
			case w.events <- *pending:
			default:
			}
			pending = nil
		}
	}

	for {
		select {
		case <-w.done:
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// Auto-watch newly created directories.
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					if err := w.fsw.Add(ev.Name); err != nil {
						log.Printf("watcher: add %s: %v", ev.Name, err)
					}
				}
			}

			kind := classify(ev.Name)
			if pending == nil || kindPriority(kind) > kindPriority(pending.Kind) {
				pending = &Event{Kind: kind, Path: ev.Name}
			}

			if timer != nil {
				timer.Reset(debounce)
			} else {
				timer = time.AfterFunc(debounce, func() {
					flush()
					timer = nil
				})
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func classify(path string) EventKind {
	base := filepath.Base(path)
	switch {
	case base == "ssgo.json":
		return KindConfig
	case filepath.Ext(path) == ".html":
		return KindTemplate
	default:
		return KindContent
	}
}

// kindPriority ensures a full rebuild (Config/Template) overrides a partial one.
func kindPriority(k EventKind) int {
	switch k {
	case KindConfig:
		return 2
	case KindTemplate:
		return 1
	default:
		return 0
	}
}
