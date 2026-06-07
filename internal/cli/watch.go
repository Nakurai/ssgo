package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ssg.nakurai/internal/builder"
	"ssg.nakurai/internal/config"
	"ssg.nakurai/internal/server"
	"ssg.nakurai/internal/watcher"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for changes and serve the dev site on localhost:8088",
	RunE:  runWatch,
}

func runWatch(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	outDir := filepath.Join(wd, "build", "dev")
	buildOpts := func(c *config.Config) builder.Options {
		return builder.Options{
			ProjectDir: wd,
			OutputDir:  outDir,
			Profile:    c.Dev,
			IsDev:      true,
			Minify:     false,
			LiveReload: true,
		}
	}

	// Initial build
	fmt.Println("Building dev site…")
	if err := builder.Build(cfg, buildOpts(cfg)); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	// SSE hub for browser live reload
	hub := server.NewHub()

	// Start HTTP server in background
	go func() {
		if err := server.Serve(outDir, ":8088", hub); err != nil {
			log.Fatalf("dev server: %v", err)
		}
	}()

	// Set up file watcher
	w, err := watcher.New()
	if err != nil {
		return fmt.Errorf("watcher: %w", err)
	}
	defer w.Close()

	addWatchTargets(w, wd, cfg.Style)

	fmt.Printf("Watching… open http://localhost:8088 (Ctrl+C to stop)\n")

	for ev := range w.Events() {
		if ev.Kind == watcher.KindConfig {
			// Reload config on ssg.json change (colors, style switch, etc.)
			newCfg, err := config.Load(wd)
			if err != nil {
				log.Printf("reload config: %v", err)
			} else {
				// If style changed, re-register watch targets
				if newCfg.Style != cfg.Style {
					addWatchTargets(w, wd, newCfg.Style)
				}
				cfg = newCfg
			}
		}

		log.Printf("change: %s — rebuilding…", ev.Path)
		if err := builder.Build(cfg, buildOpts(cfg)); err != nil {
			log.Printf("rebuild error: %v", err)
		} else {
			hub.Broadcast()
		}
	}
	return nil
}

func addWatchTargets(w *watcher.Watcher, projectDir, style string) {
	targets := []string{
		filepath.Join(projectDir, "content"),
		filepath.Join(projectDir, "template", "style", style),
	}
	for _, t := range targets {
		if err := w.AddDir(t); err != nil {
			log.Printf("watcher: add dir %s: %v", t, err)
		}
	}
	// Watch ssg.json directly
	cfgFile := filepath.Join(projectDir, "ssg.json")
	if err := w.AddFile(cfgFile); err != nil {
		log.Printf("watcher: add %s: %v", cfgFile, err)
	}
}
