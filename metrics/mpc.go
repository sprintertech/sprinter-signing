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

type MpcMetrics struct {
	totalRelayersGauge     metric.Int64ObservableGauge
	availableRelayersGauge metric.Int64ObservableGauge
	totalRelayerCount      *int64
	availableRelayerCount  *int64

	sessionTimeHistogram        metric.Float64Histogram
	initiateTimeHistogram       metric.Float64Histogram
	commSendTimeHistogram       metric.Float64Histogram
	commDnsResolveTimeHistogram metric.Float64Histogram
	sessionStartTimeCache       *ttlcache.Cache[string, time.Time]
	opts                        metric.MeasurementOption
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
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	initiateTimeHistogram, err := meter.Float64Histogram(
		"relayer.InitiateTime",
		metric.WithDescription("Duration (seconds) of the coordinator initiate handshake: broadcast -> threshold+1 peers ready"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	commSendTimeHistogram, err := meter.Float64Histogram(
		"relayer.CommSendTime",
		metric.WithDescription("Duration (seconds) of a single outbound libp2p message send, labelled by target peer"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	commDnsResolveTimeHistogram, err := meter.Float64Histogram(
		"relayer.CommDnsResolveTime",
		metric.WithDescription("Duration (seconds) of DNS resolution + libp2p Connect per outbound send"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &MpcMetrics{
		totalRelayersGauge:          totalRelayersGauge,
		availableRelayersGauge:      availableRelayersGauge,
		totalRelayerCount:           totalRelayerCount,
		availableRelayerCount:       availableRelayerCount,
		sessionTimeHistogram:        sessionTimeHistogram,
		initiateTimeHistogram:       initiateTimeHistogram,
		commSendTimeHistogram:       commSendTimeHistogram,
		commDnsResolveTimeHistogram: commDnsResolveTimeHistogram,
		sessionStartTimeCache: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](SESSION_TTL),
		),
		opts: opts,
	}, nil
}

func (m *MpcMetrics) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	*m.totalRelayerCount = int64(len(all))
	*m.availableRelayerCount = int64(len(all) - len(unavailable))
}

func (m *MpcMetrics) StartProcess(sessionID string) {
	m.sessionStartTimeCache.Set(sessionID, time.Now(), ttlcache.DefaultTTL)
}

func (m *MpcMetrics) EndProcess(sessionID string) {
	startTime := m.sessionStartTimeCache.Get(sessionID)
	if startTime == nil {
		log.Warn().Msgf("Session start time with ID %s not found", sessionID)
		return
	}

	m.sessionTimeHistogram.Record(context.Background(), time.Since(startTime.Value()).Seconds(), m.opts)
}

func (m *MpcMetrics) RecordInitiateDuration(d time.Duration) {
	m.initiateTimeHistogram.Record(context.Background(), d.Seconds(), m.opts)
}

func (m *MpcMetrics) RecordCommSend(peerID string, d time.Duration) {
	m.commSendTimeHistogram.Record(
		context.Background(),
		d.Seconds(),
		m.opts,
		metric.WithAttributes(attribute.String("peer", peerID)),
	)
}

func (m *MpcMetrics) RecordCommDnsResolve(d time.Duration) {
	m.commDnsResolveTimeHistogram.Record(context.Background(), d.Seconds(), m.opts)
}
