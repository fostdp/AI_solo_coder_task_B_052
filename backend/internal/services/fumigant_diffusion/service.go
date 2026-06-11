package fumigant_diffusion

import (
	"context"
	"math"
	"sync"
	"time"

	"ancient-wood-monitor/internal/algorithms"
	"ancient-wood-monitor/internal/pipeline"
)

type Config struct {
	DefaultReleaseRate float64 `yaml:"default_release_rate"`
	DefaultWindSpeed   float64 `yaml:"default_wind_speed"`
	DefaultWindDir     float64 `yaml:"default_wind_direction"`
	StabilityClass     string  `yaml:"stability_class"`
	GridResolution     float64 `yaml:"grid_resolution"`
	GridSizeX          int     `yaml:"grid_size_x"`
	GridSizeY          int     `yaml:"grid_size_y"`
	GridSizeZ          int     `yaml:"grid_size_z"`
	ExposureTimeHours  float64 `yaml:"exposure_time_hours"`
}

type FumigantDiffusionService struct {
	cfg     Config
	cache   map[string]pipeline.FumigantOutput
	cacheMu sync.RWMutex
	name    string
}

func NewService(cfg Config) *FumigantDiffusionService {
	return &FumigantDiffusionService{
		cfg:   cfg,
		cache: make(map[string]pipeline.FumigantOutput),
		name:  "fumigant_diffusion",
	}
}

func (s *FumigantDiffusionService) Name() string {
	return s.name
}

func (s *FumigantDiffusionService) Start(ctx context.Context, in <-chan pipeline.PipelineMessage, out chan<- pipeline.PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}

			if msg.Type != pipeline.MsgTypeTermitePrediction {
				out <- msg
				continue
			}

			processed, err := s.process(ctx, &msg)
			if err != nil {
				msg.Err = err
				out <- msg
				continue
			}

			if processed != nil {
				out <- *processed
			}
		}
	}
}

func (s *FumigantDiffusionService) process(ctx context.Context, msg *pipeline.PipelineMessage) (*pipeline.PipelineMessage, error) {
	termiteData, ok := msg.Data.(pipeline.TermiteOutput)
	if !ok {
		return msg, nil
	}

	if termiteData.RiskLevel != "high" && termiteData.RiskLevel != "critical" {
		return nil, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cacheKey := termiteData.Building + "-" + termiteData.SensorID
	s.cacheMu.RLock()
	if cached, exists := s.cache[cacheKey]; exists {
		s.cacheMu.RUnlock()
		return &pipeline.PipelineMessage{
			Type: pipeline.MsgTypeFumigantDiffusion,
			Metadata: pipeline.Metadata{
				MessageID: msg.Metadata.MessageID,
				Timestamp: time.Now(),
				Source:    s.name + "/cache",
				TraceID:   msg.Metadata.TraceID,
				Retries:   msg.Metadata.Retries,
			},
			Data: cached,
		}, nil
	}
	s.cacheMu.RUnlock()

	input := pipeline.FumigantInput{
		Building:        termiteData.Building,
		ReleaseLocation: s.getReleaseLocation(termiteData.Location),
		ReleaseRate:     s.cfg.DefaultReleaseRate,
		WindSpeed:       s.cfg.DefaultWindSpeed,
		WindDirection:   s.cfg.DefaultWindDir,
		GridResolution:  s.cfg.GridResolution,
	}

	output, err := s.simulateDiffusion(ctx, input)
	if err != nil {
		return msg, err
	}

	s.cacheMu.Lock()
	s.cache[cacheKey] = output
	s.cacheMu.Unlock()

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeFumigantDiffusion,
		Metadata: pipeline.Metadata{
			MessageID: msg.Metadata.MessageID,
			Timestamp: time.Now(),
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: output,
	}, nil
}

func (s *FumigantDiffusionService) getReleaseLocation(location string) [3]float64 {
	switch location {
	case "一层明间南檐":
		return [3]float64{0, 0, 0}
	case "二层明间北檐":
		return [3]float64{0, 8, 0}
	case "三层东":
		return [3]float64{4, 16, 0}
	case "三层西":
		return [3]float64{-4, 16, 0}
	default:
		return [3]float64{0, 5, 0}
	}
}

func (s *FumigantDiffusionService) simulateDiffusion(ctx context.Context, input pipeline.FumigantInput) (pipeline.FumigantOutput, error) {
	gridSizeX := s.cfg.GridSizeX
	gridSizeY := s.cfg.GridSizeY
	gridSizeZ := s.cfg.GridSizeZ

	plume := algorithms.NewGaussianPlume(
		input.ReleaseRate,
		input.ReleaseLocation[1],
		input.WindSpeed,
		s.cfg.StabilityClass,
	)

	plume.SetWindDirection(input.WindDirection)

	spacing := s.cfg.GridResolution
	originX := input.ReleaseLocation[0] - float64(gridSizeX)/2*spacing
	originY := input.ReleaseLocation[1] - float64(gridSizeY)/2*spacing
	originZ := input.ReleaseLocation[2] - float64(gridSizeZ)/2*spacing

	concentrations := make([]float64, gridSizeX*gridSizeY*gridSizeZ)
	maxConcentration := 0.0

	for i := 0; i < gridSizeX; i++ {
		select {
		case <-ctx.Done():
			return pipeline.FumigantOutput{}, ctx.Err()
		default:
		}

		for j := 0; j < gridSizeY; j++ {
			for k := 0; k < gridSizeZ; k++ {
				x := originX + float64(i)*spacing
				y := originY + float64(j)*spacing
				z := originZ + float64(k)*spacing

				conc := plume.CalculateConcentration(x, y, z)

				if math.IsNaN(conc) || math.IsInf(conc, 0) {
					conc = 0
				}
				if conc < 0 {
					conc = 0
				}

				idx := i*gridSizeY*gridSizeZ + j*gridSizeZ + k
				concentrations[idx] = conc

				if conc > maxConcentration {
					maxConcentration = conc
				}
			}
		}
	}

	return pipeline.FumigantOutput{
		Building:         input.Building,
		GridSize:         [3]int{gridSizeX, gridSizeY, gridSizeZ},
		GridOrigin:       [3]float64{originX, originY, originZ},
		GridSpacing:      spacing,
		Concentrations:   concentrations,
		MaxConcentration: maxConcentration,
		ExposureTime:     time.Duration(s.cfg.ExposureTimeHours * float64(time.Hour)),
	}, nil
}

func (s *FumigantDiffusionService) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = make(map[string]pipeline.FumigantOutput)
}
