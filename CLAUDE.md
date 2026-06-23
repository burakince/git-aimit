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
1. `internal/git` — checks the working directory is inside a Git repository
2. `internal/config` — loads `~/.config/git-aimit/config.json` (errors here before touching git)
3. `internal/git` — if `cfg.AutoStage` is true, runs `git add -A` before reading the diff
4. `internal/git` — runs `git diff --cached`; exits cleanly if nothing is staged
5. `os` — if `cfg.CommitTemplate` is set, reads the template file content (silently ignored if unreadable)
6. `internal/llm` — calls the configured provider (with template content baked in), prints the message, asks for confirmation
7. `internal/git` — runs `git commit -m <message>` if confirmed

**Provider interface (`internal/llm/provider.go`)** is the extension point for new LLM backends. The root command (`cmd/root.go`) holds a `llm.Provider` variable; adding a new backend only requires a new package under `internal/llm/<name>/` and a new `case` in the `switch cfg.Provider` block.

**Config (`internal/config/config.go`):** viper reads the JSON file; `encoding/json` writes it. `LoadFrom(path)` and `SaveTo(path, cfg)` accept explicit paths so tests use temp directories instead of `~/.config`. Fields: `provider`, `auto_stage` (bool, default false), `commit_template` (string, path to repo commit template file), `ollama.base_url`, `ollama.model`.

**Ollama client (`internal/llm/ollama/ollama.go`):** streams NDJSON from `POST /api/generate`, accumulating `response` tokens until `done: true`. `BuildPrompt(diff, commitTemplate string)` is a pure exported function for testability; when `commitTemplate` is non-empty it injects a `## Commit template` section into the user prompt so the model follows the repo's format. `ollamaError` reads the JSON error body and surfaces actionable messages (e.g. suggests `ollama pull` on 404). The system prompt requires a body paragraph when the diff touches multiple bounded contexts. `New(baseURL, model, commitTemplate string)` stores the template content on the `Client` struct; the root command reads the template file and passes its content at construction time.

**Testing approach:** Unit tests use `net/http/httptest` — no mocking libraries. `BuildPrompt` is a pure exported function specifically to enable network-free unit tests.

**Evals (`evals/evals_test.go`):** Opt-in tests behind `//go:build evals` that run the actual model against fixture diffs and assert output quality via a `criterion` slice. Each criterion is a named predicate over the raw message string. Two fixtures exist: `simpleDiff` (single-file, format check only) and `complexDiff` (three packages, also asserts a body paragraph is present). Add new criteria or fixtures freely — evals are deliberately flexible, not exhaustive.

**CI/Release (`.github/workflows/`):**
- `ci.yml` — runs `go vet` and `go test ./...` on every push and PR to `main`
- `release.yml` — triggered by `v*` tags; cross-compiles for 6 targets (Linux/Windows/macOS × amd64/arm64) with `CGO_ENABLED=0`, publishes binaries as GitHub release assets, then updates `Formula/git-aimit.rb` with the new tarball URL and SHA256 and commits back to `main`

**Homebrew (`Formula/git-aimit.rb`):** formula for the self-hosted tap. Users install with `brew tap burakince/git-aimit https://github.com/burakince/git-aimit && brew install git-aimit`. The formula builds from the source tarball using `go build`; the `url` and `sha256` lines are patched automatically by the release workflow on each new tag.
