package algorithms

import (
	"fmt"
	"math/rand"
	"time"

	"ancient-wood-monitor/internal/models"
)

type BirdRadarSimulator struct {
	ScanRadius          float64
	ScanInterval        time.Duration
	WoodpeckerThreshold int
	DeterrentDuration   time.Duration
	CooldownPeriod      time.Duration
	EnableUltrasonic    bool
	EnablePredatorCall  bool
	LastDeterrentTime   map[string]time.Time
	ActiveDeterrents    map[string]*models.DeterrentAction
	SimulationSpeed     float64
	randSource          *rand.Rand
}

func NewBirdRadarSimulator(scanRadius float64, scanInterval time.Duration, woodpeckerThreshold int, deterrentDuration, cooldownPeriod time.Duration, enableUltrasonic, enablePredatorCall bool, simSpeed float64) *BirdRadarSimulator {
	return &BirdRadarSimulator{
		ScanRadius:          scanRadius,
		ScanInterval:        scanInterval,
		WoodpeckerThreshold: woodpeckerThreshold,
		DeterrentDuration:   deterrentDuration,
		CooldownPeriod:      cooldownPeriod,
		EnableUltrasonic:    enableUltrasonic,
		EnablePredatorCall:  enablePredatorCall,
		LastDeterrentTime:   make(map[string]time.Time),
		ActiveDeterrents:    make(map[string]*models.DeterrentAction),
		SimulationSpeed:     simSpeed,
		randSource:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *BirdRadarSimulator) Scan(building string) []models.BirdRadarData {
	count := s.randSource.Intn(9)
	birdTypes := []string{"sparrow", "swallow", "crow"}
	results := make([]models.BirdRadarData, 0, count)
	now := time.Now()

	var activityLevel string
	switch {
	case count < 2:
		activityLevel = "low"
	case count < 4:
		activityLevel = "moderate"
	case count < 6:
		activityLevel = "high"
	default:
		activityLevel = "intense"
	}

	for i := 0; i < count; i++ {
		var birdType string
		if s.randSource.Float64() < 0.3 {
			birdType = "woodpecker"
		} else {
			birdType = birdTypes[s.randSource.Intn(len(birdTypes))]
		}

		id := fmt.Sprintf("BIRD-%s-%d-%d", building, now.Unix(), i)

		results = append(results, models.BirdRadarData{
			ID:            id,
			Timestamp:     now,
			BirdCount:     count,
			BirdType:      birdType,
			Direction:     s.randSource.Float64() * 360,
			Distance:      10 + s.randSource.Float64()*(s.ScanRadius-10),
			Altitude:      2 + s.randSource.Float64()*28,
			Speed:         5 + s.randSource.Float64()*20,
			ActivityLevel: activityLevel,
		})
	}

	return results
}

func (s *BirdRadarSimulator) EvaluateDeterrentNeed(scanData []models.BirdRadarData, building string) *models.DeterrentAction {
	woodpeckerCount := 0
	for _, bird := range scanData {
		if bird.BirdType == "woodpecker" {
			woodpeckerCount++
		}
	}

	if woodpeckerCount < s.WoodpeckerThreshold {
		return nil
	}

	if lastTime, ok := s.LastDeterrentTime[building]; ok {
		if time.Since(lastTime) < s.CooldownPeriod {
			return nil
		}
	}

	var deterrentType string
	if s.EnableUltrasonic && s.EnablePredatorCall {
		if action, exists := s.ActiveDeterrents[building]; exists && action.Type == "ultrasonic" {
			deterrentType = "predator_call"
		} else {
			deterrentType = "ultrasonic"
		}
	} else if s.EnableUltrasonic {
		deterrentType = "ultrasonic"
	} else if s.EnablePredatorCall {
		deterrentType = "predator_call"
	} else {
		return nil
	}

	now := time.Now()
	action := &models.DeterrentAction{
		ID:        fmt.Sprintf("DETER-%s-%d", building, now.UnixNano()),
		Type:      deterrentType,
		Building:  building,
		StartTime: now,
		Duration:  s.DeterrentDuration.Seconds(),
		Reason:    fmt.Sprintf("woodpecker count %d exceeds threshold %d", woodpeckerCount, s.WoodpeckerThreshold),
		Status:    "active",
		BirdCount: woodpeckerCount,
		BirdType:  "woodpecker",
	}

	s.ActiveDeterrents[building] = action
	s.LastDeterrentTime[building] = now

	return action
}

func (s *BirdRadarSimulator) GetActiveDeterrents(building string) []*models.DeterrentAction {
	s.UpdateDeterrentStatus(building)

	var active []*models.DeterrentAction
	if action, ok := s.ActiveDeterrents[building]; ok && action.Status == "active" {
		active = append(active, action)
	}

	return active
}

func (s *BirdRadarSimulator) UpdateDeterrentStatus(building string) {
	if action, ok := s.ActiveDeterrents[building]; ok {
		if time.Since(action.StartTime) >= s.DeterrentDuration {
			action.Status = "completed"
		}
	}
}

func (s *BirdRadarSimulator) GenerateSimulatedRadarData(building string, numScans int) [][]models.BirdRadarData {
	results := make([][]models.BirdRadarData, 0, numScans)
	for i := 0; i < numScans; i++ {
		scanData := s.Scan(building)
		results = append(results, scanData)
	}
	return results
}
