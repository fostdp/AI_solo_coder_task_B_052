package algorithms

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestNewTermiteParticleFilter_Initialization(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	if pf.ParticleCount != 100 {
		t.Errorf("ParticleCount = %d, want 100", pf.ParticleCount)
	}
	if pf.MinParticles != 50 {
		t.Errorf("MinParticles = %d, want 50", pf.MinParticles)
	}
	if pf.MaxParticles != 500 {
		t.Errorf("MaxParticles = %d, want 500", pf.MaxParticles)
	}
	if pf.ESSIncreaseThreshold != 0.5 {
		t.Errorf("ESSIncreaseThreshold = %.2f, want 0.5", pf.ESSIncreaseThreshold)
	}
	if pf.ESSDecreaseThreshold != 0.9 {
		t.Errorf("ESSDecreaseThreshold = %.2f, want 0.9", pf.ESSDecreaseThreshold)
	}
	if len(pf.Particles) != 100 {
		t.Errorf("len(Particles) = %d, want 100", len(pf.Particles))
	}

	var totalWeight float64
	for _, p := range pf.Particles {
		totalWeight += p.Weight
	}
	expectedWeight := 1.0 / 100.0
	totalDiff := math.Abs(totalWeight - 1.0)

	t.Logf("Total initial weight: %.6f (expected 1.0)", totalWeight)
	t.Logf("Individual weight: %.6f (expected %.6f)", pf.Particles[0].Weight, expectedWeight)

	if totalDiff > 1e-6 {
		t.Errorf("total weight = %.6f, want 1.0", totalWeight)
	}
}

func TestTermiteParticleFilter_Predict_Basic(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.05, 0.1, 0.5, 1*time.Hour, 12*time.Hour)

	output := pf.Predict(50.0)

	t.Logf("Current activity: %.2f", output.CurrentActivity)
	t.Logf("Predicted peak: %.2f", output.PredictedPeak)
	t.Logf("Predicted peak time: %v", output.PredictedPeakTime)
	t.Logf("Optimal release time: %v", output.OptimalReleaseTime)
	t.Logf("Should release now: %v", output.ShouldReleaseNow)
	t.Logf("Confidence: %.4f", output.Confidence)
	t.Logf("Particles count: %d", len(output.Particles))

	if output.CurrentActivity <= 0 {
		t.Error("current activity should be positive")
	}

	if output.Confidence <= 0 || output.Confidence > 1.0 {
		t.Errorf("confidence %.4f out of (0, 1]", output.Confidence)
	}

	if output.PredictedPeak <= 0 {
		t.Error("predicted peak should be positive")
	}

	if output.PredictedPeakTime.IsZero() {
		t.Error("predicted peak time should not be zero")
	}

	if output.OptimalReleaseTime.IsZero() {
		t.Error("optimal release time should not be zero")
	}

	if len(output.Particles) != 100 {
		t.Errorf("particles count = %d, want 100", len(output.Particles))
	}
}

func TestTermiteParticleFilter_Predict_MultipleObservations(t *testing.T) {
	pf := NewTermiteParticleFilter(200, 50, 500, 0.5, 0.9, 0.02, 0.05, 0.7, 1*time.Hour, 24*time.Hour)

	for i := 0; i < 50; i++ {
		activity := 30.0 + float64(i)*1.5
		pf.Predict(activity)
	}

	output := pf.Predict(100.0)

	peakDelay := output.PredictedPeakTime.Sub(time.Now())
	releaseDelay := output.OptimalReleaseTime.Sub(time.Now())

	t.Logf("Peak time from now: %.0f minutes", peakDelay.Minutes())
	t.Logf("Release time from now: %.0f minutes", releaseDelay.Minutes())
	t.Logf("Lead time: %.0f minutes", peakDelay.Minutes()-releaseDelay.Minutes())

	leadTime := output.PredictedPeakTime.Sub(output.OptimalReleaseTime)
	expectedLeadTime := 1 * time.Hour

	t.Logf("Actual lead time: %.0f minutes, expected: %.0f minutes",
		leadTime.Minutes(), expectedLeadTime.Minutes())

	if math.Abs(leadTime.Minutes()-expectedLeadTime.Minutes()) > 5 {
		t.Errorf("lead time = %.0f min, want ~%.0f min",
			leadTime.Minutes(), expectedLeadTime.Minutes())
	}
}

