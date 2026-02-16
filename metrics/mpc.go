package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/metric"
)

type MpcMetrics struct {
	totalRelayersGauge     metric.Int64ObservableGauge
	availableRelayersGauge metric.Int64ObservableGauge
	totalRelayerCount      *int64
	availableRelayerCount  *int64

	sessionTimeHistogram metric.Float64Histogram
	sessionStartTime     map[string]time.Time
	histogramMutex       sync.Mutex
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

	sessionTimeHistogram, err := meter.Float64Histogram("relayer.SessionTime")

	return &MpcMetrics{
		totalRelayersGauge:     totalRelayersGauge,
		availableRelayersGauge: availableRelayersGauge,
		totalRelayerCount:      totalRelayerCount,
		availableRelayerCount:  availableRelayerCount,
		sessionTimeHistogram:   sessionTimeHistogram,
		sessionStartTime:       make(map[string]time.Time),
		histogramMutex:         sync.Mutex{},
	}, nil
}

func (m *MpcMetrics) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	*m.totalRelayerCount = int64(len(all))
	*m.availableRelayerCount = int64(len(all) - len(unavailable))
}

func (m *MpcMetrics) StartProcess(sessionID string) {
	m.histogramMutex.Lock()
	defer m.histogramMutex.Unlock()

	m.sessionStartTime[sessionID] = time.Now()
}

func (m *MpcMetrics) EndProcess(sessionID string) {
	m.histogramMutex.Lock()
	defer m.histogramMutex.Unlock()

	startTime, ok := m.sessionStartTime[sessionID]
	if !ok {
		log.Warn().Msgf("Session start time with ID %s not found", sessionID)
		return
	}

	m.sessionTimeHistogram.Record(context.Background(), time.Since(startTime).Seconds())
}
