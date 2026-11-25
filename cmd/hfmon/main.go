package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcus/health-factor-monitor/internal/application"
	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/aave"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/config"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/kamino"
	"github.com/marcus/health-factor-monitor/internal/interfaces/cli"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configPath := flag.String("config", "", "path to the JSON configuration file")
	protocol := flag.String("protocol", "", "only check positions for this protocol (e.g. aave, kamino)")
	flag.Parse()

	resolvedConfigPath, err := resolveConfigPath(*configPath)
	if err != nil {
		return err
	}

	loader := config.NewReader(resolvedConfigPath)
	cfg, err := loader.Load()
	if err != nil {
		return err
	}

	providers := buildProviders(*cfg)

	svc := application.NewCheckService(cfg, providers)

	var results []domain.ProviderResult
	if *protocol != "" {
		results = svc.CheckByProtocol(ctx, *protocol)
	} else {
		results = svc.CheckAll(ctx)
	}

	fmt.Print(cli.FormatResults(results))

	if cli.HasSuccess(results) {
		return nil
	}
	return fmt.Errorf("no health factors could be retrieved for any position")
}

// resolveConfigPath resolves the configuration location for global CLI use.
// Explicit -config and HFMON_CONFIG override automatic discovery. Otherwise,
// the OS user configuration directory is preferred, followed by ./config.json.
func resolveConfigPath(explicitPath string) (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return resolveConfigPathFrom(explicitPath, os.Getenv("HFMON_CONFIG"), userConfigDir, "config.json"), nil
}

func resolveConfigPathFrom(explicitPath, envPath, userConfigDir, localPath string) string {
	if explicitPath != "" {
		return explicitPath
	}
	if envPath != "" {
		return envPath
	}
	globalPath := filepath.Join(userConfigDir, "hf", "config.json")
	if _, err := os.Stat(globalPath); err == nil {
		return globalPath
	}
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}
	return globalPath
}

// buildProviders wires the provider adapters for the MVP: Aave against the
// ethereum RPC endpoint and Kamino against its public API base URL. Providers
// are registered only when the governing network is configured.
func buildProviders(cfg domain.Config) map[string]domain.HealthFactorProvider {
	providers := map[string]domain.HealthFactorProvider{}
	if endpoint, ok := cfg.RPCEndpoints[domain.NetworkEthereum]; ok {
		providers[protocolNetworkKey(domain.ProtocolAave, domain.NetworkEthereum)] = aave.NewProvider(endpoint, domain.NetworkEthereum)
	}
	if endpoint, ok := cfg.RPCEndpoints[domain.NetworkBase]; ok {
		providers[protocolNetworkKey(domain.ProtocolAave, domain.NetworkBase)] = aave.NewProvider(endpoint, domain.NetworkBase)
	}
	if solanaRPC, ok := cfg.RPCEndpoints[domain.NetworkSolana]; ok {
		providers[protocolNetworkKey(domain.ProtocolKamino, domain.NetworkSolana)] = kamino.NewProvider(kaminoAPIBaseURL, solanaRPC)
	}
	return providers
}

// kaminoAPIBaseURL is the base URL of the Kamino public API.
const kaminoAPIBaseURL = "https://api.kamino.finance"

// protocolNetworkKey builds the "protocol:network" map key used to match
// providers to positions.
func protocolNetworkKey(protocol, network string) string {
	return protocol + ":" + network
}
