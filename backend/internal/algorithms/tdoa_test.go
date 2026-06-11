package algorithms

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"ancient-wood-monitor/internal/models"
)

const soundSpeedWood = 3300.0

func generateTDOAMeasurements(sourceX, sourceY, sourceZ float64,
	sensorPositions [][3]float64, noiseStd float64) []models.TDOAMeasurement {

	rng := rand.New(rand.NewSource(42))
	measurements := make([]models.TDOAMeasurement, len(sensorPositions))
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	distances := make([]float64, len(sensorPositions))

	for i, pos := range sensorPositions {
		dx := pos[0] - sourceX
		dy := pos[1] - sourceY
		dz := pos[2] - sourceZ
		distances[i] = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}

	for i, pos := range sensorPositions {
		arrivalTime := baseTime.Add(time.Duration(distances[i]/soundSpeedWood*1e9) * time.Nanosecond)
		if noiseStd > 0 {
			jitterSeconds := rng.NormFloat64() * noiseStd * 1e-6
			arrivalTime = arrivalTime.Add(time.Duration(jitterSeconds*1e9) * time.Nanosecond)
		}

		measurements[i] = models.TDOAMeasurement{
			SensorID:  "S" + string(rune('0'+i)),
			Timestamp: arrivalTime,
			PosX:      pos[0],
			PosY:      pos[1],
			PosZ:      pos[2],
			Amplitude: 100.0 - distances[i]*0.1,
		}
	}

	return measurements
}

func TestTDOALocator_LocateSource_NormalCase(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sensorPositions := [][3]float64{
		{0, 0, 0},
		{5, 0, 0},
		{0, 5, 0},
		{0, 0, 5},
		{5, 5, 5},
	}

	sourceX, sourceY, sourceZ := 2.5, 2.5, 2.5

	measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 0)

	x, y, z, confidence, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("LocateSource failed: %v", err)
	}

	dx := x - sourceX
	dy := y - sourceY
	dz := z - sourceZ
	errorDistance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	t.Logf("Source: (%.3f, %.3f, %.3f)", sourceX, sourceY, sourceZ)
	t.Logf("Estimated: (%.3f, %.3f, %.3f)", x, y, z)
	t.Logf("Error distance: %.4f m", errorDistance)
	t.Logf("Confidence: %.4f", confidence)

	if errorDistance >= 0.5 {
		t.Errorf("localization error %.4f m exceeds 0.5 m threshold", errorDistance)
	}

	if confidence <= 0 || confidence > 1.0 {
		t.Errorf("confidence %.4f out of valid range (0, 1]", confidence)
	}
}

func TestTDOALocator_LocateSource_WithNoise(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sensorPositions := [][3]float64{
		{0, 0, 0},
		{10, 0, 0},
		{0, 10, 0},
		{5, 5, 10},
		{10, 10, 5},
		{3, 7, 8},
	}

	sourceX, sourceY, sourceZ := 4.0, 5.0, 3.0

	totalError := 0.0
	runs := 20
	for run := 0; run < runs; run++ {
		measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 5)

		x, y, z, _, err := locator.LocateSource(measurements)
		if err != nil {
			t.Fatalf("LocateSource failed on run %d: %v", run, err)
		}

		dx := x - sourceX
		dy := y - sourceY
		dz := z - sourceZ
		totalError += math.Sqrt(dx*dx + dy*dy + dz*dz)
	}

	avgError := totalError / float64(runs)
	t.Logf("Average localization error with noise: %.4f m", avgError)

	if avgError >= 0.5 {
		t.Errorf("average localization error %.4f m exceeds 0.5 m threshold", avgError)
	}
}

func TestTDOALocator_LocateSource_BoundaryMinSensors(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	sensorPositions := [][3]float64{
		{0, 0, 0},
		{3, 0, 0},
		{0, 3, 0},
		{0, 0, 3},
	}

	sourceX, sourceY, sourceZ := 1.0, 1.0, 1.0
	measurements := generateTDOAMeasurements(sourceX, sourceY, sourceZ, sensorPositions, 0)

	if len(measurements) != locator.MinSensors {
		t.Fatalf("expected %d sensors (minimum), got %d", locator.MinSensors, len(measurements))
	}

	x, y, z, confidence, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("LocateSource with min sensors failed: %v", err)
	}

	dx := x - sourceX
	dy := y - sourceY
	dz := z - sourceZ
	errorDistance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	t.Logf("Min sensors error: %.4f m, confidence: %.4f", errorDistance, confidence)

	if errorDistance >= 0.5 {
		t.Errorf("min sensors localization error %.4f m exceeds 0.5 m", errorDistance)
	}
}

