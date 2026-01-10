package command

import (
	"context"
	"log/slog"
	"sync"

	"github.com/censys/cencli/internal/app/aggregate"
	"github.com/censys/cencli/internal/app/censeye"
	"github.com/censys/cencli/internal/app/history"
	"github.com/censys/cencli/internal/app/search"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/store"
)

// Context is the set of dependencies that are injected into each command.
type Context struct {
	config              *config.Config
	store               store.Store
	censysClient        client.Client
	logger              *slog.Logger
	colorDisabledStdout bool
	colorDisabledStderr bool
	// services
	viewSvc      view.Service
	searchSvc    search.Service
	aggregateSvc aggregate.Service
	historySvc   history.Service
	censeyeSvc   censeye.Service
}

// ContextOpts are functional options for configuring Context
type ContextOpts func(*Context)

func NewCommandContext(
	cfg *config.Config,
	st store.Store,
	opts ...ContextOpts,
) *Context {
	c := &Context{config: cfg, store: st, logger: slog.Default()}
	for _, opt := range opts {
		opt(c)
	}
	if c.config.NoColor || styles.ColorDisabled() {
		// globally disable lipgloss styles
		styles.DisableStyles()
	}
	if styles.ColorForced() {
		styles.EnableStyles()
	} else {
		if c.config.NoColor || styles.ColorDisabled() || !formatter.StdoutIsTTY() {
			c.colorDisabledStdout = true
		}
		if c.config.NoColor || styles.ColorDisabled() || !formatter.StderrIsTTY() {
			c.colorDisabledStderr = true
		}
	}
	return c
}

func (c *Context) Config() *config.Config { return c.config }
func (c *Context) Store() store.Store     { return c.store }

// SetLogger sets the logger used by commands created with this context.
func (c *Context) SetLogger(l *slog.Logger) { c.logger = l }

// SetClient sets the Context's client so that it can be used to initialize services.
func (c *Context) SetCensysClient(cli client.Client) { c.censysClient = cli }

// Logger returns a logger pre-populated with the command name field.
func (c *Context) Logger(cmdName string) *slog.Logger {
	return c.logger.With("cmd", cmdName)
}

// =====================
// Context Abilities
// =====================

// WithProgress executes fn with progress reporting enabled.
// Progress events from the application layer are displayed via spinner (if enabled)
// and logged at debug level with the provided logger.
//
// Parameters:
//   - ctx: The context to enhance with progress reporting
//   - logger: Logger that will receive progress events (inherits command context fields)
//   - initialMessage: Message to display when progress starts (e.g. "Fetching data...")
//   - fn: Function to execute with progress-enabled context
//
// The function ensures the progress display is properly stopped even if fn panics or returns early.
func (c *Context) WithProgress(
	ctx context.Context,
	logger *slog.Logger,
	initialMessage string,
	fn func(context.Context) cenclierrors.CencliError,
) cenclierrors.CencliError {
	ctxWithProgress, stop := c.startProgress(ctx, logger, initialMessage)
	var err cenclierrors.CencliError
	defer func() {
		stop(err)
	}()

	err = fn(ctxWithProgress)
	return err
}

func (c *Context) PrintData(cmd Command, data any) cenclierrors.CencliError {
	// Streaming formats are handled by WithStreamingOutput - nothing to do here
	if c.config.Streaming {
		return nil
	}

	switch c.config.OutputFormat {
	case formatter.OutputFormatShort:
		if c.colorDisabledStdout {
			enable := styles.TemporarilyDisableStyles()
			defer enable()
		}
		return cmd.RenderShort()
	case formatter.OutputFormatTemplate:
		if c.colorDisabledStdout {
			enable := styles.TemporarilyDisableStyles()
			defer enable()
		}
		return cmd.RenderTemplate()
	default:
		return formatter.PrintByFormat(data, c.config.OutputFormat, !c.colorDisabledStdout)
	}
}

// PrintYAML renders data as YAML.
func (c *Context) PrintYAML(data any) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(formatter.PrintYAML(data, !c.colorDisabledStdout))
}

// PrintDataWithTemplate renders data through a template and writes the result to stdout.
func (c *Context) PrintDataWithTemplate(entity config.TemplateEntity, data any) cenclierrors.CencliError {
	templateConfig, err := c.config.GetTemplate(entity)
	if err != nil {
		return err
	}
	return formatter.PrintDataWithTemplate(templateConfig.Path, !c.colorDisabledStdout, data)
}

