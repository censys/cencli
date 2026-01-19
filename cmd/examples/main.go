package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/censys/cencli/internal/command/aggregate"
	"github.com/censys/cencli/internal/command/censeye"
	"github.com/censys/cencli/internal/command/history"
	"github.com/censys/cencli/internal/command/root"
	"github.com/censys/cencli/internal/command/search"
	"github.com/censys/cencli/internal/command/view"
	"github.com/censys/cencli/internal/pkg/tape"
	"github.com/censys/cencli/internal/pkg/ui/spinner"
)

const (
	timeout = 30 * time.Second
	baseDir = "examples"
)

type recordableCommand interface {
	Tapes(recorder *tape.Recorder) []tape.Tape
}

func main() {
	// Get absolute path to the locally built binary
	binPath, err := filepath.Abs("./bin/censys")
	if err != nil {
		panic(err)
	}

	r, err := tape.NewTapeRecorder(
		"vhs",
		binPath,
		map[string]string{
			"FORCE_COLOR": "1",
		},
	)
	if err != nil {
		panic(err)
	}

	commands := map[string]recordableCommand{
		"search":    search.NewSearchCommand(nil),
		"root":      root.NewRootCommand(nil),
		"view":      view.NewViewCommand(nil),
		"aggregate": aggregate.NewAggregateCommand(nil),
		"censeye":   censeye.NewCenseyeCommand(nil),
		"history":   history.NewHistoryCommand(nil),
	}

	var targetCommands map[string]recordableCommand
	if len(os.Args) > 1 {
		cmdName := os.Args[1]
		if cmd, exists := commands[cmdName]; exists {
			targetCommands = map[string]recordableCommand{cmdName: cmd}
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmdName)
			fmt.Fprintf(os.Stderr, "Available commands: ")
			for name := range commands {
				fmt.Fprintf(os.Stderr, "%s ", name)
			}
			fmt.Fprintln(os.Stderr)
			os.Exit(1)
		}
	} else {
		targetCommands = commands
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	for dir, cmd := range targetCommands {
		outputDir := filepath.Join(baseDir, dir)
		// special case for root command
		if dir == "root" {
			outputDir = baseDir
		}
		for _, t := range cmd.Tapes(r) {
			ctx, cancel := context.WithTimeout(sigCtx, timeout)
			stop := spinner.Start(ctx.Done(), false, spinner.WithMessage(fmt.Sprintf("Recording tape for %s...", t.Name)))
			err = r.CreateTape(ctx, t, outputDir)
			cancel()
			stop()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}
