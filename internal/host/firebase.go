package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	register(&firebaseHost{})
}

type firebaseHost struct{}

func (f *firebaseHost) Codename() string { return "firebase" }

func (f *firebaseHost) Setup(ctx context.Context, root string, force bool) error {
	if err := checkFirebaseCLI(ctx); err != nil {
		return err
	}

	projectID, siteID, err := promptFirebaseConfig(ctx)
	if err != nil {
		return err
	}

	firebaseJSONPath := filepath.Join(root, "firebase.json")
	firebasercPath := filepath.Join(root, ".firebaserc")

	if !force {
		if _, err := os.Stat(firebaseJSONPath); err == nil {
			return fmt.Errorf("firebase.json already exists; use --force to overwrite")
		}
		if _, err := os.Stat(firebasercPath); err == nil {
			return fmt.Errorf(".firebaserc already exists; use --force to overwrite")
		}
	}

	if err := writeFirebaseJSON(firebaseJSONPath, siteID); err != nil {
		return fmt.Errorf("write firebase.json: %w", err)
	}
	if err := writeFirebaserc(firebasercPath, projectID); err != nil {
		return fmt.Errorf("write .firebaserc: %w", err)
	}

	fmt.Println("firebase.json and .firebaserc written.")
	fmt.Println("Next: run `ssgo generate` then `ssgo deploy`.")
	return nil
}

func (f *firebaseHost) Deploy(ctx context.Context, root string) error {
	if err := checkFirebaseCLI(ctx); err != nil {
		return err
	}

	for _, name := range []string{"firebase.json", ".firebaserc"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return fmt.Errorf("%s not found; run `ssgo host firebase` first", name)
		}
	}

	siteID, err := readSiteID(filepath.Join(root, "firebase.json"))
	if err != nil {
		return err
	}
	projectID, err := readProjectID(filepath.Join(root, ".firebaserc"))
	if err != nil {
		return err
	}

	onlyArg := "hosting"
	if siteID != "" {
		onlyArg = "hosting:" + siteID
	}

	cmd := exec.CommandContext(ctx, "firebase", "deploy", "--only", onlyArg)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("firebase deploy: %w", err)
	}

	fmt.Printf("\nLive at: https://%s.web.app\n", projectID)
	fmt.Printf("         https://%s.firebaseapp.com\n", projectID)
	return nil
}

// checkFirebaseCLI verifies the Firebase CLI is installed and the user is authenticated.
func checkFirebaseCLI(ctx context.Context) error {
	if _, err := exec.LookPath("firebase"); err != nil {
		return fmt.Errorf("Firebase CLI not found; install it with: npm install -g firebase-tools")
	}
	cmd := exec.CommandContext(ctx, "firebase", "projects:list")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Firebase CLI authentication check failed; run: firebase login")
	}
	return nil
}

func promptFirebaseConfig(ctx context.Context) (projectID, siteID string, err error) {
	r := bufio.NewReader(os.Stdin)

	fmt.Print("Firebase project ID (find it at https://console.firebase.google.com): ")
	projectID, err = r.ReadString('\n')
	if err != nil {
		return
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		err = fmt.Errorf("project ID is required")
		return
	}

	fmt.Print("Hosting site ID (leave blank for the project's default site): ")
	siteID, err = r.ReadString('\n')
	if err != nil {
		return
	}
	siteID = strings.TrimSpace(siteID)

	fmt.Println("Public directory: build/prod (fixed)")
	return
}

func writeFirebaseJSON(path, siteID string) error {
	type hosting struct {
		Public  string   `json:"public"`
		Site    string   `json:"site,omitempty"`
		Ignore  []string `json:"ignore"`
	}
	type firebaseConfig struct {
		Hosting hosting `json:"hosting"`
	}
	cfg := firebaseConfig{
		Hosting: hosting{
			Public: "build/prod",
			Site:   siteID,
			Ignore: []string{"firebase.json", "**/.*", "**/node_modules/**"},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func writeFirebaserc(path, projectID string) error {
	type firebaserc struct {
		Projects map[string]string `json:"projects"`
	}
	rc := firebaserc{Projects: map[string]string{"default": projectID}}
	data, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func readSiteID(firebaseJSONPath string) (string, error) {
	data, err := os.ReadFile(firebaseJSONPath)
	if err != nil {
		return "", err
	}
	var cfg struct {
		Hosting struct {
			Site string `json:"site"`
		} `json:"hosting"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Hosting.Site, nil
}

func readProjectID(firebasercPath string) (string, error) {
	data, err := os.ReadFile(firebasercPath)
	if err != nil {
		return "", err
	}
	var rc struct {
		Projects map[string]string `json:"projects"`
	}
	if err := json.Unmarshal(data, &rc); err != nil {
		return "", err
	}
	id, ok := rc.Projects["default"]
	if !ok || id == "" {
		return "", fmt.Errorf(".firebaserc has no default project")
	}
	return id, nil
}
