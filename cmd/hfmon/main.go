package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/marcus/health-factor-monitor/internal/infrastructure/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.json", "path to the JSON configuration file")
	flag.Parse()

	loader := config.NewReader(*configPath)
	if _, err := loader.Load(); err != nil {
		return err
	}

	return nil
}
