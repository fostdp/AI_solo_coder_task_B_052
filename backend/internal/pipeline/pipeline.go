package pipeline

import (
	"context"
	"sync"
	"time"

	"ancient-wood-monitor/internal/models"
)

type MessageType int

const (
	MsgTypeRawLoRa MessageType = iota
	MsgTypeDeduplicated
	MsgTypeProcessedSensor
	MsgTypeTermitePrediction
	MsgTypeTDOAStrength
	MsgTypeFumigantDiffusion
	MsgTypeAlert
	MsgTypeBirdActivity
)

type Metadata struct {
	MessageID   string    `json:"message_id"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	TraceID     string    `json:"trace_id"`
	Retries     int       `json:"retries"`
}

type PipelineMessage struct {
	Type      MessageType
	Metadata  Metadata
	Data      interface{}
	Err       error
}

type PipelineStage interface {
	Start(ctx context.Context, in <-chan PipelineMessage, out chan<- PipelineMessage) error
	Name() string
}

type Pipeline struct {
	stages []PipelineStage
	chans  []chan PipelineMessage
	mu     sync.Mutex
	wg     sync.WaitGroup
}

func NewPipeline(stages ...PipelineStage) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

func (p *Pipeline) Start(ctx context.Context) (chan<- PipelineMessage, <-chan PipelineMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.stages)
	p.chans = make([]chan PipelineMessage, n+1)

	for i := 0; i <= n; i++ {
		p.chans[i] = make(chan PipelineMessage, 4096)
	}

	for i, stage := range p.stages {
		p.wg.Add(1)
		go func(s PipelineStage, in <-chan PipelineMessage, out chan<- PipelineMessage) {
			defer p.wg.Done()
			if err := s.Start(ctx, in, out); err != nil {
				select {
				case out <- PipelineMessage{
					Type: MsgTypeAlert,
					Metadata: Metadata{
						Timestamp: time.Now(),
						Source:    s.Name(),
						TraceID:   generateTraceID(),
					},
					Err: err,
				}:
				case <-ctx.Done():
				}
			}
		}(stage, p.chans[i], p.chans[i+1])
	}

	return p.chans[0], p.chans[n], nil
}

func (p *Pipeline) Wait() {
	p.wg.Wait()
}

func (p *Pipeline) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.chans {
		if ch != nil {
			close(ch)
		}
	}
}

func generateTraceID() string {
	now := time.Now()
	return "trace-" + now.Format("20060102150405") + "-" + randomHex(8)
}

func randomHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[time.Now().UnixNano()%int64(len(hex))]
	}
	return string(b)
}

type LoRaIngestData struct {
	RawPacket  models.LoRaDataPacket
	IsDuplicate bool
	DuplicateReason string
}

type TermiteInput struct {
	SensorID         string
	Building         string
	Location         string
	HistoricalData   []map[string]float64
	CurrentEventRate float64
	CurrentMoisture  float64
}

type TermiteOutput struct {
	SensorID         string
	Building         string
	Location         string
	SmoothedRate     float64
	Trend            string
	Predictions      []models.TermitePredictionResult
	RiskLevel        string
}

type FumigantInput struct {
	Building         string
	ReleaseLocation  [3]float64
	ReleaseRate      float64
	WindSpeed        float64
	WindDirection    float64
	GridResolution   float64
}

type FumigantOutput struct {
	Building         string
	GridSize         [3]int
	GridOrigin       [3]float64
	GridSpacing      float64
	Concentrations   []float64
	MaxConcentration float64
	ExposureTime     time.Duration
}

type AlertOutput struct {
	Alert models.Alert
	Channels []string
}

type TDOAStrengthOutput struct {
	TunnelNetwork    models.TunnelNetwork
	StrengthResults  []models.WoodStrengthAssessment
	SensorID         string
	Building         string
	Location         string
	RiskLevel        string
	ParticleFilter   *models.ParticleFilterOutput
}

type BirdActivityOutput struct {
	BirdData         []models.BirdRadarData
	DeterrentActions []models.DeterrentAction
	Building         string
}
