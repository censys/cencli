package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/command/root"
	"github.com/censys/cencli/internal/config"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func dataDir() (string, error) {
	if override := os.Getenv("CENCLI_DATA_DIR"); override != "" {
		if err := os.MkdirAll(override, 0o700); err != nil {
			return "", err
		}
		return override, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, ".config", "cencli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func main() {
	os.Exit(run())
}

func run() int {
	dir, err := dataDir()
	if err != nil {
		formatter.PrintError(err, nil)
		return 1
	}

	ds, err := store.New(dir)
	if err != nil {
		formatter.PrintError(err, nil)
		return 1
	}

	cfg, err := config.New(dir)
	if err != nil {
		formatter.PrintError(err, nil)
		return 1
	}

	commandCtx := command.NewCommandContext(cfg, ds)

	// Build client and app services (optional to allow config/init before auth)
	sdkClient, err := client.NewCensysSDK(context.Background(), ds, cfg.RetryStrategy)
	if err != nil {
		if errors.Is(err, authdom.ErrAuthNotFound) {
			// user hasn't configured enough to initialize the client
		} else {
			formatter.PrintError(err, nil)
			return 1
		}
	} else {
		commandCtx.SetCensysClient(sdkClient)
	}

	rootCmd, err := command.RootCommandToCobra(root.NewRootCommand(commandCtx))
	if err != nil {
		formatter.PrintError(err, nil)
		return 1
	}

	// Signal-aware execution
	// Note: timeout is applied per-command in PersistentPreRunE unless DisableTimeout() returns true
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	cmd, err := rootCmd.ExecuteContextC(sigCtx)
	if err != nil {
		formatter.PrintError(err, cmd)
		return formatter.ExitCode(err)
	}
	return 0
}
