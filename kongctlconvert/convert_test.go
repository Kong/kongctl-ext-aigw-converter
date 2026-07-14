package kongctlconvert

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDeckToKongctlWrapsGatewayAndAdaptsFields(t *testing.T) {
	src := []byte(`
_format_version: "3.0"
consumers:
  - username: gregs-dev
    groups:
      - premium-users
    keyauth_credentials:
      - key: secret-abc
        ttl: 3600
consumer_groups:
  - name: premium-users
services:
  - name: ai-gateway
    url: http://ai-gateway.upstream.local
    routes:
      - name: openai-generate
        paths:
          - /ai/chat
        plugins:
          - name: ai-proxy-advanced
            config:
              llm_format: openai
              route_type: llm/v1/chat
              targets:
                - route_type: llm/v1/chat
                  model:
                    provider: openai
                    name: gpt-4o
                  auth:
                    header_name: Authorization
                    header_value: "{vault://env/openai-key}"
ai-models:
  - name: guarded-gpt
    alias: "@openai/guarded-gpt"
`)

	out, warnings, err := Convert(src, Options{
		From:               FormatDeck,
		To:                 FormatKongctl,
		GatewayName:        "support-ai",
		GatewayDisplayName: "Support AI",
	})
	require.NoError(t, err)
	require.NotEmpty(t, warnings)
	require.Contains(t, strings.Join(warnings, "\n"), "consumer_groups is not supported")
	require.Contains(t, strings.Join(warnings, "\n"), "api_key is write-only")

	doc := decodeYAML(t, out)
	gateways := mapsFromAny(doc[topLevelGateways])
	require.Len(t, gateways, 1)
	gateway := gateways[0]
	require.Equal(t, "support-ai", gateway["ref"])
	require.Equal(t, "support-ai", gateway["name"])
	require.Equal(t, "Support AI", gateway["display_name"])

	models := mapsFromAny(gateway["models"])
	require.NotEmpty(t, models)
	var translatedModel map[string]any
	for _, model := range models {
		if _, ok := model["target_models"]; ok {
			translatedModel = model
			break
		}
	}
	require.NotNil(t, translatedModel)
	require.NotContains(t, translatedModel, "targets")
	require.NotEmpty(t, translatedModel["display_name"])

	consumers := mapsFromAny(gateway["consumers"])
	require.Len(t, consumers, 1)
	require.NotContains(t, consumers[0], "consumer_groups")
	credentials := mapsFromAny(consumers[0]["credentials"])
	require.Len(t, credentials, 1)
	require.Equal(t, "gregs-dev-credential", credentials[0]["name"])
	require.NotContains(t, credentials[0], "api_key")
}

func TestDeckToKongctlStrictFailsOnAdapterWarnings(t *testing.T) {
	src := []byte(`
_format_version: "3.0"
consumers:
  - username: gregs-dev
    keyauth_credentials:
      - key: secret-abc
`)

	_, warnings, err := Convert(src, Options{
		From:        FormatDeck,
		To:          FormatKongctl,
		GatewayName: "support-ai",
		Strict:      true,
	})
	require.Error(t, err)
	require.Contains(t, strings.Join(warnings, "\n"), "api_key is write-only")
}

