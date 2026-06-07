package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nakurai/ssgo/internal/builder"
	"github.com/nakurai/ssgo/internal/config"
	"github.com/nakurai/ssgo/internal/host"
)

var deployBuild bool

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy build/prod/ to the configured hosting provider",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().BoolVar(&deployBuild, "build", false, "Run ssgo generate before deploying")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	if cfg.Host == "" {
		fmt.Println("No host configured; run `ssgo host` first.")
		return nil
	}

	adapter, ok := host.Get(cfg.Host)
	if !ok {
		return host.ErrUnknown(cfg.Host)
	}

	outDir := filepath.Join(wd, "build", "prod")

	if deployBuild {
		fmt.Println("Building production site…")
		opts := builder.Options{
			ProjectDir: wd,
			OutputDir:  outDir,
			Profile:    cfg.Prod,
			IsDev:      false,
			Minify:     true,
			LiveReload: false,
		}
		if err := builder.Build(cfg, opts); err != nil {
			return fmt.Errorf("build: %w", err)
		}
		fmt.Printf("Build complete → %s\n", outDir)
	} else {
		if empty, err := isDirEmpty(outDir); err != nil || empty {
			return fmt.Errorf("build/prod/ is missing or empty; run `ssgo generate` first, or `ssgo deploy --build`")
		}
	}

	fmt.Printf("Deploying to %s…\n", cfg.Host)
	return adapter.Deploy(context.Background(), wd)
}

func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return true, err
	}
	defer f.Close()
	entries, err := f.Readdirnames(1)
	if err != nil {
		return true, err
	}
	return len(entries) == 0, nil
}
