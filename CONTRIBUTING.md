# Contributing to go-blosc

Thank you for your interest in contributing to go-blosc!

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git

### Getting Started

```bash
git clone https://github.com/mrjoshuak/go-blosc.git
cd go-blosc
go test ./...
```

## Design Philosophy

### Caller Validates, Callee Assumes

Input validation should occur at API boundaries. Internal methods assume valid data has been passed. This approach:

- Reduces redundant validation
- Makes code flow clearer
- Improves performance

### Fail Fast

Return errors for conditions callers can act on (malformed data, missing files). Let programmer errors (violated invariants, nil receivers) panic to expose bugs immediately.

### Performance is a Feature

- Minimize allocations per call
- Validate once at boundaries, not on every internal call
- Use SIMD optimizations where beneficial

## Coding Standards

- Follow Go conventions with `gofmt` formatting
- Prefer short, clear functions
- Use early returns over deep nesting
- Meaningful variable and function names
- Godoc comments for all exported APIs

## Testing

- Write tests for all new functionality
- Cover both success and error paths
- Use table-driven tests where appropriate
- Run with race detector: `go test -race ./...`
- Aim for 95%+ test coverage

### Running Benchmarks

```bash
go test -bench=. -benchmem ./...
```

## Pull Request Guidelines

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Ensure all tests pass
4. Update documentation if needed
5. Submit a pull request

### Commit Messages

- Brief summary (50 chars or less)
- Reference issues when applicable
- Example: `Fix shuffle buffer overflow for small inputs`

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
