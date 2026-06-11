package termite_lstm

import (
	"context"
	"log"
	"time"

	"ancient-wood-monitor/internal/models"
	"ancient-wood-monitor/internal/pipeline"
	lstmalg "ancient-wood-monitor/internal/algorithms/lstm"
)

type Config struct {
	EWMAAcousticAlpha  float64 `yaml:"ewma_acoustic_alpha"`
	EWMAMoistureAlpha  float64 `yaml:"ewma_moisture_alpha"`
	EWMAMaxHistory     int     `yaml:"ewma_max_history"`
	SpikeThresholdSigma float64 `yaml:"spike_threshold_sigma"`
	ConsecutiveConfirm int     `yaml:"consecutive_confirm"`
	PredictionHours    int     `yaml:"prediction_hours"`
	ModelPath          string  `yaml:"model_path"`
}

type TermiteLSTMService struct {
	cfg             Config
	acousticSmoother *lstmalg.EWMASmoother
	moistureSmoother *lstmalg.EWMASmoother
	modelService    *lstmalg.ModelService
	name            string
}

func NewService(cfg Config) *TermiteLSTMService {
	modelSvc := lstmalg.GetModelService(cfg.ModelPath)
	return &TermiteLSTMService{
		cfg:              cfg,
		acousticSmoother: lstmalg.NewEWMASmoother(cfg.EWMAAcousticAlpha, cfg.EWMAMaxHistory),
		moistureSmoother: lstmalg.NewEWMASmoother(cfg.EWMAMoistureAlpha, cfg.EWMAMaxHistory),
		modelService:     modelSvc,
		name:             "termite_lstm",
	}
}

func (s *TermiteLSTMService) Name() string {
	return s.name
}

func (s *TermiteLSTMService) Start(ctx context.Context, in <-chan pipeline.PipelineMessage, out chan<- pipeline.PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}

			if msg.Type != pipeline.MsgTypeDeduplicated && msg.Type != pipeline.MsgTypeProcessedSensor {
				out <- msg
				continue
			}

			processed, err := s.process(ctx, &msg)
			if err != nil {
				msg.Err = err
				out <- msg
				continue
			}

			out <- *processed
		}
	}
}

func (s *TermiteLSTMService) process(ctx context.Context, msg *pipeline.PipelineMessage) (*pipeline.PipelineMessage, error) {
	select {
	case <-ctx.Done():
		return msg, ctx.Err()
	default:
	}

	var ingestData pipeline.LoRaIngestData
	var ok bool

	if msg.Type == pipeline.MsgTypeDeduplicated {
		ingestData, ok = msg.Data.(pipeline.LoRaIngestData)
		if !ok || ingestData.IsDuplicate {
			return msg, nil
		}
	} else {
		return msg, nil
	}

	packet := ingestData.RawPacket

	if packet.DeviceType == "acoustic_emission" {
		return s.processAcoustic(ctx, msg, &packet)
	} else if packet.DeviceType == "wood_moisture" {
		return s.processMoisture(ctx, msg, &packet)
	}

	return msg, nil
}

func (s *TermiteLSTMService) processAcoustic(ctx context.Context, msg *pipeline.PipelineMessage, packet *models.LoRaDataPacket) (*pipeline.PipelineMessage, error) {
	sensorID := packet.DeviceID
	building := packet.Data["building"].(string)
	location := packet.Data["location"].(string)
	rawEventCount := packet.Data["event_count"].(float64)

	if s.acousticSmoother.IsSpike(sensorID, rawEventCount, s.cfg.SpikeThresholdSigma) {
		log.Printf("[%s] spike detected for %s: %.1f, skipping", s.name, sensorID, rawEventCount)
		return msg, nil
	}

	smoothedRate := s.acousticSmoother.Smooth(sensorID, rawEventCount)

	history := s.acousticSmoother.GetHistory(sensorID)
	if len(history) < s.cfg.ConsecutiveConfirm {
		return msg, nil
	}

	consecutiveOver := 0
	for i := len(history) - 1; i >= len(history)-s.cfg.ConsecutiveConfirm; i-- {
		if history[i] > 100 {
			consecutiveOver++
		}
	}

	trend := "stable"
	if t, hasTrend := s.acousticSmoother.GetTrend(sensorID); hasTrend {
		if t > 0.1 {
			trend = "rising"
		} else if t < -0.1 {
			trend = "falling"
		}
	}

	historicalData := s.buildHistoricalData(sensorID, history, packet)

	predictions, err := s.predictActivity(ctx, historicalData)
	if err != nil {
		return msg, err
	}

	riskLevel := s.calculateRiskLevel(smoothedRate, trend, predictions)

	output := pipeline.TermiteOutput{
		SensorID:         sensorID,
		Building:         building,
		Location:         location,
		SmoothedRate:     smoothedRate,
		Trend:            trend,
		Predictions:      predictions,
		RiskLevel:        riskLevel,
	}

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeTermitePrediction,
		Metadata: pipeline.Metadata{
			MessageID: packet.PacketID,
			Timestamp: time.Now(),
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: output,
	}, nil
}

