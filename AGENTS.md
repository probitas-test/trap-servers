# Trap Servers

Trap servers for capturing and inspecting SMTP emails and HTTP requests. Built
for testing [Probitas](https://github.com/probitas-test/probitas) and other
client implementations.

## How it is built

- **Test first.** Red → green → refactor. Write the failing test before the
  code that makes it pass.
- **Each layer holds one thing.** Put a fact in the one layer that owns it, and
  nowhere else.

  | Layer                        | Holds                                                              |
  | ---------------------------- | ------------------------------------------------------------------ |
  | `docs/adr/`                  | one decision and why it was made                                   |
  | `docs/spec/`                 | the behaviour users and other components can rely on               |
  | `trap-*/handlers/doc/api.md` | each server's API reference, served by the server itself           |
  | Tests                        | the details that pin the behaviour down: rules, boundaries, tables |
  | Code                         | how it is done                                                     |
  | Commits                      | why a change was made                                              |
  | Comments                     | why it is not done another way                                     |

  Before writing, fixing or reviewing a record, read `docs/adr/README.md`;
  before writing, fixing or reviewing a spec, read `docs/spec/README.md`.
  Follow their writing order.
- **Dependencies are current.** Before adding or bumping one, look up its
  latest release in the registry. Do not copy a version from memory or from
  another repository.

## Quality gate

Run `just ci` before reporting a change as done. It runs each server's
`lint`, `fmt-check`, `test` and `build`, then `just fmt-check`, which checks
the formatting that spans the repository (Markdown, JSON and YAML with dprint,
Nix with nixfmt). CI runs the same recipes: the per-server build workflows run
`just <server>::ci`, and the Format workflow runs `just fmt-check`.

## Layout

| Path             | What lives there                                              |
| ---------------- | ------------------------------------------------------------- |
| `trap-smtp/`     | SMTP trap server: SMTP backend, store, HTTP API and web UI    |
| `trap-webhook/`  | Webhook trap server: HTTP capture, store, HTTP API and web UI |
| `docs/`          | ADRs and specs                                                |
| `.github/`       | per-server build and Docker workflows                         |
| `.claude/rules/` | rules that make the ADR and spec procedures get read          |

Each server is its own Go module with its own `justfile` (`lint`, `fmt`,
`fmt-check`, `test`, `build`, `run`, `ci`), `Dockerfile` and
`.golangci.yml`; the root `justfile` runs a recipe across all of them.
Images are published to `ghcr.io/probitas-test/trap-*`.

## Language

- README, docs and commit messages: English
- Code comments, log and error messages: English

## Commits

[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), as
described in `.gitmessage` (`git config commit.template .gitmessage`):

```
<type>[optional scope]: <description>
```

- Types: `feat`, `fix`, `deps`, `refactor`, `test`, `docs`, `style`, `chore`.
- The scope names the server the change is in (`fix(trap-smtp): …`); leave it
  out for a change that spans both servers or the repository (`feat: …`).
- A breaking change carries `!` after the type or scope
  (`feat(trap-smtp)!: …`).
