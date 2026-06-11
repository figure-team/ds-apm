package alertmanagerserver

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/alertmanager/types"
	"golang.org/x/sync/errgroup"

	"github.com/prometheus/alertmanager/dispatch"
	"github.com/prometheus/alertmanager/featurecontrol"
	"github.com/prometheus/alertmanager/inhibit"
	"github.com/prometheus/alertmanager/nflog"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/provider/mem"
	"github.com/prometheus/alertmanager/silence"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/timeinterval"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify"
	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/dispatchhook"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

var (
	// This is not a real file and will never be used. We need this placeholder to ensure maintenance runs on shutdown. See
	// https://github.com/prometheus/server/blob/3ee2cd0f1271e277295c02b6160507b4d193dde2/silence/silence.go#L435-L438
	// and https://github.com/prometheus/server/blob/3b06b97af4d146e141af92885a185891eb79a5b0/nflog/nflog.go#L362.
	snapfnoop string = "snapfnoop"
)

// DLQPathEnvVar is the environment variable that configures the location of
// the alertmanager dead-letter store. When unset, DLQ persistence is
// disabled and the dispatcher keeps its pre-DLQ behavior.
const DLQPathEnvVar = "SIGNOZ_DLQ_PATH"

// DefaultDLQPath is the agreed default location for the dead-letter store.
// It is intentionally NOT applied in-process when SIGNOZ_DLQ_PATH is unset
// (that would make hermetic unit tests write into the repo tree). Instead it
// is the single source of truth the deployment/bootstrap seam (compose,
// helm, cmd/community) should export as SIGNOZ_DLQ_PATH.
const DefaultDLQPath = "var/ds-apm/alert-dlq.jsonl"

// dlqSinkFromEnv resolves the dead-letter sink from SIGNOZ_DLQ_PATH.
//
// When the variable is empty the sink is disabled (nil, nil): the dispatcher
// behaves exactly as it did before DLQ persistence existed, and tests that do
// not opt in never touch the filesystem. When set, a JSONLDeadLetterSink is
// opened at that path — creating the parent directory and the file — using
// the 50 MiB production rotation default.
func dlqSinkFromEnv() (dlq.Sink, error) {
	path := os.Getenv(DLQPathEnvVar)
	if path == "" {
		return nil, nil
	}
	return dlq.NewJSONLDeadLetterSink(path, dlq.DefaultJSONLDeadLetterMaxSizeBytes)
}

type Server struct {
	// logger is the logger for the alertmanager
	logger *slog.Logger

	// registry is the prometheus registry for the alertmanager
	registry prometheus.Registerer

	// srvConfig is the server config for the alertmanager
	srvConfig Config

	// alertmanagerConfig is the config of the alertmanager
	alertmanagerConfig *alertmanagertypes.Config

	// orgID is the orgID for the alertmanager
	orgID string

	// store is the backing store for the alertmanager
	stateStore alertmanagertypes.StateStore

	// alertmanager primitives from upstream alertmanager
	alerts              *mem.Alerts
	nflog               *nflog.Log
	dispatcher          *Dispatcher
	dispatcherMetrics   *DispatcherMetrics
	inhibitor           *inhibit.Inhibitor
	silencer            *silence.Silencer
	silences            *silence.Silences
	timeIntervals       map[string][]timeinterval.TimeInterval
	pipelineBuilder     *notify.PipelineBuilder
	marker              *alertmanagertypes.MemMarker
	tmpl                *template.Template
	wg                  sync.WaitGroup
	stopc               chan struct{}
	notificationManager nfmanager.NotificationManager

	// aiHook, when non-nil, is forwarded to every Dispatcher instance
	// created by SetConfig. It runs the DS-APM AI-strategy dispatch hook
	// before delivery. Nil keeps pre-DS-APM behavior intact.
	aiHook *dispatchhook.Hook

	// dlqSink, when non-nil, is forwarded to every Dispatcher instance so
	// terminal notify-stage failures are persisted to disk and can be
	// replayed. It is resolved once at construction from SIGNOZ_DLQ_PATH
	// (see dlqSinkFromEnv) and reused across SetConfig reloads to avoid
	// leaking file handles. Nil disables DLQ persistence.
	dlqSink dlq.Sink
}

