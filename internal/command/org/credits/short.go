package credits

import (
	"fmt"
	"strings"

	appcredits "github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/formatter/short"
	"github.com/censys/cencli/internal/pkg/styles"
)

func (c *Command) showOrgCredits(result appcredits.OrganizationCreditDetailsResult) cenclierrors.CencliError {
	var out strings.Builder
	data := result.Data

	// Header
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render("━━━ Organization Credit Details ━━━"))
	out.WriteRune('\n')
	out.WriteRune('\n')

	// Balance - big and bold
	balanceLabel := styles.GlobalStyles.Primary.Render("Balance")
	balanceValue := styles.GlobalStyles.Info.Bold(true).Render(short.FormatNumber(data.Balance))
	fmt.Fprintf(&out, "  %s: %s credits\n", balanceLabel, balanceValue)

	// Auto Replenish Config
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Primary.Render("  Auto Replenish"))
	out.WriteRune('\n')

	if data.AutoReplenishConfig.Enabled {
		fmt.Fprintf(&out, "    %s: %s\n",
			styles.GlobalStyles.Comment.Render("Status"),
			styles.GlobalStyles.Secondary.Render("✓ Enabled"))
		if data.AutoReplenishConfig.Threshold.IsPresent() {
			fmt.Fprintf(&out, "    %s: %s\n",
				styles.GlobalStyles.Comment.Render("Threshold"),
				styles.GlobalStyles.Tertiary.Render(short.FormatNumber(data.AutoReplenishConfig.Threshold.MustGet())))
		}
		if data.AutoReplenishConfig.Amount.IsPresent() {
			fmt.Fprintf(&out, "    %s: %s\n",
				styles.GlobalStyles.Comment.Render("Amount"),
				styles.GlobalStyles.Tertiary.Render(short.FormatNumber(data.AutoReplenishConfig.Amount.MustGet())))
		}
	} else {
		fmt.Fprintf(&out, "    %s: %s\n",
			styles.GlobalStyles.Comment.Render("Status"),
			styles.GlobalStyles.Comment.Render("✗ Disabled"))
	}

	// Credit Expirations
	if len(data.CreditExpirations) > 0 {
		out.WriteString("\n")
		out.WriteString(styles.GlobalStyles.Primary.Render(fmt.Sprintf("  Credit Expirations (%d)", len(data.CreditExpirations))))
		out.WriteString("\n")

		for _, exp := range data.CreditExpirations {
			expBalance := styles.GlobalStyles.Info.Render(short.FormatNumber(exp.Balance))
			fmt.Fprintf(&out, "    %s %s credits", styles.GlobalStyles.Tertiary.Render("•"), expBalance)

			if exp.ExpirationDate.IsPresent() {
				expDate := exp.ExpirationDate.MustGet()
				expStr := fmt.Sprintf("(expires %s)", expDate.Format("2006-01-02"))
				fmt.Fprintf(&out, " %s", styles.GlobalStyles.Comment.Render(expStr))
			}
			out.WriteString("\n")
		}
	}

	formatter.Println(formatter.Stdout, out.String())
	return nil
}
