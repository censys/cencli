package formatter

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/styles"
)

// PrintError prints an error in a standardized format.
// Takes an optional cobra command to print usage information if the error should print usage.
func PrintError(err error, cmd *cobra.Command) {
	var cencliErr cenclierrors.CencliError
	if errors.As(err, &cencliErr) {
		printCencliError(cencliErr, cmd)
		return
	}
	fmt.Fprintln(Stderr, err.Error())
}

// printCencliError prints a domain error in a standardized format.
func printCencliError(err cenclierrors.CencliError, cmd *cobra.Command) {
	fmt.Fprintf(Stderr, "[%s]\n", styles.GlobalStyles.Danger.Render(err.Title()))
	fmt.Fprintf(Stderr, "%s\n", styles.GlobalStyles.Warning.Render(err.Error()))
	if cmd != nil && err.ShouldPrintUsage() {
		_ = cmd.Usage()
	}
}
