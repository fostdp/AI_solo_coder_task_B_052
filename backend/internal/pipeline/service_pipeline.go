package pipeline

import (
	"context"
	"log"
	"sync"

	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/services/alerter"
	fumigant "ancient-wood-monitor/internal/services/fumigant_diffusion"
	loraingest "ancient-wood-monitor/internal/services/lora_ingest"
	termitelstm "ancient-wood-monitor/internal/services/termite_lstm"
	tdoastrength "ancient-wood-monitor/internal/services/tdoa_strength"
)

type ServicePipeline struct {
	pipeline *Pipeline

	LoRaIngest        *loraingest.LoRaIngestService
	TermiteLSTM       *termitelstm.TermiteLSTMService
	TDOAStrength      *tdoastrength.TDOAStrengthService
	FumigantDiffusion *fumigant.FumigantDiffusionService
	Alerter           *alerter.AlertService

	Input  chan<- PipelineMessage
	Output <-chan PipelineMessage

	wg sync.WaitGroup
}

func NewServicePipeline(cfg *config.Config) (*ServicePipeline, error) {
	sp := &ServicePipeline{}

	sp.LoRaIngest = loraingest.NewService(loraingest.Config{
		ExpectedItems:     cfg.Pipeline.LoRaIngest.ExpectedItems,
		FalsePositiveRate: cfg.Pipeline.LoRaIngest.FalsePositiveRate,
		CacheTTL:          cfg.Pipeline.LoRaIngest.CacheTTL,
		MaxCacheSize:      cfg.Pipeline.LoRaIngest.MaxCacheSize,
	})

	sp.TermiteLSTM = termitelstm.NewService(termitelstm.Config{
		EWMAAcousticAlpha:   cfg.Pipeline.TermiteLSTM.EWMAAcousticAlpha,
		EWMAMoistureAlpha:   cfg.Pipeline.TermiteLSTM.EWMAMoistureAlpha,
		EWMAMaxHistory:      cfg.Pipeline.TermiteLSTM.EWMAMaxHistory,
		SpikeThresholdSigma: cfg.Pipeline.TermiteLSTM.SpikeThresholdSigma,
		ConsecutiveConfirm:  cfg.Pipeline.TermiteLSTM.ConsecutiveConfirm,
		PredictionHours:     cfg.Pipeline.TermiteLSTM.PredictionHours,
		ModelPath:           cfg.Model.LstmPath,
	})

	sp.TDOAStrength = tdoastrength.NewService(tdoastrength.Config{
		SoundSpeedWood:      cfg.TDOA.SoundSpeedWood,
		MinSensors:          cfg.TDOA.MinSensors,
		NodeMergeDistance:    cfg.TDOA.NodeMergeDistance,
		EdgeMaxDistance:      cfg.TDOA.EdgeMaxDistance,
		MaxNodes:            cfg.TDOA.MaxNodesPerBuilding,
		ReferenceDensity:    cfg.Strength.ReferenceDensity,
		CriticalEnergy:      cfg.Strength.CriticalEnergy,
		RequiredSafetyFactor: cfg.Strength.RequiredSafetyFactor,
		DepthRatioDefault:   cfg.Strength.DepthRatioDefault,
		ParticleCount:       cfg.ParticleFilter.ParticleCount,
		ProcessNoise:        cfg.ParticleFilter.ProcessNoise,
		MeasurementNoise:    cfg.ParticleFilter.MeasurementNoise,
		ResampleThreshold:   cfg.ParticleFilter.ResampleThreshold,
		ReleaseLeadTime:     cfg.ParticleFilter.ReleaseLeadTime,
		PredictionHorizon:   cfg.ParticleFilter.PredictionHorizon,
	})

	sp.FumigantDiffusion = fumigant.NewService(fumigant.Config{
		DefaultReleaseRate: cfg.Pipeline.FumigantDiffusion.DefaultReleaseRate,
		DefaultWindSpeed:   cfg.Pipeline.FumigantDiffusion.DefaultWindSpeed,
		DefaultWindDir:     cfg.Pipeline.FumigantDiffusion.DefaultWindDir,
		StabilityClass:     cfg.Pipeline.FumigantDiffusion.StabilityClass,
		GridResolution:     cfg.Pipeline.FumigantDiffusion.GridResolution,
		GridSizeX:          cfg.Pipeline.FumigantDiffusion.GridSizeX,
		GridSizeY:          cfg.Pipeline.FumigantDiffusion.GridSizeY,
		GridSizeZ:          cfg.Pipeline.FumigantDiffusion.GridSizeZ,
		ExposureTimeHours:  cfg.Pipeline.FumigantDiffusion.ExposureTimeHours,
	})

	sp.Alerter = alerter.NewService(alerter.Config{
		AcousticThreshold: cfg.Pipeline.Alerter.AcousticThreshold,
		MoistureThreshold: cfg.Pipeline.Alerter.MoistureThreshold,
		CooldownPeriod:    cfg.Pipeline.Alerter.CooldownPeriod,
		EnableWeChat:      cfg.Pipeline.Alerter.EnableWeChat,
		EnableSMS:         cfg.Pipeline.Alerter.EnableSMS,
		WeChatWebhookURL:  cfg.Alert.WechatWebhookURL,
		SmsAPIURL:         cfg.Alert.SmsAPIURL,
		SmsAPIKey:         cfg.Alert.SmsAPIKey,
	})

	sp.pipeline = NewPipeline(
		sp.LoRaIngest,
		sp.TermiteLSTM,
		sp.TDOAStrength,
		sp.FumigantDiffusion,
		sp.Alerter,
	)

	return sp, nil
}

func (sp *ServicePipeline) Start(ctx context.Context) error {
	input, output, err := sp.pipeline.Start(ctx)
	if err != nil {
		return err
	}
	sp.Input = input
	sp.Output = output

	sp.wg.Add(1)
	go sp.outputDrainer(ctx)

	log.Println("[Pipeline] all 5 stages started: lora_ingest -> termite_lstm -> tdoa_strength -> fumigant_diffusion -> alerter")
	return nil
}

func (sp *ServicePipeline) outputDrainer(ctx context.Context) {
	defer sp.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sp.Output:
			if !ok {
				return
			}
			if msg.Err != nil {
				log.Printf("[Pipeline] error from %s: %v", msg.Metadata.Source, msg.Err)
				continue
			}
			log.Printf("[Pipeline] message: type=%d source=%s", msg.Type, msg.Metadata.Source)
		}
	}
}

func (sp *ServicePipeline) Stop() {
	sp.pipeline.Close()
	sp.wg.Wait()
	log.Println("[Pipeline] stopped")
}

func (sp *ServicePipeline) Stats() map[string]interface{} {
	return map[string]interface{}{
		"lora_ingest":        sp.LoRaIngest.Stats(),
		"termite_lstm":       map[string]interface{}{},
		"tdoa_strength":      map[string]interface{}{},
		"fumigant_diffusion": map[string]interface{}{},
		"alerter":            sp.Alerter.Stats(),
	}
}
