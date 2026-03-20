# Contributing to Lore

Thanks for your interest in contributing to Lore! Here's how you can help.

## Reporting Bugs

1. Check [existing issues](https://github.com/GreyCoderK/lore/issues) to avoid duplicates.
2. Open a new issue with:
   - Steps to reproduce
   - Expected vs actual behavior
   - Go version (`go version`) and OS

## Suggesting Features

Open an issue with the `enhancement` label. Describe the use case and why it matters.

## Contributing Code

1. Fork the repo and create a branch from `main`.
2. Write your changes and add tests.
3. Run tests and linting:
   ```bash
   cd lore_cli
   go test ./...
   go vet ./...
   ```
4. Commit with a clear message (e.g., `fix: resolve hook path on Windows`).
5. Open a pull request against `main`.

## Code Style

- Follow standard Go conventions (`gofmt`).
- Keep functions small and focused.
- Add tests for new functionality.

## License

By contributing, you agree that your contributions will be licensed under the [AGPL-3.0](LICENSE).
