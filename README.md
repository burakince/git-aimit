# git-aimit

AI-powered Git extension that reads your staged changes and generates a
[Conventional Commits](https://www.conventionalcommits.org/) message using a
local [Ollama](https://ollama.com/) model. Once the binary is on your `PATH`
as `git-aimit`, Git exposes it as `git aimit`.

---

## What it does

1. Optionally runs `git add -A` for you when `auto_stage` is enabled in config.
2. Reads `git diff --cached` (staged changes only).
3. Sends the diff to a locally running Ollama model with a Conventional Commits
   prompt. Includes an explanatory body paragraph when the diff spans multiple
   packages or bounded contexts.
4. Prints the generated message for you to review.
5. Asks for confirmation — if you say **y**, it runs `git commit -m <message>`.

---

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap burakince/git-aimit https://github.com/burakince/git-aimit
brew install git-aimit
```

Once installed, Git will recognise the binary as a subcommand:

```bash
git aimit        # generate + commit
git aimit init   # configure
```

### Pre-built binaries

Download the binary for your platform from the
[latest release](https://github.com/burakince/git-aimit/releases/latest),
make it executable, and place it somewhere on your `PATH`:

```bash
# Example for macOS Apple Silicon
curl -L https://github.com/burakince/git-aimit/releases/latest/download/git-aimit-darwin-arm64 \
  -o /usr/local/bin/git-aimit
chmod +x /usr/local/bin/git-aimit
```

Available targets: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`,
`windows-amd64.exe`, `windows-arm64.exe`.

### Go install

```bash
go install github.com/burakince/git-aimit@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

---

## Setup

Run the interactive setup once before first use:

```bash
git aimit init
```

You will be prompted for:

| Setting | Default | Description |
|---------|---------|-------------|
| Ollama base URL | `http://localhost:11434` | HTTP endpoint of your Ollama instance |
| Model name | `llama3` | Any model you have pulled locally |
| Auto-stage | `N` | Run `git add -A` automatically before generating the message |

The tool checks whether Ollama is reachable and saves the config to
`~/.config/git-aimit/config.json`:

```json
{
  "provider": "ollama",
  "auto_stage": false,
  "ollama": {
    "base_url": "http://localhost:11434",
    "model": "llama3"
  }
}
```

Set `auto_stage` to `true` to have `git aimit` run `git add -A` automatically
before generating the message, so you don't need to stage changes manually.

---

## Usage

Stage your changes and run:

```bash
git add <files>   # skip this if auto_stage is enabled
git aimit
```

Example session:

```
Generating commit message using ollama (llama3)...

Generated commit message:

feat(auth): add JWT expiry validation

Prevents tokens with expired `exp` claims from being accepted by the
middleware, closing a gap where long-lived tokens remained valid after
the configured TTL had passed.

Commit with this message? [y/N]: y
Committed successfully.
```

---

## Development

**Prerequisites:** Go 1.21+

```bash
# Clone and install dependencies
git clone https://github.com/burakince/git-aimit.git
cd git-aimit
go mod tidy

# Build
go build -o git-aimit .

# Run all tests
go test ./...

# Run a specific test
go test ./internal/config/... -run TestSaveAndLoad
go test ./internal/llm/ollama/... -run TestGenerateCommitMessage

# Vet
go vet ./...
```

Unit tests use only the standard library (`testing`, `net/http/httptest`) — no
external test dependencies or running Ollama instance required.

### Evals

Evals test actual model output quality against a live Ollama instance. They are
excluded from `go test ./...` and must be opted into explicitly:

```bash
go test -tags evals -v ./evals/
```

Override the endpoint or model with environment variables:

```bash
OLLAMA_BASE_URL=http://localhost:11434 OLLAMA_MODEL=llama3 \
  go test -tags evals -v ./evals/
```

Each eval runs a fixture diff through the real model and checks the result
against named criteria (Conventional Commits format, 72-character subject line,
body paragraph present for complex multi-package diffs, no code fences). Evals
skip automatically when Ollama is not reachable, so they are safe to run in any
environment and will simply report `SKIP` when the model is unavailable.

---

## Adding a new LLM provider

1. Create a new package under `internal/llm/`, e.g. `internal/llm/openai/`.
2. Implement the `Provider` interface defined in `internal/llm/provider.go`:

```go
type Provider interface {
    GenerateCommitMessage(ctx context.Context, diff string) (string, error)
}
```

3. Add a new case to the `switch cfg.Provider` block in `cmd/root.go`.
4. Add the provider name and its config fields to the `Config` struct in
   `internal/config/config.go` and update `git aimit init` accordingly.

---

## Requirements

- Go 1.21+
- [Ollama](https://ollama.com/) running locally with at least one model pulled
  (`ollama pull llama3`)
