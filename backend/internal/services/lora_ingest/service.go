package lora_ingest

import (
	"context"
	"time"

	"ancient-wood-monitor/internal/models"
	"ancient-wood-monitor/internal/pipeline"
	lorad "ancient-wood-monitor/internal/services/lora"
)

type Config struct {
	ExpectedItems     uint64        `yaml:"expected_items"`
	FalsePositiveRate float64       `yaml:"false_positive_rate"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
	MaxCacheSize      int           `yaml:"max_cache_size"`
	BufferSize        int           `yaml:"buffer_size"`
}

type LoRaIngestService struct {
	cfg    Config
	dedup  *lorad.PacketDeduplicator
	name   string
}

func NewService(cfg Config) *LoRaIngestService {
	return &LoRaIngestService{
		cfg:   cfg,
		dedup: lorad.NewPacketDeduplicator(cfg.ExpectedItems, cfg.CacheTTL, cfg.MaxCacheSize),
		name:  "lora_ingest",
	}
}

func (s *LoRaIngestService) Name() string {
	return s.name
}

func (s *LoRaIngestService) Start(ctx context.Context, in <-chan pipeline.PipelineMessage, out chan<- pipeline.PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}

			if msg.Type != pipeline.MsgTypeRawLoRa {
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

func (s *LoRaIngestService) process(ctx context.Context, msg *pipeline.PipelineMessage) (*pipeline.PipelineMessage, error) {
	packet, ok := msg.Data.(models.LoRaDataPacket)
	if !ok {
		return msg, nil
	}

	select {
	case <-ctx.Done():
		return msg, ctx.Err()
	default:
	}

	dedupResult := s.dedup.CheckPacket(packet.PacketID, packet.DeviceID, packet.Timestamp)

	result := pipeline.LoRaIngestData{
		RawPacket:       packet,
		IsDuplicate:     dedupResult.IsDuplicate,
		DuplicateReason: dedupResult.Reason,
	}

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeDeduplicated,
		Metadata: pipeline.Metadata{
			MessageID: packet.PacketID,
			Timestamp: time.Now(),
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: result,
	}, nil
}

func (s *LoRaIngestService) Stats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size": len(s.dedup.GetAllKeys()),
	}
}
