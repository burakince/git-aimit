package llm

import "context"

// Provider generates commit messages from staged diffs.
// Implement this interface to add a new LLM backend.
type Provider interface {
	GenerateCommitMessage(ctx context.Context, diff string) (string, error)
}
