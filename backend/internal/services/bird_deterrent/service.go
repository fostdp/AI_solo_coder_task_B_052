package bird_deterrent

import (
	"context"
	"log"
	"sync"
	"time"

	"ancient-wood-monitor/internal/algorithms"
	"ancient-wood-monitor/internal/models"
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
	simulator   *algorithms.BirdRadarSimulator
	scanHistory map[string][]models.BirdRadarData
	mu          sync.RWMutex
	cancel      context.CancelFunc
	name        string
}

func NewService(cfg Config) *BirdDeterrentService {
	return &BirdDeterrentService{
		cfg: cfg,
		simulator: algorithms.NewBirdRadarSimulator(
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
					scanData := s.simulator.Scan(building)

					s.mu.Lock()
					s.scanHistory[building] = append(s.scanHistory[building], scanData...)
					if len(s.scanHistory[building]) > 100 {
						s.scanHistory[building] = s.scanHistory[building][len(s.scanHistory[building])-100:]
					}
					s.mu.Unlock()

					action := s.simulator.EvaluateDeterrentNeed(scanData, building)
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
	return s.simulator.Scan(building)
}

func (s *BirdDeterrentService) GetDeterrentStatus(building string) map[string]interface{} {
	activeDeterrents := s.simulator.GetActiveDeterrents(building)

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
	now := time.Now()
	action := &models.DeterrentAction{
		ID:        "DETER-" + building + "-" + now.Format("20060102150405"),
		Type:      deterrentType,
		Building:  building,
		StartTime: now,
		Duration:  s.cfg.DeterrentDuration.Seconds(),
		Reason:    "manual trigger",
		Status:    "active",
		BirdCount: 0,
		BirdType:  "",
	}

	s.simulator.ActiveDeterrents[building] = action
	s.simulator.LastDeterrentTime[building] = now

	return action
}
