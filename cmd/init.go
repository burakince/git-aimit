package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/burakince/git-aimit/internal/config"
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

	baseURL := prompt(reader, "Ollama base URL", "http://localhost:11434")
	model := prompt(reader, "Model name", "llama3")
	autoStage := promptBool(reader, "Auto-stage all changes before generating message", false)

	checkConnectivity(baseURL)

	cfg := &config.Config{
		Provider:  "ollama",
		AutoStage: autoStage,
		Ollama: config.OllamaConfig{
			BaseURL: baseURL,
			Model:   model,
		},
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if autoStage {
		fmt.Println("Configuration saved. Run `git aimit` to stage everything, generate a message, and commit.")
	} else {
		fmt.Println("Configuration saved. Run `git aimit` in a repository with staged changes.")
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
