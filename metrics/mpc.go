package metrics

import (
	"context"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	SESSION_TTL = time.Minute * 10
)

type sessionStart struct {
	at          time.Time
	processType string
}

type MpcMetrics struct {
	totalRelayersGauge     metric.Int64ObservableGauge
	availableRelayersGauge metric.Int64ObservableGauge
	totalRelayerCount      *int64
	availableRelayerCount  *int64

	sessionTimeHistogram metric.Float64Histogram
	sessionStartCache    *ttlcache.Cache[string, sessionStart]
	opts                 metric.MeasurementOption
}

// NewMpcMetrics initializes metrics related to the MPC set
func NewMpcMetrics(ctx context.Context, meter metric.Meter, opts metric.MeasurementOption) (*MpcMetrics, error) {
	totalRelayerCount := new(int64)
	availableRelayerCount := new(int64)
	totalRelayersGauge, err := meter.Int64ObservableGauge(
		"relayer.TotalRelayers",
		metric.WithInt64Callback(func(context context.Context, result metric.Int64Observer) error {
			result.Observe(*totalRelayerCount, opts)
			return nil
		}),
		metric.WithDescription("Total number of relayers currently in the subset"),
	)
	if err != nil {
		return nil, err
	}
	availableRelayersGauge, err := meter.Int64ObservableGauge(
		"relayer.AvailableRelayers",
		metric.WithInt64Callback(func(context context.Context, result metric.Int64Observer) error {
			result.Observe(*availableRelayerCount, opts)
			return nil
		}),
		metric.WithDescription("Available number of relayers currently in the subset"),
	)
	if err != nil {
		return nil, err
	}

	sessionTimeHistogram, err := meter.Float64Histogram(
		"relayer.SessionTime",
		metric.WithDescription("Duration (seconds) of a TSS process, labelled by process type (keygen/signing/resharing)"),
	)
	if err != nil {
		return nil, err
	}

	return &MpcMetrics{
		totalRelayersGauge:     totalRelayersGauge,
		availableRelayersGauge: availableRelayersGauge,
		totalRelayerCount:      totalRelayerCount,
		availableRelayerCount:  availableRelayerCount,
		sessionTimeHistogram:   sessionTimeHistogram,
		sessionStartCache: ttlcache.New(
			ttlcache.WithTTL[string, sessionStart](SESSION_TTL),
		),
		opts: opts,
	}, nil
}

func (m *MpcMetrics) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	*m.totalRelayerCount = int64(len(all))
	*m.availableRelayerCount = int64(len(all) - len(unavailable))
}

func (m *MpcMetrics) StartProcess(sessionID, processType string) {
	m.sessionStartCache.Set(
		sessionID,
		sessionStart{at: time.Now(), processType: processType},
		ttlcache.DefaultTTL,
	)
}

func (m *MpcMetrics) EndProcess(sessionID string) {
	entry := m.sessionStartCache.Get(sessionID)
	if entry == nil {
		log.Warn().Msgf("Session start time with ID %s not found", sessionID)
		return
	}

	start := entry.Value()
	m.sessionTimeHistogram.Record(
		context.Background(),
		time.Since(start.at).Seconds(),
		m.opts,
		metric.WithAttributes(attribute.String("type", start.processType)),
	)
}
