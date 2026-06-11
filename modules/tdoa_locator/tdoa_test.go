package tdoa_locator

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

const soundSpeedWood = 3300.0

func generateTDOAMeasurements(sourceX, sourceY, sourceZ float64,
	sensorPositions [][3]float64, noiseStd float64) []TDOAMeasurement {

	rng := rand.New(rand.NewSource(42))
	measurements := make([]TDOAMeasurement, len(sensorPositions))
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	refDist := math.Sqrt((sourceX-sensorPositions[0][0])*(sourceX-sensorPositions[0][0]) +
		(sourceY-sensorPositions[0][1])*(sourceY-sensorPositions[0][1]) +
		(sourceZ-sensorPositions[0][2])*(sourceZ-sensorPositions[0][2]))

	for i, pos := range sensorPositions {
		dist := math.Sqrt((sourceX-pos[0])*(sourceX-pos[0]) +
			(sourceY-pos[1])*(sourceY-pos[1]) +
			(sourceZ-pos[2])*(sourceZ-pos[2]))
		tdoa := (dist - refDist) / soundSpeedWood
		noise := rng.NormFloat64() * noiseStd
		if i > 0 {
			tdoa += noise
		}

		measurements[i] = TDOAMeasurement{
			SensorID:  fmt.Sprintf("S%d", i),
			Timestamp: baseTime.Add(time.Duration(tdoa * float64(time.Second))),
			PosX:      pos[0],
			PosY:      pos[1],
			PosZ:      pos[2],
			Amplitude: 100.0 - float64(i)*10,
		}
	}

	return measurements
}

func TestTDOALocator_LocateSource_Basic(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.0, 3.0, 1.5
	sensorPositions := [][3]float64{
		{0, 0, 0},
		{5, 0, 0},
		{0, 5, 0},
		{5, 5, 0},
	}

	measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 0)

	result, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("LocateSource failed: %v", err)
	}

	error := math.Sqrt((result.X-sourceX)*(result.X-sourceX) +
		(result.Y-sourceY)*(result.Y-sourceY) +
		(result.Z-sourceZ)*(result.Z-sourceZ))

	t.Logf("Basic localization error: %.4f m (confidence: %.4f)", error, result.Confidence)

	if error >= 0.01 {
		t.Errorf("localization error %.4f should be < 0.01 m for noise-free data", error)
	}

	if result.Confidence <= 0.99 {
		t.Errorf("confidence %.4f should be > 0.99 for noise-free data", result.Confidence)
	}
}

func TestTDOALocator_LocateSource_WithNoise(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.5, 1.5, 1.0
	sensorPositions := [][3]float64{
		{0, 0, 0},
		{5, 0, 0},
		{0, 5, 0},
		{5, 5, 0},
		{2.5, 2.5, 2},
	}

	runCount := 20
	var totalError float64
	failCount := 0

	for run := 0; run < runCount; run++ {
		measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 0.5e-3)

		result, err := locator.LocateSource(measurements)
		if err != nil {
			failCount++
			continue
		}

		error := math.Sqrt((result.X-sourceX)*(result.X-sourceX) +
			(result.Y-sourceY)*(result.Y-sourceY) +
			(result.Z-sourceZ)*(result.Z-sourceZ))
		totalError += error
	}

	avgError := totalError / float64(runCount-failCount)
	t.Logf("WLS average error over %d runs: %.4f m (failed: %d)", runCount, avgError, failCount)
	t.Logf("Convergence rate: %.1f%%", float64(runCount-failCount)/float64(runCount)*100)

	if avgError >= 0.5 {
		t.Errorf("average error %.4f should be < 0.5 m", avgError)
	}
}

func TestTDOALocator_LocateSource_InsufficientSensors(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	measurements := []TDOAMeasurement{
		{SensorID: "S0", PosX: 0, PosY: 0, PosZ: 0, Timestamp: time.Now()},
		{SensorID: "S1", PosX: 1, PosY: 0, PosZ: 0, Timestamp: time.Now()},
		{SensorID: "S2", PosX: 0, PosY: 1, PosZ: 0, Timestamp: time.Now()},
	}

	_, err := locator.LocateSource(measurements)
	if err == nil {
		t.Error("expected error for insufficient sensors, got nil")
	}
}

func TestBuildTunnelNetwork_Basic(t *testing.T) {
	nodes := []TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0},
		{ID: "N2", PositionX: 2, PositionY: 0, PositionZ: 0},
		{ID: "N3", PositionX: 0, PositionY: 2, PositionZ: 0},
		{ID: "N4", PositionX: 10, PositionY: 10, PositionZ: 0},
	}

	edges := BuildTunnelNetwork(nodes, 3.0)

	if len(edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(edges))
	}

	if edges[0].Strength < edges[len(edges)-1].Strength {
		t.Error("edges should be sorted by strength descending")
	}
}

func TestBuildTunnelNetwork_EmptyNodes(t *testing.T) {
	edges := BuildTunnelNetwork([]TunnelNode{}, 3.0)
	if len(edges) != 0 {
		t.Errorf("expected 0 edges for empty nodes, got %d", len(edges))
	}
}

func TestMergeNode_MergeCloseNode(t *testing.T) {
	existing := []TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0, LastSeen: time.Now()},
	}

	newNode := TunnelNode{
		ID: "N2", PositionX: 0.3, PositionY: 0.0, PositionZ: 0.0,
		LastSeen: time.Now().Add(1 * time.Hour),
	}

	result, merged := MergeNode(existing, newNode, 0.5)

	if !merged {
		t.Error("expected merge for close node, got no merge")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 node after merge, got %d", len(result))
	}

	expectedX := (0 + 0.3) / 2.0
	if math.Abs(result[0].PositionX-expectedX) > 1e-9 {
		t.Errorf("merged node X position = %.4f, want %.4f", result[0].PositionX, expectedX)
	}
}

func TestMergeNode_AddNewNode(t *testing.T) {
	existing := []TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0},
	}

	newNode := TunnelNode{
		ID: "N2", PositionX: 5.0, PositionY: 0.0, PositionZ: 0.0,
		LastSeen: time.Now(),
	}

	result, merged := MergeNode(existing, newNode, 0.5)

	if merged {
		t.Error("expected no merge for distant node")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result))
	}
}

func TestTDOALocator_GonumVsCustom(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.0, 2.0, 1.0
	sensorPositions := [][3]float64{
		{0, 0, 0},
		{4, 0, 0},
		{0, 4, 0},
		{4, 4, 0},
		{2, 2, 2},
	}

	measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 0.3e-3)

	result, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("gonum solver failed: %v", err)
	}

	error := math.Sqrt((result.X-sourceX)*(result.X-sourceX) +
		(result.Y-sourceY)*(result.Y-sourceY) +
		(result.Z-sourceZ)*(result.Z-sourceZ))

	t.Logf("gonum solver error: %.4f m, confidence: %.4f", error, result.Confidence)

	if error >= 0.5 {
		t.Errorf("gonum solver error %.4f should be < 0.5 m", error)
	}
}
