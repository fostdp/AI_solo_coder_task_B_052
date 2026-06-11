package algorithms

import (
	"math"
	"testing"

	"ancient-wood-monitor/internal/models"
)

const (
	testRefDensity     = 450.0
	testCriticalEnergy = 50000.0
	testRequiredSF     = 1.5
	testDepthRatio     = 0.3
)

func newTestEvaluator() *WoodStrengthEvaluator {
	return NewWoodStrengthEvaluator(testRefDensity, testCriticalEnergy, testRequiredSF, testDepthRatio)
}

func TestWoodStrengthEvaluator_AssessStrength_LightDamage(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 5000, 450, 0.3)

	t.Logf("Light damage - CumulativeEnergy: %.0f, SafetyFactor: %.4f, Level: %s",
		result.CumulativeEnergy, result.SafetyFactor, result.StrengthLevel)

	if result.StrengthLevel != "safe" && result.StrengthLevel != "caution" {
		t.Errorf("light damage expected safe/caution, got %s", result.StrengthLevel)
	}

	if result.SafetyFactor < 1.5 {
		t.Errorf("light damage safety factor %.2f should be >= 1.5", result.SafetyFactor)
	}

	if result.DamageIndex < 0.8 {
		t.Errorf("light damage damage index %.2f should be >= 0.8", result.DamageIndex)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_SevereDamage(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 45000, 450, 0.3)

	t.Logf("Severe damage - CumulativeEnergy: %.0f, SafetyFactor: %.4f, Level: %s",
		result.CumulativeEnergy, result.SafetyFactor, result.StrengthLevel)

	if result.SafetyFactor >= 1.0 {
		t.Errorf("severe damage safety factor %.2f should be < 1.0", result.SafetyFactor)
	}

	if result.StrengthLevel != "warning" && result.StrengthLevel != "danger" && result.StrengthLevel != "critical" {
		t.Errorf("severe damage expected warning/danger/critical, got %s", result.StrengthLevel)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_ZeroEnergy(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 0, 450, 0.3)

	t.Logf("Zero energy - SafetyFactor: %.4f, Level: %s, DamageIndex: %.4f",
		result.SafetyFactor, result.StrengthLevel, result.DamageIndex)

	if result.DamageIndex != 1.0 {
		t.Errorf("zero energy damage index = %.4f, want 1.0", result.DamageIndex)
	}

	if result.StrengthLevel != "safe" {
		t.Errorf("zero damage should be safe, got %s", result.StrengthLevel)
	}

	expectedSF := (450.0/testRefDensity) * 1.0 * (1.0 - 0.3) * 3.0
	if math.Abs(result.SafetyFactor-expectedSF) > 1e-10 {
		t.Errorf("zero energy safety factor = %.4f, want %.4f", result.SafetyFactor, expectedSF)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_CriticalEnergy(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", testCriticalEnergy, 450, 0.3)

	t.Logf("Critical energy - SafetyFactor: %.4f, Level: %s, DamageIndex: %.4f",
		result.SafetyFactor, result.StrengthLevel, result.DamageIndex)

	if result.DamageIndex != 0.0 {
		t.Errorf("critical energy damage index = %.4f, want 0.0", result.DamageIndex)
	}

	if result.SafetyFactor != 0.0 {
		t.Errorf("critical energy safety factor = %.4f, want 0.0", result.SafetyFactor)
	}

	if result.StrengthLevel != "critical" {
		t.Errorf("critical energy should be critical level, got %s", result.StrengthLevel)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_ExceedsCriticalEnergy(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 80000, 450, 0.3)

	t.Logf("Exceeds critical - SafetyFactor: %.4f, Level: %s, DamageIndex: %.4f",
		result.SafetyFactor, result.StrengthLevel, result.DamageIndex)

	if result.DamageIndex < 0 || result.DamageIndex > 1.0 {
		t.Errorf("damage index %.4f out of [0, 1] range", result.DamageIndex)
	}

	if result.SafetyFactor < 0 {
		t.Errorf("safety factor %.4f should not be negative", result.SafetyFactor)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_LowDensity(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 10000, 250, 0.3)

	t.Logf("Low density wood - Density: %.0f, SafetyFactor: %.4f, Level: %s",
		result.WoodDensity, result.SafetyFactor, result.StrengthLevel)

	if result.ResidualStrengthIndex >= 1.0 {
		t.Errorf("low density RSI %.4f should be < 1.0", result.ResidualStrengthIndex)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_HighDensity(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 10000, 700, 0.3)

	t.Logf("High density wood - Density: %.0f, SafetyFactor: %.4f, Level: %s",
		result.WoodDensity, result.SafetyFactor, result.StrengthLevel)

	if result.SafetyFactor <= 2.0 {
		t.Errorf("high density safety factor %.2f should be > 2.0 for mild damage", result.SafetyFactor)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_BoundaryLevels(t *testing.T) {
	evaluator := newTestEvaluator()

	testCases := []struct {
		name         string
		energy       float64
		expectedMinSF float64
		expectedMaxSF float64
		expectedLevel string
	}{
		{"Safe level", 1000, 2.0, 2.1, "safe"},
		{"Caution level", 10000, 1.5, 2.0, "caution"},
		{"Warning level", 20000, 1.0, 1.5, "warning"},
		{"Danger level", 35000, 0.5, 1.0, "danger"},
		{"Critical level", 48000, 0.0, 0.5, "critical"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := evaluator.AssessStrength("S1", "应县木塔", "一层", tc.energy, 450, 0.3)

			t.Logf("%s: SF=%.4f, Level=%s", tc.name, result.SafetyFactor, result.StrengthLevel)

			if result.SafetyFactor < tc.expectedMinSF || result.SafetyFactor > tc.expectedMaxSF {
				t.Errorf("%s: SF %.4f not in [%.2f, %.2f]",
					tc.name, result.SafetyFactor, tc.expectedMinSF, tc.expectedMaxSF)
			}

			if result.StrengthLevel != tc.expectedLevel {
				t.Errorf("%s: expected level %s, got %s", tc.name, tc.expectedLevel, result.StrengthLevel)
			}
		})
	}
}

func TestWoodStrengthEvaluator_AssessStrength_DepthRatio(t *testing.T) {
	evaluator := newTestEvaluator()

	shallow := evaluator.AssessStrength("S1", "应县木塔", "一层", 20000, 450, 0.1)
	deep := evaluator.AssessStrength("S2", "应县木塔", "二层", 20000, 450, 0.5)

	t.Logf("Shallow (depth=0.1): SF=%.4f", shallow.SafetyFactor)
	t.Logf("Deep (depth=0.5): SF=%.4f", deep.SafetyFactor)

	if deep.SafetyFactor >= shallow.SafetyFactor {
		t.Error("deeper damage should have lower safety factor")
	}
}

func TestWoodStrengthEvaluator_BatchAssess(t *testing.T) {
	evaluator := newTestEvaluator()

	sensors := []SensorStrengthInput{
		{SensorID: "S1", Building: "应县木塔", Location: "一层", CumulativeEnergy: 5000, WoodDensity: 450, DepthRatio: 0.3},
		{SensorID: "S2", Building: "应县木塔", Location: "二层", CumulativeEnergy: 40000, WoodDensity: 400, DepthRatio: 0.3},
		{SensorID: "S3", Building: "应县木塔", Location: "三层", CumulativeEnergy: 25000, WoodDensity: 500, DepthRatio: 0.3},
	}

	results := evaluator.BatchAssess(sensors)

	if len(results) != len(sensors) {
		t.Fatalf("expected %d results, got %d", len(sensors), len(results))
	}

	for i, r := range results {
		if r.SensorID != sensors[i].SensorID {
			t.Errorf("result[%d] sensor_id = %s, want %s", i, r.SensorID, sensors[i].SensorID)
		}
		if r.Building != sensors[i].Building {
			t.Errorf("result[%d] building = %s, want %s", i, r.Building, sensors[i].Building)
		}
		if r.CumulativeEnergy != sensors[i].CumulativeEnergy {
			t.Errorf("result[%d] energy = %.0f, want %.0f",
				i, r.CumulativeEnergy, sensors[i].CumulativeEnergy)
		}
	}
}

func TestWoodStrengthEvaluator_BatchAssessEmpty(t *testing.T) {
	evaluator := newTestEvaluator()

	results := evaluator.BatchAssess([]SensorStrengthInput{})

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

func TestSimulateWoodDensity_Normal(t *testing.T) {
	density := SimulateWoodDensity(450, 968, 20)

	t.Logf("Simulated density: %.2f kg/m³ (base=450, age=968, moisture=20)", density)

	if density <= 200 || density >= 800 {
		t.Errorf("density %.2f out of reasonable range [200, 800]", density)
	}

	if density >= 450 {
		t.Errorf("aged wood density %.2f should be less than base density 450", density)
	}
}

func TestSimulateWoodDensity_AgeEffect(t *testing.T) {
	newWood := SimulateWoodDensity(450, 10, 12)
	oldWood := SimulateWoodDensity(450, 1000, 12)

	t.Logf("New wood (10y): %.2f, Old wood (1000y): %.2f", newWood, oldWood)

	if oldWood >= newWood {
		t.Error("older wood should have lower density")
	}
}

func TestSimulateWoodDensity_MoistureEffect(t *testing.T) {
	dryWood := SimulateWoodDensity(450, 100, 8)
	wetWood := SimulateWoodDensity(450, 100, 25)

	t.Logf("Dry wood (8%%): %.2f, Wet wood (25%%): %.2f", dryWood, wetWood)

	if wetWood <= dryWood {
		t.Error("wetter wood should have higher density")
	}
}

func TestSimulateWoodDensity_BoundaryClamping(t *testing.T) {
	veryLow := SimulateWoodDensity(100, 2000, 5)
	veryHigh := SimulateWoodDensity(900, 10, 30)

	t.Logf("Very low input result: %.2f, Very high input result: %.2f", veryLow, veryHigh)

	if veryLow < 200 {
		t.Errorf("density %.2f should be clamped to minimum 200", veryLow)
	}
	if veryHigh > 800 {
		t.Errorf("density %.2f should be clamped to maximum 800", veryHigh)
	}
}

func TestWoodStrengthAssessment_FieldConsistency(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("SENSOR-001", "佛光寺", "斗拱", 15000, 480, 0.25)

	if result.SensorID != "SENSOR-001" {
		t.Errorf("SensorID = %s, want SENSOR-001", result.SensorID)
	}
	if result.Building != "佛光寺" {
		t.Errorf("Building = %s, want 佛光寺", result.Building)
	}
	if result.Location != "斗拱" {
		t.Errorf("Location = %s, want 斗拱", result.Location)
	}
	if result.WoodDensity != 480 {
		t.Errorf("WoodDensity = %.0f, want 480", result.WoodDensity)
	}
	if result.CumulativeEnergy != 15000 {
		t.Errorf("CumulativeEnergy = %.0f, want 15000", result.CumulativeEnergy)
	}

	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestWoodStrengthEvaluator_AssessStrength_NegativeEnergy(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", -1000, 450, 0.3)

	t.Logf("Negative energy - DamageIndex: %.4f, SafetyFactor: %.4f",
		result.DamageIndex, result.SafetyFactor)

	if result.DamageIndex > 1.0 || result.DamageIndex < 0 {
		t.Errorf("negative energy damage index %.4f should be clamped to [0,1]", result.DamageIndex)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_ZeroDensity(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 10000, 0, 0.3)

	t.Logf("Zero density - SafetyFactor: %.4f, RSI: %.4f",
		result.SafetyFactor, result.ResidualStrengthIndex)

	if result.SafetyFactor != 0 {
		t.Errorf("zero density safety factor = %.4f, want 0", result.SafetyFactor)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_MaxDepthRatio(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", 0, 450, 1.0)

	t.Logf("Depth ratio 1.0 - SafetyFactor: %.4f", result.SafetyFactor)

	if result.SafetyFactor != 0 {
		t.Errorf("full depth damage safety factor = %.4f, want 0", result.SafetyFactor)
	}
}

func TestWoodStrengthEvaluator_AllLevelsPresent(t *testing.T) {
	evaluator := newTestEvaluator()

	levels := make(map[string]bool)
	for energy := 0.0; energy <= testCriticalEnergy; energy += 1000 {
		result := evaluator.AssessStrength("S1", "test", "test", energy, 450, 0.3)
		levels[result.StrengthLevel] = true
	}

	expectedLevels := []string{"safe", "caution", "warning", "danger", "critical"}
	for _, level := range expectedLevels {
		if !levels[level] {
			t.Errorf("level %s not present in energy sweep", level)
		}
	}

	t.Logf("All levels present: safe=%v caution=%v warning=%v danger=%v critical=%v",
		levels["safe"], levels["caution"], levels["warning"], levels["danger"], levels["critical"])
}

var _ = models.WoodStrengthAssessment{}