// New creates a new alertmanager Server.
//
// aiHook is optional: when non-nil, the DS-APM AI-strategy dispatch hook is
// forwarded to each Dispatcher so alerts get their SOP and AI strategy
// annotations populated before delivery. Pass nil for non-DS-APM deployments.
func New(ctx context.Context, logger *slog.Logger, registry prometheus.Registerer, srvConfig Config, orgID string, stateStore alertmanagertypes.StateStore, nfManager nfmanager.NotificationManager, aiHook *dispatchhook.Hook) (*Server, error) {
	server := &Server{
		logger:              logger.With(slog.String("pkg", "go.signoz.io/pkg/alertmanager/alertmanagerserver")),
		registry:            registry,
		srvConfig:           srvConfig,
		orgID:               orgID,
		stateStore:          stateStore,
		stopc:               make(chan struct{}),
		notificationManager: nfManager,
		aiHook:              aiHook,
	}

	// Resolve the dead-letter sink once at construction and reuse it across
	// SetConfig reloads (opening a fresh sink per reload would leak file
	// handles). A DLQ misconfiguration must not prevent the alertmanager from
	// booting — alert delivery takes precedence — so a resolution error is
	// logged and DLQ persistence is left disabled rather than fatal.
	if sink, err := dlqSinkFromEnv(); err != nil {
		server.logger.ErrorContext(ctx, "failed to open dead-letter sink; DLQ persistence disabled", slog.String("env", DLQPathEnvVar), errors.Attr(err))
	} else {
		server.dlqSink = sink
	}

	signozRegisterer := prometheus.WrapRegistererWithPrefix("signoz_", registry)
	signozRegisterer = prometheus.WrapRegistererWith(prometheus.Labels{"org_id": server.orgID}, signozRegisterer)
	// initialize marker
	server.marker = alertmanagertypes.NewMarker(signozRegisterer)

	// get silences for initial state
	state, err := server.stateStore.Get(ctx, server.orgID)
	if err != nil && !errors.Ast(err, errors.TypeNotFound) {
		return nil, err
	}

	silencesSnapshot := ""
	if state != nil {
		silencesSnapshot, err = state.Get(alertmanagertypes.SilenceStateName)
		if err != nil && !errors.Ast(err, errors.TypeNotFound) {
			return nil, err
		}
	}
	// Initialize silences
	server.silences, err = silence.New(silence.Options{
		SnapshotReader: strings.NewReader(silencesSnapshot),
		Retention:      srvConfig.Silences.Retention,
		Limits: silence.Limits{
			MaxSilences:         func() int { return srvConfig.Silences.Max },
			MaxSilenceSizeBytes: func() int { return srvConfig.Silences.MaxSizeBytes },
		},
		Metrics: signozRegisterer,
		Logger:  server.logger,
	})
	if err != nil {
		return nil, err
	}

	nflogSnapshot := ""
	if state != nil {
		nflogSnapshot, err = state.Get(alertmanagertypes.NFLogStateName)
		if err != nil && !errors.Ast(err, errors.TypeNotFound) {
			return nil, err
		}
	}

	// Initialize notification log
	server.nflog, err = nflog.New(nflog.Options{
		SnapshotReader: strings.NewReader(nflogSnapshot),
		Retention:      server.srvConfig.NFLog.Retention,
		Metrics:        signozRegisterer,
		Logger:         server.logger,
	})
	if err != nil {
		return nil, err
	}

	// Start maintenance for silences
	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		server.silences.Maintenance(server.srvConfig.Silences.MaintenanceInterval, snapfnoop, server.stopc, func() (int64, error) {
			// Delete silences older than the retention period.
			if _, err := server.silences.GC(); err != nil {
				server.logger.ErrorContext(ctx, "silence garbage collection", errors.Attr(err))
				// Don't return here - we need to snapshot our state first.
			}

			storableSilences, err := server.stateStore.Get(ctx, server.orgID)
			if err != nil && !errors.Ast(err, errors.TypeNotFound) {
				return 0, err
			}

			if storableSilences == nil {
				storableSilences = alertmanagertypes.NewStoreableState(server.orgID)
			}

			c, err := storableSilences.Set(alertmanagertypes.SilenceStateName, server.silences)
			if err != nil {
				return 0, err
			}

			return c, server.stateStore.Set(ctx, storableSilences)
		})

	}()

	// Start maintenance for notification logs
	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		server.nflog.Maintenance(server.srvConfig.NFLog.MaintenanceInterval, snapfnoop, server.stopc, func() (int64, error) {
			if _, err := server.nflog.GC(); err != nil {
				server.logger.ErrorContext(ctx, "notification log garbage collection", errors.Attr(err))
				// Don't return without saving the current state.
			}

			storableNFLog, err := server.stateStore.Get(ctx, server.orgID)
			if err != nil && !errors.Ast(err, errors.TypeNotFound) {
				return 0, err
			}

			if storableNFLog == nil {
				storableNFLog = alertmanagertypes.NewStoreableState(server.orgID)
			}

			c, err := storableNFLog.Set(alertmanagertypes.NFLogStateName, server.nflog)
			if err != nil {
				return 0, err
			}

			return c, server.stateStore.Set(ctx, storableNFLog)
		})
	}()

	server.alerts, err = mem.NewAlerts(ctx, server.marker, server.srvConfig.Alerts.GCInterval, 0, nil, server.logger, signozRegisterer, nil)
	if err != nil {
		return nil, err
	}

	server.pipelineBuilder = notify.NewPipelineBuilder(signozRegisterer, featurecontrol.NoopFlags{})
	server.dispatcherMetrics = NewDispatcherMetrics(false, signozRegisterer)

	return server, nil
}

