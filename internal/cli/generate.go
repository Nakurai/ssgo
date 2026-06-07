package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nakurai/ssgo/internal/builder"
	"github.com/nakurai/ssgo/internal/config"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Build the production site into build/prod/",
	RunE:  runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	outDir := filepath.Join(wd, "build", "prod")
	opts := builder.Options{
		ProjectDir: wd,
		OutputDir:  outDir,
		Profile:    cfg.Prod,
		IsDev:      false,
		Minify:     true,
		LiveReload: false,
	}

	fmt.Println("Building production site…")
	if err := builder.Build(cfg, opts); err != nil {
		return err
	}
	fmt.Printf("Done → %s\n", outDir)
	return nil
}