func TestKongctlToDeckSelectsGatewayAndRootChildren(t *testing.T) {
	src := []byte(`
ai_gateways:
  - ref: support-ref
    name: support-ai
    display_name: Support AI
    providers:
      - ref: openai-provider
        name: openai-provider
        type: openai
        display_name: OpenAI Provider
        config:
          auth:
            type: basic
            headers:
              - name: Authorization
                value: "{vault://env/openai-key}"
    models:
      - ref: support-model
        type: model
        name: support-model
        display_name: Support Model
        capabilities:
          - generate
        formats:
          - type: openai
        target_models:
          - name: gpt-4o
            provider: openai-provider
            config:
              type: openai
        config:
          route:
            paths:
              - /ai
          model:
            alias: "@openai/support"
  - ref: other-ref
    name: other-ai
    display_name: Other AI
ai_gateway_mcp_servers:
  - ref: tools
    ai_gateway: support-ref
    type: passthrough-listener
    name: tools
    display_name: Tools
    config:
      url: https://tools.example.com/mcp
      route:
        paths:
          - /tools
    tools:
      - name: lookup
        description: Lookup
ai_gateway_consumer_credentials:
  - ref: support-key
    ai_gateway_consumer: support-consumer
    name: support-key
    type: api-key
    display_name: Support Key
    ttl: 0
ai_gateway_consumers:
  - ref: support-consumer
    ai_gateway: support-ref
    name: support-consumer
    type: api-key
    display_name: Support Consumer
`)

	out, warnings, err := Convert(src, Options{
		From:        FormatKongctl,
		To:          FormatDeck,
		GatewayName: "support-ai",
	})
	require.NoError(t, err)
	require.Empty(t, warnings)

	doc := decodeYAML(t, out)
	require.Equal(t, "3.0", doc["_format_version"])
	require.NotEmpty(t, mapsFromAny(doc["services"]))

	aiModels := mapsFromAny(doc["ai_models"])
	require.Len(t, aiModels, 1)
	require.Equal(t, "support-model", aiModels[0]["id"])

	consumers := mapsFromAny(doc["consumers"])
	require.Len(t, consumers, 1)
	require.Equal(t, "support-consumer", consumers[0]["id"])
	credentials := mapsFromAny(consumers[0]["keyauth_credentials"])
	require.Len(t, credentials, 1)
	require.Equal(t, "support-key", credentials[0]["id"])

	rendered := string(out)
	require.Contains(t, rendered, "name: ai-proxy-advanced")
	require.Contains(t, rendered, "name: ai-mcp-proxy")
	require.Contains(t, rendered, "keyauth_credentials:")
	require.NotContains(t, rendered, "target_models:")
	require.NotContains(t, rendered, "ai_gateway:")
}

func TestKongctlToDeckMapsMCPAccess(t *testing.T) {
	src := []byte(`
ai_gateways:
  - ref: support-ref
    name: support-ai
    display_name: Support AI
    mcp_servers:
      - ref: flights-mcp
        type: conversion-listener
        name: flights-mcp
        display_name: Flights MCP
        access:
          default_tool_acls:
            allow: [premium-users]
        config:
          route:
            paths: [/mcp/flights]
        tools:
          - name: search-flights
            description: Search available flights
            method: GET
            path: /flights
            access:
              acls:
                allow: [agents]
      - ref: oauth-mcp
        type: conversion-listener
        name: oauth-mcp
        display_name: OAuth MCP
        config:
          route:
            paths: [/mcp/oauth]
          access:
            acl_attribute_type: oauth_access_token
            access_token_claim_field: .sub
            acls:
              deny: [blocked-users]
            default_tool_acls:
              allow: [trusted-users]
        tools: []
`)

	out, warnings, err := Convert(src, Options{
		From:        FormatKongctl,
		To:          FormatDeck,
		GatewayName: "support-ai",
	})
	require.NoError(t, err)
	require.Empty(t, warnings)

	doc := decodeYAML(t, out)
	flightsConfig := mcpPluginConfigForService(t, doc, "flights-mcp")
	defaultACL := mapsFromAny(flightsConfig["default_acl"])
	require.Len(t, defaultACL, 1)
	require.Equal(t, []any{"premium-users"}, defaultACL[0]["allow"])
	tools := mapsFromAny(flightsConfig["tools"])
	require.Len(t, tools, 1)
	require.Equal(t, []any{"agents"}, mapFromAny(tools[0]["acl"])["allow"])

	oauthConfig := mcpPluginConfigForService(t, doc, "oauth-mcp")
	require.Equal(t, "oauth_access_token", oauthConfig["acl_attribute_type"])
	require.Equal(t, ".sub", oauthConfig["access_token_claim_field"])
	require.NotContains(t, oauthConfig, "include_consumer_groups")
	oauthDefaultACL := mapsFromAny(oauthConfig["default_acl"])
	require.Len(t, oauthDefaultACL, 1)
	require.Equal(t, []any{"trusted-users"}, oauthDefaultACL[0]["allow"])
	require.Equal(t, []any{"blocked-users"}, oauthDefaultACL[0]["deny"])
}

