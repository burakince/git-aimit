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
2. `internal/config` — loads `~/.config/git-aimit/config.json`; warns if `config_version` is outdated
3. `internal/git` — if `cfg.AutoStage` is true, runs `git add -A` before reading the diff
4. `internal/git` — runs `git diff --cached`; exits cleanly if nothing is staged
5. `os` — if `cfg.CommitTemplate` is set, reads the template file content (silently ignored if unreadable)
6. `internal/llm` — calls the configured provider (template content passed at construction), prints the message, asks for confirmation
7. `internal/git` — runs `git commit -m <message>` if confirmed

**Provider interface (`internal/llm/provider.go`)** is the extension point for new LLM backends. The root command (`cmd/root.go`) holds a `llm.Provider` variable; adding a new backend only requires a new package under `internal/llm/<name>/` and a new `case` in the `switch cfg.Provider` block.

**Config (`internal/config/config.go`):** viper reads the JSON file; `encoding/json` writes it. `LoadFrom(path)` and `SaveTo(path, cfg)` accept explicit paths so tests use temp directories instead of `~/.config`. Fields: `config_version` (int, bumped when the schema changes), `provider`, `auto_stage` (bool, default false), `commit_template` (string, path to commit template file), `ollama.base_url`, `ollama.model`.

**Default commit template (`internal/config/defaults.go`):** `WriteDefaultTemplate(configDir string)` writes the built-in template (embedded via `//go:embed assets/commit-template.txt`) to `~/.config/git-aimit/commit-template.txt` if the file does not yet exist, preserving any user edits on subsequent runs.

**Init command (`cmd/init.go`):** interactive setup wizard that prompts for Ollama URL, model name, connectivity check, auto-stage preference, and commit template. Template discovery order: (1) `git.FindCommitTemplate()` checks `git config commit.template` then common filenames (`.gitmessage`, `.git-commit-template`, `.commit-template`); (2) if none found, offer the built-in default via `WriteDefaultTemplate`; (3) if declined, prompt for a custom path. Saves `config_version: CurrentConfigVersion` so stale configs can be detected on next run.

**Ollama client (`internal/llm/ollama/ollama.go`):** streams NDJSON from `POST /api/generate`, accumulating `response` tokens until `done: true`.

- `BuildPrompt(diff, commitTemplate string)` — pure exported function for testability. Calls `extractPaths(diff)` to parse `diff --git` headers and injects a `<changed_files>` block listing every changed path before the `<staged_diff>` block, so the model sees file paths before reading diff content. Prepends `<commit_template>` when non-empty.
- `extractPaths(diff string)` — extracts file paths from `diff --git a/X b/X` header lines using a compiled regexp.
- `isContentFile(path string)` — returns true for paths under `_posts/`, `docs/`, `articles/`, or `content/`; used in classification logic.
- `ollamaError` — reads the JSON error body and surfaces actionable messages (e.g. suggests `ollama pull` on 404).
- `New(baseURL, model, commitTemplate string)` — stores template content on the `Client` struct.
- Output post-processing pipeline: `stripTrailingCommentary(stripCodeFences(stripPreamble(strings.TrimSpace(raw))))` — removes LLM preamble, code fences, and trailing meta-commentary.

**System prompt structure:** Step 1–4 sequential workflow — (1) classify each file from `<changed_files>` (content files derive subject from filename slug; code files read diff hunks), (2) write subject line, (3) decide whether a body is needed, (4) add footers only when explicit evidence exists. Rules-only system prompt; data is passed as XML-tagged user content.

**Testing approach:** Unit tests use `net/http/httptest` — no mocking libraries. `BuildPrompt`, `extractPaths`, and `isContentFile` are pure exported/unexported functions specifically to enable network-free unit tests.

**Evals (`evals/evals_test.go`):** Opt-in tests behind `//go:build evals` that run the actual model against fixture diffs and assert output quality via a `criterion` slice. Each criterion is a named predicate over the raw message string. Two fixtures exist: `simpleDiff` (single-file, format check only) and `complexDiff` (three packages, also asserts a body paragraph is present). Add new criteria or fixtures freely — evals are deliberately flexible, not exhaustive.

**CI/Release (`.github/workflows/`):**
- `ci.yml` — runs `go vet` and `go test ./...` on every push and PR to `main`
- `release.yml` — triggered by `v*` tags; cross-compiles for 6 targets (Linux/Windows/macOS × amd64/arm64) with `CGO_ENABLED=0`, injects the tag as `cmd.version` via `-ldflags "-X github.com/burakince/git-aimit/cmd.version=$TAG"`, publishes binaries as GitHub release assets, then updates `Formula/git-aimit.rb` with the new tarball URL and SHA256 and commits back to `main`

**Homebrew (`Formula/git-aimit.rb`):** formula for the self-hosted tap. Users install with `brew tap burakince/git-aimit https://github.com/burakince/git-aimit && brew install git-aimit`. The formula builds from the source tarball using `go build`; the `url` and `sha256` lines are patched automatically by the release workflow on each new tag.
