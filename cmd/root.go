package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/burakince/git-aimit/internal/config"
	"github.com/burakince/git-aimit/internal/git"
	"github.com/burakince/git-aimit/internal/llm"
	"github.com/burakince/git-aimit/internal/llm/ollama"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "git-aimit",
	Short: "AI-powered Git commit message generator",
	Long: `git-aimit reads staged changes and uses a local Ollama model to generate
a Conventional Commits message, then optionally commits for you.

Install it to PATH as "git-aimit" and Git will expose it as "git aimit".`,
	Version:      version,
	RunE:         runRoot,
	SilenceUsage: true,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	if !git.IsRepo() {
		return fmt.Errorf("not inside a Git repository")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.ConfigVersion < config.CurrentConfigVersion {
		fmt.Fprintln(os.Stderr, "Warning: your git-aimit config is outdated. Run `git aimit init` to update it.")
	}

	if cfg.AutoStage {
		fmt.Println("Auto-staging all changes...")
		if err := git.StageAll(); err != nil {
			return err
		}
	}

	diff, err := git.StagedDiff()
	if err != nil {
		return err
	}
	if diff == "" {
		fmt.Println("No staged changes found. Stage your changes with `git add` first.")
		return nil
	}

	var templateContent string
	if cfg.CommitTemplate != "" {
		if b, err := os.ReadFile(cfg.CommitTemplate); err == nil {
			templateContent = strings.TrimSpace(string(b))
		}
	}

	var provider llm.Provider
	switch cfg.Provider {
	case "ollama", "":
		provider = ollama.New(cfg.Ollama.BaseURL, cfg.Ollama.Model, templateContent)
	default:
		return fmt.Errorf("unknown provider %q -- check your config or run `git aimit init`", cfg.Provider)
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		spin(fmt.Sprintf("Generating with %s (%s)...", cfg.Provider, cfg.Ollama.Model), stop)
		close(stopped)
	}()
	message, err := provider.GenerateCommitMessage(context.Background(), diff)
	close(stop)
	<-stopped // wait for spinner to clear its line before printing

	if err != nil {
		return err
	}

	fmt.Printf("\nGenerated commit message:\n\n%s\n\n", message)

	fmt.Print("Commit with this message? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Commit cancelled.")
		return nil
	}

	if err := git.Commit(message); err != nil {
		return err
	}

	fmt.Println("Committed successfully.")
	return nil
}

func spin(label string, stop <-chan struct{}) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-stop:
			fmt.Print("\r\033[K") // clear the spinner line
			return
		case <-ticker.C:
			fmt.Printf("\r%s %s", frames[i%len(frames)], label)
			i++
		}
	}
}
