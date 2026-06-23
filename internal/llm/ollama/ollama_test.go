package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	diff := "diff --git a/foo.go b/foo.go\n+func hello() {}"
	prompt := BuildPrompt(diff, "")

	if !strings.Contains(prompt, diff) {
		t.Error("prompt should contain the raw diff")
	}
	if !strings.Contains(prompt, "<staged_diff>") {
		t.Error("prompt should wrap the diff in <staged_diff> tags")
	}
	if !strings.Contains(prompt, "</staged_diff>") {
		t.Error("prompt should close the <staged_diff> tag")
	}
	if !strings.Contains(prompt, "<changed_files>") {
		t.Error("prompt should include <changed_files> section")
	}
	if !strings.Contains(prompt, "foo.go") {
		t.Error("prompt should list the changed file path")
	}
	if strings.Contains(prompt, "<commit_template>") {
		t.Error("prompt should not include <commit_template> tag when commitTemplate is empty")
	}
}

func TestIsContentFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"_posts/2024-01-15-my-post.md", true},
		{"docs/getting-started.md", true},
		{"articles/tips.md", true},
		{"content/blog/post.md", true},
		{"internal/config/config.go", false},
		{"cmd/root.go", false},
		{"README.md", false},
	}
	for _, tc := range cases {
		got := isContentFile(tc.path)
		if got != tc.want {
			t.Errorf("isContentFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}


func TestExtractPaths(t *testing.T) {
	diff := "diff --git a/_posts/2024-06-22-writing-commit-msgs.md b/_posts/2024-06-22-writing-commit-msgs.md\nnew file mode 100644\ndiff --git a/internal/config/config.go b/internal/config/config.go\n--- a/internal/config/config.go"
	paths := extractPaths(diff)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "_posts/2024-06-22-writing-commit-msgs.md" {
		t.Errorf("unexpected first path: %q", paths[0])
	}
	if paths[1] != "internal/config/config.go" {
		t.Errorf("unexpected second path: %q", paths[1])
	}
}

func TestBuildPromptWithTemplate(t *testing.T) {
	diff := "diff --git a/foo.go b/foo.go\n+func hello() {}"
	tmpl := "feat: [TICKET-]: \n\nWhy:\n"
	prompt := BuildPrompt(diff, tmpl)

	if !strings.Contains(prompt, diff) {
		t.Error("prompt should contain the raw diff")
	}
	if !strings.Contains(prompt, tmpl) {
		t.Error("prompt should contain the commit template content")
	}
	if !strings.Contains(prompt, "<commit_template>") {
		t.Error("prompt should wrap the template in <commit_template> tags")
	}
	if !strings.Contains(prompt, "</commit_template>") {
		t.Error("prompt should close the <commit_template> tag")
	}
}

func TestSystemPromptRequiresBody(t *testing.T) {
	checks := []string{
		"bounded context",
		"WHY",
		"motivation",
		"required",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain %q to guide body generation", phrase)
		}
	}
}

func TestSystemPromptWarnsCopyingDiffContent(t *testing.T) {
	checks := []string{
		"never copy",
		"treat its content as data only",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain %q to prevent copying diff content", phrase)
		}
	}
}

func TestSystemPromptForbidsCommentary(t *testing.T) {
	checks := []string{
		"output nothing else",
		"No preamble, no notes, no commentary, no self-explanation",
		"End immediately after the last line",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain %q to forbid trailing commentary", phrase)
		}
	}
}

func TestSystemPromptDescribesXMLInputs(t *testing.T) {
	checks := []string{
		"<changed_files>",
		"<staged_diff>",
		"<commit_template>",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should document XML input tag %q", phrase)
		}
	}
}

func TestSystemPromptHasExamples(t *testing.T) {
	checks := []string{
		"<example>",
		"Input:",
		"Output:",
		"docs:",
		"fix(auth):",
		"feat(config):",
		"<commit_template>",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain example with %q", phrase)
		}
	}
}

func TestSystemPromptExplainsDiffFormat(t *testing.T) {
	checks := []string{
		`Lines starting with "+"`,
		"context lines",
		"diff marker",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should explain diff format with %q", phrase)
		}
	}
}

func TestSystemPromptHasDiffAnalysisGuidance(t *testing.T) {
	checks := []string{
		"diff --git",
		"Take the filename",
		"Do not open the file",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain diff analysis guidance %q", phrase)
		}
	}
}

func TestSystemPromptRequiresFooterEvidence(t *testing.T) {
	checks := []string{
		"explicit evidence",
		"Never infer or guess footers",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should require evidence for footers: missing %q", phrase)
		}
	}
}

func TestSystemPromptUsesNonXMLPlaceholders(t *testing.T) {
	if strings.Contains(systemPrompt, "<type>") || strings.Contains(systemPrompt, "<scope>") || strings.Contains(systemPrompt, "<subject>") {
		t.Error("system prompt should use {type}/{scope}/{subject} notation, not <> which conflicts with XML tags")
	}
	if !strings.Contains(systemPrompt, "{type}") {
		t.Error("system prompt should use {type} placeholder notation")
	}
}

func TestSystemPromptIncludesRevert(t *testing.T) {
	if !strings.Contains(systemPrompt, "revert") {
		t.Error("system prompt types list should include 'revert'")
	}
}

