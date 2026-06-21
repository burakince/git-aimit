package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/burakince/git-aimit/internal/config"
	"github.com/burakince/git-aimit/internal/git"
	"github.com/burakince/git-aimit/internal/llm"
	"github.com/burakince/git-aimit/internal/llm/ollama"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-aimit",
	Short: "AI-powered Git commit message generator",
	Long: `git-aimit reads staged changes and uses a local Ollama model to generate
a Conventional Commits message, then optionally commits for you.

Install it to PATH as "git-aimit" and Git will expose it as "git aimit".`,
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

	diff, err := git.StagedDiff()
	if err != nil {
		return err
	}
	if diff == "" {
		fmt.Println("No staged changes found. Stage your changes with `git add` first.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var provider llm.Provider
	switch cfg.Provider {
	case "ollama", "":
		provider = ollama.New(cfg.Ollama.BaseURL, cfg.Ollama.Model)
	default:
		return fmt.Errorf("unknown provider %q -- check your config or run `git aimit init`", cfg.Provider)
	}

	fmt.Printf("Generating commit message using %s (%s)...\n", cfg.Provider, cfg.Ollama.Model)

	message, err := provider.GenerateCommitMessage(context.Background(), diff)
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
