package formatter

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	handlebars "github.com/aymerick/raymond"

	"github.com/censys/cencli/internal/config/templates/helpers"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/styles"
)

// avoid panics when registering helpers multiple times
var once = sync.Once{}

// PrintDataWithTemplate renders data through a template and writes the result to stdout.
// Callers must provide what kind of data needs to be rendered,
// so its correspoinding template can be used.
// Returns error if the template does not exist for the given entity,
// or the template fails to render.
func PrintDataWithTemplate(templatePath string, colored bool, data any) cenclierrors.CencliError {
	once.Do(func() {
		registerTemplateHelpers(colored)
	})
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return newTemplateFailureError(templatePath, err)
	}
	data, err = dataToJSON(data)
	if err != nil {
		return newTemplateFailureError(templatePath, err)
	}
	result, err := handlebars.Render(string(templateBytes), data)
	if err != nil {
		return newTemplateFailureError(templatePath, err)
	}
	Stdout.Write([]byte(result))
	return nil
}

// registerTemplateHelpers registers the template helpers for the template engine.
func registerTemplateHelpers(colored bool) {
	var helpersToRegister []helpers.HandlebarsHelper

	helpersToRegister = append(helpersToRegister,
		helpers.NewLengthHelper(),
		helpers.NewLessThanHelper(),
		helpers.NewGreaterThanHelper(),
		helpers.NewEqualHelper(),
		helpers.NewCapitalizeHelper(),
		helpers.NewLookupURLHelper(colored),
		helpers.NewJoinHelper(),
		helpers.NewPluckHelper(),
		helpers.NewConcatHelper(),
		helpers.NewSoftwareHelper(),
		helpers.NewSoftwareListHelper(colored),
		helpers.NewHasComponentsHelper(),
		helpers.NewOSHelper(colored),
		helpers.NewLocationHelper(colored),
		helpers.NewClickableIPHelper(colored, colored),
	)

	helpersToRegister = append(helpersToRegister,
		helpers.NewColorHelpers(
			colored,
			helpers.ColorHelperConfig{Name: "red", Color: styles.ColorRed},
			helpers.ColorHelperConfig{Name: "blue", Color: styles.ColorBlue},
			helpers.ColorHelperConfig{Name: "orange", Color: styles.ColorOrange},
			helpers.ColorHelperConfig{Name: "yellow", Color: styles.ColorGold},
		)...,
	)

	helpers.RegisterHelpers(helpersToRegister...)
}

// dataToJSON converts the data to a "JSON-style" Go object,
// allowing template engines to use JSON-style keys.
// This uses JSON marshal/unmarshal to ensure all nested structs
// are converted to maps with JSON field names.
func dataToJSON(data any) (any, error) {
	if data == nil {
		return nil, nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}
	var result interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}
	return result, nil
}

type TemplateFailureError interface {
	cenclierrors.CencliError
}

type templateFailureError struct {
	path string
	err  error
}

var _ TemplateFailureError = &templateFailureError{}

func newTemplateFailureError(path string, err error) TemplateFailureError {
	return &templateFailureError{path: path, err: err}
}

func (e *templateFailureError) Error() string {
	if e.path == "" {
		return fmt.Sprintf("failed to render template: %v", e.err)
	}
	return fmt.Sprintf("failed to render template at path '%s': %v", e.path, e.err)
}

func (e *templateFailureError) Title() string {
	return "Template Failure"
}

func (e *templateFailureError) ShouldPrintUsage() bool {
	return false
}
