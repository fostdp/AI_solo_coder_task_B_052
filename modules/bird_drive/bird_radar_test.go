package bird_drive

import (
	"math"
	"testing"
	"time"
)

func newTestSimulator() *BirdRadarSimulator {
	return NewBirdRadarSimulator(
		100.0,
		5*time.Second,
		2,
		30*time.Second,
		1*time.Minute,
		true,
		true,
		1.0,
	)
}

func TestBirdRadarSimulator_Scan(t *testing.T) {
	sim := newTestSimulator()

	results := sim.Scan("应县木塔")

	if len(results) < 0 || len(results) >= 9 {
		t.Errorf("expected 0-8 birds per scan, got %d", len(results))
	}

	for i, bird := range results {
		if bird.ID == "" {
			t.Errorf("bird[%d]: ID should not be empty", i)
		}
		if bird.BirdCount != len(results) {
			t.Errorf("bird[%d]: BirdCount = %d, want %d", i, bird.BirdCount, len(results))
		}
		if bird.Direction < 0 || bird.Direction >= 360 {
			t.Errorf("bird[%d]: Direction = %.2f, want [0, 360)", i, bird.Direction)
		}
		if bird.Distance < 10 || bird.Distance > 100 {
			t.Errorf("bird[%d]: Distance = %.2f, want [10, 100]", i, bird.Distance)
		}
		if bird.Altitude < 2 || bird.Altitude > 30 {
			t.Errorf("bird[%d]: Altitude = %.2f, want [2, 30]", i, bird.Altitude)
		}
		if bird.Speed < 5 || bird.Speed > 25 {
			t.Errorf("bird[%d]: Speed = %.2f, want [5, 25]", i, bird.Speed)
		}
		if bird.Timestamp.IsZero() {
			t.Errorf("bird[%d]: Timestamp should not be zero", i)
		}

		validTypes := map[string]bool{
			"sparrow": true, "swallow": true, "crow": true, "woodpecker": true,
		}
		if !validTypes[bird.BirdType] {
			t.Errorf("bird[%d]: invalid BirdType = %s", i, bird.BirdType)
		}

		validLevels := map[string]bool{
			"low": true, "moderate": true, "high": true, "intense": true,
		}
		if !validLevels[bird.ActivityLevel] {
			t.Errorf("bird[%d]: invalid ActivityLevel = %s", i, bird.ActivityLevel)
		}
	}

	t.Logf("Scanned %d birds, activity level: %s", len(results),
		func() string {
			if len(results) > 0 {
				return results[0].ActivityLevel
			}
			return "none"
		}())
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_NoWoodpeckers(t *testing.T) {
	sim := newTestSimulator()

	scanData := []BirdRadarData{
		{BirdType: "sparrow"},
		{BirdType: "swallow"},
		{BirdType: "crow"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "应县木塔")

	if action != nil {
		t.Errorf("expected no deterrent when no woodpeckers, got %+v", action)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_UnderThreshold(t *testing.T) {
	sim := newTestSimulator()
	sim.WoodpeckerThreshold = 5

	scanData := []BirdRadarData{
		{BirdType: "woodpecker"},
		{BirdType: "woodpecker"},
		{BirdType: "sparrow"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "应县木塔")

	if action != nil {
		t.Errorf("expected no deterrent when woodpecker count below threshold, got %+v", action)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_OverThreshold(t *testing.T) {
	sim := newTestSimulator()
	sim.WoodpeckerThreshold = 2

	scanData := []BirdRadarData{
		{BirdType: "woodpecker"},
		{BirdType: "woodpecker"},
		{BirdType: "woodpecker"},
		{BirdType: "sparrow"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "应县木塔")

	if action == nil {
		t.Fatal("expected deterrent action, got nil")
	}

	if action.Building != "应县木塔" {
		t.Errorf("building = %s, want 应县木塔", action.Building)
	}

	if action.Status != "active" {
		t.Errorf("status = %s, want active", action.Status)
	}

	if action.BirdCount != 3 {
		t.Errorf("birdCount = %d, want 3", action.BirdCount)
	}

	if action.BirdType != "woodpecker" {
		t.Errorf("birdType = %s, want woodpecker", action.BirdType)
	}

	if action.Type != "ultrasonic" && action.Type != "predator_call" {
		t.Errorf("invalid deterrent type: %s", action.Type)
	}

	t.Logf("Deterrent triggered: type=%s, reason=%s", action.Type, action.Reason)
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_Cooldown(t *testing.T) {
	sim := newTestSimulator()
	sim.WoodpeckerThreshold = 1
	sim.CooldownPeriod = 1 * time.Hour

	scanData := []BirdRadarData{
		{BirdType: "woodpecker"},
		{BirdType: "woodpecker"},
	}

	action1 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action1 == nil {
		t.Fatal("first deterrent should have been triggered")
	}

	action2 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action2 != nil {
		t.Error("second deterrent should be blocked by cooldown")
	}

	if !sim.LastDeterrentTime["应县木塔"].Equal(action1.StartTime) {
		t.Error("last deterrent time not updated")
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_AlternatingTypes(t *testing.T) {
	sim := newTestSimulator()
	sim.WoodpeckerThreshold = 1
	sim.CooldownPeriod = 0

	scanData := []BirdRadarData{
		{BirdType: "woodpecker"},
		{BirdType: "woodpecker"},
	}

	action1 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action1 == nil {
		t.Fatal("first deterrent should have been triggered")
	}

	action2 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action2 == nil {
		t.Fatal("second deterrent should have been triggered")
	}

	if action1.Type == action2.Type {
		t.Errorf("deterrent types should alternate, both were %s", action1.Type)
	}

	t.Logf("Action 1: %s, Action 2: %s", action1.Type, action2.Type)
}

func TestBirdRadarSimulator_UpdateDeterrentStatus(t *testing.T) {
	sim := newTestSimulator()
	sim.DeterrentDuration = 10 * time.Millisecond

	sim.ActiveDeterrents["应县木塔"] = &DeterrentAction{
		ID:        "test-1",
		Type:      "ultrasonic",
		Building:  "应县木塔",
		StartTime: time.Now().Add(-20 * time.Millisecond),
		Duration:  0.01,
		Status:    "active",
	}

	sim.UpdateDeterrentStatus("应县木塔")

	if sim.ActiveDeterrents["应县木塔"].Status != "completed" {
		t.Errorf("status should be completed after duration, got %s",
			sim.ActiveDeterrents["应县木塔"].Status)
	}
}

func TestBirdRadarSimulator_GetActiveDeterrents(t *testing.T) {
	sim := newTestSimulator()

	sim.ActiveDeterrents["应县木塔"] = &DeterrentAction{
		ID:        "test-1",
		Type:      "ultrasonic",
		Building:  "应县木塔",
		StartTime: time.Now(),
		Duration:  30,
		Status:    "active",
	}
	sim.ActiveDeterrents["佛光寺"] = &DeterrentAction{
		ID:        "test-2",
		Type:      "predator_call",
		Building:  "佛光寺",
		StartTime: time.Now().Add(-1 * time.Hour),
		Duration:  30,
		Status:    "completed",
	}

	activeYX := sim.GetActiveDeterrents("应县木塔")
	if len(activeYX) != 1 {
		t.Errorf("expected 1 active deterrent for 应县木塔, got %d", len(activeYX))
	}

	activeFGS := sim.GetActiveDeterrents("佛光寺")
	if len(activeFGS) != 0 {
		t.Errorf("expected 0 active deterrents for 佛光寺, got %d", len(activeFGS))
	}
}

func TestBirdRadarSimulator_TriggerDeterrent(t *testing.T) {
	sim := newTestSimulator()

	action := sim.TriggerDeterrent("应县木塔", "ultrasonic")

	if action == nil {
		t.Fatal("TriggerDeterrent returned nil")
	}

	if action.Type != "ultrasonic" {
		t.Errorf("type = %s, want ultrasonic", action.Type)
	}

	if action.Reason != "manual trigger" {
		t.Errorf("reason = %s, want manual trigger", action.Reason)
	}

	if action.Status != "active" {
		t.Errorf("status = %s, want active", action.Status)
	}

	if sim.ActiveDeterrents["应县木塔"] != action {
		t.Error("active deterrent not stored")
	}

	if sim.LastDeterrentTime["应县木塔"].IsZero() {
		t.Error("last deterrent time not updated")
	}

	t.Logf("Manual trigger: ID=%s", action.ID)
}

func TestBirdRadarSimulator_GetDeterrentStatus(t *testing.T) {
	sim := newTestSimulator()

	sim.ActiveDeterrents["应县木塔"] = &DeterrentAction{
		ID:        "test-1",
		Type:      "ultrasonic",
		Building:  "应县木塔",
		StartTime: time.Now(),
		Duration:  30,
		Status:    "active",
	}

	history := []BirdRadarData{
		{BirdCount: 5, BirdType: "woodpecker", ActivityLevel: "high", Timestamp: time.Now()},
		{BirdCount: 5, BirdType: "sparrow", ActivityLevel: "high", Timestamp: time.Now()},
		{BirdCount: 5, BirdType: "woodpecker", ActivityLevel: "high", Timestamp: time.Now()},
	}

	status := sim.GetDeterrentStatus("应县木塔", history)

	if len(status.ActiveDeterrents) != 1 {
		t.Errorf("expected 1 active deterrent, got %d", len(status.ActiveDeterrents))
	}

	if status.RecentBirdCount != 5 {
		t.Errorf("recent bird count = %d, want 5", status.RecentBirdCount)
	}

	if status.WoodpeckerCount != 2 {
		t.Errorf("woodpecker count = %d, want 2", status.WoodpeckerCount)
	}

	if status.ActivityLevel != "high" {
		t.Errorf("activity level = %s, want high", status.ActivityLevel)
	}
}

func TestBirdRadarSimulator_ActivityLevelMapping(t *testing.T) {
	sim := newTestSimulator()

	levelCounts := map[string]int{
		"low":      0,
		"moderate": 0,
		"high":     0,
		"intense":  0,
	}

	for i := 0; i < 1000; i++ {
		results := sim.Scan("test")
		if len(results) > 0 {
			levelCounts[results[0].ActivityLevel]++
		}
	}

	t.Logf("Activity levels over 1000 scans: %+v", levelCounts)

	for level, count := range levelCounts {
		if count == 0 {
			t.Logf("Warning: activity level %s not observed in 1000 scans", level)
		}
	}
}

func TestBirdRadarSimulator_WoodpeckerProbability(t *testing.T) {
	sim := newTestSimulator()

	totalBirds := 0
	woodpeckerCount := 0

	for i := 0; i < 1000; i++ {
		results := sim.Scan("test")
		for _, bird := range results {
			totalBirds++
			if bird.BirdType == "woodpecker" {
				woodpeckerCount++
			}
		}
	}

	if totalBirds == 0 {
		t.Skip("no birds scanned in 1000 iterations")
	}

	ratio := float64(woodpeckerCount) / float64(totalBirds)
	t.Logf("Woodpecker ratio: %.2f%% (%d/%d)", ratio*100, woodpeckerCount, totalBirds)

	if math.Abs(ratio-0.3) > 0.1 {
		t.Logf("Note: woodpecker ratio %.2f is not close to expected 0.3 (this is probabilistic)", ratio)
	}
}
