# kongctl AI Gateway converter extension

This repository contains and publishes the `kongctl` AI Gateway converter
extension. It uses
`Kong/kong-ai-migration-tool` as its migration library.

The extension adds:

```sh
kongctl convert ai-gateway
```

Use it to migrate Kong Gateway 3.x decK YAML into a directory of kongctl AI
Gateway declarative resources.

## Install

Install the latest compatible release:

```sh
kongctl install extension Kong/kongctl-ext-aigw-converter
```

Install a specific release:

```sh
kongctl install extension Kong/kongctl-ext-aigw-converter@v0.1.0
```

## Usage

Migrate a Kong Gateway 3.x decK file:

```sh
kongctl convert ai-gateway \
  --input kong.yaml \
  --config ./config \
  --out ./out
```

`--input` is required. The remaining flags are optional:

- `--config` selects the manual migration config directory (default
  `./config`).
- `--ref` overrides the bundled AI Gateway OpenAPI schema used to validate
  vaults.
- `--out` selects the generated resource directory (default `./out`).
- `--label-tag-prefix` selects the tag prefix lifted into labels.
- `--namespace-prefix` selects the generated kongctl namespace prefix (default
  `ai-gateway`).

The output includes a managed AI Gateway and separate files for each non-empty
child resource kind. Apply the complete migration in one operation:

```sh
kongctl apply -f ./out
```

Show the extension and embedded conversion library versions:

```sh
kongctl convert ai-gateway version
```

## Development

Build the extension runtime before linking it. kongctl extensions run an
existing executable; they do not compile source during install or link.

```sh
make test
make build
kongctl link extension .
```

The migration library is currently private. Local builds require GitHub read
access and these Go settings:

```sh
export GOPRIVATE=github.com/Kong/kong-ai-migration-tool
export GONOSUMDB=github.com/Kong/kong-ai-migration-tool
```

The extension manifest is [`kongctl-extension.yaml`](kongctl-extension.yaml).
The runtime is built at `bin/kongctl-ext-ai-gateway-converter`.

## Releases

Each release publishes platform-specific archives that `kongctl install
extension` can consume directly. Archives contain:

```text
kongctl-extension.yaml
README.md
bin/kongctl-ext-ai-gateway-converter
```

Pushing a `v*` tag builds and publishes the release artifacts from this
repository. Releases include per-platform SPDX SBOMs, SHA-256 checksums, and
GitHub build provenance for each archive. CI and release workflows require the
`GH_TOKEN_PRIVATE_READ` secret to read the migration library. After downloading
an archive, verify its provenance with:

```sh
gh attestation verify \
  kongctl-ext-ai-gateway-converter-linux-amd64.tar.gz \
  --repo Kong/kongctl-ext-aigw-converter
```
