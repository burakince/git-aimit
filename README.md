# git-aimit

AI-powered Git extension that reads your staged changes and generates a
[Conventional Commits](https://www.conventionalcommits.org/) message using a
local [Ollama](https://ollama.com/) model. Once the binary is on your `PATH`
as `git-aimit`, Git exposes it as `git aimit`.

---

## What it does

1. Reads `git diff --cached` (staged changes only).
2. Sends the diff to a locally running Ollama model with a Conventional Commits
   prompt.
3. Prints the generated message for you to review.
4. Asks for confirmation — if you say **y**, it runs `git commit -m <message>`.

---

## Installation

```bash
go install github.com/burakince/git-aimit@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`. Once it is, Git will
recognise the binary as a subcommand:

```bash
git aimit        # generate + commit
git aimit init   # configure
```

---

## Setup

Run the interactive setup once before first use:

```bash
git aimit init
```

You will be prompted for:

| Setting | Default |
|---------|---------|
| Ollama base URL | `http://localhost:11434` |
| Model name | `llama3` |

The tool checks whether Ollama is reachable and saves the config to
`~/.config/git-aimit/config.json`:

```json
{
  "provider": "ollama",
  "ollama": {
    "base_url": "http://localhost:11434",
    "model": "llama3"
  }
}
```

---

## Usage

Stage your changes as usual, then run:

```bash
git add <files>
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
