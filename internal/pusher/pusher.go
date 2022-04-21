package pusher

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/grafana/synthetic-monitoring-agent/internal/pkg/logproto"
	"github.com/grafana/synthetic-monitoring-agent/internal/pkg/loki"
	"github.com/grafana/synthetic-monitoring-agent/internal/pkg/prom"
	"github.com/grafana/synthetic-monitoring-agent/internal/version"
	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog"
)

const defaultBufferCapacity = 10 * 1024

var bufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, defaultBufferCapacity)
		return &buf
	},
}

type remoteTarget struct {
	Events  *prom.Client
	Metrics *prom.Client
}

type Payload interface {
	Tenant() int64
	Metrics() []prompb.TimeSeries
	Streams() []logproto.Stream
}

type Publisher struct {
	tenantManager *TenantManager
	logger        zerolog.Logger
	publishCh     <-chan Payload
	clientsMutex  sync.Mutex
	clients       map[int64]*remoteTarget
	pushCounter   *prometheus.CounterVec
	errorCounter  *prometheus.CounterVec
	failedCounter *prometheus.CounterVec
	bytesOut      *prometheus.CounterVec
}

func NewPublisher(tm *TenantManager, publishCh <-chan Payload, logger zerolog.Logger, promRegisterer prometheus.Registerer) *Publisher {
	pushCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sm_agent",
			Subsystem: "publisher",
			Name:      "push_total",
			Help:      "Total number of push events.",
		},
		[]string{"type", "tenantID"})

	promRegisterer.MustRegister(pushCounter)

	errorCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sm_agent",
			Subsystem: "publisher",
			Name:      "push_errors_total",
			Help:      "Total number of push errors.",
		},
		[]string{"type", "tenantID", "status"})

	promRegisterer.MustRegister(errorCounter)

	failedCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sm_agent",
			Subsystem: "publisher",
			Name:      "push_failed_total",
			Help:      "Total number of push failed.",
		},
		[]string{"type", "tenantID"})

	promRegisterer.MustRegister(failedCounter)

	bytesOut := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sm_agent",
			Subsystem: "publisher",
			Name:      "push_bytes",
			Help:      "Total number of bytes pushed.",
		},
		[]string{"target", "tenantID"})

	promRegisterer.MustRegister(bytesOut)

	return &Publisher{
		tenantManager: tm,
		publishCh:     publishCh,
		clients:       make(map[int64]*remoteTarget),
		logger:        logger,
		pushCounter:   pushCounter,
		errorCounter:  errorCounter,
		failedCounter: failedCounter,
		bytesOut:      bytesOut,
	}
}

func (p *Publisher) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		case payload := <-p.publishCh:
			go p.publish(ctx, payload)
		}
	}
}

func (p *Publisher) publish(ctx context.Context, payload Payload) {
	tenantID := payload.Tenant()
	tenantStr := strconv.FormatInt(tenantID, 10)
	newClient := false

	logger := p.logger.With().Int64("tenant", tenantID).Logger()

	for retry := 2; retry > 0; retry-- {
		client, err := p.getClient(ctx, tenantID, newClient)
		if err != nil {
			logger.Error().Err(err).Msg("get client failed")
			p.failedCounter.WithLabelValues("client", tenantStr).Inc()
			return
		}

		if len(payload.Streams()) > 0 {
			if n, err := p.pushEvents(ctx, client.Events, payload.Streams()); err != nil {
				httpStatusCode, hasStatusCode := prom.GetHttpStatusCode(err)
				logger.Error().Err(err).Msg("publish events")
				p.errorCounter.WithLabelValues("logs", tenantStr, strconv.Itoa(httpStatusCode)).Inc()
				if hasStatusCode && httpStatusCode == http.StatusUnauthorized {
					// Retry to get a new client, credentials might be stale.
					newClient = true
					continue
				}
			} else {
				p.pushCounter.WithLabelValues("logs", tenantStr).Inc()
				p.bytesOut.WithLabelValues("logs", tenantStr).Add(float64(n))
			}
		}

		if len(payload.Metrics()) > 0 {
			if n, err := p.pushMetrics(ctx, client.Metrics, payload.Metrics()); err != nil {
				httpStatusCode, hasStatusCode := prom.GetHttpStatusCode(err)
				logger.Error().Err(err).Msg("publish metrics")
				p.errorCounter.WithLabelValues("metrics", tenantStr, strconv.Itoa(httpStatusCode)).Inc()
				if hasStatusCode && httpStatusCode == http.StatusUnauthorized {
					// Retry to get a new client, credentials might be stale.
					newClient = true
					continue
				}
			} else {
				p.pushCounter.WithLabelValues("metrics", tenantStr).Inc()
				p.bytesOut.WithLabelValues("metrics", tenantStr).Add(float64(n))
			}
		}

		// if we make it here we have sent everything we could send, we are done.
		return
	}

	// if we are here, we retried and failed
	p.failedCounter.WithLabelValues("retry_exhausted", tenantStr).Inc()
	logger.Warn().Msg("failed to push payload")
}

