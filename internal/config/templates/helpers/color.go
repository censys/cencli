package helpers

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/styles"
)

type ColorHelperConfig struct {
	Name  string
	Color styles.Color
}

func NewColorHelpers(enabled bool, configs ...ColorHelperConfig) []HandlebarsHelper {
	helpers := make([]HandlebarsHelper, len(configs))
	for i, config := range configs {
		helpers[i] = newColorHelper(config.Name, config.Color, enabled)
	}
	return helpers
}

type colorHelper struct {
	name    string
	color   styles.Color
	enabled bool
}

var _ HandlebarsHelper = &colorHelper{}

func newColorHelper(name string, color styles.Color, enabled bool) HandlebarsHelper {
	return &colorHelper{name: name, color: color, enabled: enabled}
}

func (h *colorHelper) Name() string {
	return h.name
}

func (h *colorHelper) Function() any {
	return func(v any) string {
		if !h.enabled {
			return fmt.Sprint(v)
		}
		return styles.NewStyle(h.color).Render(fmt.Sprint(v))
	}
}
