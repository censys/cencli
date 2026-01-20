package details

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

func (c *Command) showOrgDetails(result organizations.OrganizationDetailsResult) cenclierrors.CencliError {
	var out strings.Builder
	data := result.Data

	// Header
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render("━━━ Organization Details ━━━"))
	out.WriteRune('\n')
	out.WriteRune('\n')

	// Organization Name - big and bold
	nameLabel := fmt.Sprintf("%-8s", "Name:")
	nameLabelStyled := styles.GlobalStyles.Primary.Render(nameLabel)
	nameValue := styles.GlobalStyles.Info.Bold(true).Render(data.Name)
	fmt.Fprintf(&out, "  %s %s\n", nameLabelStyled, nameValue)

	// Organization ID
	idLabel := fmt.Sprintf("%-8s", "ID:")
	idLabelStyled := styles.GlobalStyles.Primary.Render(idLabel)
	idValue := styles.GlobalStyles.Comment.Render(data.ID.String())
	fmt.Fprintf(&out, "  %s %s\n", idLabelStyled, idValue)

	// Created At
	if data.CreatedAt.IsPresent() {
		createdLabel := fmt.Sprintf("%-8s", "Created:")
		createdLabelStyled := styles.GlobalStyles.Primary.Render(createdLabel)
		createdValue := styles.GlobalStyles.Comment.Render(data.CreatedAt.MustGet().Format("2006-01-02 15:04:05 MST"))
		fmt.Fprintf(&out, "  %s %s\n", createdLabelStyled, createdValue)
	}

	// Member Counts
	if data.MemberCounts != nil {
		out.WriteRune('\n')
		out.WriteString(styles.GlobalStyles.Primary.Render("  Member Counts"))
		out.WriteRune('\n')

		totalLabel := fmt.Sprintf("%-11s", "Total:")
		fmt.Fprintf(&out, "    %s %s\n",
			styles.GlobalStyles.Comment.Render(totalLabel),
			styles.GlobalStyles.Tertiary.Render(fmt.Sprintf("%d", data.MemberCounts.Total)))

		// Show role breakdown if available
		if data.MemberCounts.ByRole.Admin != nil {
			adminsLabel := fmt.Sprintf("%-11s", "Admins:")
			fmt.Fprintf(&out, "    %s %s\n",
				styles.GlobalStyles.Comment.Render(adminsLabel),
				styles.GlobalStyles.Tertiary.Render(fmt.Sprintf("%d", *data.MemberCounts.ByRole.Admin)))
		}
		if data.MemberCounts.ByRole.APIAccess != nil {
			apiAccessLabel := fmt.Sprintf("%-11s", "API Access:")
			fmt.Fprintf(&out, "    %s %s\n",
				styles.GlobalStyles.Comment.Render(apiAccessLabel),
				styles.GlobalStyles.Tertiary.Render(fmt.Sprintf("%d", *data.MemberCounts.ByRole.APIAccess)))
		}
	}

	// Preferences
	if data.Preferences != nil {
		out.WriteRune('\n')
		out.WriteString(styles.GlobalStyles.Primary.Render("  Preferences"))
		out.WriteRune('\n')

		if data.Preferences.MfaRequired != nil {
			mfaStatus := "Disabled"
			if *data.Preferences.MfaRequired {
				mfaStatus = "Required"
			}
			mfaLabel := fmt.Sprintf("%-12s", "MFA:")
			fmt.Fprintf(&out, "    %s %s\n",
				styles.GlobalStyles.Comment.Render(mfaLabel),
				styles.GlobalStyles.Tertiary.Render(mfaStatus))
		}
		if data.Preferences.AiOptIn != nil {
			aiStatus := "Opted Out"
			if *data.Preferences.AiOptIn {
				aiStatus = "Opted In"
			}
			aiFeaturesLabel := fmt.Sprintf("%-12s", "AI Features:")
			fmt.Fprintf(&out, "    %s %s\n",
				styles.GlobalStyles.Comment.Render(aiFeaturesLabel),
				styles.GlobalStyles.Tertiary.Render(aiStatus))
		}
		if data.Preferences.AiTraining != nil {
			aiTrainingStatus := "Disabled"
			if *data.Preferences.AiTraining {
				aiTrainingStatus = "Enabled"
			}
			aiTrainingLabel := fmt.Sprintf("%-12s", "AI Training:")
			fmt.Fprintf(&out, "    %s %s\n",
				styles.GlobalStyles.Comment.Render(aiTrainingLabel),
				styles.GlobalStyles.Tertiary.Render(aiTrainingStatus))
		}
	}

	formatter.Println(formatter.Stdout, out.String())
	return nil
}
