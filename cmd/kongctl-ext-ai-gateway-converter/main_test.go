package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kong/kong-ai-migration-tool/migrate"

	"github.com/stretchr/testify/require"
)

const testDeck = `
_format_version: "3.0"
services:
  - name: openai-chat
    url: https://api.openai.com
    routes:
      - name: openai-chat
        paths: [/chat]
        plugins:
          - name: ai-proxy
            config:
              route_type: llm/v1/chat
              model:
                provider: openai
                name: gpt-4o
              auth:
                header_name: Authorization
                header_value: "{vault://env/openai-key}"
`

func TestRunMigratesIntoOutputDirectory(t *testing.T) {
	input := filepath.Join(t.TempDir(), "deck.yaml")
	config := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.WriteFile(input, []byte(testDeck), 0o600))

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--input", input,
		"--config", config,
		"--out", out,
	}, &stdout, &stderr)

	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "migration complete: wrote output to "+out)
	require.Contains(t, stderr.String(), "findings:")
	require.Contains(t, stderr.String(), filepath.Join(out, "migration-report.md"))
	require.FileExists(t, filepath.Join(out, "migration-report.json"))
	require.FileExists(t, filepath.Join(out, "gateway.yaml"))
	require.FileExists(t, filepath.Join(out, "models.yaml"))
	require.FileExists(t, filepath.Join(out, "providers.yaml"))

	models, err := os.ReadFile(filepath.Join(out, "models.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(models), "ai_gateway_models:")
	require.Contains(t, string(models), "ai_gateway: !ref ai-gateway-deck#id")
	require.Contains(t, string(models), "targets:")
}

func TestRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--out", t.TempDir()}, &stdout, &stderr)
	require.ErrorContains(t, err, "--input is required")
}

func TestRunRejectsFlagWithoutValue(t *testing.T) {
	_, err := parseArgs([]string{"--input", "--out", t.TempDir()})

	require.ErrorContains(t, err, "flag --input requires a value")
}

func TestRunShowsHelpWithNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Usage:")
	require.Contains(t, stdout.String(), "--namespace-prefix string")
	require.Empty(t, stderr.String())
}

func TestRunRejectsRemovedAndPositionalArguments(t *testing.T) {
	tests := [][]string{
		{"deck.yaml"},
		{"--from", "deck"},
		{"--to", "kongctl"},
		{"--gateway-name", "support-ai"},
		{"--output-file", "out.yaml"},
	}
	for _, args := range tests {
		_, err := parseArgs(args)
		require.Error(t, err, "args: %v", args)
	}
}

func TestParseArgsSupportsMigrationFlags(t *testing.T) {
	opts, err := parseArgs([]string{
		"--input=deck.yaml",
		"--config", "./manual",
		"--ref=./schema.json",
		"--out", "./generated",
		"--label-tag-prefix=aigw/",
		"--namespace-prefix", "support-ai",
		"--input-format=terraform",
		"--source-name", "support",
		"--format=kongctl,deck,kong-operator,terraform",
		"--operator-gateway", "onprem",
		"--k8s-namespace", "ai",
		"--konnect-server-url=eu.api.konghq.com",
		"--strict",
	})

	require.NoError(t, err)
	require.Equal(t, "deck.yaml", opts.migrate.InputPath)
	require.Equal(t, "./manual", opts.migrate.ConfigDir)
	require.Equal(t, "./schema.json", opts.migrate.RefPath)
	require.Equal(t, "./generated", opts.migrate.OutDir)
	require.Equal(t, "aigw/", opts.migrate.LabelTagPrefix)
	require.Equal(t, "support-ai", opts.migrate.NamespacePrefix)
	require.Equal(t, migrate.InputTerraform, opts.migrate.InputFormat)
	require.Equal(t, "support", opts.migrate.SourceName)
	require.Equal(t, []migrate.Format{
		migrate.FormatKongctl, migrate.FormatDeck,
		migrate.FormatKongOperator, migrate.FormatTerraform,
	}, opts.migrate.Formats)
	require.Equal(t, migrate.OperatorGatewayOnPrem, opts.migrate.OperatorGateway)
	require.Equal(t, "ai", opts.migrate.K8sNamespace)
	require.Equal(t, "eu.api.konghq.com", opts.migrate.KonnectServerURL)
	require.True(t, opts.migrate.Strict)
}

