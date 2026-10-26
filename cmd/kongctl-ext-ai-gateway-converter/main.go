// Command kongctl-ext-ai-gateway-converter migrates Kong Gateway AI
// configuration into AI Gateway resources.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/Kong/kong-ai-migration-tool/migrate"
)

const (
	conversionCommandID = "convert_ai_gateway"
	versionCommandID    = "convert_ai_gateway_version"
	extensionContextEnv = "KONGCTL_EXTENSION_CONTEXT"
	migrationToolModule = "github.com/Kong/kong-ai-migration-tool"
	unknownVersion      = "unknown"
)

const helpText = `Migrate Kong Gateway AI configuration into AI Gateway resources.

Usage:
  kongctl convert ai-gateway --input PATH [flags]

Flags:
      --input string           Source decK file, Kong Operator manifests, or Terraform config. Required.
      --input-format string    Input format: auto, deck, kong-operator, terraform. (default "auto")
      --config string          Directory holding manual migration config. (default "./config")
      --ref string             Path to an AI Gateway OpenAPI spec override.
      --out string             Output directory for migrated resources. (default "./out")
      --label-tag-prefix string
                               Tag prefix lifted back into labels.
      --namespace-prefix string
                               kongctl namespace prefix for generated files. (default "ai-gateway")
      --source-name string     Migrated control plane name. (default: input filename)
      --strict                 Fail on migration findings instead of collecting them.
      --format string          Comma-separated outputs: kongctl, deck, kong-operator, terraform. (default "kongctl")
      --operator-gateway string
                               Kong Operator parent: konnect or onprem. (default "konnect")
      --k8s-namespace string    Namespace for Kong Operator manifests.
      --konnect-server-url string
                               Konnect API host for Operator and Terraform. (default "us.api.konghq.com")

Example:
  kongctl convert ai-gateway --input kong.yaml --config ./config --out ./out
`

type cliOptions struct {
	migrate migrate.Options
}

func main() {
	commandID, err := matchedCommandID()
	if err == nil {
		err = runCommand(commandID, os.Args[1:], os.Stdout, os.Stderr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	return runCommand("", args, stdout, stderr)
}

func runCommand(commandID string, args []string, stdout, stderr io.Writer) error {
	switch commandID {
	case versionCommandID:
		return writeVersion(stdout)
	case "", conversionCommandID:
	default:
		return fmt.Errorf("unsupported extension command %q", commandID)
	}

	if shouldShowHelp(args) {
		_, err := fmt.Fprint(stdout, helpText)
		return err
	}

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	result, err := migrate.Run(opts.migrate)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "%s: %s\n", warning.Severity, warning.String())
	}
	if err != nil {
		return err
	}
	if result.Report != nil {
		errors, warnings, notes := result.Report.Counts()
		if _, err := fmt.Fprintf(stderr, "findings: %d error(s), %d warning(s), %d note(s)\n",
			errors, warnings, notes); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(stderr, "migration complete: wrote output to %s\n"+
		"read %s before applying: it lists the URL and request-body changes clients must make\n",
		opts.migrate.OutDir, filepath.Join(opts.migrate.OutDir, "migration-report.md"))
	return err
}

type extensionContext struct {
	MatchedCommandPath struct {
		ID string `json:"id"`
	} `json:"matched_command_path"`
}

func matchedCommandID() (string, error) {
	path := strings.TrimSpace(os.Getenv(extensionContextEnv))
	if path == "" {
		return "", nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open extension context: %w", err)
	}
	defer file.Close()

	var context extensionContext
	if err := json.NewDecoder(file).Decode(&context); err != nil {
		return "", fmt.Errorf("decode extension context: %w", err)
	}
	if context.MatchedCommandPath.ID == "" {
		return "", fmt.Errorf("extension context is missing matched_command_path.id")
	}
	return context.MatchedCommandPath.ID, nil
}

func writeVersion(output io.Writer) error {
	extensionVersion, migrationVersion := buildVersions()
	_, err := fmt.Fprintf(
		output,
		"ai-gateway-converter: %s\nkong-ai-migration-tool: %s\n",
		extensionVersion,
		migrationVersion,
	)
	return err
}

func buildVersions() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unknownVersion, unknownVersion
	}

	extensionVersion := normalizedVersion(info.Main.Version)
	migrationVersion := unknownVersion
	for _, dependency := range info.Deps {
		if dependency.Path != migrationToolModule {
			continue
		}
		if dependency.Replace != nil {
			dependency = dependency.Replace
		}
		migrationVersion = normalizedVersion(dependency.Version)
		break
	}
	return extensionVersion, migrationVersion
}

func normalizedVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return unknownVersion
	}
	return version
}

func parseArgs(args []string) (cliOptions, error) {
	opts := cliOptions{migrate: migrate.Options{
		ConfigDir:        "./config",
		OutDir:           "./out",
		NamespacePrefix:  "ai-gateway",
		Formats:          []migrate.Format{migrate.FormatKongctl},
		OperatorGateway:  migrate.OperatorGatewayKonnect,
		KonnectServerURL: migrate.DefaultKonnectServerURL,
	}}

	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "--") {
			return cliOptions{}, fmt.Errorf("unexpected positional argument %q", token)
		}

		name, value, hasValue := strings.Cut(strings.TrimPrefix(token, "--"), "=")
		if name == "strict" {
			opts.migrate.Strict = true
			if hasValue {
				strict, err := strconv.ParseBool(value)
				if err != nil {
					return cliOptions{}, fmt.Errorf("invalid value for --strict: %w", err)
				}
				opts.migrate.Strict = strict
			}
			continue
		}
		var err error
		value, i, err = nextFlagValue(args, i, name, value, hasValue)
		if err != nil {
			return cliOptions{}, err
		}

		switch name {
		case "input":
			opts.migrate.InputPath = value
		case "input-format":
			opts.migrate.InputFormat, err = migrate.ParseInputFormat(value)
		case "source-name":
			opts.migrate.SourceName = value
		case "format":
			opts.migrate.Formats, err = migrate.ParseFormats(value)
		case "operator-gateway":
			if value != migrate.OperatorGatewayKonnect && value != migrate.OperatorGatewayOnPrem {
				return cliOptions{}, fmt.Errorf("--operator-gateway must be %s or %s",
					migrate.OperatorGatewayKonnect, migrate.OperatorGatewayOnPrem)
			}
			opts.migrate.OperatorGateway = value
		case "k8s-namespace":
			opts.migrate.K8sNamespace = value
		case "konnect-server-url":
			opts.migrate.KonnectServerURL = value
		case "config":
			opts.migrate.ConfigDir = value
		case "ref":
			opts.migrate.RefPath = value
		case "out":
			opts.migrate.OutDir = value
		case "label-tag-prefix":
			opts.migrate.LabelTagPrefix = value
		case "namespace-prefix":
			opts.migrate.NamespacePrefix = value
		default:
			return cliOptions{}, fmt.Errorf("unknown flag --%s", name)
		}
		if err != nil {
			return cliOptions{}, err
		}
	}

	opts.migrate.InputPath = strings.TrimSpace(opts.migrate.InputPath)
	if opts.migrate.InputPath == "" {
		return cliOptions{}, fmt.Errorf("--input is required")
	}
	return opts, nil
}

func nextFlagValue(args []string, index int, name, value string, hasValue bool) (string, int, error) {
	if hasValue {
		return value, index, nil
	}
	next := index + 1
	if next >= len(args) || strings.HasPrefix(args[next], "--") {
		return "", index, fmt.Errorf("flag --%s requires a value", name)
	}
	return args[next], next, nil
}

func shouldShowHelp(args []string) bool {
	return len(args) == 0 || slices.Contains(args, "--help") || slices.Contains(args, "-h")
}