func TestKongctlToDeckMapsEmbeddingsModelConfig(t *testing.T) {
	src := []byte(`
ai_gateways:
  - ref: support-ref
    name: support-ai
    display_name: Support AI
    providers:
      - ref: openai-main
        type: openai
        name: openai-main
        display_name: OpenAI Main
        config:
          auth:
            type: basic
            headers:
              - name: Authorization
                value: "{vault://env/openai-key}"
      - ref: embed-main
        type: openai
        name: embed-main
        display_name: Embeddings
        config:
          auth:
            type: basic
            headers:
              - name: Authorization
                value: "{vault://env/embed-key}"
    models:
      - ref: semantic-lb
        type: model
        name: semantic-lb
        display_name: Semantic LB
        capabilities: [generate]
        formats:
          - type: openai
        target_models:
          - name: gpt-4o
            provider: openai-main
            config:
              type: openai
        config:
          route:
            paths: [/ai]
          balancer:
            algorithm: semantic
            embeddings:
              provider: embed-main
              model:
                name: text-embedding-3-small
                config:
                  type: openai
`)

	out, warnings, err := Convert(src, Options{
		From:        FormatKongctl,
		To:          FormatDeck,
		GatewayName: "support-ai",
	})
	require.NoError(t, err)
	require.Empty(t, warnings)

	config := rootPluginConfig(t, decodeYAML(t, out), "ai-proxy-advanced")
	embeddings := mapFromAny(config["embeddings"])
	model := mapFromAny(embeddings["model"])
	require.Equal(t, "text-embedding-3-small", model["name"])
	require.Equal(t, "openai", model["provider"])
	require.Equal(t, "{vault://env/embed-key}", mapFromAny(embeddings["auth"])["header_value"])
}

func TestKongctlToDeckRequiresUniqueGatewayName(t *testing.T) {
	src := []byte(`
ai_gateways:
  - ref: one
    name: support-ai
    display_name: One
  - ref: two
    name: support-ai
    display_name: Two
`)

	_, _, err := Convert(src, Options{
		From:        FormatKongctl,
		To:          FormatDeck,
		GatewayName: "support-ai",
	})
	require.ErrorContains(t, err, `matched 2 gateways`)
}

func TestAdaptNativeMCPServerHandlesMalformedConfig(t *testing.T) {
	item := map[string]any{
		"name":         "tools",
		"upstream_url": "https://tools.example.com/mcp",
		"config":       "invalid",
	}
	var warnings []string

	adaptNativeChildToKongctl("mcp_servers", item, &warnings)

	require.Equal(t, map[string]any{"url": "https://tools.example.com/mcp"}, item["config"])
	require.NotContains(t, item, "upstream_url")
}

func TestAdaptKongctlMCPServerRemovesURLFromNativeConfig(t *testing.T) {
	item := map[string]any{
		"name": "tools",
		"config": map[string]any{
			"url": "https://tools.example.com/mcp",
			"route": map[string]any{
				"paths": []any{"/tools"},
			},
		},
	}

	adaptKongctlChildToNative("mcp_servers", item)

	require.Equal(t, "https://tools.example.com/mcp", item["upstream_url"])
	config := mapFromAny(item["config"])
	require.NotContains(t, config, "url")
	require.Contains(t, config, "route")
}

func decodeYAML(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(data, &doc))
	return doc
}

func mcpPluginConfigForService(t *testing.T, doc map[string]any, serviceName string) map[string]any {
	t.Helper()
	for _, service := range mapsFromAny(doc["services"]) {
		if stringField(service, "name") != serviceName {
			continue
		}
		for _, route := range mapsFromAny(service["routes"]) {
			for _, plugin := range mapsFromAny(route["plugins"]) {
				if stringField(plugin, "name") == "ai-mcp-proxy" {
					return mapFromAny(plugin["config"])
				}
			}
		}
	}
	t.Fatalf("ai-mcp-proxy plugin for service %q not found", serviceName)
	return nil
}

func rootPluginConfig(t *testing.T, doc map[string]any, pluginName string) map[string]any {
	t.Helper()
	for _, plugin := range mapsFromAny(doc["plugins"]) {
		if stringField(plugin, "name") == pluginName {
			return mapFromAny(plugin["config"])
		}
	}
	t.Fatalf("root plugin %q not found", pluginName)
	return nil
}