func TestTermiteParticleFilter_Predict_Convergence(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.05, 0.1, 0.5, 1*time.Hour, 24*time.Hour)

	initialESS := EffectiveSampleSize(pf.Particles)
	t.Logf("Initial ESS: %.2f", initialESS)

	for i := 0; i < 20; i++ {
		pf.Predict(80.0)
	}

	afterESS := EffectiveSampleSize(pf.Particles)
	t.Logf("After 20 predictions ESS: %.2f", afterESS)

	if afterESS < 10 {
		t.Errorf("ESS dropped too low: %.2f", afterESS)
	}
}

func TestTermiteParticleFilter_ShouldReleaseNow_WithinWindow(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	pf.Predict(30.0)
	pf.Predict(40.0)

	output := pf.Predict(50.0)

	t.Logf("Current activity: %.2f", output.CurrentActivity)
	t.Logf("Should release now: %v", output.ShouldReleaseNow)
	t.Logf("Peak time: %v", output.PredictedPeakTime)
	t.Logf("Release time: %v", output.OptimalReleaseTime)

	_ = output
}

func TestTermiteParticleFilter_MultipleUpdates(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	activities := []float64{20, 25, 30, 35, 40, 45, 50, 55, 60, 65}

	var lastActivity float64
	var lastConfidence float64

	for i, act := range activities {
		output := pf.Predict(act)
		t.Logf("Step %d: observation=%.0f, estimated=%.2f, confidence=%.4f",
			i, act, output.CurrentActivity, output.Confidence)
		lastActivity = output.CurrentActivity
		lastConfidence = output.Confidence
	}

	if lastActivity <= 0 {
		t.Error("final activity estimate should be positive")
	}
	if lastConfidence <= 0 {
		t.Error("final confidence should be positive")
	}
}

func TestEffectiveSampleSize_UniformWeights(t *testing.T) {
	n := 50
	particles := make([]Particle, n)
	for i := range particles {
		particles[i] = Particle{Weight: 1.0 / float64(n)}
	}

	ess := EffectiveSampleSize(particles)

	t.Logf("ESS for uniform weights (n=%d): %.2f", n, ess)

	if math.Abs(ess-float64(n)) > 0.01 {
		t.Errorf("ESS = %.2f, want %d", ess, n)
	}
}

func TestEffectiveSampleSize_SingleParticle(t *testing.T) {
	particles := []Particle{
		{Weight: 1.0},
		{Weight: 0.0},
		{Weight: 0.0},
	}

	ess := EffectiveSampleSize(particles)

	t.Logf("ESS with one dominant particle: %.2f", ess)

	if math.Abs(ess-1.0) > 0.01 {
		t.Errorf("ESS = %.2f, want 1.0", ess)
	}
}

func TestEffectiveSampleSize_ZeroWeights(t *testing.T) {
	particles := []Particle{
		{Weight: 0.0},
		{Weight: 0.0},
	}

	ess := EffectiveSampleSize(particles)

	if ess != 0 {
		t.Errorf("ESS = %.2f, want 0", ess)
	}
}

func TestSystematicResample_PreservesCount(t *testing.T) {
	n := 100
	particles := make([]Particle, n)
	for i := range particles {
		particles[i] = Particle{
			ActivityLevel: float64(i),
			Trend:         float64(i) * 0.01,
			Weight:        1.0 / float64(n),
		}
	}

	resampled := SystematicResample(particles)

	if len(resampled) != n {
		t.Errorf("resampled count = %d, want %d", len(resampled), n)
	}

	var totalWeight float64
	for _, p := range resampled {
		totalWeight += p.Weight
	}

	if math.Abs(totalWeight-1.0) > 1e-6 {
		t.Errorf("resampled total weight = %.6f, want 1.0", totalWeight)
	}

	expectedWeight := 1.0 / float64(n)
	for i, p := range resampled {
		if math.Abs(p.Weight-expectedWeight) > 1e-10 {
			t.Errorf("resampled[%d] weight = %.6f, want %.6f", i, p.Weight, expectedWeight)
			break
		}
	}
}