func (s *TermiteLSTMService) processMoisture(ctx context.Context, msg *pipeline.PipelineMessage, packet *models.LoRaDataPacket) (*pipeline.PipelineMessage, error) {
	sensorID := packet.DeviceID
	building := packet.Data["building"].(string)
	location := packet.Data["location"].(string)
	rawMoisture := packet.Data["moisture"].(float64)

	if s.moistureSmoother.IsSpike(sensorID, rawMoisture, 2.0) {
		log.Printf("[%s] moisture spike detected for %s: %.1f, skipping", s.name, sensorID, rawMoisture)
		return msg, nil
	}

	smoothedMoisture := s.moistureSmoother.Smooth(sensorID, rawMoisture)

	history := s.moistureSmoother.GetHistory(sensorID)
	if len(history) < s.cfg.ConsecutiveConfirm {
		return msg, nil
	}

	consecutiveOver := 0
	for i := len(history) - 1; i >= len(history)-s.cfg.ConsecutiveConfirm; i-- {
		if history[i] > 25.0 {
			consecutiveOver++
		}
	}

	trend := "stable"
	if t, hasTrend := s.moistureSmoother.GetTrend(sensorID); hasTrend {
		if t > 0.05 {
			trend = "rising"
		} else if t < -0.05 {
			trend = "falling"
		}
	}

	output := pipeline.TermiteOutput{
		SensorID:     sensorID,
		Building:     building,
		Location:     location,
		SmoothedRate: smoothedMoisture,
		Trend:        trend,
		RiskLevel:    s.getMoistureRisk(smoothedMoisture, consecutiveOver),
	}

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeTermitePrediction,
		Metadata: pipeline.Metadata{
			MessageID: packet.PacketID,
			Timestamp: time.Now(),
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: output,
	}, nil
}

func (s *TermiteLSTMService) buildHistoricalData(sensorID string, history []float64, packet *models.LoRaDataPacket) []map[string]float64 {
	result := make([]map[string]float64, len(history))
	for i, rate := range history {
		result[i] = map[string]float64{
			"event_count": rate,
			"energy":      packet.Data["energy"].(float64),
			"amplitude":   packet.Data["amplitude"].(float64),
			"duration":    packet.Data["duration"].(float64),
			"peak_freq":   packet.Data["frequency_peak"].(float64),
			"hour":        float64(packet.Timestamp.Hour()),
		}
	}
	return result
}

func (s *TermiteLSTMService) predictActivity(ctx context.Context, data []map[string]float64) ([]models.TermitePredictionResult, error) {
	s.modelService.ResetState()

	predictions := make([]models.TermitePredictionResult, s.cfg.PredictionHours)
	now := time.Now()

	for i := 0; i < s.cfg.PredictionHours; i++ {
		var lastValue float64
		if len(data) > 0 {
			lastValue = data[len(data)-1]["event_count"] / 100.0
		}

		input := []float64{
			lastValue,
			0.8,
			0.5,
			0.0,
			0.5 + 0.3*float64(i)/float64(s.cfg.PredictionHours),
			0.5,
			float64(i) / float64(s.cfg.PredictionHours),
			0.3,
		}

		output := s.modelService.Predict(input)
		activityLevel := output[0] * 150.0
		activityLevel = s.clamp(activityLevel, 0, 200)

		riskLevel := s.getRiskLevel(activityLevel)
		confidence := 0.6 + 0.3*(1-float64(i)/float64(s.cfg.PredictionHours))

		predictions[i] = models.TermitePredictionResult{
			Timestamp:     now.Add(time.Duration(i+1) * time.Hour),
			ActivityLevel: activityLevel,
			RiskLevel:     riskLevel,
			Confidence:    confidence,
			Trend:         "stable",
		}
	}

	return predictions, nil
}

func (s *TermiteLSTMService) calculateRiskLevel(smoothedRate float64, trend string, predictions []models.TermitePredictionResult) string {
	maxPredicted := 0.0
	for _, p := range predictions {
		if p.ActivityLevel > maxPredicted {
			maxPredicted = p.ActivityLevel
		}
	}

	maxValue := smoothedRate
	if maxPredicted > maxValue {
		maxValue = maxPredicted
	}

	switch {
	case maxValue > 150:
		return "critical"
	case maxValue > 120:
		return "high"
	case maxValue > 100:
		return "medium"
	case maxValue > 50:
		return "low"
	default:
		return "normal"
	}
}

func (s *TermiteLSTMService) getRiskLevel(activity float64) string {
	switch {
	case activity > 150:
		return "critical"
	case activity > 120:
		return "high"
	case activity > 100:
		return "medium"
	case activity > 50:
		return "low"
	default:
		return "normal"
	}
}

func (s *TermiteLSTMService) getMoistureRisk(moisture float64, consecutiveOver int) string {
	if consecutiveOver >= s.cfg.ConsecutiveConfirm {
		switch {
		case moisture > 30:
			return "critical"
		case moisture > 27:
			return "high"
		case moisture > 25:
			return "medium"
		default:
			return "low"
		}
	}
	return "normal"
}

func (s *TermiteLSTMService) clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
