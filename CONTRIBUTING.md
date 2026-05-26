# Contributing

Thanks for considering contributing to Gherkio!

## How to contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes (`git commit -am 'feat: add something'`)
4. Push to the branch (`git push origin feat/my-feature`)
5. Open a Pull Request

## Commit style

Please use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add retry backoff strategy
fix: handle empty response body
docs: update README with timing examples
refactor: extract assertion evaluator
test: add schema validation tests
```

## Development

```bash
go build -o gherkio .                   # Build binary
go test ./...                            # Run all tests
go run . run <test-file>                 # Run a test scenario
go run . run <test-file> --verbose       # With full payloads
```

## Code style

- Follow standard Go formatting (`gofmt` / `go fmt`)
- Wrap errors with context: `fmt.Errorf("doing x: %w", err)`
- No panics in production paths
- Keep the DSL declarative — no arbitrary code execution in the YAML layer

## Pull Request checklist

- [ ] Tests pass (`go test ./...`)
- [ ] New features include tests
- [ ] Documentation updated if needed
- [ ] Commit messages follow Conventional Commits