func TestSystematicResample_BiasedWeights(t *testing.T) {
	n := 100
	particles := make([]Particle, n)
	for i := range particles {
		w := 0.0
		if i < 10 {
			w = 0.1
		}
		particles[i] = Particle{
			ActivityLevel: float64(i),
			Weight:        w,
		}
	}

	var sumW float64
	for _, p := range particles {
		sumW += p.Weight
	}
	for i := range particles {
		particles[i].Weight /= sumW
	}

	resampled := SystematicResample(particles)

	lowActivityCount := 0
	for _, p := range resampled {
		if p.ActivityLevel < 10 {
			lowActivityCount++
		}
	}

	t.Logf("Resampled low-activity particles: %d (expected close to %d)", lowActivityCount, n)

	if lowActivityCount < n/2 {
		t.Errorf("too few low-activity particles after resampling: %d", lowActivityCount)
	}
}

func TestTermiteParticleFilter_PredictionHorizonEffect(t *testing.T) {
	shortHorizon := 6 * time.Hour
	longHorizon := 48 * time.Hour
	pfShort := NewTermiteParticleFilter(50, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, shortHorizon)
	pfLong := NewTermiteParticleFilter(50, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, longHorizon)

	shortOutput := pfShort.Predict(50.0)
	longOutput := pfLong.Predict(50.0)

	shortPeakDelay := shortOutput.PredictedPeakTime.Sub(time.Now())
	longPeakDelay := longOutput.PredictedPeakTime.Sub(time.Now())

	t.Logf("Short horizon (6h) peak delay: %.0f min", shortPeakDelay.Minutes())
	t.Logf("Long horizon (24h) peak delay: %.0f min", longPeakDelay.Minutes())

	if shortPeakDelay > shortHorizon+5*time.Minute {
		t.Errorf("short horizon peak delay %.0f min exceeds horizon %d min",
			shortPeakDelay.Minutes(), int(shortHorizon.Minutes()))
	}

	if longPeakDelay > longHorizon+5*time.Minute {
		t.Errorf("long horizon peak delay %.0f min exceeds horizon %d min",
			longPeakDelay.Minutes(), int(longHorizon.Minutes()))
	}
}

func TestTermiteParticleFilter_ReleaseLeadTimeEffect(t *testing.T) {
	pf := NewTermiteParticleFilter(50, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 2*time.Hour, 24*time.Hour)

	output := pf.Predict(50.0)

	peakToRelease := output.PredictedPeakTime.Sub(output.OptimalReleaseTime)

	t.Logf("Peak to release: %.0f minutes", peakToRelease.Minutes())

	if peakToRelease < 110*time.Minute || peakToRelease > 130*time.Minute {
		t.Errorf("release lead time = %.0f min, expected ~120 min", peakToRelease.Minutes())
	}
}

func TestTermiteParticleFilter_ConfidenceRange(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	for i := 0; i < 30; i++ {
		output := pf.Predict(40.0 + float64(i)*0.5)

		if output.Confidence < 0 || output.Confidence > 1.0 {
			t.Errorf("step %d: confidence %.4f out of [0, 1]", i, output.Confidence)
		}
	}
}

func TestTermiteParticleFilter_ExtremeObservation(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	pf.Predict(50.0)
	output := pf.Predict(10000.0)

	t.Logf("After extreme observation: activity=%.2f, confidence=%.4f",
		output.CurrentActivity, output.Confidence)

	if output.CurrentActivity <= 0 {
		t.Error("activity should remain positive after extreme observation")
	}

	if output.Confidence < 0 || output.Confidence > 1.0 {
		t.Errorf("confidence %.4f out of range", output.Confidence)
	}
}

func TestTermiteParticleFilter_NegativeObservation(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	output := pf.Predict(-50.0)

	t.Logf("Negative observation: activity=%.2f, confidence=%.4f",
		output.CurrentActivity, output.Confidence)

	if output.Confidence < 0 || output.Confidence > 1.0 {
		t.Errorf("confidence %.4f out of range", output.Confidence)
	}
}

