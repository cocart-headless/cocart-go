<!-- CoCart SDK Support Policy Template v1 -->

# Support & Versioning Policy

> **Note:** This SDK is currently in development. The full support lifecycle (maintenance phase for previous major versions, EOL (End-of-Life) grace periods) takes effect once the SDK is declared stable and production-ready.

## Versioning

This SDK follows [Semantic Versioning](https://semver.org/) (SemVer):

- **Major** (X.0.0) — Breaking changes to the public API
- **Minor** (x.Y.0) — New features that are backward-compatible
- **Patch** (x.y.Z) — Bug fixes and security patches

Only the **latest major version** receives active development. Older major versions remain available for install but receive no updates. Migration guides are provided in the `docs/` folder for major version upgrades.

### What constitutes a breaking change

- Removing or renaming an exported type, function, method, or constant
- Changing a function or method signature
- Changing the behavior of an exported function in a way that breaks existing callers
- Bumping the minimum Go version in `go.mod`

### What is NOT a breaking change

- Adding new exported types, functions, or methods
- Adding new fields to option structs
- Internal refactors that do not affect the public API
- Adding a new Go version to the supported matrix
- Bug fixes that correct behavior to match documentation

## SDK Lifecycle

| Phase | Description | Duration |
|---|---|---|
| **Active** | New features, bug fixes, security patches | Current major version |
| **Maintenance** | Security patches and critical bug fixes only | Previous major version, 12 months |
| **Deprecated** | No updates; remains installable | After maintenance ends |

## Supported Go Versions

| Go | Status | SDK Support | Notes |
|---|---|---|---|
| 1.25 | Current | Supported | Tested in CI |
| 1.24 | Supported | Supported | Tested in CI |
| 1.23 | Supported | Minimum version | Required for `iter.Seq2`; tested in CI |
| 1.22 and below | EOL | Not supported | Missing range-over-func support |

### Version support policy

We follow Go's official [release policy](https://go.dev/doc/devel/release#policy) of supporting the **two most recent major releases**.

Because this SDK requires `iter.Seq2` (introduced in Go 1.23), Go 1.23 is the absolute minimum regardless of Go's rolling window. When Go 1.26 releases, we will drop Go 1.23 support (moving the minimum to 1.24).

Go's strong backward compatibility promise means upgrades are low-friction, so no extended grace period is offered after a Go version leaves the supported window.

## Deprecation Notices

We communicate deprecations through:

1. **Doc comments** — `// Deprecated: Use X instead.` comments recognized by `go vet` and IDEs
2. **Changelog entry** — Every deprecation is noted in release notes
3. **Minimum one minor release** — A deprecation notice ships at least one minor version before the deprecated feature is removed
4. **Migration guide** — Major version upgrades include a migration guide in the `docs/` folder

## Getting Help

- **Documentation:** https://cocartapi.com/docs
- **Community:** https://cocartapi.com/community
- **Issues:** https://github.com/cocart-headless/cocart-go/issues
