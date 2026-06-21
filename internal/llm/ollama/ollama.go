package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const systemPrompt = `You are a Git commit message generator. Analyse the staged diff carefully before writing.

## Subject line (always required)
- Format: <type>(<optional scope>): <short subject>
- Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
- Max 72 characters, imperative mood, no trailing period

## Body (required when any of the following apply)
- The diff touches more than one bounded context, package, or architectural layer
- The motivation or trade-off behind the change is not obvious from the diff alone
- Multiple distinct concerns are addressed in the same staged set of files

When a body is required:
- Separate it from the subject with a blank line
- Write one paragraph explaining WHY this change was made — the motivation, constraint, or context a reviewer needs to understand
- If multiple bounded contexts are affected, describe what changed in each and why they are coupled in this commit
- Do not restate what the diff already shows; explain the reasoning behind it
- Wrap lines at 72 characters

## Output
Return ONLY the raw commit message text. No code fences, no preamble, no explanation.`

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
	return fmt.Sprintf(
		"Analyse the following staged diff. Identify how many bounded contexts or packages are affected, then write the commit message:\n\n%s",
		diff,
	)
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
		return "", ollamaError(resp)
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

// ollamaError reads the response body and returns a descriptive error.
// Ollama encodes errors as {"error": "..."} in the body.
func ollamaError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		if resp.StatusCode == http.StatusNotFound {
			// Extract the model name from the error when possible; fall back to the raw message.
			return fmt.Errorf("%s -- try: ollama pull <model>", apiErr.Error)
		}
		return fmt.Errorf("Ollama error: %s", apiErr.Error)
	}

	return fmt.Errorf("Ollama returned unexpected status %d", resp.StatusCode)
}