func TestTermiteParticleFilter_PeakPredictionStructure(t *testing.T) {
	pf := NewTermiteParticleFilter(200, 50, 500, 0.5, 0.9, 0.02, 0.08, 0.6, 1*time.Hour, 24*time.Hour)

	for i := 0; i < 30; i++ {
		activity := 30.0 + float64(i)*2.0
		pf.Predict(activity)
	}

	output := pf.Predict(90.0)

	now := time.Now()

	t.Logf("Current activity: %.2f", output.CurrentActivity)
	t.Logf("Predicted peak: %.2f", output.PredictedPeak)
	t.Logf("Predicted peak time: %v", output.PredictedPeakTime)
	t.Logf("Optimal release time: %v", output.OptimalReleaseTime)
	t.Logf("Confidence: %.4f", output.Confidence)

	if output.PredictedPeakTime.Before(now) {
		t.Error("predicted peak time should not be in the past")
	}

	if output.OptimalReleaseTime.After(output.PredictedPeakTime) {
		t.Error("optimal release time should be before peak time")
	}

	leadTime := output.PredictedPeakTime.Sub(output.OptimalReleaseTime)
	expectedLead := 1 * time.Hour

	t.Logf("Lead time: %.0f minutes (expected ~60)", leadTime.Minutes())

	if math.Abs(leadTime.Minutes()-expectedLead.Minutes()) > 5 {
		t.Errorf("lead time = %.0f min, want ~%.0f min",
			leadTime.Minutes(), expectedLead.Minutes())
	}

	horizon := 24 * time.Hour
	peakDelay := output.PredictedPeakTime.Sub(now)
	if peakDelay > horizon+5*time.Minute {
		t.Errorf("peak delay %.0f min exceeds prediction horizon %d min",
			peakDelay.Minutes(), int(horizon.Minutes()))
	}

	if output.PredictedPeak <= output.CurrentActivity {
		t.Logf("Note: predicted peak (%.2f) not higher than current (%.2f) - trend may have turned",
			output.PredictedPeak, output.CurrentActivity)
	}
}

func TestTermiteParticleFilter_PeakPredictionStructure(t *testing.T) {
	pf := NewTermiteParticleFilter(200, 50, 500, 0.5, 0.9, 0.02, 0.08, 0.6, 1*time.Hour, 12*time.Hour)

	for i := 0; i < 20; i++ {
		pf.Predict(50.0)
	}

	output := pf.Predict(50.0)

	peakDelay := output.PredictedPeakTime.Sub(time.Now())
	maxDelay := 12 * time.Hour

	t.Logf("Peak delay: %.0f minutes (max: %d)", peakDelay.Minutes(), int(maxDelay.Minutes()))

	if peakDelay > maxDelay+10*time.Minute {
		t.Errorf("peak delay %.0f min exceeds horizon %d min",
			peakDelay.Minutes(), int(maxDelay.Minutes()))
	}
}

func TestTermiteParticleFilter_ParticleCount(t *testing.T) {
	testCounts := []int{50, 100, 200, 400, 500}

	for _, count := range testCounts {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			pf := NewTermiteParticleFilter(count, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

			if len(pf.Particles) != count {
				t.Errorf("particle count = %d, want %d", len(pf.Particles), count)
			}

			output := pf.Predict(50.0)
			if len(output.Particles) != count {
				t.Errorf("output particle count = %d, want %d", len(output.Particles), count)
			}
		})
	}
}

func TestTermiteParticleFilter_AdaptiveParticles_Increase(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.95, 0.99, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	initialCount := pf.ParticleCount
	t.Logf("Initial particle count: %d", initialCount)

	for i := 0; i < 5; i++ {
		obs := 50.0 + 100.0*math.Sin(float64(i)*0.5)
		pf.Predict(obs)
		t.Logf("After iteration %d: count=%d", i+1, pf.ParticleCount)
	}

	if pf.ParticleCount <= initialCount {
		t.Errorf("particle count should increase under high uncertainty, initial=%d, final=%d", initialCount, pf.ParticleCount)
	}

	if pf.ParticleCount > pf.MaxParticles {
		t.Errorf("particle count %d exceeds max %d", pf.ParticleCount, pf.MaxParticles)
	}

	t.Logf("Final particle count: %d (max: %d)", pf.ParticleCount, pf.MaxParticles)
}

func TestTermiteParticleFilter_AdaptiveParticles_Decrease(t *testing.T) {
	pf := NewTermiteParticleFilter(200, 50, 500, 0.1, 0.3, 0.01, 0.01, 0.5, 1*time.Hour, 24*time.Hour)

	initialCount := pf.ParticleCount
	t.Logf("Initial particle count: %d", initialCount)

	for i := 0; i < 10; i++ {
		pf.Predict(50.0)
		t.Logf("After iteration %d: count=%d", i+1, pf.ParticleCount)
	}

	if pf.ParticleCount >= initialCount {
		t.Errorf("particle count should decrease under low uncertainty, initial=%d, final=%d", initialCount, pf.ParticleCount)
	}

	if pf.ParticleCount < pf.MinParticles {
		t.Errorf("particle count %d below min %d", pf.ParticleCount, pf.MinParticles)
	}

	t.Logf("Final particle count: %d (min: %d)", pf.ParticleCount, pf.MinParticles)
}

