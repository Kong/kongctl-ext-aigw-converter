# kongctl AI Gateway converter extension

This repository publishes release artifacts for the `kongctl` AI Gateway
converter extension.

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

## Releases

Each release publishes platform-specific archives that `kongctl install
extension` can consume directly. Archives contain:

```text
kongctl-extension.yaml
README.md
bin/kongctl-ext-ai-gateway-converter
```

The source for these release artifacts is maintained in
`Kong/ai-deck-converter`.