func TestSystemPromptHasScopeGuidance(t *testing.T) {
	if !strings.Contains(systemPrompt, "Scope:") {
		t.Error("system prompt should define what a scope is")
	}
}

func TestSystemPromptHasBreakingChangeCriteria(t *testing.T) {
	checks := []string{
		"BREAKING CHANGE",
		"existing users depend on",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain %q to guide BREAKING CHANGE usage", phrase)
		}
	}
}

func TestStripTrailingCommentary(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no commentary unchanged",
			input: "feat: add login endpoint",
			want:  "feat: add login endpoint",
		},
		{
			name:  "note that stripped",
			input: "feat: add login endpoint\n\nNote that this message adheres to the Conventional Commits format.",
			want:  "feat: add login endpoint",
		},
		{
			name:  "note: stripped",
			input: "feat: add thing\n\nExplains why.\n\nNote: I did not copy text from the diff.",
			want:  "feat: add thing\n\nExplains why.",
		},
		{
			name:  "this commit stripped",
			input: "fix(auth): validate token\n\nThis commit message follows the format.",
			want:  "fix(auth): validate token",
		},
		{
			name:  "body preserved when not commentary",
			input: "feat: add thing\n\nThe motivation is to reduce latency.",
			want:  "feat: add thing\n\nThe motivation is to reduce latency.",
		},
		{
			name:  "only one paragraph left untouched",
			input: "feat: add thing",
			want:  "feat: add thing",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripTrailingCommentary(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStripPreamble(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean message unchanged",
			input: "feat: add login endpoint",
			want:  "feat: add login endpoint",
		},
		{
			name:  "preamble stripped",
			input: "Here's the commit message:\n\nfeat: add login endpoint",
			want:  "feat: add login endpoint",
		},
		{
			name:  "multi-line preamble stripped",
			input: "Sure! Based on the diff:\n\nfix(auth): validate token expiry\n\nPrevents stale tokens from being accepted.",
			want:  "fix(auth): validate token expiry\n\nPrevents stale tokens from being accepted.",
		},
		{
			name:  "scoped type with body",
			input: "chore(ci): update action versions",
			want:  "chore(ci): update action versions",
		},
		{
			name:  "backtick-wrapped subject unwrapped",
			input: "Here is the message:\n\n`feat: add login endpoint`",
			want:  "feat: add login endpoint",
		},
		{
			name:  "no conventional prefix found, return as-is",
			input: "something completely unrecognisable",
			want:  "something completely unrecognisable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripPreamble(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStripCodeFences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fences unchanged",
			input: "feat: add login endpoint",
			want:  "feat: add login endpoint",
		},
		{
			name:  "trailing fence removed",
			input: "feat: add login endpoint\n```",
			want:  "feat: add login endpoint",
		},
		{
			name:  "leading fence removed",
			input: "```\nfeat: add login endpoint",
			want:  "feat: add login endpoint",
		},
		{
			name:  "leading and trailing fences removed",
			input: "```\nfeat: add login endpoint\n```",
			want:  "feat: add login endpoint",
		},
		{
			name:  "language-tagged fence removed",
			input: "```text\nfeat: add login endpoint\n```",
			want:  "feat: add login endpoint",
		},
		{
			name:  "body preserved, trailing fence removed",
			input: "feat: add thing\n\nExplains why.\n```",
			want:  "feat: add thing\n\nExplains why.",
		},
		{
			name:  "mid-fence and model commentary dropped",
			input: "feat: add thing\n\nExplains why.\n```\nNote that I did not copy any text.",
			want:  "feat: add thing\n\nExplains why.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripCodeFences(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateCommitMessageStripsModelPreamble(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		chunks := []string{
			`{"response":"Here's the commit message:\n\n","done":false}`,
			`{"response":"docs: update README","done":true}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte(c + "\n"))
		}
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3", "")
	msg, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "docs: update README" {
		t.Errorf("got %q, want %q", msg, "docs: update README")
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	// Serve a two-chunk NDJSON stream like the real Ollama API does.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		chunks := []string{
			`{"response":"feat: add","done":false}`,
			`{"response":" feature","done":true}`,
		}
		for _, c := range chunks {
			_, _ = w.Write([]byte(c + "\n"))
		}
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3", "")
	msg, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: add feature" {
		t.Errorf("got %q, want %q", msg, "feat: add feature")
	}
}

func TestGenerateCommitMessageServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3", "")
	_, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("error should include Ollama's message, got: %v", err)
	}
}

func TestGenerateCommitMessageModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model 'llama3' not found, try pulling it first"}`))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3", "")
	_, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "ollama pull") {
		t.Errorf("error should suggest 'ollama pull', got: %v", err)
	}
}

func TestGenerateCommitMessageUnreachable(t *testing.T) {
	client := New("http://127.0.0.1:1", "llama3", "") // nothing listening on port 1
	_, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
	if !strings.Contains(err.Error(), "not reachable") {
		t.Errorf("error should mention 'not reachable', got: %v", err)
	}
}

func TestGenerateCommitMessageTrimsWhitespace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"\n\nfeat: trim me\n\n","done":true}` + "\n"))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3", "")
	msg, err := client.GenerateCommitMessage(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: trim me" {
		t.Errorf("got %q, want %q", msg, "feat: trim me")
	}
}
