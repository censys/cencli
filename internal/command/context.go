package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/aggregate"
	"github.com/censys/cencli/internal/app/censeye"
	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/app/enrich"
	"github.com/censys/cencli/internal/app/history"
	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/app/rescan"
	"github.com/censys/cencli/internal/app/search"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/credential"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
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
	enrichSvc    enrich.Service
	searchSvc    search.Service
	aggregateSvc aggregate.Service
	historySvc   history.Service
	censeyeSvc   censeye.Service
	creditsSvc   credits.Service
	orgSvc       organizations.Service
	tagsSvc      tags.Service
	rescanSvc    rescan.Service
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
	c.updateColorSettings()
	return c
}

// updateColorSettings evaluates and updates the color settings based on current config.
// This should be called after config is loaded or re-unmarshaled.
func (c *Context) updateColorSettings() {
	if c.config.NoColor || styles.ColorDisabled() {
		// globally disable lipgloss styles
		styles.DisableStyles()
	}
	if styles.ColorForced() {
		styles.EnableStyles()
	} else {
		if c.config.NoColor || styles.ColorDisabled() || !formatter.StdoutIsTTY() {
			c.colorDisabledStdout = true
		} else {
			c.colorDisabledStdout = false
		}
		if c.config.NoColor || styles.ColorDisabled() || !formatter.StderrIsTTY() {
			c.colorDisabledStderr = true
		} else {
			c.colorDisabledStderr = false
		}
	}
}

func (c *Context) Config() *config.Config { return c.config }
func (c *Context) Store() store.Store     { return c.store }

// SetLogger sets the logger used by commands created with this context.
func (c *Context) SetLogger(l *slog.Logger) { c.logger = l }

// SetClient sets the Context's client so that it can be used to initialize services.
func (c *Context) SetCensysClient(cli client.Client) { c.censysClient = cli }

// HasOrgID returns true if the context has a configured organization ID.
func (c *Context) HasOrgID() bool {
	return c.censysClient != nil && c.censysClient.HasOrgID()
}

// credentialInfo returns the active credential as reported by the client, or a
// zero (KindNone) value when no client is set (e.g. before auth, or a test that
// doesn't inject one).
func (c *Context) credentialInfo() credential.Info {
	if c.censysClient == nil {
		return credential.Info{Kind: credential.KindNone}
	}
	return c.censysClient.CredentialInfo()
}

// ResolveOrgID determines the organization a command should target.
//
// The two manual sources — the --org-id flag and the stored org-id global —
// apply only to credentials that permit it (see credential.Info.AllowsManualOrg,
// which allowlists personal access tokens). Every other credential kind carries
// its own organization binding: that organization is used, and --org-id is
// rejected as a usage error because it cannot apply.
func (c *Context) ResolveOrgID(ctx context.Context, flagOrgID mo.Option[identifiers.OrganizationID]) (mo.Option[identifiers.OrganizationID], cenclierrors.CencliError) {
	zero := mo.None[identifiers.OrganizationID]()
	info := c.credentialInfo()

	// ---- Not organization-scoped: the caller picks the organization per request.
	if info.AllowsManualOrg() {
		if flagOrgID.IsPresent() {
			return flagOrgID, nil
		}
		return c.storedOrgID(ctx)
	}

	// ---- Credential carries its own organization; flags/config cannot override it.
	if flagOrgID.IsPresent() {
		return zero, cenclierrors.NewOrgIDNotApplicableError(credentialScopeTarget(info))
	}
	if info.OrgID == "" {
		return zero, nil // e.g. a free-account OAuth session: no organization to target
	}
	boundOrg, err := parseOrgID(info.OrgID)
	if err != nil {
		return zero, err
	}
	return mo.Some(boundOrg), nil
}

// ResolveRequiredOrgID is ResolveOrgID for commands that cannot run without an
// organization. It distinguishes the two reasons an organization may be missing,
// so the message names the real problem:
//
//   - the credential is locked to the free account and can never target an
//     organization, or
//   - the credential can target one, but none was given or stored.
func (c *Context) ResolveRequiredOrgID(
	cmd *cobra.Command,
	flagOrgID mo.Option[identifiers.OrganizationID],
) (identifiers.OrganizationID, cenclierrors.CencliError) {
	// Resolve first, so an inapplicable --org-id is always reported as such,
	// consistently with every other command that accepts the flag.
	orgID, err := c.ResolveOrgID(cmd.Context(), flagOrgID)
	if err != nil {
		return identifiers.OrganizationID{}, err
	}
	if orgID.IsPresent() {
		return orgID.MustGet(), nil
	}

	// Nothing to target. Say which of the two reasons applies.
	info := c.credentialInfo()
	if info.IsBoundToFreeAccount() {
		return identifiers.OrganizationID{}, cenclierrors.NewOrganizationRequiredError(
			cmd.CommandPath(), credentialScopeTarget(info))
	}
	return identifiers.OrganizationID{}, cenclierrors.NewNoOrgIDError()
}

// EnsureFreeAccountAccess errors when the active credential is locked to an
// organization, and so cannot read the user's free account. alternative is an
// optional sentence pointing at the organization equivalent of the command.
func (c *Context) EnsureFreeAccountAccess(cmd *cobra.Command, alternative string) cenclierrors.CencliError {
	info := c.credentialInfo()
	if info.IsBoundToOrg() {
		return cenclierrors.NewFreeAccountRequiredError(
			cmd.CommandPath(), credentialScopeTarget(info), alternative)
	}
	return nil
}

