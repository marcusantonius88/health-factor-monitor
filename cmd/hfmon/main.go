package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/marcus/health-factor-monitor/internal/application"
	"github.com/marcus/health-factor-monitor/internal/domain"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/aave"
	"github.com/marcus/health-factor-monitor/internal/infrastructure/config"
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
	configPath := flag.String("config", "config.json", "path to the JSON configuration file")
	protocol := flag.String("protocol", "", "only check positions for this protocol (e.g. aave, kamino)")
	flag.Parse()

	loader := config.NewReader(*configPath)
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

// buildProviders wires the provider adapters for the MVP. For now only the
// Aave provider is available, registered against the ethereum RPC endpoint.
func buildProviders(cfg domain.Config) map[string]domain.HealthFactorProvider {
	providers := map[string]domain.HealthFactorProvider{}
	if endpoint, ok := cfg.RPCEndpoints[domain.NetworkEthereum]; ok {
		providers[aaveProtocolKey] = aave.NewProvider(endpoint)
	}
	return providers
}

const aaveProtocolKey = "aave:ethereum"
