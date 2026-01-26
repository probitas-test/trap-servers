mod trap-webhook
mod trap-smtp

[private]
default:
    @just --list

# Run linter on all packages
lint: trap-webhook::lint trap-smtp::lint
    dprint check

# Run tests on all packages
test: trap-webhook::test trap-smtp::test

# Build all packages
build: trap-webhook::build trap-smtp::build

# Format all code (Go + Markdown/JSON/YAML)
fmt: trap-webhook::fmt trap-smtp::fmt
    dprint fmt

# Clean all packages
clean: trap-webhook::clean trap-smtp::clean

# Tidy all packages
tidy: trap-webhook::tidy trap-smtp::tidy
