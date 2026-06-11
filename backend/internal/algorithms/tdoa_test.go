package algorithms

import (
	"fmt"
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

func TestTDOALocator_WLS_WeightCalculation(t *testing.T) {
	locator := NewTDOALocator(3300.0, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.0, 3.0, 1.5
	baseTime := time.Now()

	measurements := []models.TDOAMeasurement{
		{SensorID: "S0", PosX: 0, PosY: 0, PosZ: 0, Timestamp: baseTime.Add(time.Duration(0)), Amplitude: 100.0},
		{SensorID: "S1", PosX: 5, PosY: 0, PosZ: 0, Timestamp: baseTime.Add(time.Duration(0)), Amplitude: 80.0},
		{SensorID: "S2", PosX: 0, PosY: 5, PosZ: 0, Timestamp: baseTime.Add(time.Duration(0)), Amplitude: 60.0},
		{SensorID: "S3", PosX: 5, PosY: 5, PosZ: 0, Timestamp: baseTime.Add(time.Duration(0)), Amplitude: 40.0},
	}

	for i := 1; i < len(measurements); i++ {
		dx := sourceX - measurements[i].PosX
		dy := sourceY - measurements[i].PosY
		dz := sourceZ - measurements[i].PosZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		d0 := math.Sqrt(sourceX*sourceX + sourceY*sourceY + sourceZ*sourceZ)
		tdoa := (dist - d0) / 3300.0
		measurements[i].Timestamp = measurements[0].Timestamp.Add(time.Duration(tdoa * float64(time.Second)))
	}

	x, y, z, confidence, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("WLS localization failed: %v", err)
	}

	error := math.Sqrt((x-sourceX)*(x-sourceX) + (y-sourceY)*(y-sourceY) + (z-sourceZ)*(z-sourceZ))
	t.Logf("WLS localization error: %.4f m (confidence: %.4f)", error, confidence)

	if error >= 0.5 {
		t.Errorf("WLS localization error %.4f should be < 0.5 m", error)
	}

	if confidence <= 0.8 {
		t.Errorf("WLS confidence %.4f should be > 0.8", confidence)
	}
}

func TestTDOALocator_WLS_HighAmplitudeWeight(t *testing.T) {
	locator := NewTDOALocator(3300.0, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 1.0, 1.0, 0.5
	baseTime := time.Now()

	measurements := []models.TDOAMeasurement{
		{SensorID: "S0", PosX: 0, PosY: 0, PosZ: 0, Timestamp: baseTime, Amplitude: 100.0},
		{SensorID: "S1", PosX: 3, PosY: 0, PosZ: 0, Timestamp: baseTime, Amplitude: 90.0},
		{SensorID: "S2", PosX: 0, PosY: 3, PosZ: 0, Timestamp: baseTime, Amplitude: 85.0},
		{SensorID: "S3", PosX: 3, PosY: 3, PosZ: 0, Timestamp: baseTime, Amplitude: 10.0},
	}

	for i := 1; i < len(measurements); i++ {
		dx := sourceX - measurements[i].PosX
		dy := sourceY - measurements[i].PosY
		dz := sourceZ - measurements[i].PosZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		d0 := math.Sqrt(sourceX*sourceX + sourceY*sourceY + sourceZ*sourceZ)
		tdoa := (dist - d0) / 3300.0
		measurements[i].Timestamp = measurements[0].Timestamp.Add(time.Duration(tdoa * float64(time.Second)))
	}

	measurements[3].Timestamp = measurements[3].Timestamp.Add(2 * time.Millisecond)

	x, y, z, confidence, err := locator.LocateSource(measurements)
	if err != nil {
		t.Fatalf("WLS localization failed: %v", err)
	}

	error := math.Sqrt((x-sourceX)*(x-sourceX) + (y-sourceY)*(y-sourceY) + (z-sourceZ)*(z-sourceZ))
	t.Logf("WLS with weighted low-amplitude sensor: error=%.4f m, confidence=%.4f", error, confidence)

	if error >= 0.5 {
		t.Errorf("WLS should down-weight low-amplitude noisy sensor, error=%.4f >= 0.5", error)
	}
}

func TestTDOALocator_WLS_NoiseRobustness(t *testing.T) {
	locator := NewTDOALocator(3300.0, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.5, 1.5, 1.0
	baseTime := time.Now()

	sensors := []struct {
		x, y, z  float64
		amplitude float64
	}{
		{0, 0, 0, 100.0},
		{5, 0, 0, 95.0},
		{0, 5, 0, 90.0},
		{5, 5, 0, 85.0},
		{2.5, 2.5, 2, 80.0},
	}

	noiseStdDev := 0.5e-3
	rng := rand.New(rand.NewSource(42))

	runCount := 20
	var totalError float64
	errorCount := 0

	for run := 0; run < runCount; run++ {
		measurements := make([]models.TDOAMeasurement, len(sensors))
		for i, s := range sensors {
			dx := sourceX - s.x
			dy := sourceY - s.y
			dz := sourceZ - s.z
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
			d0 := math.Sqrt(sourceX*sourceX + sourceY*sourceY + sourceZ*sourceZ)
			tdoa := (dist - d0) / 3300.0

			noise := rng.NormFloat64() * noiseStdDev
			if i > 0 {
				tdoa += noise
			}

			measurements[i] = models.TDOAMeasurement{
				SensorID:  fmt.Sprintf("S%d", i),
				PosX:      s.x,
				PosY:      s.y,
				PosZ:      s.z,
				Timestamp: baseTime.Add(time.Duration(tdoa * float64(time.Second))),
				Amplitude: s.amplitude,
			}
		}

		x, y, z, _, err := locator.LocateSource(measurements)
		if err != nil {
			errorCount++
			continue
		}

		error := math.Sqrt((x-sourceX)*(x-sourceX) + (y-sourceY)*(y-sourceY) + (z-sourceZ)*(z-sourceZ))
		totalError += error
	}

	avgError := totalError / float64(runCount-errorCount)
	t.Logf("WLS average error over %d runs: %.4f m (failed: %d)", runCount, avgError, errorCount)
	t.Logf("WLS convergence rate: %.1f%%", float64(runCount-errorCount)/float64(runCount)*100)

	if avgError >= 0.5 {
		t.Errorf("WLS average error %.4f should be < 0.5 m", avgError)
	}

	if errorCount > runCount/2 {
		t.Errorf("WLS failed %d/%d times, too many failures", errorCount, runCount)
	}
}

func TestTDOALocator_WLS_ConfidenceMetric(t *testing.T) {
	locator := NewTDOALocator(3300.0, 4, 0.5, 3.0, 100)

	sourceX, sourceY, sourceZ := 2.0, 2.0, 1.0
	baseTime := time.Now()

	cleanMeasurements := []models.TDOAMeasurement{
		{SensorID: "S0", PosX: 0, PosY: 0, PosZ: 0, Timestamp: baseTime, Amplitude: 100.0},
		{SensorID: "S1", PosX: 4, PosY: 0, PosZ: 0, Timestamp: baseTime, Amplitude: 100.0},
		{SensorID: "S2", PosX: 0, PosY: 4, PosZ: 0, Timestamp: baseTime, Amplitude: 100.0},
		{SensorID: "S3", PosX: 4, PosY: 4, PosZ: 0, Timestamp: baseTime, Amplitude: 100.0},
	}

	for i := 1; i < len(cleanMeasurements); i++ {
		dx := sourceX - cleanMeasurements[i].PosX
		dy := sourceY - cleanMeasurements[i].PosY
		dz := sourceZ - cleanMeasurements[i].PosZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		d0 := math.Sqrt(sourceX*sourceX + sourceY*sourceY + sourceZ*sourceZ)
		tdoa := (dist - d0) / 3300.0
		cleanMeasurements[i].Timestamp = cleanMeasurements[0].Timestamp.Add(time.Duration(tdoa * float64(time.Second)))
	}

	_, _, _, cleanConfidence, _ := locator.LocateSource(cleanMeasurements)

	noisyMeasurements := make([]models.TDOAMeasurement, len(cleanMeasurements))
	copy(noisyMeasurements, cleanMeasurements)
	for i := 1; i < len(noisyMeasurements); i++ {
		noisyMeasurements[i].Timestamp = noisyMeasurements[i].Timestamp.Add(3 * time.Millisecond)
	}

	_, _, _, noisyConfidence, _ := locator.LocateSource(noisyMeasurements)

	t.Logf("Clean data confidence: %.4f", cleanConfidence)
	t.Logf("Noisy data confidence: %.4f", noisyConfidence)

	if cleanConfidence <= noisyConfidence {
		t.Errorf("clean data should have higher confidence than noisy data: clean=%.4f, noisy=%.4f",
			cleanConfidence, noisyConfidence)
	}

	if cleanConfidence <= 0.8 {
		t.Errorf("clean data confidence %.4f should be > 0.8", cleanConfidence)
	}
}
