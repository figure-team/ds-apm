package signoz

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/alertmanager"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager/nfroutingstore/sqlroutingstore"
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/apiserver"
	"github.com/SigNoz/signoz/pkg/auditor"
	"github.com/SigNoz/signoz/pkg/authn"
	"github.com/SigNoz/signoz/pkg/authn/authnstore/sqlauthnstore"
	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/cache"
	"github.com/SigNoz/signoz/pkg/emailing"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/gateway"
	"github.com/SigNoz/signoz/pkg/global"
	"github.com/SigNoz/signoz/pkg/identn"
	"github.com/SigNoz/signoz/pkg/instrumentation"
	"github.com/SigNoz/signoz/pkg/licensing"
	"github.com/SigNoz/signoz/pkg/modules/cloudintegration"
	"github.com/SigNoz/signoz/pkg/modules/dashboard"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/modules/organization/implorganization"
	"github.com/SigNoz/signoz/pkg/modules/rulestatehistory"
	"github.com/SigNoz/signoz/pkg/modules/serviceaccount"
	"github.com/SigNoz/signoz/pkg/modules/serviceaccount/implserviceaccount"
	"github.com/SigNoz/signoz/pkg/modules/user/impluser"
	"github.com/SigNoz/signoz/pkg/prometheus"
	"github.com/SigNoz/signoz/pkg/querier"
	"github.com/SigNoz/signoz/pkg/queryparser"
	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/sqlaiconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/dispatchhook"
	"github.com/SigNoz/signoz/pkg/ruler/aihistorystore/sqlaihistorystore"
	"github.com/SigNoz/signoz/pkg/ruler/sopstore/sqlsopstore"
	"github.com/SigNoz/signoz/pkg/sharder"
	"github.com/SigNoz/signoz/pkg/sqlmigration"
	"github.com/SigNoz/signoz/pkg/sqlmigrator"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/statsreporter"
	"github.com/SigNoz/signoz/pkg/telemetryaudit"
	"github.com/SigNoz/signoz/pkg/telemetrylogs"
	"github.com/SigNoz/signoz/pkg/telemetrymetadata"
	"github.com/SigNoz/signoz/pkg/telemetrymeter"
	"github.com/SigNoz/signoz/pkg/telemetrymetrics"
	"github.com/SigNoz/signoz/pkg/telemetrystore"
	"github.com/SigNoz/signoz/pkg/telemetrytraces"
	pkgtokenizer "github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/version"
	"github.com/SigNoz/signoz/pkg/zeus"

	"github.com/SigNoz/signoz/pkg/web"

	codercaauditor "github.com/SigNoz/signoz/pkg/ruler/coderca/auditor"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseservicemapstore"
	codercadelivery "github.com/SigNoz/signoz/pkg/ruler/coderca/delivery"
	codercaengine "github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/reporesolver"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	codercasourcestate "github.com/SigNoz/signoz/pkg/ruler/coderca/sourcestate"
	codercatrigger "github.com/SigNoz/signoz/pkg/ruler/coderca/trigger"
	codercaworker "github.com/SigNoz/signoz/pkg/ruler/coderca/worker"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore/sqlremediationstore"
	"github.com/SigNoz/signoz/pkg/ruler/signozruler"
	"path/filepath"
)

type SigNoz struct {
	*factory.Registry
	Instrumentation        instrumentation.Instrumentation
	Analytics              analytics.Analytics
	Cache                  cache.Cache
	Web                    web.Web
	SQLStore               sqlstore.SQLStore
	TelemetryStore         telemetrystore.TelemetryStore
	TelemetryMetadataStore telemetrytypes.MetadataStore
	Prometheus             prometheus.Prometheus
	Alertmanager           alertmanager.Alertmanager
	Querier                querier.Querier
	APIServer              apiserver.APIServer
	Zeus                   zeus.Zeus
	Licensing              licensing.Licensing
	Emailing               emailing.Emailing
	Sharder                sharder.Sharder
	StatsReporter          statsreporter.StatsReporter
	Tokenizer              pkgtokenizer.Tokenizer
	IdentNResolver         identn.IdentNResolver
	Authz                  authz.AuthZ
	Ruler                  ruler.Ruler
	Modules                Modules
	Handlers               Handlers
	QueryParser            queryparser.QueryParser
	Flagger                flagger.Flagger
	Gateway                gateway.Gateway
	Auditor                auditor.Auditor
}

