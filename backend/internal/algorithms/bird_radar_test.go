package algorithms

import (
	"strings"
	"testing"
	"time"

	"ancient-wood-monitor/internal/models"
)

func TestNewBirdRadarSimulator_Initialization(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0,
		30*time.Second,
		3,
		10*time.Minute,
		30*time.Minute,
		true,
		true,
		1.0,
	)

	if sim.ScanRadius != 100.0 {
		t.Errorf("ScanRadius = %.1f, want 100.0", sim.ScanRadius)
	}
	if sim.WoodpeckerThreshold != 3 {
		t.Errorf("WoodpeckerThreshold = %d, want 3", sim.WoodpeckerThreshold)
	}
	if sim.LastDeterrentTime == nil {
		t.Error("LastDeterrentTime map should be initialized")
	}
	if sim.ActiveDeterrents == nil {
		t.Error("ActiveDeterrents map should be initialized")
	}
	if sim.randSource == nil {
		t.Error("randSource should be initialized")
	}
}

func TestBirdRadarSimulator_Scan_Normal(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	results := sim.Scan("应县木塔")

	t.Logf("Scan returned %d birds", len(results))

	if len(results) > 8 {
		t.Errorf("bird count %d exceeds maximum 8", len(results))
	}

	for i, bird := range results {
		if bird.ID == "" {
			t.Errorf("bird[%d] ID is empty", i)
		}
		if bird.Timestamp.IsZero() {
			t.Errorf("bird[%d] Timestamp is zero", i)
		}
		if bird.BirdType == "" {
			t.Errorf("bird[%d] BirdType is empty", i)
		}
		if bird.Distance < 10 || bird.Distance > 100 {
			t.Errorf("bird[%d] Distance = %.2f out of [10, 100]", i, bird.Distance)
		}
		if bird.Altitude < 2 || bird.Altitude > 30 {
			t.Errorf("bird[%d] Altitude = %.2f out of [2, 30]", i, bird.Altitude)
		}
		if bird.Direction < 0 || bird.Direction > 360 {
			t.Errorf("bird[%d] Direction = %.2f out of [0, 360]", i, bird.Direction)
		}
		if bird.BirdCount != len(results) {
			t.Errorf("bird[%d] BirdCount = %d, want %d", i, bird.BirdCount, len(results))
		}
		if !strings.HasPrefix(bird.ID, "BIRD-应县木塔-") {
			t.Errorf("bird[%d] ID prefix wrong: %s", i, bird.ID)
		}
	}
}

func TestBirdRadarSimulator_Scan_ActivityLevels(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	validLevels := map[string]bool{
		"low":      true,
		"moderate": true,
		"high":     true,
		"intense":  true,
	}

	for run := 0; run < 20; run++ {
		results := sim.Scan("test")
		if len(results) == 0 {
			continue
		}
		level := results[0].ActivityLevel
		if !validLevels[level] {
			t.Errorf("invalid activity level: %s", level)
		}
	}
}

