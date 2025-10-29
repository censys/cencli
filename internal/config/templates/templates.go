package templates

import (
	"embed"
	"encoding"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

//go:embed defaults
var defaultTemplates embed.FS

const (
	defaultsDir = "defaults"
	templateDir = "templates"
)

type TemplateEntity string

const (
	TemplateEntityHost         TemplateEntity = "host"
	TemplateEntityCertificate  TemplateEntity = "certificate"
	TemplateEntityWebProperty  TemplateEntity = "webproperty"
	TemplateEntitySearchResult TemplateEntity = "searchresult"
)

var ErrUnsupportedTemplateEntity = fmt.Errorf("unsupported template entity type")

func (a TemplateEntity) String() string {
	return string(a)
}

type TemplateConfig struct {
	Path string `yaml:"path" mapstructure:"path" doc:"Path to the template file"`
}

var DefaultTemplateConfig = map[TemplateEntity]TemplateConfig{
	// will be potentially updated at runtime
	TemplateEntityHost:         {},
	TemplateEntityCertificate:  {},
	TemplateEntityWebProperty:  {},
	TemplateEntitySearchResult: {},
}

var _ encoding.TextUnmarshaler = (*TemplateEntity)(nil)

func (a *TemplateEntity) UnmarshalText(text []byte) error {
	s := string(text)
	switch s {
	case TemplateEntityHost.String():
		*a = TemplateEntityHost
	case TemplateEntityCertificate.String():
		*a = TemplateEntityCertificate
	case TemplateEntityWebProperty.String():
		*a = TemplateEntityWebProperty
	case TemplateEntitySearchResult.String():
		*a = TemplateEntitySearchResult
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedTemplateEntity, s)
	}
	return nil
}

// InitTemplates validates existing template paths and creates default templates if needed.
// Returns the updated template configuration map.
// If a template file is missing, it will set the path but not fail - the error will occur
// when actually trying to use the template.
func InitTemplates(dataDir string, templateConfigs map[TemplateEntity]TemplateConfig) (map[TemplateEntity]TemplateConfig, cenclierrors.CencliError) {
	templatesDir := filepath.Join(dataDir, templateDir)
	// the templates directory always exists, even if unused
	if err := ensureTemplatesDirectory(templatesDir); err != nil {
		return nil, err
	}

	// Create a copy of the template configs to modify
	updatedConfigs := make(map[TemplateEntity]TemplateConfig)
	for k, v := range templateConfigs {
		updatedConfigs[k] = v
	}

	// validate each template
	for entity, template := range updatedConfigs {
		// if the path is set, just keep it (don't validate during init)
		// validation will happen when the template is actually used
		if template.Path != "" {
			continue
		}
		// the path needs to be set
		// look for existing template files in templates directory
		existingTemplate, err := findExistingTemplateInDir(entity, templatesDir)
		if err != nil {
			return nil, err
		}
		// if an existing template is found, set the path
		if existingTemplate != "" {
			template.Path = filepath.Join(templatesDir, existingTemplate)
			updatedConfigs[entity] = template
			continue
		}
		// if no existing template is found, copy the default template
		defaultTemplateName, err := findDefaultTemplateInEmbedded(entity)
		if err != nil {
			return nil, err
		}
		if err := CopyDefaultTemplate(defaultTemplateName, templatesDir); err != nil {
			return nil, err
		}
		// set the path
		template.Path = filepath.Join(templatesDir, defaultTemplateName)
		updatedConfigs[entity] = template
	}

	return updatedConfigs, nil
}

// ResetTemplates replaces all template files with the latest defaults.
// This is useful when templates are in a broken state or the user wants to update to new defaults.
// Unlike InitTemplates, this will overwrite existing files and ignore any errors about missing files.
func ResetTemplates(dataDir string) ([]string, cenclierrors.CencliError) {
	templatesDir := filepath.Join(dataDir, templateDir)

	// Ensure templates directory exists
	if err := ensureTemplatesDirectory(templatesDir); err != nil {
		return nil, err
	}

	// Get list of default templates
	defaultTemplateNames, err := ListDefaultTemplates()
	if err != nil {
		return nil, cenclierrors.NewCencliError(fmt.Errorf("failed to list default templates: %w", err))
	}

	var reset []string
	for _, templateName := range defaultTemplateNames {
		// Copy default template, overwriting any existing file
		if err := CopyDefaultTemplate(templateName, templatesDir); err != nil {
			// Continue on error - we want to reset as many as possible
			continue
		}
		reset = append(reset, templateName)
	}

	return reset, nil
}

// validateExistingTemplatePath checks if a configured template path exists and is valid.
func validateExistingTemplatePath(entity TemplateEntity, templateConfig TemplateConfig) cenclierrors.CencliError {
	if _, err := os.Stat(templateConfig.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newTemplateNotFoundError(string(entity), templateConfig.Path)
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to check template file for entity '%s': %w", entity, err))
	}

	return nil
}

// findExistingTemplateInDir searches for template files matching the entity name pattern.
func findExistingTemplateInDir(entity TemplateEntity, templatesDir string) (string, cenclierrors.CencliError) {
	pattern := regexp.MustCompile(fmt.Sprintf("^%s\\.", regexp.QuoteMeta(string(entity))))

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return "", newTemplateDirectoryError("read", templatesDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && pattern.MatchString(entry.Name()) {
			return entry.Name(), nil
		}
	}

	return "", nil
}

