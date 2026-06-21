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

**Testing approach:** HTTP tests use `net/http/httptest` — no mocking libraries. `BuildPrompt` is a pure exported function specifically to enable network-free unit tests.
