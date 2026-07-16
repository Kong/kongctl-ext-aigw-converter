package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunConvertsStdinToKongctl(t *testing.T) {
	input := `
_format_version: "3.0"
services: []
`
	var stdout, stderr bytes.Buffer

	err := run([]string{
		"--from", "deck",
		"--to", "kongctl",
		"--gateway-name", "support-ai",
	}, bytes.NewBufferString(input), &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ai_gateways:")
	require.Contains(t, stdout.String(), "name: support-ai")
	require.Empty(t, stderr.String())
}

func TestRunWritesOutputFile(t *testing.T) {
	input := filepath.Join(t.TempDir(), "deck.yaml")
	output := filepath.Join(t.TempDir(), "aigw.yaml")
	require.NoError(t, os.WriteFile(input, []byte(`_format_version: "3.0"`), 0o644))

	var stdout, stderr bytes.Buffer
	err := run([]string{
		input,
		"--from=deck",
		"--to=kongctl",
		"--gateway-name=support-ai",
		"--output-file", output,
	}, bytes.NewBuffer(nil), &stdout, &stderr)

	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(data), "ai_gateways:")
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRunRestrictsExistingOutputFilePermissions(t *testing.T) {
	output := filepath.Join(t.TempDir(), "aigw.yaml")
	require.NoError(t, os.WriteFile(output, []byte("old content"), 0o644))
	require.NoError(t, os.Chmod(output, 0o644))

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--from", "deck",
		"--to", "kongctl",
		"--gateway-name", "support-ai",
		"--output-file", output,
	}, bytes.NewBufferString(`_format_version: "3.0"`), &stdout, &stderr)

	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRunRequiresDirectionAndGateway(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--from", "deck", "--to", "kongctl"}, bytes.NewBufferString(""), &stdout, &stderr)
	require.ErrorContains(t, err, "--gateway-name is required")
}

func TestRunIgnoresKongctlOutputContext(t *testing.T) {
	contextPath := filepath.Join(t.TempDir(), "context.json")
	require.NoError(t, os.WriteFile(contextPath, []byte(`{
  "resolved": {"output": "yaml"},
  "output": {"format": "yaml"}
}`), 0o600))
	t.Setenv("KONGCTL_EXTENSION_CONTEXT", contextPath)

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--from", "deck",
		"--to", "kongctl",
		"--gateway-name", "support-ai",
	}, bytes.NewBufferString(`_format_version: "3.0"`), &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ai_gateways:")
	require.Empty(t, stderr.String())
}

func TestRunShowsHelpWithNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, bytes.NewBufferString(""), &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Usage:")
	require.Contains(t, stdout.String(), "kongctl convert ai-gateway <file>")
	require.Contains(t, stdout.String(), "--gateway-display-name string")
	require.Contains(t, stdout.String(), "--label-tag-prefix string")
	require.Empty(t, stderr.String())
}

func TestRunShowsHelpWhenCombinedWithOtherArguments(t *testing.T) {
	for _, helpFlag := range []string{"--help", "-h"} {
		t.Run(helpFlag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run([]string{"missing.yaml", helpFlag}, bytes.NewBufferString(""), &stdout, &stderr)

			require.NoError(t, err)
			require.Contains(t, stdout.String(), "Usage:")
			require.Empty(t, stderr.String())
		})
	}
}

func TestRunVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runCommand(versionCommandID, nil, bytes.NewBuffer(nil), &stdout, &stderr)

	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ai-gateway-converter: (devel)\n")
	require.Contains(t, stdout.String(), "ai-deck-converter: v0.4.0\n")
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

func TestParseArgsAllowsFlagsAfterInput(t *testing.T) {
	opts, err := parseArgs([]string{
		"deck.yaml",
		"--from", "deck",
		"--to", "kongctl",
		"--gateway-name", "support-ai",
		"--strict",
	})

	require.NoError(t, err)
	require.Equal(t, "deck.yaml", opts.input)
	require.Equal(t, "deck", opts.convert.From)
	require.Equal(t, "kongctl", opts.convert.To)
	require.Equal(t, "support-ai", opts.convert.GatewayName)
	require.True(t, opts.convert.Strict)
}
