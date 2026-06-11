package bird_deterrent

import (
	"context"
	"log"
	"sync"
	"time"

	"ancient-wood-monitor/internal/models"

	bird "github.com/ancient-wood/bird_drive"
)

type Config struct {
	ScanRadius          float64
	ScanInterval        time.Duration
	WoodpeckerThreshold int
	DeterrentDuration   time.Duration
	CooldownPeriod      time.Duration
	EnableUltrasonic    bool
	EnablePredatorCall  bool
	SimulationSpeed     float64
}

type BirdDeterrentService struct {
	cfg         Config
	simulator   *bird.BirdRadarSimulator
	scanHistory map[string][]models.BirdRadarData
	mu          sync.RWMutex
	cancel      context.CancelFunc
	name        string
}

func NewService(cfg Config) *BirdDeterrentService {
	return &BirdDeterrentService{
		cfg: cfg,
		simulator: bird.NewBirdRadarSimulator(
			cfg.ScanRadius,
			cfg.ScanInterval,
			cfg.WoodpeckerThreshold,
			cfg.DeterrentDuration,
			cfg.CooldownPeriod,
			cfg.EnableUltrasonic,
			cfg.EnablePredatorCall,
			cfg.SimulationSpeed,
		),
		scanHistory: make(map[string][]models.BirdRadarData),
		name:        "bird_deterrent",
	}
}

func toModelBirdRadarData(b []bird.BirdRadarData) []models.BirdRadarData {
	result := make([]models.BirdRadarData, len(b))
	for i, bd := range b {
		result[i] = models.BirdRadarData{
			ID:            bd.ID,
			Timestamp:     bd.Timestamp,
			BirdCount:     bd.BirdCount,
			BirdType:      bd.BirdType,
			Direction:     bd.Direction,
			Distance:      bd.Distance,
			Altitude:      bd.Altitude,
			Speed:         bd.Speed,
			ActivityLevel: bd.ActivityLevel,
		}
	}
	return result
}

func toModelDeterrentAction(d *bird.DeterrentAction) *models.DeterrentAction {
	if d == nil {
		return nil
	}
	return &models.DeterrentAction{
		ID:        d.ID,
		Type:      d.Type,
		Building:  d.Building,
		StartTime: d.StartTime,
		Duration:  d.Duration,
		Reason:    d.Reason,
		Status:    d.Status,
		BirdCount: d.BirdCount,
		BirdType:  d.BirdType,
	}
}

func toModelDeterrentActions(ds []*bird.DeterrentAction) []*models.DeterrentAction {
	result := make([]*models.DeterrentAction, len(ds))
	for i, d := range ds {
		result[i] = toModelDeterrentAction(d)
	}
	return result
}

func (s *BirdDeterrentService) Name() string {
	return s.name
}

func (s *BirdDeterrentService) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(s.cfg.ScanInterval)
		defer ticker.Stop()

		buildings := []string{"应县木塔", "佛光寺"}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, building := range buildings {
					scanDataMod := s.simulator.Scan(building)
					scanData := toModelBirdRadarData(scanDataMod)

					s.mu.Lock()
					s.scanHistory[building] = append(s.scanHistory[building], scanData...)
					if len(s.scanHistory[building]) > 100 {
						s.scanHistory[building] = s.scanHistory[building][len(s.scanHistory[building])-100:]
					}
					s.mu.Unlock()

					action := s.simulator.EvaluateDeterrentNeed(scanDataMod, building)
					if action != nil {
						log.Printf("[bird_deterrent] deterrent triggered for %s: type=%s reason=%s", building, action.Type, action.Reason)
					}
				}
			}
		}
	}()
}

func (s *BirdDeterrentService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *BirdDeterrentService) ScanBuilding(building string) []models.BirdRadarData {
	modData := s.simulator.Scan(building)
	return toModelBirdRadarData(modData)
}

func (s *BirdDeterrentService) GetDeterrentStatus(building string) map[string]interface{} {
	activeDeterrentsMod := s.simulator.GetActiveDeterrents(building)
	activeDeterrents := toModelDeterrentActions(activeDeterrentsMod)

	s.mu.RLock()
	history := s.scanHistory[building]
	s.mu.RUnlock()

	var recentCount int
	var woodpeckerCount int
	var activityLevel string

	if len(history) > 0 {
		lastEntry := history[len(history)-1]
		recentCount = lastEntry.BirdCount
		activityLevel = lastEntry.ActivityLevel

		lastTimestamp := lastEntry.Timestamp
		for i := len(history) - 1; i >= 0; i-- {
			if !history[i].Timestamp.Equal(lastTimestamp) {
				break
			}
			if history[i].BirdType == "woodpecker" {
				woodpeckerCount++
			}
		}
	}

	return map[string]interface{}{
		"active_deterrents":  activeDeterrents,
		"recent_bird_count":  recentCount,
		"woodpecker_count":   woodpeckerCount,
		"activity_level":     activityLevel,
	}
}

func (s *BirdDeterrentService) GetScanHistory(building string, limit int) []models.BirdRadarData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.scanHistory[building]
	if limit <= 0 || limit >= len(history) {
		return history
	}

	return history[len(history)-limit:]
}

func (s *BirdDeterrentService) TriggerDeterrent(building string, deterrentType string) *models.DeterrentAction {
	modAction := s.simulator.TriggerDeterrent(building, deterrentType)
	return toModelDeterrentAction(modAction)
}
