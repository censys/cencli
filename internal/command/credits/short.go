package credits

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/formatter/short"
	"github.com/censys/cencli/internal/pkg/styles"
)

func (c *Command) showUserCredits(result credits.UserCreditDetailsResult) cenclierrors.CencliError {
	var out strings.Builder
	data := result.Data

	// Header
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render("━━━ Your Free User Credit Details ━━━"))
	out.WriteRune('\n')
	out.WriteRune('\n')

	// Balance
	balanceLabel := styles.GlobalStyles.Primary.Render("Balance")
	balanceValue := styles.GlobalStyles.Info.Bold(true).Render(short.FormatNumber(data.Balance))
	fmt.Fprintf(&out, "  %s: %s credits", balanceLabel, balanceValue)

	// Resets At
	if data.ResetsAt.IsPresent() {
		resetTime := data.ResetsAt.MustGet()
		resetStr := fmt.Sprintf("(resets %s)", resetTime.Format("2006-01-02"))
		fmt.Fprintf(&out, " %s", styles.GlobalStyles.Comment.Render(resetStr))
	}

	out.WriteRune('\n')
	formatter.Println(formatter.Stdout, out.String())
	return nil
}
