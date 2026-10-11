# Run `nix develop` (or use direnv) first so every tool is on PATH.

set shell := ["bash", "-euo", "pipefail", "-c"]

mod trap-webhook
mod trap-smtp

[private]
default:
    @just --list

# Run linter on all packages
lint: trap-webhook::lint trap-smtp::lint

# Run tests on all packages
test: trap-webhook::test trap-smtp::test

# Build all packages
build: trap-webhook::build trap-smtp::build

# Format all code (Go + Markdown/JSON/YAML + Nix)
fmt: trap-webhook::fmt trap-smtp::fmt
    dprint fmt
    git ls-files -z '*.nix' | xargs -0 nixfmt

# Clean all packages
clean: trap-webhook::clean trap-smtp::clean

# Tidy all packages
tidy: trap-webhook::tidy trap-smtp::tidy

# Check the formatting that spans the repository: Markdown/JSON/YAML (dprint)
# and Nix. Each server checks its Go formatting in its own fmt-check.
fmt-check:
    dprint check
    git ls-files -z '*.nix' | xargs -0 nixfmt --check

# Everything CI runs: each server's lint, fmt-check, test and build, plus the
# repository-wide formatting
ci: trap-webhook::ci trap-smtp::ci fmt-check
