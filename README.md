# kongctl AI Gateway converter extension

This repository contains and publishes the `kongctl` AI Gateway converter
extension. It uses
[`Kong/ai-deck-converter`](https://github.com/Kong/ai-deck-converter) as its
conversion library.

The extension adds:

```sh
kongctl convert ai-gateway
```

Use it to convert AI Gateway configuration between Kong Gateway decK YAML and
the kongctl AI Gateway declarative format.

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

Convert a Kong Gateway 3.x decK file into kongctl AI Gateway declarative YAML:

```sh
kongctl convert ai-gateway deck.yaml \
  --from deck \
  --to kongctl \
  --gateway-name support-ai \
  --output-file aigw.yaml
```

Convert a kongctl AI Gateway declarative file back into Kong Gateway decK YAML:

```sh
kongctl convert ai-gateway aigw.yaml \
  --from kongctl \
  --to deck \
  --gateway-name support-ai \
  --output-file deck.yaml
```

## Development

Build the extension runtime before linking it. kongctl extensions run an
existing executable; they do not compile source during install or link.

```sh
make test
make build
kongctl link extension .
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
GitHub build provenance for each archive. After downloading an archive, verify
its provenance with:

```sh
gh attestation verify \
  kongctl-ext-ai-gateway-converter-linux-amd64.tar.gz \
  --repo Kong/kongctl-ext-aigw-converter
```
