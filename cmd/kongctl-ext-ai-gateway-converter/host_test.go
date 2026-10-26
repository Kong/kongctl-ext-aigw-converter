package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Run with KONGCTL_BIN pointing to a built host. Each test uses its own
// extension store; no user configuration is changed.
func TestKongctlIntegration(t *testing.T) {
	host := os.Getenv("KONGCTL_BIN")
	if host == "" {
		t.Skip("set KONGCTL_BIN to run host integration tests")
	}
	host, err := filepath.Abs(host)
	require.NoError(t, err)
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "bin", "kongctl-ext-ai-gateway-converter"))
	require.NoError(t, err, "run make build before host integration tests")

	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "KONGCTL_") && !strings.HasPrefix(entry, "XDG_") {
			env = append(env, entry)
		}
	}
	env = append(env, "XDG_CONFIG_HOME="+t.TempDir(), "XDG_DATA_HOME="+t.TempDir(),
		"XDG_CACHE_HOME="+t.TempDir(), "XDG_STATE_HOME="+t.TempDir(), "KONGCTL_NO_TELEMETRY=true")
	invoke := func(args ...string) (string, string, error) {
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, host, args...)
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}
	stdout, stderr, err := invoke("link", "extension", root)
	require.NoError(t, err, "%s\n%s", stdout, stderr)
	stdout, stderr, err = invoke("list", "extensions", "--output=json")
	require.NoError(t, err, "%s", stderr)
	require.Contains(t, stdout, "ai-gateway-converter")
	stdout, stderr, err = invoke("convert", "ai-gateway", "--help")
	require.NoError(t, err, "%s", stderr)
	for _, flag := range []string{
		"--input-format", "--source-name", "--strict", "--format",
		"--operator-gateway", "--k8s-namespace", "--konnect-server-url",
	} {
		require.Contains(t, stdout, flag)
	}
	stdout, stderr, err = invoke("convert", "aigw", "version")
	require.NoError(t, err, "%s", stderr)
	require.Contains(t, stdout, "kong-ai-migration-tool: v0.0.0-20261026163831-b5cc58d56ed6")

	for _, fixture := range []struct {
		format, name, content string
	}{
		{"deck", "deck.yaml", testDeck},
		{"kong-operator", "operator.yaml", testOperator},
		{"terraform", "main.tf", testTerraform},
	} {
		t.Run(fixture.format, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), fixture.name)
			require.NoError(t, os.WriteFile(input, []byte(fixture.content), 0o600))
			for _, inputFormat := range []string{"auto", fixture.format} {
				out := filepath.Join(t.TempDir(), "out")
				stdout, stderr, err := invoke("convert", "aigw", "--input", input,
					"--input-format", inputFormat, "--config", t.TempDir(), "--out", out,
					"--source-name=host-test", "--format=kongctl,deck,kong-operator,terraform",
					"--operator-gateway=onprem", "--k8s-namespace=ai", "--strict=false")
				require.NoError(t, err, "%s\n%s", stdout, stderr)
				require.Empty(t, stdout)
				require.Contains(t, stderr, "migration complete:")
				for _, file := range []string{
					"kongctl/models.yaml", "deck/ai-gateway.yaml",
					"kong-operator/models.yaml", "terraform/models.tf", "migration-report.md",
				} {
					require.FileExists(t, filepath.Join(out, file))
				}
			}
		})
	}
	_, stderr, err = invoke("convert", "ai-gateway", "--input=unused", "--format=invalid")
	require.Error(t, err)
	require.Contains(t, stderr, "unknown output format")
	input := filepath.Join(t.TempDir(), "strict.yaml")
	require.NoError(t, os.WriteFile(input, []byte(testDeck+`
plugins:
- name: ai-proxy
  config:
    route_type: llm/v1/chat
    model: {provider: openai, name: gpt-4o}
`), 0o600))
	out := filepath.Join(t.TempDir(), "out")
	_, stderr, err = invoke("convert", "ai-gateway", "--input", input,
		"--config", t.TempDir(), "--out", out, "--strict")
	require.Error(t, err)
	require.Contains(t, stderr, "strict mode:")
	require.NoDirExists(t, out)
}

const testOperator = `apiVersion: configuration.konghq.com/v1alpha1
kind: KongService
metadata:
  name: openai
spec:
  host: api.openai.com
---
apiVersion: configuration.konghq.com/v1alpha1
kind: KongRoute
metadata:
  name: chat
  annotations:
    konghq.com/plugins: ai
spec:
  serviceRef:
    type: namespacedRef
    namespacedRef:
      name: openai
  paths: [/chat]
---
apiVersion: configuration.konghq.com/v1
kind: KongPlugin
metadata:
  name: ai
plugin: ai-proxy
config:
  route_type: llm/v1/chat
  model: {provider: openai, name: gpt-4o}
  auth:
    header_name: Authorization
    header_value: '{vault://env/openai-key}'
`

const testTerraform = `resource "konnect_gateway_service" "openai" {
  name = "openai"
  host = "api.openai.com"
}
resource "konnect_gateway_route" "chat" {
  name = "chat"
  paths = ["/chat"]
  service = { id = konnect_gateway_service.openai.id }
}
resource "konnect_gateway_plugin_ai_proxy" "ai" {
  route = { id = konnect_gateway_route.chat.id }
  config = {
    route_type = "llm/v1/chat"
    model = { provider = "openai", name = "gpt-4o" }
    auth = {
      header_name = "Authorization"
      header_value = "{vault://env/openai-key}"
    }
  }
}
`
