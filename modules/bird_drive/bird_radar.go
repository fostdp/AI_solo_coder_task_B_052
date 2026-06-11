package bird_drive

import (
	"fmt"
	"math/rand"
	"time"
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
	ActiveDeterrents    map[string]*DeterrentAction
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
		ActiveDeterrents:    make(map[string]*DeterrentAction),
		SimulationSpeed:     simSpeed,
		randSource:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *BirdRadarSimulator) Scan(building string) []BirdRadarData {
	count := s.randSource.Intn(9)
	birdTypes := []string{"sparrow", "swallow", "crow"}
	results := make([]BirdRadarData, 0, count)
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

		results = append(results, BirdRadarData{
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

func (s *BirdRadarSimulator) EvaluateDeterrentNeed(scanData []BirdRadarData, building string) *DeterrentAction {
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
	action := &DeterrentAction{
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

func (s *BirdRadarSimulator) GetActiveDeterrents(building string) []*DeterrentAction {
	s.UpdateDeterrentStatus(building)

	var active []*DeterrentAction
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

func (s *BirdRadarSimulator) GenerateSimulatedRadarData(building string, numScans int) [][]BirdRadarData {
	results := make([][]BirdRadarData, 0, numScans)
	for i := 0; i < numScans; i++ {
		scanData := s.Scan(building)
		results = append(results, scanData)
	}
	return results
}

func (s *BirdRadarSimulator) GetDeterrentStatus(building string, history []BirdRadarData) DeterrentStatus {
	activeDeterrents := s.GetActiveDeterrents(building)

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

	return DeterrentStatus{
		ActiveDeterrents: activeDeterrents,
		RecentBirdCount:  recentCount,
		WoodpeckerCount:  woodpeckerCount,
		ActivityLevel:    activityLevel,
	}
}

func (s *BirdRadarSimulator) TriggerDeterrent(building string, deterrentType string) *DeterrentAction {
	now := time.Now()
	action := &DeterrentAction{
		ID:        "DETER-" + building + "-" + now.Format("20060102150405"),
		Type:      deterrentType,
		Building:  building,
		StartTime: now,
		Duration:  s.DeterrentDuration.Seconds(),
		Reason:    "manual trigger",
		Status:    "active",
		BirdCount: 0,
		BirdType:  "",
	}

	s.ActiveDeterrents[building] = action
	s.LastDeterrentTime[building] = now

	return action
}