func TestTDOALocator_LocateSource_ErrorInsufficientSensors(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	measurements := []models.TDOAMeasurement{
		{SensorID: "S1", Timestamp: time.Now(), PosX: 0, PosY: 0, PosZ: 0},
		{SensorID: "S2", Timestamp: time.Now().Add(1 * time.Millisecond), PosX: 5, PosY: 0, PosZ: 0},
		{SensorID: "S3", Timestamp: time.Now().Add(2 * time.Millisecond), PosX: 0, PosY: 5, PosZ: 0},
	}

	_, _, _, _, err := locator.LocateSource(measurements)
	if err == nil {
		t.Error("expected error for insufficient sensors, got nil")
	}
}

func TestTDOALocator_LocateSource_ErrorEmptyMeasurements(t *testing.T) {
	locator := NewTDOALocator(soundSpeedWood, 4, 0.5, 3.0, 100)

	_, _, _, _, err := locator.LocateSource([]models.TDOAMeasurement{})
	if err == nil {
		t.Error("expected error for empty measurements, got nil")
	}
}

func TestBuildTunnelNetwork_NormalCase(t *testing.T) {
	nodes := []models.TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0},
		{ID: "N2", PositionX: 2, PositionY: 0, PositionZ: 0},
		{ID: "N3", PositionX: 0, PositionY: 2, PositionZ: 0},
		{ID: "N4", PositionX: 5, PositionY: 5, PositionZ: 5},
	}

	edges := BuildTunnelNetwork(nodes, 3.0)

	t.Logf("Generated %d edges", len(edges))

	if len(edges) == 0 {
		t.Error("expected at least one edge, got 0")
	}

	for _, edge := range edges {
		if edge.Length > 3.0 {
			t.Errorf("edge %s-%s length %.2f exceeds max distance 3.0",
				edge.FromNodeID, edge.ToNodeID, edge.Length)
		}
		if edge.Strength < 0 || edge.Strength > 1.0 {
			t.Errorf("edge strength %.2f out of range [0, 1]", edge.Strength)
		}
	}

	prevStrength := 1.0
	for _, edge := range edges {
		if edge.Strength > prevStrength {
			t.Error("edges not sorted by strength descending")
		}
		prevStrength = edge.Strength
	}
}

func TestBuildTunnelNetwork_NoEdges(t *testing.T) {
	nodes := []models.TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0},
		{ID: "N2", PositionX: 10, PositionY: 0, PositionZ: 0},
		{ID: "N3", PositionX: 0, PositionY: 10, PositionZ: 0},
	}

	edges := BuildTunnelNetwork(nodes, 3.0)

	if len(edges) != 0 {
		t.Errorf("expected 0 edges when all nodes are far apart, got %d", len(edges))
	}
}

func TestBuildTunnelNetwork_EmptyNodes(t *testing.T) {
	edges := BuildTunnelNetwork([]models.TunnelNode{}, 3.0)
	if len(edges) != 0 {
		t.Errorf("expected 0 edges for empty nodes, got %d", len(edges))
	}
}

func TestMergeNode_MergeCloseNode(t *testing.T) {
	existing := []models.TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0, LastSeen: time.Now()},
	}

	newNode := models.TunnelNode{
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
	existing := []models.TunnelNode{
		{ID: "N1", PositionX: 0, PositionY: 0, PositionZ: 0},
	}

	newNode := models.TunnelNode{
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

func TestMergeNode_EmptyExisting(t *testing.T) {
	newNode := models.TunnelNode{
		ID: "N1", PositionX: 1.0, PositionY: 2.0, PositionZ: 3.0,
		LastSeen: time.Now(),
	}

	result, merged := MergeNode([]models.TunnelNode{}, newNode, 0.5)

	if merged {
		t.Error("expected no merge for empty existing list")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 node, got %d", len(result))
	}
	if result[0].PositionX != 1.0 {
		t.Errorf("node position X = %.2f, want 1.0", result[0].PositionX)
	}
}

func TestSolve4x4_Identity(t *testing.T) {
	mat := [][]float64{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	rhs := []float64{5, 3, 7, 2}

	sol, err := solve4x4(mat, rhs)
	if err != nil {
		t.Fatalf("solve4x4 failed: %v", err)
	}

	expected := []float64{5, 3, 7, 2}
	for i := range sol {
		if math.Abs(sol[i]-expected[i]) > 1e-10 {
			t.Errorf("sol[%d] = %.6f, want %.6f", i, sol[i], expected[i])
		}
	}
}

func TestSolve4x4_Singular(t *testing.T) {
	mat := [][]float64{
		{1, 2, 3, 4},
		{2, 4, 6, 8},
		{1, 0, 0, 0},
		{0, 1, 0, 0},
	}
	rhs := []float64{1, 2, 3, 4}

	_, err := solve4x4(mat, rhs)
	if err == nil {
		t.Error("expected error for singular matrix, got nil")
	}
}