// findDefaultTemplateInEmbedded searches for default template files in the embedded FS.
func findDefaultTemplateInEmbedded(entity TemplateEntity) (string, cenclierrors.CencliError) {
	pattern := regexp.MustCompile(fmt.Sprintf("^%s\\.", regexp.QuoteMeta(string(entity))))

	embeddedEntries, err := defaultTemplates.ReadDir(defaultsDir)
	if err != nil {
		return "", cenclierrors.NewCencliError(fmt.Errorf("failed to read embedded templates directory: %w", err))
	}

	for _, entry := range embeddedEntries {
		if !entry.IsDir() && pattern.MatchString(entry.Name()) {
			return entry.Name(), nil
		}
	}

	return "", NewDefaultTemplateNotFoundError(string(entity))
}

// CopyDefaultTemplate copies a default template from embedded FS to the templates directory.
// This is exported so it can be used by the migration command.
func CopyDefaultTemplate(templateName, templatesDir string) cenclierrors.CencliError {
	// for some reason filepath.Join doesn't work with embedded FS
	// on windows, but path.Join does
	defaultTemplate, err := defaultTemplates.ReadFile(path.Join(defaultsDir, templateName))
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to read default template '%s': %w", templateName, err))
	}

	templatePath := filepath.Join(templatesDir, templateName)
	if err := os.WriteFile(templatePath, defaultTemplate, 0o644); err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to write template '%s': %w", templateName, err))
	}

	return nil
}

// ensureTemplatesDirectory creates the templates directory if it doesn't exist.
func ensureTemplatesDirectory(templatesDir string) cenclierrors.CencliError {
	if _, err := os.Stat(templatesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(templatesDir, 0o700); err != nil {
				return newTemplateDirectoryError("create", templatesDir, err)
			}
		} else {
			return newTemplateDirectoryError("check", templatesDir, err)
		}
	}
	return nil
}

// GetTemplatesDir returns the path to the templates directory for a given data directory.
func GetTemplatesDir(dataDir string) string {
	return filepath.Join(dataDir, templateDir)
}

// ListDefaultTemplates returns a list of all default template names available in the embedded FS.
func ListDefaultTemplates() ([]string, error) {
	entries, err := defaultTemplates.ReadDir(defaultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates directory: %w", err)
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			templates = append(templates, entry.Name())
		}
	}
	return templates, nil
}

type TemplateNotRegisteredError interface {
	cenclierrors.CencliError
}

type templateNotRegisteredError struct {
	entity string
}

var _ TemplateNotRegisteredError = &templateNotRegisteredError{}

func NewTemplateNotRegisteredError(entity string) TemplateNotRegisteredError {
	return &templateNotRegisteredError{
		entity: entity,
	}
}

func (e *templateNotRegisteredError) Error() string {
	return fmt.Sprintf("template not registered for entity: %s", e.entity)
}

func (e *templateNotRegisteredError) Title() string {
	return "Template Not Registered"
}

func (e *templateNotRegisteredError) ShouldPrintUsage() bool {
	return false
}

type TemplateNotFoundError interface {
	cenclierrors.CencliError
}

type templateNotFoundError struct {
	entity string
	path   string
}

var _ TemplateNotFoundError = &templateNotFoundError{}

func newTemplateNotFoundError(entity, path string) TemplateNotFoundError {
	return &templateNotFoundError{
		entity: entity,
		path:   path,
	}
}

func (e *templateNotFoundError) Error() string {
	return fmt.Sprintf("template file not found for entity '%s' at path: %s", e.entity, e.path)
}

func (e *templateNotFoundError) Title() string {
	return "Template Not Found"
}

func (e *templateNotFoundError) ShouldPrintUsage() bool {
	return false
}

type DefaultTemplateNotFoundError interface {
	cenclierrors.CencliError
	Entity() string
}

type defaultTemplateNotFoundError struct {
	entity string
}

var _ DefaultTemplateNotFoundError = &defaultTemplateNotFoundError{}

func NewDefaultTemplateNotFoundError(entity string) DefaultTemplateNotFoundError {
	return &defaultTemplateNotFoundError{
		entity: entity,
	}
}

func (e *defaultTemplateNotFoundError) Entity() string {
	return e.entity
}

func (e *defaultTemplateNotFoundError) Error() string {
	return fmt.Sprintf("no default template found for entity: %s", e.entity)
}

func (e *defaultTemplateNotFoundError) Title() string {
	return "Default Template Not Found"
}

func (e *defaultTemplateNotFoundError) ShouldPrintUsage() bool {
	return false
}

type TemplateDirectoryError interface {
	cenclierrors.CencliError
}

type templateDirectoryError struct {
	operation string
	path      string
	err       error
}

var _ TemplateDirectoryError = &templateDirectoryError{}

func newTemplateDirectoryError(operation, path string, err error) TemplateDirectoryError {
	return &templateDirectoryError{
		operation: operation,
		path:      path,
		err:       err,
	}
}

func (e *templateDirectoryError) Error() string {
	return fmt.Sprintf("failed to %s template directory: %v", e.operation, e.err)
}

func (e *templateDirectoryError) Title() string {
	return "Template Directory Error"
}

func (e *templateDirectoryError) ShouldPrintUsage() bool {
	return false
}

func (e *templateDirectoryError) Unwrap() error {
	return e.err
}
