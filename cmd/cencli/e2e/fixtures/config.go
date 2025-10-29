package fixtures

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/templates"
)

var ConfigFixtures = []Fixture{
	{
		Name:      "template-reset",
		Args:      []string{"templates", "reset", "--yes"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: false,
		Setup: mo.Some(func(t *testing.T, dataDir string) {
			// modify one of the default templates and ensure it is migrated
			templatePath := filepath.Join(dataDir, "templates", "host.hbs")
			assert.FileExists(t, templatePath)
			require.NoError(t, os.WriteFile(templatePath, []byte("AAAAAA"), 0o644))
			contents, err := os.ReadFile(templatePath)
			require.NoError(t, err)
			assert.Equal(t, "AAAAAA", string(contents))
			// delete one of the other default templates
			os.Remove(filepath.Join(dataDir, "templates", "certificate.hbs"))
		}),
		Assert: func(t *testing.T, dataDir string, stdout, stderr []byte) {
			assert.Contains(t, string(stdout), "Successfully reset 4 template(s):")
			assert.Contains(t, string(stdout), "host.hbs")
			assert.Contains(t, string(stdout), "certificate.hbs")
			assert.Contains(t, string(stdout), "webproperty.hbs")
			assert.Contains(t, string(stdout), "searchresult.hbs")
			// make sure the template files are all correct
			hostTemplate, err := os.ReadFile(filepath.Join(dataDir, "templates", "host.hbs"))
			require.NoError(t, err)
			assert.Equal(t, templates.HostTemplate, hostTemplate)
			certificateTemplate, err := os.ReadFile(filepath.Join(dataDir, "templates", "certificate.hbs"))
			require.NoError(t, err)
			assert.Equal(t, templates.CertificateTemplate, certificateTemplate)
			webpropertyTemplate, err := os.ReadFile(filepath.Join(dataDir, "templates", "webproperty.hbs"))
			require.NoError(t, err)
			assert.Equal(t, templates.WebPropertyTemplate, webpropertyTemplate)
			searchresultTemplate, err := os.ReadFile(filepath.Join(dataDir, "templates", "searchresult.hbs"))
			require.NoError(t, err)
			assert.Equal(t, templates.SearchResultTemplate, searchresultTemplate)
		},
	},
}
