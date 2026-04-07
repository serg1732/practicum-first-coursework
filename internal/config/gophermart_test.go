package config

import (
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetFlags(t *testing.T, args []string) func() {
	t.Helper()

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flag.CommandLine = fs
	os.Args = args

	return func() {
		flag.CommandLine = oldCommandLine
		os.Args = oldArgs
	}
}

func TestGetGophermartConfig_FromEnv(t *testing.T) {
	restore := resetFlags(t, []string{"cmd"})
	defer restore()

	t.Setenv("RUN_ADDRESS", "localhost:8080")
	t.Setenv("DATABASE_URI", "postgres://user:pass@localhost:5432/db")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "localhost:9090")
	t.Setenv("SECRET", "my-secret")
	t.Setenv("RATE_LIMIT", "10")
	t.Setenv("RATE_LIMIT_DELAY", "120")

	cfg, err := GetGophermartConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RunAddr != "localhost:8080" {
		t.Errorf("RunAddr = %q, want %q", cfg.RunAddr, "localhost:8080")
	}
	if cfg.DSN != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("DSN = %q, want %q", cfg.DSN, "postgres://user:pass@localhost:5432/db")
	}
	if cfg.AccrualAddress != "localhost:9090" {
		t.Errorf("AccrualAddress = %q, want %q", cfg.AccrualAddress, "localhost:9090")
	}
	if cfg.Secret != "my-secret" {
		t.Errorf("Secret = %q, want %q", cfg.Secret, "my-secret")
	}
	if cfg.RateLimit != 10 {
		t.Errorf("RateLimit = %d, want %d", cfg.RateLimit, 10)
	}
	if cfg.RateLimitDelaySec != 120 {
		t.Errorf("RateLimitDelaySec = %d, want %d", cfg.RateLimitDelaySec, 120)
	}
}

func TestGetGophermartConfig_Defaults(t *testing.T) {
	restore := resetFlags(t, []string{"cmd"})
	defer restore()

	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("SECRET", "")
	t.Setenv("RATE_LIMIT", "")
	t.Setenv("RATE_LIMIT_DELAY", "")

	cfg, err := GetGophermartConfig()
	assert.NoError(t, err)
	assert.Equal(t, "secret", cfg.Secret)
	assert.Equal(t, 5, cfg.RateLimit)
	assert.Equal(t, 60, cfg.RateLimitDelaySec)
}

func TestGetGophermartConfigSuccessFlagsOverrideEnv(t *testing.T) {
	restore := resetFlags(t, []string{
		"cmd",
		"-a", "localhost:8090",
		"-d", "postgres://flaguser:pass@localhost:5432/db",
		"-r", "https://accrual:8083",
	})
	defer restore()

	t.Setenv("RUN_ADDRESS", "localhost:8080")
	t.Setenv("DATABASE_URI", "postgres://user:pass@localhost:5432/db")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "https://accrual:8082")
	t.Setenv("SECRET", "ochen-slojnyi-secret")
	t.Setenv("RATE_LIMIT", "10")
	t.Setenv("RATE_LIMIT_DELAY", "60")

	cfg, err := GetGophermartConfig()
	assert.NoError(t, err, "Не ожидали получить ошибку")
	assert.Equal(t, cfg.RunAddr, "localhost:8090")
	assert.Equal(t, cfg.DSN, "postgres://flaguser:pass@localhost:5432/db")
	assert.Equal(t, cfg.AccrualAddress, "https://accrual:8083")
	assert.Equal(t, cfg.Secret, "ochen-slojnyi-secret")
	assert.Equal(t, cfg.RateLimit, 10)
	assert.Equal(t, cfg.RateLimitDelaySec, 60)
}

func TestGetGophermartConfigErrorInvalidEnv(t *testing.T) {
	restore := resetFlags(t, []string{"cmd"})
	defer restore()

	t.Setenv("RATE_LIMIT", "string-value")

	cfg, err := GetGophermartConfig()
	assert.NotNil(t, err)
	assert.Nil(t, cfg)
}
