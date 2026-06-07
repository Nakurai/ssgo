package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"ssg.nakurai/internal/config"
	"ssg.nakurai/internal/host"
)

var hostForce bool

var hostCmd = &cobra.Command{
	Use:   "host [provider]",
	Short: "Configure a hosting provider (interactive)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHost,
}

func init() {
	hostCmd.Flags().BoolVar(&hostForce, "force", false, "Overwrite existing provider config files")
}

func runHost(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	codename := ""
	if len(args) == 1 {
		codename = args[0]
	}

	if codename == "" {
		codename, err = pickProvider()
		if err != nil {
			return err
		}
	}

	adapter, ok := host.Get(codename)
	if !ok {
		return host.ErrUnknown(codename)
	}

	if err := adapter.Setup(context.Background(), wd, hostForce); err != nil {
		return err
	}

	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}
	cfg.Host = codename
	if err := config.Save(wd, cfg); err != nil {
		return fmt.Errorf("save ssg.json: %w", err)
	}
	fmt.Printf("ssg.json updated: host = %q\n", codename)
	return nil
}

func pickProvider() (string, error) {
	supported := host.Supported()
	fmt.Println("Available hosting providers:")
	for i, s := range supported {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Print("Choose a provider: ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	choice := strings.TrimSpace(line)
	for _, s := range supported {
		if s == choice {
			return s, nil
		}
	}
	// Accept numeric choice too.
	for i, s := range supported {
		if choice == fmt.Sprintf("%d", i+1) {
			return s, nil
		}
	}
	return "", fmt.Errorf("unknown provider %q; supported: %v", choice, supported)
}
