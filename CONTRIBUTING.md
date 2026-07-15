# Contributing

This repository contains and publishes the kongctl AI Gateway converter
extension.

## Where to Contribute

Use this repository for extension command behavior, the kongctl format adapter,
tests, manifest changes, release automation, and released artifacts. For changes
to the underlying conversion engine, contribute to
[`Kong/ai-deck-converter`](https://github.com/Kong/ai-deck-converter).

Useful issue reports include:

- missing or incorrect release assets
- install failures from
  `kongctl install extension Kong/kongctl-ext-aigw-converter`
- release metadata, checksums, or repository documentation problems

## Reporting Issues

When reporting an install or release artifact issue, include:

- the release version or command used
- operating system and architecture
- `kongctl` version
- the exact error output
- steps to reproduce

## Release Artifacts

Each release should publish platform-specific archives that `kongctl install
extension` can consume directly. Archives must contain:

```text
kongctl-extension.yaml
README.md
bin/kongctl-ext-ai-gateway-converter
```

The release workflow in this repository builds and publishes these assets when
a `v*` tag is pushed. It also publishes per-platform SPDX SBOMs, SHA-256
checksums, and GitHub build provenance for each archive.

## Security

For security vulnerabilities, see [SECURITY.md](SECURITY.md).

## License

By contributing to this project, you agree that your contributions will be
licensed under its Apache 2.0 License.
