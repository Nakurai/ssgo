package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"ssg.nakurai/internal/config"
	"ssg.nakurai/internal/template"
)

var styleCmd = &cobra.Command{
	Use:   "style",
	Short: "Manage template styles",
}

var (
	styleAddFrom    string
	styleAddName    string
	styleAddForce   bool
	styleSwitchName string
)

func init() {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a template style from a URL, .zip, or directory",
		RunE:  runStyleAdd,
	}
	addCmd.Flags().StringVar(&styleAddFrom, "from", "", "Template source: http(s) URL, local .zip, or directory (required)")
	addCmd.Flags().StringVar(&styleAddName, "name", "", "Name to install the style as (defaults to the source name)")
	addCmd.Flags().BoolVar(&styleAddForce, "force", false, "Replace an existing style without confirmation")
	_ = addCmd.MarkFlagRequired("from")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed template styles",
		RunE:  runStyleList,
	}

	switchCmd := &cobra.Command{
		Use:   "switch",
		Short: "Set the active template style",
		RunE:  runStyleSwitch,
	}
	switchCmd.Flags().StringVar(&styleSwitchName, "name", "", "Name of the style to activate (required)")
	_ = switchCmd.MarkFlagRequired("name")

	styleCmd.AddCommand(addCmd, listCmd, switchCmd)
}

func runStyleAdd(cmd *cobra.Command, args []string) error {
	wd, err := requireProject()
	if err != nil {
		return err
	}

	name := styleAddName
	if name == "" {
		name = template.StyleName(styleAddFrom)
	}
	dest := filepath.Join(wd, "template", "style", name)

	if _, err := os.Stat(dest); err == nil {
		if !styleAddForce {
			ok, err := confirm(fmt.Sprintf("Style %q already exists; replace its files?", name))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}
		}
		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("replace style %q: %w", name, err)
		}
	}

	if err := template.Fetch(styleAddFrom, dest); err != nil {
		os.RemoveAll(dest)
		return fmt.Errorf("add style %q: %w", name, err)
	}

	fmt.Printf("Added style %q to template/style/%s\n", name, name)
	fmt.Printf("  Activate it with: ssg style switch --name %s\n", name)
	return nil
}

func runStyleList(cmd *cobra.Command, args []string) error {
	wd, err := requireProject()
	if err != nil {
		return err
	}
	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	styleRoot := filepath.Join(wd, "template", "style")
	entries, err := os.ReadDir(styleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No styles installed (using the embedded default).")
			return nil
		}
		return err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		fmt.Println("No styles installed (using the embedded default).")
		return nil
	}
	sort.Strings(names)

	for _, n := range names {
		marker := "  "
		if n == cfg.Style {
			marker = "* "
		}
		fmt.Printf("%s%s\n", marker, n)
	}
	return nil
}

func runStyleSwitch(cmd *cobra.Command, args []string) error {
	wd, err := requireProject()
	if err != nil {
		return err
	}
	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	dest := filepath.Join(wd, "template", "style", styleSwitchName)
	if _, err := os.Stat(dest); err != nil {
		// "default" is always available via the embedded fallback.
		if styleSwitchName != "default" {
			return fmt.Errorf("style %q not found; run 'ssg style list' to see installed styles", styleSwitchName)
		}
	}

	if cfg.Style == styleSwitchName {
		fmt.Printf("Style %q is already active.\n", styleSwitchName)
		return nil
	}
	cfg.Style = styleSwitchName
	if err := config.Save(wd, cfg); err != nil {
		return err
	}
	fmt.Printf("Switched active style to %q\n", styleSwitchName)
	return nil
}

// requireProject returns the working directory if it contains an ssg.json.
func requireProject() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(wd, config.Filename)); err != nil {
		return "", fmt.Errorf("no %s found in %s; run 'ssg init' first", config.Filename, wd)
	}
	return wd, nil
}

// confirm prompts on stdin and returns true only for an affirmative answer.
func confirm(prompt string) (bool, error) {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