func (server *Server) GetAlerts(ctx context.Context, params alertmanagertypes.GettableAlertsParams) (alertmanagertypes.GettableAlerts, error) {
	return alertmanagertypes.NewGettableAlertsFromAlertProvider(server.alerts, server.alertmanagerConfig, server.marker.Status, func(labels model.LabelSet) {
		server.inhibitor.Mutes(ctx, labels)
		server.silencer.Mutes(ctx, labels)
	}, params)
}

func (server *Server) PutAlerts(ctx context.Context, postableAlerts alertmanagertypes.PostableAlerts) error {
	alerts, err := alertmanagertypes.NewAlertsFromPostableAlerts(ctx, postableAlerts, time.Duration(server.srvConfig.Global.ResolveTimeout), time.Now())
	// Notification sending alert takes precedence over validation errors.
	if err := server.alerts.Put(ctx, alerts...); err != nil {
		return err
	}

	if err != nil {
		return errors.Join(err...)
	}

	return nil
}

func (server *Server) SetConfig(ctx context.Context, alertmanagerConfig *alertmanagertypes.Config) error {
	config := alertmanagerConfig.AlertmanagerConfig()

	var err error
	server.tmpl, err = alertmanagertypes.FromGlobs(config.Templates)
	if err != nil {
		return err
	}

	server.tmpl.ExternalURL = server.srvConfig.ExternalURL

	// Build the routing tree and record which receivers are used.
	routes := dispatch.NewRoute(config.Route, nil)
	activeReceivers := make(map[string]struct{})
	routes.Walk(func(r *dispatch.Route) {
		activeReceivers[r.RouteOpts.Receiver] = struct{}{}
	})

	// Build the map of receiver to integrations.
	receivers := make(map[string][]notify.Integration, len(activeReceivers))
	var integrationsNum int
	for _, rcv := range config.Receivers {
		if _, found := activeReceivers[rcv.Name]; !found {
			// No need to build a receiver if no route is using it.
			server.logger.InfoContext(ctx, "skipping creation of receiver not referenced by any route", slog.String("receiver", rcv.Name))
			continue
		}
		integrations, err := alertmanagernotify.NewReceiverIntegrations(rcv, server.tmpl, server.logger)
		if err != nil {
			return err
		}
		// rcv.Name is guaranteed to be unique across all receivers.
		receivers[rcv.Name] = integrations
		integrationsNum += len(integrations)
	}

	// Build the map of time interval names to time interval definitions.
	timeIntervals := make(map[string][]timeinterval.TimeInterval, len(config.MuteTimeIntervals)+len(config.TimeIntervals))
	for _, ti := range config.MuteTimeIntervals {
		timeIntervals[ti.Name] = ti.TimeIntervals
	}

	for _, ti := range config.TimeIntervals {
		timeIntervals[ti.Name] = ti.TimeIntervals
	}

	intervener := timeinterval.NewIntervener(timeIntervals)

	if server.inhibitor != nil {
		server.inhibitor.Stop()
	}
	if server.dispatcher != nil {
		server.dispatcher.Stop()
	}

	server.inhibitor = inhibit.NewInhibitor(server.alerts, config.InhibitRules, server.marker, server.logger)
	server.timeIntervals = timeIntervals
	server.silencer = silence.NewSilencer(server.silences, server.marker, server.logger)

	var pipelinePeer notify.Peer
	pipeline := server.pipelineBuilder.New(
		receivers,
		func() time.Duration { return 0 },
		server.inhibitor,
		server.silencer,
		intervener,
		server.marker,
		server.nflog,
		pipelinePeer,
	)

	timeoutFunc := func(d time.Duration) time.Duration {
		if d < notify.MinTimeout {
			d = notify.MinTimeout
		}
		return d
	}

	server.dispatcher = NewDispatcher(
		server.alerts,
		routes,
		pipeline,
		server.marker,
		timeoutFunc,
		nil,
		server.logger,
		server.dispatcherMetrics,
		server.notificationManager,
		server.orgID,
		// DLQ sink resolved once from SIGNOZ_DLQ_PATH at construction (see
		// dlqSinkFromEnv) and reused across reloads. Nil when unset, which
		// preserves the pre-DLQ behavior — terminal failures are still
		// logged, just not persisted.
		server.dlqSink,
		server.aiHook,
	)

	// Do not try to add these to server.wg as there seems to be a race condition if
	// we call Start() and Stop() in quick succession.
	// Both these goroutines will run indefinitely.
	go server.dispatcher.Run()
	go server.inhibitor.Run()

	server.alertmanagerConfig = alertmanagerConfig
	return nil
}

