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
	prompt := BuildPrompt(diff)

	if !strings.Contains(prompt, diff) {
		t.Error("prompt should contain the raw diff")
	}
	if !strings.Contains(prompt, "bounded contexts") {
		t.Error("prompt should ask the model to identify bounded contexts")
	}
	if !strings.Contains(prompt, "NEW Conventional Commits message") {
		t.Error("prompt should instruct the model to write a NEW commit message")
	}
	if !strings.Contains(prompt, "<staged_diff>") {
		t.Error("prompt should wrap the diff in <staged_diff> tags")
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
		"do not copy",
		"Start your response directly with the commit type",
	}
	for _, phrase := range checks {
		if !strings.Contains(systemPrompt, phrase) {
			t.Errorf("system prompt should contain %q to prevent copying diff content", phrase)
		}
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

	client := New(srv.URL, "llama3")
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

	client := New(srv.URL, "llama3")
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

	client := New(srv.URL, "llama3")
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

	client := New(srv.URL, "llama3")
	_, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "ollama pull") {
		t.Errorf("error should suggest 'ollama pull', got: %v", err)
	}
}

func TestGenerateCommitMessageUnreachable(t *testing.T) {
	client := New("http://127.0.0.1:1", "llama3") // nothing listening on port 1
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

	client := New(srv.URL, "llama3")
	msg, err := client.GenerateCommitMessage(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: trim me" {
		t.Errorf("got %q, want %q", msg, "feat: trim me")
	}
}