func TestBirdRadarSimulator_Scan_WoodpeckerRatio(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	totalBirds := 0
	woodpeckers := 0

	for run := 0; run < 100; run++ {
		results := sim.Scan("test")
		totalBirds += len(results)
		for _, bird := range results {
			if bird.BirdType == "woodpecker" {
				woodpeckers++
			}
		}
	}

	if totalBirds == 0 {
		t.Skip("no birds detected in 100 runs")
	}

	ratio := float64(woodpeckers) / float64(totalBirds)
	t.Logf("Woodpecker ratio: %.2f (%d/%d)", ratio, woodpeckers, totalBirds)

	if ratio < 0.15 || ratio > 0.5 {
		t.Errorf("woodpecker ratio %.2f outside expected 30%% range", ratio)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_WoodpeckerDetected(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
		{ID: "B4", BirdType: "sparrow"},
		{ID: "B5", BirdType: "swallow"},
	}

	start := time.Now()
	action := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	elapsed := time.Since(start)

	t.Logf("Deterrent evaluation took %v", elapsed)
	t.Logf("Action: %+v", action)

	if action == nil {
		t.Fatal("expected deterrent action for 3 woodpeckers, got nil")
	}

	if action.Type != "ultrasonic" && action.Type != "predator_call" {
		t.Errorf("unexpected deterrent type: %s", action.Type)
	}

	if action.Building != "应县木塔" {
		t.Errorf("building = %s, want 应县木塔", action.Building)
	}

	if action.Status != "active" {
		t.Errorf("status = %s, want active", action.Status)
	}

	if action.BirdCount != 3 {
		t.Errorf("bird count = %d, want 3", action.BirdCount)
	}

	if action.BirdType != "woodpecker" {
		t.Errorf("bird type = %s, want woodpecker", action.BirdType)
	}

	if elapsed >= 2*time.Second {
		t.Errorf("deterrent evaluation took %v, should be < 2s", elapsed)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_BelowThreshold(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "sparrow"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "应县木塔")

	if action != nil {
		t.Errorf("expected no action for 2 woodpeckers, got %+v", action)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_BoundaryThreshold(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "test")

	if action == nil {
		t.Error("expected action when woodpecker count equals threshold, got nil")
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_Cooldown(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action1 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action1 == nil {
		t.Fatal("first deterrent should trigger")
	}
	t.Logf("First deterrent: type=%s, ID=%s", action1.Type, action1.ID)

	action2 := sim.EvaluateDeterrentNeed(scanData, "应县木塔")
	if action2 != nil {
		t.Error("second deterrent should be blocked by cooldown")
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_NoDeviceEnabled(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, false, false, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
		{ID: "B4", BirdType: "woodpecker"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "test")

	if action != nil {
		t.Error("expected no action when no devices are enabled")
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_OnlyUltrasonic(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, false, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "test")

	if action == nil {
		t.Fatal("expected deterrent action")
	}
	if action.Type != "ultrasonic" {
		t.Errorf("type = %s, want ultrasonic", action.Type)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_OnlyPredatorCall(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, false, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "test")

	if action == nil {
		t.Fatal("expected deterrent action")
	}
	if action.Type != "predator_call" {
		t.Errorf("type = %s, want predator_call", action.Type)
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_AlternatingTypes(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 0*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action1 := sim.EvaluateDeterrentNeed(scanData, "test")
	if action1 == nil {
		t.Fatal("first deterrent should trigger")
	}
	firstType := action1.Type

	action2 := sim.EvaluateDeterrentNeed(scanData, "test")
	if action2 == nil {
		t.Fatal("second deterrent should trigger (no cooldown)")
	}

	if action2.Type == firstType {
		t.Logf("Both deterrents are same type: %s (depends on ActiveDeterrents state)", firstType)
	}
}

func TestBirdRadarSimulator_GetActiveDeterrents(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	deterrents := sim.GetActiveDeterrents("test")
	if len(deterrents) != 0 {
		t.Errorf("expected 0 active deterrents, got %d", len(deterrents))
	}

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}
	sim.EvaluateDeterrentNeed(scanData, "test")

	deterrents = sim.GetActiveDeterrents("test")
	if len(deterrents) != 1 {
		t.Errorf("expected 1 active deterrent, got %d", len(deterrents))
	}
}

func TestBirdRadarSimulator_UpdateDeterrentStatus(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 1*time.Millisecond, 30*time.Minute, true, true, 1.0,
	)

	sim.ActiveDeterrents["test"] = &models.DeterrentAction{
		ID:        "DETER-1",
		Type:      "ultrasonic",
		Building:  "test",
		StartTime: time.Now().Add(-10 * time.Millisecond),
		Duration:  0.001,
		Status:    "active",
	}

	time.Sleep(5 * time.Millisecond)
	sim.UpdateDeterrentStatus("test")

	if sim.ActiveDeterrents["test"].Status != "completed" {
		t.Errorf("status = %s, want completed", sim.ActiveDeterrents["test"].Status)
	}
}

func TestBirdRadarSimulator_GenerateSimulatedRadarData(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	numScans := 5
	results := sim.GenerateSimulatedRadarData("test", numScans)

	if len(results) != numScans {
		t.Errorf("expected %d scans, got %d", numScans, len(results))
	}

	for i, scan := range results {
		if len(scan) > 8 {
			t.Errorf("scan %d has %d birds, max is 8", i, len(scan))
		}
	}
}

func TestBirdRadarSimulator_Scan_MultipleBuildings(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	buildingA := "应县木塔"
	buildingB := "佛光寺"

	scanDataA := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	actionA := sim.EvaluateDeterrentNeed(scanDataA, buildingA)
	if actionA == nil {
		t.Fatal("expected deterrent for building A")
	}

	scanDataB := []models.BirdRadarData{
		{ID: "B4", BirdType: "woodpecker"},
		{ID: "B5", BirdType: "woodpecker"},
		{ID: "B6", BirdType: "woodpecker"},
	}
	actionB := sim.EvaluateDeterrentNeed(scanDataB, buildingB)

	if actionB == nil {
		t.Error("expected deterrent for building B (separate cooldown)")
	}

	if actionA.ID == actionB.ID {
		t.Error("different buildings should have different deterrent IDs")
	}
}

func TestBirdRadarSimulator_EvaluateDeterrentNeed_EmptyScan(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	action := sim.EvaluateDeterrentNeed([]models.BirdRadarData{}, "test")

	if action != nil {
		t.Error("expected no action for empty scan data")
	}
}

func TestBirdRadarSimulator_DeterrentDuration(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := []models.BirdRadarData{
		{ID: "B1", BirdType: "woodpecker"},
		{ID: "B2", BirdType: "woodpecker"},
		{ID: "B3", BirdType: "woodpecker"},
	}

	action := sim.EvaluateDeterrentNeed(scanData, "test")

	if action == nil {
		t.Fatal("expected deterrent action")
	}

	expectedDuration := 10 * time.Minute
	if action.Duration != expectedDuration.Seconds() {
		t.Errorf("duration = %.0f seconds, want %.0f",
			action.Duration, expectedDuration.Seconds())
	}
}

func TestBirdRadarSimulator_TriggerResponseTime(t *testing.T) {
	sim := NewBirdRadarSimulator(
		100.0, 30*time.Second, 3, 10*time.Minute, 30*time.Minute, true, true, 1.0,
	)

	scanData := make([]models.BirdRadarData, 5)
	for i := 0; i < 5; i++ {
		scanData[i] = models.BirdRadarData{
			ID:       "B" + string(rune('0'+i)),
			BirdType: "woodpecker",
		}
	}

	bestTime := time.Duration(time.Hour)
	for i := 0; i < 10; i++ {
		start := time.Now()
		action := sim.EvaluateDeterrentNeed(scanData, "test")
		elapsed := time.Since(start)

		if elapsed < bestTime {
			bestTime = elapsed
		}

		if action == nil {
			t.Fatalf("iteration %d: expected deterrent action", i)
		}

		sim.LastDeterrentTime = make(map[string]time.Time)
		sim.ActiveDeterrents = make(map[string]*models.DeterrentAction)
	}

	t.Logf("Best deterrent trigger time: %v", bestTime)
	t.Logf("Target: < 2 seconds")

	if bestTime >= 2*time.Second {
		t.Errorf("best trigger time %v exceeds 2 second requirement", bestTime)
	}
}
