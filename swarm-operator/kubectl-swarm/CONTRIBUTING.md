# Contributing to kubectl-swarm

Thank you for your interest in contributing to kubectl-swarm!

## Development Setup

### Prerequisites

- Go 1.21+
- kubectl
- Access to a Kubernetes cluster (kind, minikube, or real cluster)
- Swarm CRDs installed

### Building from Source

```bash
# Clone the repository
git clone https://github.com/claude-flow/kubectl-swarm.git
cd kubectl-swarm

# Install dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Install locally
make install
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires cluster)
make test-integration

# Linting
make lint
```

## Making Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to your fork (`git push origin feature/amazing-feature`)
8. Create a Pull Request

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes
- Add comments for exported functions
- Keep functions focused and testable

## Adding New Commands

When adding a new command:

1. Create a new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Implement the command following the existing pattern
3. Add the command to `cmd/root.go`
4. Add tests in `cmd/newcommand_test.go`
5. Update README.md with usage examples
6. Add shell completion support

## Testing Guidelines

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Test error cases
- Aim for >80% code coverage

## Documentation

- Update README.md for user-facing changes
- Add inline documentation for complex logic
- Include examples for new features
- Update command help text

## Release Process

1. Update version in Makefile
2. Update CHANGELOG.md
3. Create git tag: `git tag v0.1.0`
4. Push tag: `git push origin v0.1.0`
5. GitHub Actions will build and release

## Questions?

Feel free to open an issue for questions or discussions!