package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const systemPrompt = `You are a Git commit message generator. Given a unified diff of staged changes, produce a commit message that follows the Conventional Commits specification:

- Format: <type>(<optional scope>): <short subject>
- Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
- Subject line: max 72 characters, imperative mood, no trailing period
- Optional body: explain WHY, not WHAT, wrapped at 72 characters
- Separate subject from body with a blank line

Return ONLY the commit message text with no extra commentary, code fences, or explanation.`

// Client calls the Ollama HTTP API to generate commit messages.
type Client struct {
	BaseURL string
	Model   string
	http    *http.Client
}

// New creates a Client for the given Ollama base URL and model name.
func New(baseURL, model string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		http:    &http.Client{},
	}
}

// BuildPrompt constructs the user prompt sent to the model.
func BuildPrompt(diff string) string {
	return fmt.Sprintf("Generate a commit message for the following staged changes:\n\n%s", diff)
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// GenerateCommitMessage implements llm.Provider.
func (c *Client) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	body, err := json.Marshal(generateRequest{
		Model:  c.Model,
		Prompt: BuildPrompt(diff),
		System: systemPrompt,
		Stream: true,
	})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama is not reachable at %s -- is it running?", c.BaseURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned unexpected status %d", resp.StatusCode)
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk generateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		sb.WriteString(chunk.Response)
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading Ollama response: %w", err)
	}

	return strings.TrimSpace(sb.String()), nil
}
