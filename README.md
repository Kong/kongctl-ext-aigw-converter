# kongctl AI Gateway converter extension

This repository contains and publishes the `kongctl` AI Gateway converter
extension. It uses the [Kong AI migration tool][migration-tool] as its migration
library.

The extension adds:

```sh
kongctl convert ai-gateway
```

Use it to migrate Kong Gateway 3.x decK, Kong Operator, or Terraform
configuration into AI Gateway resources. The default output is a directory of
kongctl declarative resources; decK, Kong Operator, and Terraform outputs are
also available.

## Install

Install the latest compatible release:

```sh
kongctl install extension Kong/kongctl-ext-aigw-converter
```

Install the opt-in preview release:

```sh
kongctl install extension Kong/kongctl-ext-aigw-converter@v0.6.0-rc.1
```

Prereleases require an explicit version. Unversioned installs continue to use
GitHub's latest stable release.

## Usage

Choose the output format for the destination where you will apply the
migration. Input and output formats are independent: decK input does not
select decK output. `--input-format` defaults to automatic detection, while
`--format` defaults to `kongctl` regardless of the input format.

Conversion runs locally and does not require a Konnect login. Before applying
any generated configuration, review `migration-report.md`, resolve error
findings, and populate required secrets.

### Konnect

Generate kongctl resources for a Konnect-managed AI Gateway:

```sh
kongctl convert ai-gateway \
  --input kong.yaml \
  --config ./config \
  --format kongctl \
  --out ./out
```

Authenticate with Konnect using `kongctl login` or a configured personal
access token before applying the reviewed output:

```sh
kongctl apply -f ./out
```

### Self-managed AI Gateway

For a self-managed (on-premises) AI Gateway, explicitly select decK output:

```sh
kongctl convert ai-gateway \
  --input kong.yaml \
  --config ./config \
  --format deck \
  --out ./out
```

Apply the reviewed output using decK, configured to connect to your AI
Gateway:

```sh
deck ai sync ./out/ai-gateway.yaml
```

`kongctl apply` targets Konnect and cannot apply this migration to a
self-managed AI Gateway. A Konnect login is not needed for this decK flow.

### Options

`--input` is required and accepts a decK file, Kong Operator manifest file or
directory, or Terraform file or directory. The remaining flags are optional:

- `--input-format` selects `auto` (default), `deck`, `kong-operator`, or
  `terraform`. Auto detection uses the input contents.
- `--config` selects the manual migration config directory (default
  `./config`).
- `--ref` overrides the bundled AI Gateway OpenAPI schema used to validate
  vaults.
- `--out` selects the generated resource directory (default `./out`).
- `--label-tag-prefix` selects the tag prefix lifted into labels.
- `--namespace-prefix` selects the generated kongctl namespace prefix (default
  `ai-gateway`).
- `--source-name` identifies the migrated control plane (default: input
  filename). It distinguishes generated gateway names and references.
- `--strict` fails on migration findings instead of collecting them. Use
  `--strict=false` to disable it explicitly.
- `--format` selects comma-separated output formats: `kongctl` (default),
  `deck`, `kong-operator`, and `terraform`.
- `--operator-gateway` selects `konnect` (default) or `onprem` for Kong
  Operator output.
- `--k8s-namespace` sets the namespace in Kong Operator manifests.
- `--konnect-server-url` sets the Konnect API host for Operator and Terraform
  output (default `us.api.konghq.com`).

For example, migrate Terraform input to kongctl resources:

```sh
kongctl convert ai-gateway \
  --input ./terraform --input-format terraform --out ./out
```

Generate several output formats from Kong Operator input:

```sh
kongctl convert ai-gateway \
  --input ./manifests --input-format kong-operator \
  --format kongctl,deck,kong-operator,terraform \
  --operator-gateway onprem --k8s-namespace ai --out ./out
```

With one format, resources are written directly into `--out`. With several,
each format is written into `--out/<format>`. Shared migration reports and
secret scaffolding are written at the root of `--out`.

The default kongctl output includes a managed AI Gateway and separate files
for each non-empty child resource kind. decK output contains a single
`ai-gateway.yaml` file for `deck ai sync`.

Compared with v0.5.0, generated gateways now incorporate the source name in
both their reference and display name. For an existing migration, supply your
previous `gateway.yaml` in `--config` to preserve its identity, and review the
plan before applying.

Read `migration-report.md` before applying. It describes changes to client
URLs and request bodies, skipped resources, and required secret values.
Findings retain their `error`, `warn`, or `info` severity on stderr, followed
by a count summary. A successful non-strict conversion can include error
findings; resolve them before applying. Strict mode turns findings into
conversion failures according to the migration library's rules.

The updated library also writes secret scaffolding where needed. Populate the
required values before applying and treat generated secret files as sensitive.
For several formats, apply the appropriate subdirectory for the destination:
`kongctl apply -f ./out/kongctl` for Konnect, or
`deck ai sync ./out/deck/ai-gateway.yaml` for a self-managed AI Gateway.

Show the extension and migration library versions:

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

To run the optional host integration tests against a built kongctl executable:

```sh
KONGCTL_BIN=/absolute/path/to/kongctl go test ./...
```

These tests use isolated configuration and exercise linking, help, version,
aliases, all input formats, output selection, and failure propagation.

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
repository. Tags with a prerelease suffix, such as `v0.6.0-rc.1`, publish as
GitHub prereleases with `latest` explicitly disabled. Stable tags retain the
normal latest-release behavior.

Releases include per-platform SPDX SBOMs, SHA-256 checksums, and
GitHub build provenance for each archive. CI and release workflows require the
`GH_TOKEN_PRIVATE_READ` secret to read the migration library. After downloading
an archive, verify its provenance with:

```sh
gh attestation verify \
  kongctl-ext-ai-gateway-converter-linux-amd64.tar.gz \
  --repo Kong/kongctl-ext-aigw-converter
```

[migration-tool]: https://github.com/Kong/kong-ai-migration-tool
