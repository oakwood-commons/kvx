---
description: "Go coding conventions for kvx: error handling, design principles, functional options, context/timeouts, and formatting. Use when writing or editing Go code."
applyTo: "**/*.go"
---

# Go Conventions

## Struct Tags

Always add JSON/YAML tags on exported structs used for configuration or data serialization.

## Error Handling

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}
```

## Design Principles

- Accept interfaces, return structs
- Keep interfaces small (1-3 methods)
- Define interfaces where they are used, not where they are implemented

## Dependency Injection

Use constructor functions to inject dependencies:

```go
func NewUserService(repo UserRepository, logger Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}
```

## Functional Options

kvx uses this pattern extensively in `pkg/core.New()`:

```go
type Option func(*Engine)

func WithEvaluator(e Evaluator) Option {
    return func(eng *Engine) { eng.evaluator = e }
}

func New(opts ...Option) (*Engine, error) {
    eng := &Engine{}
    for _, opt := range opts {
        opt(eng)
    }
    return eng, nil
}
```

## Context & Timeouts

Always use `context.Context` for timeout control:

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

## Formatting

- **gofumpt** and **goimports** are mandatory -- no style debates
- Never use magic strings or numbers; always define constants or use settings

## TUI Patterns

- Bubbletea models implement `Init()`, `Update()`, and `View()` -- keep `Update` focused on state transitions and `View` on rendering
- Use lipgloss for all terminal styling -- never raw ANSI codes
- TUI state belongs in `internal/ui/`; reusable library-level TUI entry points in `pkg/tui/`

## Reference

See skill: `golang-patterns` for comprehensive Go idioms and patterns.
