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
