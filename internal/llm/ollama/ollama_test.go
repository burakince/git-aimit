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
	if !strings.HasPrefix(prompt, "Generate a commit message") {
		t.Errorf("prompt should start with expected prefix, got: %q", prompt[:50])
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
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	_, err := client.GenerateCommitMessage(context.Background(), "some diff")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500, got: %v", err)
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
