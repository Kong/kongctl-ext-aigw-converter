package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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
	out := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.WriteFile(input, []byte(testDeck), 0o600))

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--input", input,
		"--config", filepath.Join(t.TempDir(), "missing-config"),
		"--out", out,
	}, &stdout, &stderr)

	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "migration complete: wrote output to "+out)
	require.FileExists(t, filepath.Join(out, "gateway.yaml"))
	require.FileExists(t, filepath.Join(out, "models.yaml"))
	require.FileExists(t, filepath.Join(out, "providers.yaml"))

	models, err := os.ReadFile(filepath.Join(out, "models.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(models), "ai_gateway_models:")
	require.Contains(t, string(models), "ai_gateway: !ref ai-gateway#id")
	require.Contains(t, string(models), "targets:")
}

func TestRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--out", t.TempDir()}, &stdout, &stderr)
	require.ErrorContains(t, err, "--input is required")
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
		{"--strict"},
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
	})

	require.NoError(t, err)
	require.Equal(t, "deck.yaml", opts.migrate.InputPath)
	require.Equal(t, "./manual", opts.migrate.ConfigDir)
	require.Equal(t, "./schema.json", opts.migrate.RefPath)
	require.Equal(t, "./generated", opts.migrate.OutDir)
	require.Equal(t, "aigw/", opts.migrate.LabelTagPrefix)
	require.Equal(t, "support-ai", opts.migrate.NamespacePrefix)
}

func TestParseArgsUsesMigrationDefaults(t *testing.T) {
	opts, err := parseArgs([]string{"--input", "deck.yaml"})

	require.NoError(t, err)
	require.Equal(t, "./config", opts.migrate.ConfigDir)
	require.Empty(t, opts.migrate.RefPath)
	require.Equal(t, "./out", opts.migrate.OutDir)
	require.Equal(t, "ai-gateway", opts.migrate.NamespacePrefix)
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
