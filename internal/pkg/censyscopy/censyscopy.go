package censyscopy

import (
	"fmt"
	"io"
	"strings"

	"github.com/censys/cencli/internal/pkg/term"
)

type CencliLink string

const (
	CencliRepo              CencliLink = "https://github.com/censys/cencli"
	CencliDocs              CencliLink = "https://docs.censys.com/docs/platform-cli"
	CensysPATInstructions   CencliLink = "https://docs.censys.com/reference/get-started#step-2-create-a-personal-access-token"
	CensysOrgIDInstructions CencliLink = "https://docs.censys.com/reference/get-started#step-3-find-and-use-your-organization-id-optional"
)

func (l CencliLink) String() string {
	return string(l)
}

// Render returns a terminal-friendly clickable link.
// This should only be used if the result is being written to a TTY.
// If no anchor text is provided, the link's URL will be used as
// the anchor text, with the protocol stripped.
func (l CencliLink) Render(anchorText string) string {
	url := string(l)
	if anchorText == "" {
		anchorText = strings.TrimPrefix(url, "https://")
	}
	return term.RenderLink(url, anchorText)
}

// Title returns the copy for the title of this CLI.
func Title() string {
	return "The Censys CLI"
}

// Description returns the copy for the description of this CLI.
func Description() string {
	sb := strings.Builder{}
	sb.WriteString("The Censys CLI brings the authority of internet intelligence to your terminal.")
	sb.WriteRune('\n')
	sb.WriteString("Leverage the Censys Platform to analyze assets, perform searches, hunt threats,")
	sb.WriteRune('\n')
	sb.WriteString("all from the command line.")
	return sb.String()
}

// DocumentationCLI returns the copy for the documentation of this CLI.
// Takes a writer to determine if the output is a TTY for link rendering.
func DocumentationCLI(w io.Writer) string {
	sb := strings.Builder{}
	if term.IsTTY(w) {
		sb.WriteString("For more information, please refer to the ")
		sb.WriteString(CencliDocs.Render("documentation"))
		sb.WriteString(fmt.Sprintf(" (%s)", CencliDocs.Render("")))
		sb.WriteString(".")
	} else {
		sb.WriteString("For more information, please refer to the documentation: ")
		sb.WriteString(CencliDocs.String())
	}
	return sb.String()
}

// DocumentationPAT returns the copy for the documentation for obtaining a personal access token.
// Takes a writer to determine if the output is a TTY for link rendering.
func DocumentationPAT(w io.Writer) string {
	sb := strings.Builder{}
	if term.IsTTY(w) {
		sb.WriteString("Click ")
		sb.WriteString(CensysPATInstructions.Render("here"))
		sb.WriteString(" to learn how to obtain a personal access token")
		sb.WriteRune('\n')
		sb.WriteString(fmt.Sprintf("(or go to %s)", CensysPATInstructions.Render("")))
	} else {
		sb.WriteString("Click here to learn how to obtain a personal access token: ")
		sb.WriteString(CensysPATInstructions.String())
	}
	return sb.String()
}

// DocumentationOrgID returns the copy for the documentation for obtaining an organization ID.
// Takes a writer to determine if the output is a TTY for link rendering.
func DocumentationOrgID(w io.Writer) string {
	sb := strings.Builder{}
	if term.IsTTY(w) {
		sb.WriteString("Click ")
		sb.WriteString(CensysOrgIDInstructions.Render("here"))
		sb.WriteString(" to learn where to find your organization ID")
		sb.WriteRune('\n')
		sb.WriteString(fmt.Sprintf("(or go to %s)", CensysOrgIDInstructions.Render("")))
	} else {
		sb.WriteString("Click here to learn how to obtain an organization ID: ")
		sb.WriteString(CensysOrgIDInstructions.String())
	}
	return sb.String()
}
