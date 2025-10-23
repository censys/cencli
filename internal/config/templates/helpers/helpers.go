package helpers

import (
	"sync"

	handlebars "github.com/aymerick/raymond"
)

// HandlebarsHelper is used to register helpers with the Handlebars template engine.
type HandlebarsHelper interface {
	Name() string
	Function() any
}

var (
	registeredHelpers = make(map[string]bool)
	mu                sync.Mutex
)

// RegisterHelpers registers helpers with the Handlebars template engine.
// It keeps track of already registered helpers to avoid panics from re-registration.
func RegisterHelpers(helpers ...HandlebarsHelper) {
	mu.Lock()
	defer mu.Unlock()

	for _, helper := range helpers {
		name := helper.Name()
		if !registeredHelpers[name] {
			handlebars.RegisterHelper(name, helper.Function())
			registeredHelpers[name] = true
		}
	}
}