func (server *Server) TestReceiver(ctx context.Context, receiver alertmanagertypes.Receiver) error {
	testAlert := alertmanagertypes.NewTestAlert(receiver, time.Now(), time.Now())
	return alertmanagertypes.TestReceiver(ctx, receiver, alertmanagernotify.NewReceiverIntegrations, server.alertmanagerConfig, server.tmpl, server.logger, testAlert.Labels, testAlert)
}

func (server *Server) TestAlert(ctx context.Context, receiversMap map[*alertmanagertypes.PostableAlert][]string, config *alertmanagertypes.NotificationConfig) error {
	if len(receiversMap) == 0 {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"expected at least 1 alert, got 0")
	}

	postableAlerts := make(alertmanagertypes.PostableAlerts, 0, len(receiversMap))
	for alert := range receiversMap {
		postableAlerts = append(postableAlerts, alert)
	}

	alerts, err := alertmanagertypes.NewAlertsFromPostableAlerts(
		ctx,
		postableAlerts,
		time.Duration(server.srvConfig.Global.ResolveTimeout),
		time.Now(),
	)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"failed to construct alerts from postable alerts: %v", err)
	}

	type alertGroup struct {
		groupLabels model.LabelSet
		alerts      []*types.Alert
		receivers   map[string]struct{}
	}

	groupMap := make(map[model.Fingerprint]*alertGroup)

	for i, alert := range alerts {
		labels := getGroupLabels(alert, config.NotificationGroup, config.GroupByAll)
		fp := labels.Fingerprint()

		postableAlert := postableAlerts[i]
		alertReceivers := receiversMap[postableAlert]

		if group, exists := groupMap[fp]; exists {
			group.alerts = append(group.alerts, alert)
			for _, r := range alertReceivers {
				group.receivers[r] = struct{}{}
			}
		} else {
			receiverSet := make(map[string]struct{})
			for _, r := range alertReceivers {
				receiverSet[r] = struct{}{}
			}
			groupMap[fp] = &alertGroup{
				groupLabels: labels,
				alerts:      []*types.Alert{alert},
				receivers:   receiverSet,
			}
		}
	}

	var mu sync.Mutex
	var errs []error

	g, gCtx := errgroup.WithContext(ctx)
	for _, group := range groupMap {
		for receiverName := range group.receivers {
			group := group
			receiverName := receiverName

			g.Go(func() error {
				receiver, err := server.alertmanagerConfig.GetReceiver(receiverName)
				if err != nil {
					mu.Lock()
					errs = append(errs, errors.WrapInternalf(err, errors.CodeInternal, "failed to get receiver %q", receiverName))
					mu.Unlock()
					return nil // Return nil to continue processing other goroutines
				}

				err = alertmanagertypes.TestReceiver(
					gCtx,
					receiver,
					alertmanagernotify.NewReceiverIntegrations,
					server.alertmanagerConfig,
					server.tmpl,
					server.logger,
					group.groupLabels,
					group.alerts...,
				)
				if err != nil {
					mu.Lock()
					errs = append(errs, errors.WrapInternalf(err, errors.CodeInternal, "receiver %q test failed", receiverName))
					mu.Unlock()
				}
				return nil // Return nil to continue processing other goroutines
			})
		}
	}
	_ = g.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (server *Server) Hash() string {
	if server.alertmanagerConfig == nil {
		return ""
	}

	return server.alertmanagerConfig.StoreableConfig().Hash
}

func (server *Server) Stop(ctx context.Context) error {
	if server.dispatcher != nil {
		server.dispatcher.Stop()
	}

	if server.inhibitor != nil {
		server.inhibitor.Stop()
	}

	// Close the alert provider.
	server.alerts.Close()

	// Signals maintenance goroutines of server states to stop.
	close(server.stopc)

	// Wait for all goroutines to finish.
	server.wg.Wait()

	// Close the dead-letter sink last: the dispatcher (the only writer) has
	// already been stopped above, so no further Write can race with Close.
	if closer, ok := server.dlqSink.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			server.logger.ErrorContext(ctx, "failed to close dead-letter sink", errors.Attr(err))
		}
	}

	return nil
}