func New(
	ctx context.Context,
	config Config,
	zeusConfig zeus.Config,
	zeusProviderFactory factory.ProviderFactory[zeus.Zeus, zeus.Config],
	licenseConfig licensing.Config,
	licenseProviderFactory func(sqlstore.SQLStore, zeus.Zeus, organization.Getter, analytics.Analytics) factory.ProviderFactory[licensing.Licensing, licensing.Config],
	emailingProviderFactories factory.NamedMap[factory.ProviderFactory[emailing.Emailing, emailing.Config]],
	cacheProviderFactories factory.NamedMap[factory.ProviderFactory[cache.Cache, cache.Config]],
	webProviderFactories factory.NamedMap[factory.ProviderFactory[web.Web, web.Config]],
	sqlSchemaProviderFactories func(sqlstore.SQLStore) factory.NamedMap[factory.ProviderFactory[sqlschema.SQLSchema, sqlschema.Config]],
	sqlstoreProviderFactories factory.NamedMap[factory.ProviderFactory[sqlstore.SQLStore, sqlstore.Config]],
	telemetrystoreProviderFactories factory.NamedMap[factory.ProviderFactory[telemetrystore.TelemetryStore, telemetrystore.Config]],
	authNsCallback func(ctx context.Context, providerSettings factory.ProviderSettings, store authtypes.AuthNStore, licensing licensing.Licensing) (map[authtypes.AuthNProvider]authn.AuthN, error),
	authzCallback func(context.Context, sqlstore.SQLStore, licensing.Licensing, []authz.OnBeforeRoleDelete, dashboard.Module) (factory.ProviderFactory[authz.AuthZ, authz.Config], error),
	dashboardModuleCallback func(sqlstore.SQLStore, factory.ProviderSettings, analytics.Analytics, organization.Getter, queryparser.QueryParser, querier.Querier, licensing.Licensing) dashboard.Module,
	gatewayProviderFactory func(licensing.Licensing) factory.ProviderFactory[gateway.Gateway, gateway.Config],
	auditorProviderFactories func(licensing.Licensing) factory.NamedMap[factory.ProviderFactory[auditor.Auditor, auditor.Config]],
	querierHandlerCallback func(factory.ProviderSettings, querier.Querier, analytics.Analytics) querier.Handler,
	cloudIntegrationCallback func(sqlstore.SQLStore, global.Global, zeus.Zeus, gateway.Gateway, licensing.Licensing, serviceaccount.Module, cloudintegration.Config) (cloudintegration.Module, error),
	rulerProviderFactories func(cache.Cache, alertmanager.Alertmanager, sqlstore.SQLStore, telemetrystore.TelemetryStore, telemetrytypes.MetadataStore, prometheus.Prometheus, organization.Getter, rulestatehistory.Module, querier.Querier, queryparser.QueryParser) factory.NamedMap[factory.ProviderFactory[ruler.Ruler, ruler.Config]],
) (*SigNoz, error) {
	// Initialize instrumentation
	instrumentation, err := instrumentation.New(ctx, config.Instrumentation, version.Info, "signoz")
	if err != nil {
		return nil, err
	}

	instrumentation.Logger().InfoContext(ctx, "starting signoz", slog.String("version", version.Info.Version()), slog.String("variant", version.Info.Variant()), slog.String("commit", version.Info.Hash()), slog.String("branch", version.Info.Branch()), slog.String("go", version.Info.GoVersion()), slog.String("time", version.Info.Time()))
	instrumentation.Logger().DebugContext(ctx, "loaded signoz config", slog.Any("config", config))

	// Get the provider settings from instrumentation
	providerSettings := instrumentation.ToProviderSettings()

	pprofService, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.PProf,
		NewPProfProviderFactories(),
		config.PProf.Provider(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize analytics just after instrumentation, as providers might require it
	analytics, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Analytics,
		NewAnalyticsProviderFactories(),
		config.Analytics.Provider(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize zeus from the available zeus provider factory. This is not config controlled
	// and depends on the variant of the build.
	zeus, err := zeusProviderFactory.New(
		ctx,
		providerSettings,
		zeusConfig,
	)
	if err != nil {
		return nil, err
	}

	// Initialize emailing from the available emailing provider factories
	emailing, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Emailing,
		emailingProviderFactories,
		config.Emailing.Provider(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize cache from the available cache provider factories
	cache, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Cache,
		cacheProviderFactories,
		config.Cache.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Initialize flagger from the available flagger provider factories
	flaggerRegistry := flagger.MustNewRegistry()
	flaggerProviderFactories := NewFlaggerProviderFactories(flaggerRegistry)
	flagger, err := flagger.New(
		ctx,
		providerSettings,
		config.Flagger,
		flaggerRegistry,
		flaggerProviderFactories.GetInOrder()...,
	)
	if err != nil {
		return nil, err
	}

	// Initialize web from the available web provider factories
	web, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Web,
		webProviderFactories,
		config.Web.Provider(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize sqlstore from the available sqlstore provider factories
	sqlstore, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.SQLStore,
		sqlstoreProviderFactories,
		config.SQLStore.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Initialize telemetrystore from the available telemetrystore provider factories
	telemetrystore, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.TelemetryStore,
		telemetrystoreProviderFactories,
		config.TelemetryStore.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Initialize prometheus from the available prometheus provider factories
	prometheus, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Prometheus,
		NewPrometheusProviderFactories(telemetrystore),
		config.Prometheus.Provider(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize querier from the available querier provider factories
	querier, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Querier,
		NewQuerierProviderFactories(telemetrystore, prometheus, cache, flagger),
		config.Querier.Provider(),
	)
	if err != nil {
		return nil, err
	}

	sqlschema, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.SQLSchema,
		sqlSchemaProviderFactories(sqlstore),
		config.SQLStore.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Run migrations on the sqlstore
	sqlmigrations, err := sqlmigration.New(
		ctx,
		providerSettings,
		config.SQLMigration,
		NewSQLMigrationProviderFactories(sqlstore, sqlschema, telemetrystore, providerSettings),
	)
	if err != nil {
		return nil, err
	}

	err = sqlmigrator.New(ctx, providerSettings, sqlstore, sqlmigrations, config.SQLMigrator).Migrate(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize sharder from the available sharder provider factories
	sharder, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Sharder,
		NewSharderProviderFactories(),
		config.Sharder.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Initialize organization getter
	orgGetter := implorganization.NewGetter(implorganization.NewStore(sqlstore), sharder)

	// Initialize tokenizer from the available tokenizer provider factories
	tokenizer, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Tokenizer,
		NewTokenizerProviderFactories(cache, sqlstore, orgGetter),
		config.Tokenizer.Provider,
	)
	if err != nil {
		return nil, err
	}

	// Initialize user store
	userStore := impluser.NewStore(sqlstore, providerSettings)

	// Initialize user role store
	userRoleStore := impluser.NewUserRoleStore(sqlstore, providerSettings)

	licensingProviderFactory := licenseProviderFactory(sqlstore, zeus, orgGetter, analytics)
	licensing, err := licensingProviderFactory.New(
		ctx,
		providerSettings,
		licenseConfig,
	)
	if err != nil {
		return nil, err
	}

	// Initialize query parser (needed for dashboard module)
	queryParser := queryparser.New(providerSettings)

	// Initialize dashboard module (needed for authz registry)
	dashboard := dashboardModuleCallback(sqlstore, providerSettings, analytics, orgGetter, queryParser, querier, licensing)

	// Initialize user getter
	userGetter := impluser.NewGetter(userStore, userRoleStore, flagger)

	// Initialize service account getter
	serviceAccountGetter := implserviceaccount.NewGetter(implserviceaccount.NewStore(sqlstore))

	// Build pre-delete callbacks from modules
	onBeforeRoleDelete := []authz.OnBeforeRoleDelete{
		userGetter.OnBeforeRoleDelete,
		serviceAccountGetter.OnBeforeRoleDelete,
	}

	// Initialize authz
	authzProviderFactory, err := authzCallback(ctx, sqlstore, licensing, onBeforeRoleDelete, dashboard)
	if err != nil {
		return nil, err
	}
	authz, err := authzProviderFactory.New(ctx, providerSettings, config.Authz)
	if err != nil {
		return nil, err
	}

	// Initialize notification manager from the available notification manager provider factories
	nfManager, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		nfmanager.Config{},
		NewNotificationManagerProviderFactories(sqlroutingstore.NewStore(sqlstore)),
		"rulebased",
	)
	if err != nil {
		return nil, err
	}

	// Build the AI generator and dispatch hook before the alertmanager so they
	// can be wired in at construction time. The hook is nil-safe: if the
	// generator is disabled (DS_APM_AI_GENERATOR unset) or the sopStore is
	// unavailable, passing nil keeps existing alertmanager behavior intact.
	aiGen, err := aigenerator.New(aigenerator.Config{
		Provider:          os.Getenv("DS_APM_AI_GENERATOR"),
		MockFixtureDir:    os.Getenv("DS_APM_AI_MOCK_FIXTURE_DIR"),
		LLMProvider:       os.Getenv("DS_APM_LLM_PROVIDER"),
		LLMTransport:      os.Getenv("DS_APM_LLM_TRANSPORT"),
		LLMModel:          os.Getenv("DS_APM_LLM_MODEL"),
		LLMTimeoutSeconds: envInt("DS_APM_LLM_TIMEOUT_SECONDS"),
		LLMAPIKey:         pickAPIKey(os.Getenv("DS_APM_LLM_PROVIDER")),
		LLMBinary:         os.Getenv("DS_APM_LLM_BINARY"),
		LLMEndpoint:       os.Getenv("DS_APM_LLM_ENDPOINT"),
		LLMMaxOutputTokens: noticeMaxOutputTokens(
			envInt("DS_APM_AINOTICE_MAX_OUTPUT_TOKENS"),
			envFloat("DS_APM_AINOTICE_MAX_BUDGET_USD"),
			envFloat("DS_APM_AINOTICE_OUTPUT_USD_PER_MTOK"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("ds-apm ai generator: %w", err)
	}

	// Build AI config store, cipher, and StoreAware generator for per-org AI
	// configuration (AI Module Settings page).
	aiConfigStore := sqlaiconfigstore.New(sqlstore)
	aiCipher, insecure, err := secretbox.FromEnv()
	if err != nil {
		return nil, fmt.Errorf("ds-apm ai config cipher: %w", err)
	}
	if insecure {
		instrumentation.Logger().WarnContext(ctx, "ds-apm ai config: running with plaintext API-key storage; set DS_APM_AI_CONFIG_ENCRYPTION_KEY for AES-256 at-rest encryption")
	}
	storeAware := aigenerator.NewStoreAware(aiConfigStore, aiCipher, aiGen)

	// Build a RunbookDrafter from the same env-driven config as aiGen.
	// Falls back to mock when no LLM provider is configured.
	aiCfg := aigenerator.Config{
		Provider:          os.Getenv("DS_APM_AI_GENERATOR"),
		LLMProvider:       os.Getenv("DS_APM_LLM_PROVIDER"),
		LLMTransport:      os.Getenv("DS_APM_LLM_TRANSPORT"),
		LLMModel:          os.Getenv("DS_APM_LLM_MODEL"),
		LLMTimeoutSeconds: envInt("DS_APM_LLM_TIMEOUT_SECONDS"),
		LLMAPIKey:         pickAPIKey(os.Getenv("DS_APM_LLM_PROVIDER")),
		LLMBinary:         os.Getenv("DS_APM_LLM_BINARY"),
		LLMEndpoint:       os.Getenv("DS_APM_LLM_ENDPOINT"),
	}
	// Reuse the per-org AI-module credential when present; the env-built drafter
	// is the fallback.
	runbookDrafter := aigenerator.NewStoreAwareRunbookDrafter(
		aiConfigStore, aiCipher, aigenerator.NewRunbookDrafter(aiCfg),
	)

	var aiDispatchHook *dispatchhook.Hook
	if aiGen != nil {
		// Best-effort AI enrichment must not stall alert dispatch, so the hook
		// caps generation time. The default suits a fast/local generator; slow
		// LLM transports (e.g. a CLI agent) need a larger budget, configurable
		// via DS_APM_AI_DISPATCH_TIMEOUT_SECONDS.
		dispatchAITimeout := 5 * time.Second
		if secs := envInt("DS_APM_AI_DISPATCH_TIMEOUT_SECONDS"); secs > 0 {
			dispatchAITimeout = time.Duration(secs) * time.Second
		}
		aiDispatchHook = dispatchhook.New(
			sqlsopstore.NewSOPStore(sqlstore),
			sqlaihistorystore.NewAIStrategyHistoryStore(sqlstore),
			storeAware,
			providerSettings.Logger,
			dispatchAITimeout,
		)
	}

	// Initialize alertmanager from the available alertmanager provider factories
	alertmanager, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Alertmanager,
		NewAlertmanagerProviderFactories(sqlstore, orgGetter, nfManager, aiDispatchHook),
		config.Alertmanager.Provider,
	)
	if err != nil {
		return nil, err
	}

	gatewayFactory := gatewayProviderFactory(licensing)
	gateway, err := gatewayFactory.New(ctx, providerSettings, config.Gateway)
	if err != nil {
		return nil, err
	}

	// Initialize auditor from the variant-specific provider factories
	auditor, err := factory.NewProviderFromNamedMap(ctx, providerSettings, config.Auditor, auditorProviderFactories(licensing), config.Auditor.Provider)
	if err != nil {
		return nil, err
	}

	// ── CF-11 code RCA (coderca) — integration wiring (design §11) ──────────
	// Per-org stores + cost-control run store.
	codercaRepoStore := sqlcodebaseconfigstore.New(sqlstore)
	codercaMapStore := sqlcodebaseservicemapstore.New(sqlstore)
	codercaCfgStore := sqlcodebasercaconfigstore.New(sqlstore)
	codercaRunStore := codercarunstore.New(sqlstore)

	// Build remediation store (used by proposer, handler, and verifier).
	remStore := sqlremediationstore.New(sqlstore)

	// Trigger: injected into the dispatch hook (fail-open, fire-and-forget).
	if aiDispatchHook != nil {
		aiDispatchHook.SetCodeRCATrigger(codercatrigger.New(
			codercaCfgStore, codercaMapStore, codercaRunStore,
			providerSettings.Logger, nil,
		))

		// Dispatch-hook proposer: wraps the Proposer in a MaybePropose adapter.
		// baseURL is the SigNoz external URL (the host operators reach the UI on)
		// so the approval deep link is absolute and renders as a clickable link in
		// Slack/Teams/Email. Configurable via SIGNOZ_ALERTMANAGER_SIGNOZ_EXTERNAL__URL.
		remediationBaseURL := ""
		if config.Alertmanager.Signoz.ExternalURL != nil {
			remediationBaseURL = config.Alertmanager.Signoz.ExternalURL.String()
		}
		proposer := remediation.NewProposer(remStore, remediationBaseURL, time.Now)
		aiDispatchHook.SetRemediationProposer(remediationHookAdapter{
			proposer: proposer,
			store:    remStore,
		})
	}

	// Engine: deployment-level agent/model/auth from env (per-org thresholds
	// live in ds_codebase_config and are enforced at admission).
	codercaAgent := os.Getenv("DS_APM_CODERCA_AGENT")
	if codercaAgent == "" {
		codercaAgent = "claude"
	}
	// The clirunner requires a model (ErrNoModel otherwise). Default the claude
	// agent to its documented model so on-demand RCA works without extra env,
	// matching how agent/budget/auth already default. codex models are
	// account-specific (ChatGPT-subscription codex rejects gpt-5/gpt-5-codex),
	// so we never guess one — codex must set DS_APM_CODERCA_MODEL explicitly.
	codercaModel := os.Getenv("DS_APM_CODERCA_MODEL")
	if codercaModel == "" && codercaAgent == "claude" {
		codercaModel = "claude-sonnet-4-6"
	}
	// MaxTurns is the primary cost bound (deterministic): the agent's read-only
	// explore loop stops after this many turns. MaxBudgetUSD stays as a safety net
	// for the rare runaway. $0.50 alone was too tight — a turn-uncapped explore of
	// a real repo nondeterministically blew it before reaching a conclusion.
	codercaBudget := os.Getenv("DS_APM_CODERCA_MAX_BUDGET_USD")
	if codercaBudget == "" {
		codercaBudget = "2.00"
	}
	codercaMaxTurns := 40
	if v := os.Getenv("DS_APM_CODERCA_MAX_TURNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			codercaMaxTurns = n
		}
	}
	codercaAuth := os.Getenv("DS_APM_CODERCA_AUTH_TOKEN")
	if codercaAuth == "" {
		codercaAuth = pickAPIKey(os.Getenv("DS_APM_LLM_PROVIDER"))
	}
	codercaBaseDir := os.Getenv("DS_APM_CODERCA_DIR")
	if codercaBaseDir == "" {
		codercaBaseDir = filepath.Join(os.TempDir(), "ds-coderca")
	}
	hostname, _ := os.Hostname()

	gitRunner := codercasourcestate.NewShellGitRunner(filepath.Join(codercaBaseDir, "mirrors"))
	codercaEngine := codercaengine.New(
		codercaengine.Config{
			Scope:        "global",
			InstanceID:   hostname,
			Agent:        clirunner.Agent(codercaAgent),
			Model:        codercaModel,
			MaxBudgetUSD: codercaBudget,
			MaxTurns:     codercaMaxTurns,
			AuthToken:    codercaAuth,
		},
		codercaengine.Deps{
			Runs:    codercaRunStore,
			Repos:   reporesolver.New(codercaMapStore, codercaRepoStore, aiCipher.DecryptFunc()),
			Source:  codercasourcestate.NewManager(gitRunner, filepath.Join(codercaBaseDir, "checkouts")),
			CLI:     clirunner.NewRunner(),
			Deliver: codercadelivery.New(codercadelivery.NewAlertmanagerSink(alertmanager)),
			Auditor: codercaauditor.New(codercaauditor.NewDSSink(auditor.Audit), nil),
			// Reuse the per-org AI-module credential (env Config is the fallback).
			Creds: newCodercaCredsResolver(aiConfigStore, aiCipher),
		},
	)
	codercaWorker := codercaworker.New(codercaEngine, codercaRunStore, "global", 0, 0, providerSettings.Logger, nil)

	// Initialize authns
	store := sqlauthnstore.NewStore(sqlstore)
	authNs, err := authNsCallback(ctx, providerSettings, store, licensing)
	if err != nil {
		return nil, err
	}

	// Initialize telemetry metadata store
	// TODO: consolidate other telemetrymetadata.NewTelemetryMetaStore initializations to reuse this instance instead.
	telemetryMetadataStore := telemetrymetadata.NewTelemetryMetaStore(
		providerSettings,
		telemetrystore,
		telemetrytraces.DBName,
		telemetrytraces.TagAttributesV2TableName,
		telemetrytraces.SpanAttributesKeysTblName,
		telemetrytraces.SpanIndexV3TableName,
		telemetrymetrics.DBName,
		telemetrymetrics.AttributesMetadataTableName,
		telemetrymeter.DBName,
		telemetrymeter.SamplesAgg1dTableName,
		telemetrylogs.DBName,
		telemetrylogs.LogsV2TableName,
		telemetrylogs.TagAttributesV2TableName,
		telemetrylogs.LogAttributeKeysTblName,
		telemetrylogs.LogResourceKeysTblName,
		telemetryaudit.DBName,
		telemetryaudit.AuditLogsTableName,
		telemetryaudit.TagAttributesTableName,
		telemetryaudit.LogAttributeKeysTblName,
		telemetryaudit.LogResourceKeysTblName,
		telemetrymetadata.DBName,
		telemetrymetadata.AttributesMetadataLocalTableName,
		telemetrymetadata.ColumnEvolutionMetadataTableName,
	)

	global, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.Global,
		NewGlobalProviderFactories(config.IdentN),
		"signoz",
	)
	if err != nil {
		return nil, err
	}

	serviceAccount := implserviceaccount.NewModule(implserviceaccount.NewStore(sqlstore), authz, cache, analytics, providerSettings, config.ServiceAccount)

	cloudIntegrationModule, err := cloudIntegrationCallback(sqlstore, global, zeus, gateway, licensing, serviceAccount, config.CloudIntegration)
	if err != nil {
		return nil, err
	}

	// Initialize all modules
	modules := NewModules(sqlstore, tokenizer, emailing, providerSettings, orgGetter, alertmanager, analytics, querier, telemetrystore, telemetryMetadataStore, authNs, authz, cache, queryParser, config, dashboard, userGetter, userRoleStore, serviceAccount, cloudIntegrationModule)

	// Initialize ruler from the variant-specific provider factories
	rulerInstance, err := factory.NewProviderFromNamedMap(ctx, providerSettings, config.Ruler, rulerProviderFactories(cache, alertmanager, sqlstore, telemetrystore, telemetryMetadataStore, prometheus, orgGetter, modules.RuleStateHistory, querier, queryParser), "signoz")
	if err != nil {
		return nil, err
	}

	// Initialize identN resolver
	identNFactories := NewIdentNProviderFactories(tokenizer, serviceAccount, orgGetter, userGetter, config.User)
	identNResolver, err := identn.NewIdentNResolver(ctx, providerSettings, config.IdentN, identNFactories)
	if err != nil {
		return nil, err
	}

	userService := impluser.NewService(providerSettings, impluser.NewStore(sqlstore, providerSettings), modules.UserGetter, modules.UserSetter, orgGetter, authz, config.User.Root)

	// Initialize the querier handler via callback (allows EE to decorate with anomaly detection)
	querierHandler := querierHandlerCallback(providerSettings, querier, analytics)

	// Create a list of all stats collectors
	statsCollectors := []statsreporter.StatsCollector{
		alertmanager,
		rulerInstance,
		modules.Dashboard,
		modules.SavedView,
		modules.UserSetter,
		licensing,
		tokenizer,
		config,
		modules.AuthDomain,
		serviceAccount,
		cloudIntegrationModule,
	}

	// Initialize stats reporter from the available stats reporter provider factories
	statsReporter, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.StatsReporter,
		NewStatsReporterProviderFactories(telemetrystore, statsCollectors, orgGetter, userGetter, tokenizer, version.Info, config.Analytics),
		config.StatsReporter.Provider(),
	)
	if err != nil {
		return nil, err
	}

	registry, err := factory.NewRegistry(
		ctx,
		instrumentation.Logger(),
		factory.NewNamedService(factory.MustNewName("instrumentation"), instrumentation),
		factory.NewNamedService(factory.MustNewName("pprof"), pprofService),
		factory.NewNamedService(factory.MustNewName("analytics"), analytics),
		factory.NewNamedService(factory.MustNewName("alertmanager"), alertmanager),
		factory.NewNamedService(factory.MustNewName("licensing"), licensing),
		factory.NewNamedService(factory.MustNewName("statsreporter"), statsReporter),
		factory.NewNamedService(factory.MustNewName("tokenizer"), tokenizer),
		factory.NewNamedService(factory.MustNewName("authz"), authz),
		factory.NewNamedService(factory.MustNewName("user"), userService, factory.MustNewName("authz")),
		factory.NewNamedService(factory.MustNewName("auditor"), auditor),
		factory.NewNamedService(factory.MustNewName("codercaworker"), codercaWorker),
		factory.NewNamedService(factory.MustNewName("ruler"), rulerInstance),
	)
	if err != nil {
		return nil, err
	}

	// Start the remediation verifier as a background goroutine. It polls active
	// orgs every 30 s, expiring stale proposals and promoting succeeded→verified
	// (or unresolved) based on alert state. alertStateLookup uses GetAlerts from
	// the alertmanager service (active-only filter). orgLister uses the shared
	// orgGetter that already drives the alertmanager sync loop.
	verifier := remediation.NewVerifier(remStore, alertStateLookup{am: alertmanager}, time.Now)
	go verifier.Run(ctx, 30*time.Second, orgLister(orgGetter, providerSettings.Logger))

	// Initialize all handlers for the modules
	registryHandler := factory.NewHandler(registry)
	newExec := func(d time.Duration) signozruler.RemediationRunner {
		return remediation.NewExecutor(d)
	}
	handlers := NewHandlers(modules, providerSettings, analytics, querierHandler, licensing, global, flagger, gateway, telemetryMetadataStore, authz, zeus, registryHandler, alertmanager, rulerInstance, sqlstore, storeAware, aiConfigStore, aiCipher, storeAware, runbookDrafter, codercaRepoStore, codercaMapStore, codercaCfgStore, codercaRunStore, insecure, remStore, newExec)

	// Initialize the API server (after registry so it can access service health)
	apiserverInstance, err := factory.NewProviderFromNamedMap(
		ctx,
		providerSettings,
		config.APIServer,
		NewAPIServerProviderFactories(orgGetter, authz, modules, handlers),
		"signoz",
	)
	if err != nil {
		return nil, err
	}

	return &SigNoz{
		Registry:               registry,
		Analytics:              analytics,
		Instrumentation:        instrumentation,
		Cache:                  cache,
		Web:                    web,
		SQLStore:               sqlstore,
		TelemetryStore:         telemetrystore,
		TelemetryMetadataStore: telemetryMetadataStore,
		Prometheus:             prometheus,
		Alertmanager:           alertmanager,
		Querier:                querier,
		APIServer:              apiserverInstance,
		Zeus:                   zeus,
		Licensing:              licensing,
		Emailing:               emailing,
		Sharder:                sharder,
		Tokenizer:              tokenizer,
		IdentNResolver:         identNResolver,
		Authz:                  authz,
		Ruler:                  rulerInstance,
		Modules:                modules,
		Handlers:               handlers,
		QueryParser:            queryParser,
		Flagger:                flagger,
		Gateway:                gateway,
		Auditor:                auditor,
	}, nil
}

func envInt(key string) int {
	s := os.Getenv(key)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// envFloat parses an environment variable as float64; returns 0 when unset or
// unparseable.
func envFloat(key string) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

func pickAPIKey(llmProvider string) string {
	switch llmProvider {
	case "claude":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "codex":
		return os.Getenv("OPENAI_API_KEY")
	default:
		return ""
	}
}
