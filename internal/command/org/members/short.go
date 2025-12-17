package members

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
	"github.com/censys/cencli/internal/pkg/ui/table"
)

func (c *Command) showRawTable(result organizations.OrganizationMembersResult) cenclierrors.CencliError {
	if len(result.Data.Members) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo members found.\n")
		return nil
	}

	columns := []rawtable.Column[organizations.OrganizationMember]{
		{
			Title: "Email",
			String: func(m organizations.OrganizationMember) string {
				if m.Email.IsPresent() {
					return m.Email.MustGet()
				}
				return "-"
			},
			Style: func(s string, m organizations.OrganizationMember) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title:  "Name",
			String: formatName,
			Style: func(s string, m organizations.OrganizationMember) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title: "Roles",
			String: func(m organizations.OrganizationMember) string {
				if len(m.Roles) > 0 {
					return strings.Join(m.Roles, ", ")
				}
				return "-"
			},
			Style: func(s string, m organizations.OrganizationMember) string {
				return styles.NewStyle(styles.ColorSage).Render(s)
			},
		},
		{
			Title: "First Login",
			String: func(m organizations.OrganizationMember) string {
				if m.FirstLoginTime.IsPresent() {
					return m.FirstLoginTime.MustGet().Format("2006-01-02 15:04")
				}
				return "Never"
			},
			Style: func(s string, m organizations.OrganizationMember) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title: "Last Login",
			String: func(m organizations.OrganizationMember) string {
				if m.LatestLoginTime.IsPresent() {
					return m.LatestLoginTime.MustGet().Format("2006-01-02 15:04")
				}
				return "Never"
			},
			Style: func(s string, m organizations.OrganizationMember) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[organizations.OrganizationMember](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[organizations.OrganizationMember](!formatter.StdoutIsTTY()),
	)

	title := styles.GlobalStyles.Signature.Bold(true).Render(fmt.Sprintf("Organization Members (%d)", len(result.Data.Members)))
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(result.Data.Members))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

func (c *Command) showInteractiveTable(result organizations.OrganizationMembersResult) cenclierrors.CencliError {
	if len(result.Data.Members) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo members found.\n")
		return nil
	}

	tbl := table.NewTable[organizations.OrganizationMember](
		[]string{"Email", "Name", "Roles", "First Login", "Last Login"},
		func(m organizations.OrganizationMember) []string {
			email := "-"
			if m.Email.IsPresent() {
				email = m.Email.MustGet()
			}
			name := formatName(m)
			roles := "-"
			if len(m.Roles) > 0 {
				roles = strings.Join(m.Roles, ", ")
			}
			firstLogin := "Never"
			if m.FirstLoginTime.IsPresent() {
				firstLogin = m.FirstLoginTime.MustGet().Format("2006-01-02 15:04")
			}
			lastLogin := "Never"
			if m.LatestLoginTime.IsPresent() {
				lastLogin = m.LatestLoginTime.MustGet().Format("2006-01-02 15:04")
			}
			return []string{email, name, roles, firstLogin, lastLogin}
		},
		table.WithColumnWidths[organizations.OrganizationMember]([]int{30, 25, 20, 20, 20}),
		table.WithTitle[organizations.OrganizationMember](fmt.Sprintf("Organization Members (%d)", len(result.Data.Members))),
	)

	if err := tbl.Run(result.Data.Members); err != nil {
		return cenclierrors.NewCencliError(
			fmt.Errorf("failed to display interactive table: %w", err),
		)
	}
	return nil
}

// formatName combines first and last name, handling optional values.
func formatName(m organizations.OrganizationMember) string {
	var parts []string
	if m.FirstName.IsPresent() {
		parts = append(parts, m.FirstName.MustGet())
	}
	if m.LastName.IsPresent() {
		parts = append(parts, m.LastName.MustGet())
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}
