//go:build windows
// +build windows

package main

import (
	"os"

	"github.com/mattn/go-colorable"
	"golang.org/x/sys/windows"

	"github.com/censys/cencli/internal/pkg/formatter"
)

func init() {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)
	formatter.Stdout = colorable.NewColorableStdout()
	formatter.Stderr = colorable.NewColorableStderr()
}
