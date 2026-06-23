package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/burakince/git-aimit/internal/config"
	"github.com/burakince/git-aimit/internal/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Configure git-aimit interactively",
	Long:         "Prompts for Ollama connection details and saves them to ~/.config/git-aimit/config.json.",
	RunE:         runInit,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	path, err := config.ConfigPath()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(path)

	baseURL := prompt(reader, "Ollama base URL", "http://localhost:11434")
	model := prompt(reader, "Model name", "llama3.1")
	checkConnectivity(baseURL)
	autoStage := promptBool(reader, "Auto-stage all changes before generating message", false)

	commitTemplate := git.FindCommitTemplate()
	if commitTemplate != "" {
		fmt.Printf("Found commit template: %s\n", commitTemplate)
		if !promptBool(reader, "Use this commit template", true) {
			commitTemplate = ""
		}
	}

	if commitTemplate == "" {
		if promptBool(reader, "Use built-in best-practices commit template", true) {
			p, werr := config.WriteDefaultTemplate(configDir)
			if werr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write default template: %v\n", werr)
			} else {
				commitTemplate = p
				fmt.Printf("Default commit template written to %s\n", p)
			}
		} else {
			commitTemplate = prompt(reader, "Commit template path", "")
			if commitTemplate != "" {
				checkTemplatePath(commitTemplate)
			}
		}
	}

	cfg := &config.Config{
		ConfigVersion:  config.CurrentConfigVersion,
		Provider:       "ollama",
		AutoStage:      autoStage,
		CommitTemplate: commitTemplate,
		Ollama: config.OllamaConfig{
			BaseURL: baseURL,
			Model:   model,
		},
	}

	if err := config.SaveTo(path, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", path)
	if autoStage {
		fmt.Println("Run `git aimit` to stage everything, generate a message, and commit.")
	} else {
		fmt.Println("Run `git aimit` in a repository with staged changes.")
	}
	return nil
}

func prompt(r *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptBool(r *bufio.Reader, label string, defaultVal bool) bool {
	defaultStr := "N"
	if defaultVal {
		defaultStr = "y"
	}
	fmt.Printf("%s [%s]: ", label, defaultStr)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultVal
	}
	return line == "y" || line == "yes"
}

func checkTemplatePath(path string) {
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: commit template not found at %s -- verify the path.\n", path)
	}
}

func checkConnectivity(baseURL string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Warning: Ollama is not reachable at %s -- is it running?\n", baseURL)
		fmt.Fprintln(os.Stderr, "Saving config anyway. Fix connectivity and try again.")
		return
	}
	resp.Body.Close()
	fmt.Println("Ollama is reachable.")
}
