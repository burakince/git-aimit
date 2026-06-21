//go:build evals

// Package evals_test runs quality evaluations against a live Ollama instance.
// These tests are excluded from the default test run. To execute them:
//
//	go test -tags evals -v ./evals/
//
// Override the Ollama endpoint and model with environment variables:
//
//	OLLAMA_BASE_URL=http://localhost:11434 OLLAMA_MODEL=llama3 go test -tags evals -v ./evals/
package evals_test

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/burakince/git-aimit/internal/llm/ollama"
)

// conventionalCommitRe matches the mandatory subject line format.
var conventionalCommitRe = regexp.MustCompile(
	`^(feat|fix|docs|style|refactor|test|chore|perf|ci|build)(\([^)]+\))?: .+`,
)

// criterion is a named quality check applied to a generated commit message.
type criterion struct {
	name  string
	check func(msg string) bool
}

// commonCriteria are applied to every eval case regardless of diff complexity.
var commonCriteria = []criterion{
	{
		name: "subject line follows Conventional Commits format",
		check: func(msg string) bool {
			subject := strings.SplitN(msg, "\n", 2)[0]
			return conventionalCommitRe.MatchString(subject)
		},
	},
	{
		name: "subject line is 72 characters or fewer",
		check: func(msg string) bool {
			subject := strings.SplitN(msg, "\n", 2)[0]
			return len(subject) <= 72
		},
	},
	{
		name: "output contains no code fences",
		check: func(msg string) bool {
			return !strings.Contains(msg, "```")
		},
	},
	{
		name: "output is not empty",
		check: func(msg string) bool {
			return strings.TrimSpace(msg) != ""
		},
	},
}

// hasBodyParagraph checks that the message contains a blank-line-separated body.
var hasBodyParagraph = criterion{
	name: "complex diff produces a body paragraph separated by a blank line",
	check: func(msg string) bool {
		parts := strings.SplitN(msg, "\n\n", 2)
		return len(parts) == 2 && strings.TrimSpace(parts[1]) != ""
	},
}

// simpleDiff represents a focused, single-concern change in one package.
// A subject-only message is acceptable here; a body is not required.
const simpleDiff = `diff --git a/internal/git/git.go b/internal/git/git.go
index a1b2c3d..e4f5a6b 100644
--- a/internal/git/git.go
+++ b/internal/git/git.go
@@ -20,6 +20,15 @@ func StagedDiff() (string, error) {
 	return strings.TrimSpace(string(out)), nil
 }

+// StashList returns the names of all stash entries.
+func StashList() ([]string, error) {
+	out, err := exec.Command("git", "stash", "list", "--format=%gd").Output()
+	if err != nil {
+		return nil, fmt.Errorf("running git stash list: %w", err)
+	}
+	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
+	return lines, nil
+}
+
 func Commit(message string) error {`

// complexDiff represents a coordinated change across three bounded contexts:
// the config schema, the Ollama HTTP client, and the init CLI command.
// The model should recognise the coupling and include an explanatory body.
const complexDiff = `diff --git a/internal/config/config.go b/internal/config/config.go
index a1b2c3d..e4f5a6b 100644
--- a/internal/config/config.go
+++ b/internal/config/config.go
@@ -12,6 +12,7 @@ import (
 type OllamaConfig struct {
 	BaseURL string
 	Model   string
+	Timeout int
 }

diff --git a/internal/llm/ollama/ollama.go b/internal/llm/ollama/ollama.go
index b2c3d4e..f5a6b7c 100644
--- a/internal/llm/ollama/ollama.go
+++ b/internal/llm/ollama/ollama.go
@@ -44,7 +44,10 @@ func New(baseURL, model string, timeout int) *Client {
 	return &Client{
 		BaseURL: strings.TrimRight(baseURL, "/"),
 		Model:   model,
-		http:    &http.Client{},
+		http: &http.Client{
+			Timeout: time.Duration(timeout) * time.Second,
+		},
 	}
 }

diff --git a/cmd/init.go b/cmd/init.go
index c3d4e5f..a6b7c8d 100644
--- a/cmd/init.go
+++ b/cmd/init.go
@@ -30,6 +30,11 @@ func runInit(cmd *cobra.Command, args []string) error {
 	baseURL := prompt(reader, "Ollama base URL", "http://localhost:11434")
 	model := prompt(reader, "Model name", "llama3")
+	raw := prompt(reader, "Request timeout in seconds", "30")
+	timeout, err := strconv.Atoi(raw)
+	if err != nil || timeout <= 0 {
+		timeout = 30
+	}

 	checkConnectivity(baseURL)`

// ollamaClient returns a configured client and skips the test if Ollama is not reachable.
func ollamaClient(t *testing.T) *ollama.Client {
	t.Helper()

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3"
	}

	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(baseURL + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("skipping eval: Ollama not reachable at %s (set OLLAMA_BASE_URL / OLLAMA_MODEL to override)", baseURL)
	}
	resp.Body.Close()

	return ollama.New(baseURL, model)
}

// evaluate runs each criterion against msg and records failures.
func evaluate(t *testing.T, msg string, criteria []criterion) {
	t.Helper()
	for _, c := range criteria {
		if !c.check(msg) {
			t.Errorf("FAIL [%s]\n--- generated message ---\n%s\n---", c.name, msg)
		}
	}
}

// TestSimpleDiffCommitFormat checks that a focused single-file change produces
// a correctly formatted Conventional Commits subject line.
func TestSimpleDiffCommitFormat(t *testing.T) {
	client := ollamaClient(t)

	msg, err := client.GenerateCommitMessage(context.Background(), simpleDiff)
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	t.Logf("generated message:\n%s", msg)

	evaluate(t, msg, commonCriteria)
}

// TestComplexDiffHasBodyParagraph checks that a multi-package change produces
// a commit message with an explanatory body paragraph.
func TestComplexDiffHasBodyParagraph(t *testing.T) {
	client := ollamaClient(t)

	msg, err := client.GenerateCommitMessage(context.Background(), complexDiff)
	if err != nil {
		t.Fatalf("GenerateCommitMessage: %v", err)
	}
	t.Logf("generated message:\n%s", msg)

	evaluate(t, msg, append(commonCriteria, hasBodyParagraph))
}
