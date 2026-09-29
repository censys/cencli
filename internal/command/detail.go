package command

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/pkg/styles"
)

// WriteDetailField appends a padded label / value line to a detail view.
func WriteDetailField(out *strings.Builder, label, value string) {
	labelStyled := styles.GlobalStyles.Primary.Render(fmt.Sprintf("%-13s", label+":"))
	valueStyled := styles.GlobalStyles.Comment.Render(value)
	fmt.Fprintf(out, "  %s %s\n", labelStyled, valueStyled)
}
