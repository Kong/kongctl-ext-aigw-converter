# Contributing

This repository publishes release artifacts for the kongctl AI Gateway converter
extension.

## Where to Contribute

For code, conversion behavior, tests, release workflow changes, or extension
manifest changes, open an issue so maintainers can route the work to the
appropriate source repository.

Use this repository for issues specific to released extension artifacts, such
as:

- missing or incorrect release assets
- install failures from `kongctl install extension Kong/kongctl-ext-aigw-converter`
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

Kong-maintained automation is responsible for building and publishing these
assets.

## Security

For security vulnerabilities, see [SECURITY.md](SECURITY.md).

## License

By contributing to this project, you agree that your contributions will be
licensed under its Apache 2.0 License.