// credentialScopeTarget names what the active credential is scoped to, for use
// after "…is scoped to " in error text: "the organization [X]", or "your free
// account" when the credential carries no organization.
func credentialScopeTarget(info credential.Info) string {
	switch {
	case info.OrgName != "":
		return fmt.Sprintf("the organization [%s]", info.OrgName)
	case info.OrgID != "":
		return fmt.Sprintf("the organization [%s]", info.OrgID)
	default:
		return "your free account"
	}
}

// storedOrgID reads the org-id global. Personal-access-token path only: an
// OAuth login takes its organization from the login itself.
func (c *Context) storedOrgID(ctx context.Context) (mo.Option[identifiers.OrganizationID], cenclierrors.CencliError) {
	zero := mo.None[identifiers.OrganizationID]()
	storedOrgID, err := c.store.GetLastUsedGlobalByName(ctx, config.OrgIDGlobalName)
	if err != nil {
		if errors.Is(err, store.ErrGlobalNotFound) {
			return zero, nil
		}
		return zero, cenclierrors.NewCencliError(err)
	}
	parsed, perr := parseOrgID(storedOrgID.Value)
	if perr != nil {
		return zero, perr
	}
	return mo.Some(parsed), nil
}

func parseOrgID(value string) (identifiers.OrganizationID, cenclierrors.CencliError) {
	parsedUUID, err := uuid.Parse(value)
	if err != nil {
		return identifiers.OrganizationID{}, cenclierrors.NewCencliError(err)
	}
	return identifiers.NewOrganizationID(parsedUUID), nil
}

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

// PrintValueByFormat renders a single structured value according to the
// configured output format. For formats that rely on per-asset rendering
// (short, template) or that require streaming (ndjson), it falls back to the
// provided plain-text representation. Unlike PrintData, it does not short-circuit
// on the streaming flag, so it is safe on paths that never set up a streaming
// emitter (e.g. the search --count path).
func (c *Context) PrintValueByFormat(data any, plain string) cenclierrors.CencliError {
	switch c.config.OutputFormat {
	case formatter.OutputFormatShort, formatter.OutputFormatTemplate, formatter.OutputFormatNDJSON:
		formatter.Println(formatter.Stdout, plain)
		return nil
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

// EnrichService attempts to provide an EnrichService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) EnrichService() (enrich.Service, cenclierrors.CencliError) {
	if c.enrichSvc != nil {
		return c.enrichSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.enrichSvc = enrich.New(c.censysClient)
	return c.enrichSvc, nil
}

// WithEnrichService injects an instantiated EnrichService to the Context.
// This should only be used in tests, as in the application,
// the EnrichService will be instantiated on demand.
func WithEnrichService(svc enrich.Service) ContextOpts {
	return func(c *Context) { c.enrichSvc = svc }
}

// TagsService attempts to provide a TagsService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) TagsService() (tags.Service, cenclierrors.CencliError) {
	if c.tagsSvc != nil {
		return c.tagsSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.tagsSvc = tags.New(c.censysClient)
	return c.tagsSvc, nil
}

// WithTagsService injects an instantiated TagsService to the Context.
// This should only be used in tests, as in the application,
// the TagsService will be instantiated on demand.
func WithTagsService(svc tags.Service) ContextOpts {
	return func(c *Context) { c.tagsSvc = svc }
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

// CreditsService attempts to provide a CreditsService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) CreditsService() (credits.Service, cenclierrors.CencliError) {
	if c.creditsSvc != nil {
		return c.creditsSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.creditsSvc = credits.New(c.censysClient)
	return c.creditsSvc, nil
}

// WithCreditsService injects an instantiated CreditsService to the Context.
// This should only be used in tests, as in the application,
// the CreditsService will be instantiated on demand.
func WithCreditsService(svc credits.Service) ContextOpts {
	return func(c *Context) { c.creditsSvc = svc }
}

// OrganizationsService attempts to provide an OrganizationsService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) OrganizationsService() (organizations.Service, cenclierrors.CencliError) {
	if c.orgSvc != nil {
		return c.orgSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	// Memoize the service instance since it's stateless and thread-safe for reuse
	c.orgSvc = organizations.New(c.censysClient)
	return c.orgSvc, nil
}

// WithOrganizationsService injects an instantiated OrganizationsService to the Context.
// This should only be used in tests, as in the application,
// the OrganizationsService will be instantiated on demand.
func WithOrganizationsService(svc organizations.Service) ContextOpts {
	return func(c *Context) { c.orgSvc = svc }
}

// RescanService attempts to provide a RescanService to the caller.
// If it is not already set and is unable to be instantiated, it will return an error.
func (c *Context) RescanService() (rescan.Service, cenclierrors.CencliError) {
	if c.rescanSvc != nil {
		return c.rescanSvc, nil
	}
	if c.censysClient == nil {
		return nil, client.NewCensysClientNotConfiguredError()
	}
	c.rescanSvc = rescan.New(c.censysClient)
	return c.rescanSvc, nil
}

// WithRescanService injects an instantiated RescanService to the Context.
// This should only be used in tests, as in the application,
// the RescanService will be instantiated on demand.
func WithRescanService(svc rescan.Service) ContextOpts {
	return func(c *Context) { c.rescanSvc = svc }
}
