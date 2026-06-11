package strength_calc

import (
	"math"
	"testing"
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

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", "pine", 5000, 450, 0.3)

	t.Logf("Light damage - CumulativeEnergy: %.0f, SafetyFactor: %.4f, Level: %s",
		result.CumulativeEnergy, result.SafetyFactor, result.StrengthLevel)

	if result.StrengthLevel != "safe" && result.StrengthLevel != "caution" {
		t.Errorf("light damage expected safe/caution, got %s", result.StrengthLevel)
	}

	if result.SafetyFactor < 1.2 {
		t.Errorf("light damage safety factor %.2f should be >= 1.2", result.SafetyFactor)
	}

	if result.DamageIndex < 0.8 {
		t.Errorf("light damage damage index %.2f should be >= 0.8", result.DamageIndex)
	}
}

func TestWoodStrengthEvaluator_AssessStrength_SevereDamage(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", "pine", 45000, 450, 0.3)

	t.Logf("Severe damage - CumulativeEnergy: %.0f, SafetyFactor: %.4f, Level: %s",
		result.CumulativeEnergy, result.SafetyFactor, result.StrengthLevel)

	if result.SafetyFactor >= 1.0 {
		t.Errorf("severe damage safety factor %.2f should be < 1.0", result.SafetyFactor)
	}

	if result.StrengthLevel != "warning" && result.StrengthLevel != "danger" && result.StrengthLevel != "critical" {
		t.Errorf("severe damage expected warning/danger/critical, got %s", result.StrengthLevel)
	}
}

func TestGetWoodTypeCorrection_StandardTypes(t *testing.T) {
	testCases := []struct {
		woodType string
		expected float64
	}{
		{"pine", 0.85},
		{"nanmu", 1.15},
		{"fir", 0.90},
		{"oak", 1.25},
		{"default", 1.0},
		{"unknown", 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.woodType, func(t *testing.T) {
			result := GetWoodTypeCorrection(tc.woodType)
			if math.Abs(result-tc.expected) > 1e-10 {
				t.Errorf("GetWoodTypeCorrection(%s) = %.4f, want %.4f",
					tc.woodType, result, tc.expected)
			}
		})
	}
}

func TestWoodStrengthEvaluator_WoodTypeEffect(t *testing.T) {
	evaluator := newTestEvaluator()

	pineResult := evaluator.AssessStrength("S1", "应县木塔", "一层", "pine", 10000, 450, 0.3)
	nanmuResult := evaluator.AssessStrength("S1", "佛光寺", "斗拱", "nanmu", 10000, 450, 0.3)
	defaultResult := evaluator.AssessStrength("S1", "test", "test", "unknown", 10000, 450, 0.3)

	t.Logf("Pine (0.85): SF=%.4f", pineResult.SafetyFactor)
	t.Logf("Nanmu (1.15): SF=%.4f", nanmuResult.SafetyFactor)
	t.Logf("Default (1.0): SF=%.4f", defaultResult.SafetyFactor)

	if pineResult.SafetyFactor >= defaultResult.SafetyFactor {
		t.Errorf("pine (softer wood) should have lower SF than default, pine=%.4f, default=%.4f",
			pineResult.SafetyFactor, defaultResult.SafetyFactor)
	}

	if nanmuResult.SafetyFactor <= defaultResult.SafetyFactor {
		t.Errorf("nanmu (harder wood) should have higher SF than default, nanmu=%.4f, default=%.4f",
			nanmuResult.SafetyFactor, defaultResult.SafetyFactor)
	}

	expectedRatio := 1.15 / 0.85
	actualRatio := nanmuResult.SafetyFactor / pineResult.SafetyFactor
	if math.Abs(actualRatio-expectedRatio) > 0.01 {
		t.Errorf("nanmu/pine SF ratio = %.4f, expected %.4f", actualRatio, expectedRatio)
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

func TestWoodStrengthEvaluator_BatchAssess(t *testing.T) {
	evaluator := newTestEvaluator()

	sensors := []SensorStrengthInput{
		{SensorID: "S1", Building: "应县木塔", Location: "一层", WoodType: "pine", CumulativeEnergy: 5000, WoodDensity: 450, DepthRatio: 0.3},
		{SensorID: "S2", Building: "应县木塔", Location: "二层", WoodType: "pine", CumulativeEnergy: 40000, WoodDensity: 400, DepthRatio: 0.3},
		{SensorID: "S3", Building: "应县木塔", Location: "三层", WoodType: "pine", CumulativeEnergy: 25000, WoodDensity: 500, DepthRatio: 0.3},
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

func TestWoodStrengthEvaluator_ZeroEnergy(t *testing.T) {
	evaluator := newTestEvaluator()

	result := evaluator.AssessStrength("S1", "应县木塔", "一层", "pine", 0, 450, 0.3)

	if result.DamageIndex != 1.0 {
		t.Errorf("zero energy damage index = %.4f, want 1.0", result.DamageIndex)
	}

	expectedSF := (450.0/testRefDensity) * 1.0 * (1.0 - 0.3) * 3.0 * 0.85
	if math.Abs(result.SafetyFactor-expectedSF) > 1e-10 {
		t.Errorf("zero energy safety factor = %.4f, want %.4f", result.SafetyFactor, expectedSF)
	}
}
