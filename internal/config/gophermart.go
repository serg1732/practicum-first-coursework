package config

import (
	"flag"

	"github.com/caarlos0/env/v11"
)

type GophermartConfig struct {
	RunAddr           string `env:"RUN_ADDRESS"`
	DSN               string `env:"DATABASE_URI"`
	AccuralAddress    string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	Secret            string `env:"SECRET" envDefault:"secret"`
	RateLimit         int    `env:"RATE_LIMIT" envDefault:"5"`
	RateLimitDelaySec int    `env:"RATE_LIMIT_DELAY" envDefault:"60"`
}

func GetGophermartConfig() (*GophermartConfig, error) {
	var serverConfig GophermartConfig
	if err := env.Parse(&serverConfig); err != nil {
		return nil, err
	}

	flag.StringVar(&serverConfig.RunAddr, "a", serverConfig.RunAddr, "address and port to run server")
	flag.StringVar(&serverConfig.DSN, "d", serverConfig.DSN, "database connection string")
	flag.StringVar(&serverConfig.AccuralAddress, "r", serverConfig.AccuralAddress, "accural address to use for accural service")
	flag.Parse()
	return &serverConfig, nil
}