// PrintAppResponseMeta renders application-level response metadata to stderr.
// If the quiet flag is set, this is a no-op.
// If the debug flag is set, this will also print the headers.
func (c *Context) PrintAppResponseMeta(meta *responsemeta.ResponseMeta) {
	if !c.config.Quiet && meta != nil {
		formatter.PrintAppResponseMeta(styles.GlobalStyles, meta, c.config.Debug, !c.colorDisabledStderr)
	}
}

// WithStreamingOutput sets up streaming output infrastructure when streaming mode is enabled.
// For non-streaming mode, this is a no-op.
//
// Returns a context with a streaming emitter attached (if streaming) and a stop function
// that must be called to properly clean up resources. The stop function should be deferred
// immediately after calling WithStreamingOutput.
//
// Example usage:
//
//	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
//	defer stopStreaming(nil)
func (c *Context) WithStreamingOutput(
	ctx context.Context,
	logger *slog.Logger,
) (context.Context, func(error)) {
	// No-op for non-streaming formats
	if !c.config.Streaming {
		return ctx, func(error) {}
	}

	emitter, items := streaming.NewChannelEmitter(1)
	ctx = streaming.WithEmitter(ctx, emitter)

	// Start goroutine to consume and write items immediately
	done := make(chan struct{})
	go func() {
		defer close(done)
		for item := range items {
			if item.Done {
				break
			}
			if item.Err != nil {
				logger.Debug("streaming item error", "error", item.Err)
				continue
			}
			if err := formatter.WriteNDJSONItem(formatter.Stdout, item.Data, !c.colorDisabledStdout); err != nil {
				logger.Debug("failed to write streaming item", "error", err)
			}
		}
	}()

	var once sync.Once
	stop := func(finalErr error) {
		once.Do(func() {
			emitter.Close(finalErr)
			<-done
		})
	}

	return ctx, stop
}

// =====================
// Service-specific
// =====================

// ViewService attempts to provide a ViewService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) ViewService() (view.Service, cenclierrors.CencliError) {
	if c.viewSvc != nil {
		return c.viewSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.viewSvc = view.New(c.censysClient)
	return c.viewSvc, nil
}

// WithViewService injects an instantiated ViewService to the Context.
// This should only be used in tests, as in the application,
// the ViewService will be instantiated on demand.
func WithViewService(svc view.Service) ContextOpts {
	return func(c *Context) { c.viewSvc = svc }
}

// SearchService attempts to provide a SearchService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) SearchService() (search.Service, cenclierrors.CencliError) {
	if c.searchSvc != nil {
		return c.searchSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.searchSvc = search.New(c.censysClient)
	return c.searchSvc, nil
}

// WithSearchService injects an instantiated SearchService to the Context.
// This should only be used in tests, as in the application,
// the SearchService will be instantiated on demand.
func WithSearchService(svc search.Service) ContextOpts {
	return func(c *Context) { c.searchSvc = svc }
}

// CenseyeService attempts to provide a CenseyeService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) CenseyeService() (censeye.Service, cenclierrors.CencliError) {
	if c.censeyeSvc != nil {
		return c.censeyeSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize
	c.censeyeSvc = censeye.New(c.censysClient)
	return c.censeyeSvc, nil
}

// WithCenseyeService injects an instantiated CenseyeService to the Context.
// This should only be used in tests; in the app the service is instantiated on demand.
func WithCenseyeService(svc censeye.Service) ContextOpts {
	return func(c *Context) { c.censeyeSvc = svc }
}

// HistoryService attempts to provide a HistoryService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) HistoryService() (history.Service, cenclierrors.CencliError) {
	if c.historySvc != nil {
		return c.historySvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize
	c.historySvc = history.New(c.censysClient)
	return c.historySvc, nil
}

// WithHistoryService injects an instantiated HistoryService to the Context.
// This should only be used in tests; in the app the service is instantiated on demand.
func WithHistoryService(svc history.Service) ContextOpts {
	return func(c *Context) { c.historySvc = svc }
}

// AggregateService attempts to provide a AggregateService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) AggregateService() (aggregate.Service, cenclierrors.CencliError) {
	if c.aggregateSvc != nil {
		return c.aggregateSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.aggregateSvc = aggregate.New(c.censysClient)
	return c.aggregateSvc, nil
}

// WithAggregateService injects an instantiated AggregateService to the Context.
// This should only be used in tests, as in the application,
// the AggregateService will be instantiated on demand.
func WithAggregateService(svc aggregate.Service) ContextOpts {
	return func(c *Context) { c.aggregateSvc = svc }
}