func TestParseArgsUsesMigrationDefaults(t *testing.T) {
	opts, err := parseArgs([]string{"--input", "deck.yaml"})

	require.NoError(t, err)
	require.Equal(t, "./config", opts.migrate.ConfigDir)
	require.Empty(t, opts.migrate.RefPath)
	require.Equal(t, "./out", opts.migrate.OutDir)
	require.Equal(t, "ai-gateway", opts.migrate.NamespacePrefix)
	require.Empty(t, opts.migrate.InputFormat)
	require.Empty(t, opts.migrate.SourceName)
	require.Equal(t, []migrate.Format{migrate.FormatKongctl}, opts.migrate.Formats)
	require.Equal(t, migrate.OperatorGatewayKonnect, opts.migrate.OperatorGateway)
	require.Empty(t, opts.migrate.K8sNamespace)
	require.Equal(t, migrate.DefaultKonnectServerURL, opts.migrate.KonnectServerURL)
	require.False(t, opts.migrate.Strict)
}

func TestRunVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCommand(versionCommandID, nil, &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ai-gateway-converter: (devel)\n")
	require.Contains(t, stdout.String(), "kong-ai-migration-tool:")
	require.Empty(t, stderr.String())
}

func TestMatchedCommandID(t *testing.T) {
	contextPath := filepath.Join(t.TempDir(), "context.json")
	require.NoError(t, os.WriteFile(contextPath, []byte(`{
  "matched_command_path": {"id": "convert_ai_gateway_version"}
}`), 0o600))
	t.Setenv(extensionContextEnv, contextPath)

	commandID, err := matchedCommandID()

	require.NoError(t, err)
	require.Equal(t, versionCommandID, commandID)
}

func TestParseArgsRejectsInvalidMigrationOptions(t *testing.T) {
	for _, args := range [][]string{
		{"--input-format=unknown"},
		{"--format=unknown"},
		{"--operator-gateway=unknown"},
		{"--strict=maybe"},
		{"--strict", "false"},
		{"--input-format"},
		{"--format"},
	} {
		_, err := parseArgs(append([]string{"--input=deck.yaml"}, args...))
		require.Error(t, err, "args: %v", args)
	}
}

func TestParseArgsStrictBoolean(t *testing.T) {
	for _, value := range []string{"false", "0", "true", "1"} {
		opts, err := parseArgs([]string{"--input=deck.yaml", "--strict=" + value})
		require.NoError(t, err)
		require.Equal(t, value == "true" || value == "1", opts.migrate.Strict)
	}
	opts, err := parseArgs([]string{"--strict", "--strict=false", "--input=deck.yaml"})
	require.NoError(t, err)
	require.False(t, opts.migrate.Strict)
}

func TestRunMultipleOutputFormats(t *testing.T) {
	input := filepath.Join(t.TempDir(), "deck.yaml")
	require.NoError(t, os.WriteFile(input, []byte(testDeck), 0o600))
	out := filepath.Join(t.TempDir(), "out")
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--input", input, "--config", t.TempDir(), "--out", out,
		"--format=kongctl,deck,kong-operator,terraform", "--source-name=support",
		"--operator-gateway=onprem", "--k8s-namespace=ai",
		"--konnect-server-url=eu.api.konghq.com",
	}, &stdout, &stderr)
	require.NoError(t, err)
	require.Empty(t, stdout.String())
	for _, file := range []string{
		"kongctl/models.yaml", "deck/ai-gateway.yaml",
		"kong-operator/models.yaml", "terraform/models.tf", "migration-report.md",
	} {
		require.FileExists(t, filepath.Join(out, file))
	}
	gateway, err := os.ReadFile(filepath.Join(out, "kong-operator", "gateway.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(gateway), "OnPremAIGateway")
	require.Contains(t, string(gateway), "namespace: ai")
}

func TestRunStrictFindings(t *testing.T) {
	// A globally scoped AI plugin cannot be represented in AI Gateway 2.x.
	input := filepath.Join(t.TempDir(), "deck.yaml")
	deck := testDeck + `
plugins:
- name: ai-proxy
  config:
    route_type: llm/v1/chat
    model: {provider: openai, name: gpt-4o}
`
	require.NoError(t, os.WriteFile(input, []byte(deck), 0o600))
	for _, strict := range []bool{false, true} {
		out := filepath.Join(t.TempDir(), "out")
		args := []string{"--input", input, "--config", t.TempDir(), "--out", out}
		if strict {
			args = append(args, "--strict")
		}
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		if strict {
			require.Error(t, err)
			require.NoDirExists(t, out)
			require.NotContains(t, stderr.String(), "migration complete:")
		} else {
			require.NoError(t, err)
			require.Contains(t, stderr.String(), "error:")
			require.Contains(t, stderr.String(), "findings:")
			require.FileExists(t, filepath.Join(out, "migration-report.md"))
		}
	}
}
