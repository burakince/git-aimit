# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run a single test
go test ./internal/config/... -run TestSaveAndLoad
go test ./internal/llm/ollama/... -run TestBuildPrompt

# Vet
go vet ./...

# Sync dependencies
go mod tidy

# Run evals (requires a live Ollama instance; excluded from go test ./...)
go test -tags evals -v ./evals/

# Override Ollama endpoint or model for evals
OLLAMA_BASE_URL=http://localhost:11434 OLLAMA_MODEL=llama3 go test -tags evals -v ./evals/
```

## Architecture

`git-aimit` is a Cobra CLI with two commands: the root command (generate + commit) and `init` (interactive config setup). It is named `git-aimit` so Git exposes it as `git aimit`.

**Request flow (root command):**
1. `internal/git` — checks repo, gets `git diff --cached`
2. `internal/config` — loads `~/.config/git-aimit/config.json`
3. `internal/llm` — calls the configured provider, prints the message, asks for confirmation
4. `internal/git` — runs `git commit -m <message>` if confirmed

**Provider interface (`internal/llm/provider.go`)** is the extension point for new LLM backends. The root command (`cmd/root.go`) holds a `llm.Provider` variable; adding a new backend only requires a new package under `internal/llm/<name>/` and a new `case` in the `switch cfg.Provider` block.

**Config (`internal/config/config.go`):** viper reads the JSON file; `encoding/json` writes it. `LoadFrom(path)` and `SaveTo(path, cfg)` accept explicit paths so tests use temp directories instead of `~/.config`. The Ollama client (`internal/llm/ollama/ollama.go`) streams NDJSON from `POST /api/generate`, accumulating `response` fields until `done: true`.

**Testing approach:** Unit tests use `net/http/httptest` — no mocking libraries. `BuildPrompt` is a pure exported function specifically to enable network-free unit tests.

**Evals (`evals/evals_test.go`):** Opt-in tests behind `//go:build evals` that run the actual model against fixture diffs and assert output quality via a `criterion` slice. Each criterion is a named predicate over the raw message string. Two fixtures exist: `simpleDiff` (single-file, format check only) and `complexDiff` (three packages, also asserts a body paragraph is present). Add new criteria or fixtures freely — evals are deliberately flexible, not exhaustive.