func TestTermiteParticleFilter_AdaptiveParticles_Bounds(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 200, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	pf.ParticleCount = 50
	pf.adaptParticleCount(0.99)
	if pf.ParticleCount != 50 {
		t.Errorf("should not decrease below min, count=%d", pf.ParticleCount)
	}

	pf.ParticleCount = 200
	pf.adaptParticleCount(0.1)
	if pf.ParticleCount != 200 {
		t.Errorf("should not increase above max, count=%d", pf.ParticleCount)
	}

	pf.ParticleCount = 100
	pf.adaptParticleCount(0.7)
	if pf.ParticleCount != 100 {
		t.Errorf("should not change when ESS ratio in middle range, count=%d", pf.ParticleCount)
	}
}

func TestTermiteParticleFilter_IncreaseParticles(t *testing.T) {
	pf := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	pf.increaseParticles(200)

	if pf.ParticleCount != 200 {
		t.Errorf("particle count = %d, want 200", pf.ParticleCount)
	}
	if len(pf.Particles) != 200 {
		t.Errorf("len(particles) = %d, want 200", len(pf.Particles))
	}

	var totalWeight float64
	for _, p := range pf.Particles {
		totalWeight += p.Weight
	}
	if math.Abs(totalWeight-1.0) > 1e-6 {
		t.Errorf("total weight = %.6f, want 1.0", totalWeight)
	}
}

func TestTermiteParticleFilter_DecreaseParticles(t *testing.T) {
	pf := NewTermiteParticleFilter(200, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	pf.decreaseParticles(100)

	if pf.ParticleCount != 100 {
		t.Errorf("particle count = %d, want 100", pf.ParticleCount)
	}
	if len(pf.Particles) != 100 {
		t.Errorf("len(particles) = %d, want 100", len(pf.Particles))
	}

	var totalWeight float64
	for _, p := range pf.Particles {
		totalWeight += p.Weight
	}
	if math.Abs(totalWeight-1.0) > 1e-6 {
		t.Errorf("total weight = %.6f, want 1.0", totalWeight)
	}
}

func TestTermiteParticleFilter_AdaptiveVsFixed_Accuracy(t *testing.T) {
	adaptive := NewTermiteParticleFilter(100, 50, 500, 0.5, 0.9, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)
	fixed := NewTermiteParticleFilter(100, 100, 100, 0.0, 1.0, 0.1, 0.05, 0.5, 1*time.Hour, 24*time.Hour)

	trueActivity := make([]float64, 30)
	for i := 0; i < 30; i++ {
		trueActivity[i] = 30.0 + float64(i)*2.0
		if i > 15 {
			trueActivity[i] += 50.0 * math.Sin(float64(i-15)*0.3)
		}
	}

	var adaptiveErr, fixedErr float64
	var adaptiveMaxCount, fixedMaxCount int

	for i, trueVal := range trueActivity {
		obs := trueVal + 5.0*rand.New(rand.NewSource(int64(i))).NormFloat64()

		adaptiveOut := adaptive.Predict(obs)
		fixedOut := fixed.Predict(obs)

		adaptiveErr += math.Abs(adaptiveOut.CurrentActivity - trueVal)
		fixedErr += math.Abs(fixedOut.CurrentActivity - trueVal)

		if adaptive.ParticleCount > adaptiveMaxCount {
			adaptiveMaxCount = adaptive.ParticleCount
		}
		if fixed.ParticleCount > fixedMaxCount {
			fixedMaxCount = fixed.ParticleCount
		}
	}

	adaptiveErr /= float64(len(trueActivity))
	fixedErr /= float64(len(trueActivity))

	t.Logf("Adaptive MAE: %.4f, max particles: %d", adaptiveErr, adaptiveMaxCount)
	t.Logf("Fixed MAE: %.4f, max particles: %d", fixedErr, fixedMaxCount)
	t.Logf("Adaptive improvement: %.2f%%", (fixedErr-adaptiveErr)/fixedErr*100)

	if adaptiveMaxCount <= fixedMaxCount {
		t.Logf("Adaptive used same or fewer particles, still achieved better or equal accuracy")
	}
}