func (p *Publisher) pushEvents(ctx context.Context, client *prom.Client, streams []logproto.Stream) (int, error) {
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := loki.SendStreamsWithBackoff(ctx, client, streams, buf); err != nil {
		return 0, fmt.Errorf("sending events: %w", err)
	}

	return len(*buf), nil
}

func (p *Publisher) pushMetrics(ctx context.Context, client *prom.Client, metrics []prompb.TimeSeries) (int, error) {
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := prom.SendSamplesWithBackoff(ctx, client, metrics, buf); err != nil {
		return 0, fmt.Errorf("sending timeseries: %w", err)
	}

	return len(*buf), nil
}

func (p *Publisher) getClient(ctx context.Context, tenantId int64, newClient bool) (*remoteTarget, error) {
	var (
		client *remoteTarget
		found  bool
	)

	p.clientsMutex.Lock()
	if newClient {
		p.logger.Info().Int64("tenantId", tenantId).Msg("removing tenant from cache")
		delete(p.clients, tenantId)
	} else {
		client, found = p.clients[tenantId]
	}
	p.clientsMutex.Unlock()

	if found {
		return client, nil
	}

	p.logger.Info().Int64("tenantId", tenantId).Msg("fetching tenant credentials")

	req := sm.TenantInfo{
		Id: tenantId,
	}
	tenant, err := p.tenantManager.GetTenant(ctx, &req)
	if err != nil {
		return nil, err
	}

	return p.updateClient(tenant)
}

func (p *Publisher) updateClient(tenant *sm.Tenant) (*remoteTarget, error) {
	mClientCfg, err := clientFromRemoteInfo(tenant.Id, tenant.MetricsRemote)
	if err != nil {
		return nil, fmt.Errorf("creating metrics client configuration: %w", err)
	}

	mClient, err := prom.NewClient(tenant.MetricsRemote.Name, mClientCfg)
	if err != nil {
		return nil, fmt.Errorf("creating metrics client: %w", err)
	}

	eClientCfg, err := clientFromRemoteInfo(tenant.Id, tenant.EventsRemote)
	if err != nil {
		return nil, fmt.Errorf("creating events client configuration: %w", err)
	}

	eClient, err := prom.NewClient(tenant.EventsRemote.Name, eClientCfg)
	if err != nil {
		return nil, fmt.Errorf("creating events client: %w", err)
	}

	clients := &remoteTarget{
		Metrics: mClient,
		Events:  eClient,
	}

	p.clientsMutex.Lock()
	p.clients[tenant.Id] = clients
	p.clientsMutex.Unlock()
	p.logger.Debug().Int64("tenantId", tenant.Id).Int64("stackId", tenant.StackId).Msg("updated client")

	return clients, nil
}

func clientFromRemoteInfo(tenantId int64, remote *sm.RemoteInfo) (*prom.ClientConfig, error) {
	// TODO(mem): this is hacky.
	//
	// it's trying to deal with the fact that the URL shown to users
	// is not the push URL but the base for the API endpoints
	u, err := url.Parse(remote.Url + "/push")
	if err != nil {
		// XXX(mem): do we really return an error here?
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	clientCfg := prom.ClientConfig{
		URL:       u,
		Timeout:   5 * time.Second,
		UserAgent: version.UserAgent(),
	}

	if remote.Username != "" {
		clientCfg.HTTPClientConfig.BasicAuth = &prom.BasicAuth{
			Username: remote.Username,
			Password: remote.Password,
		}
	}

	if clientCfg.Headers == nil {
		clientCfg.Headers = make(map[string]string)
	}

	clientCfg.Headers["X-Prometheus-Remote-Write-Version"] = "0.1.0"
	// TODO: check if grafana cloud looks for this headers? or gets OrgID from BasicAuth
	clientCfg.Headers["X-Scope-OrgID"] = strconv.FormatInt(tenantId, 10)

	return &clientCfg, nil
}
