package particle_filter_timing

import (
	"context"
	"math"
	"testing"
	"time"
)

func newTestFilter() *TermiteParticleFilter {
	return NewTermiteParticleFilter(
		100, 50, 500,
		0.3, 0.7,
		2.0, 5.0, 0.5,
		2*time.Hour, 8*time.Hour,
	)
}

func TestTermiteParticleFilter_NewFilter(t *testing.T) {
	filter := newTestFilter()

	if filter.ParticleCount != 100 {
		t.Errorf("expected 100 particles, got %d", filter.ParticleCount)
	}

	if len(filter.Particles) != 100 {
		t.Errorf("expected 100 particles in slice, got %d", len(filter.Particles))
	}

	if filter.MinParticles != 50 {
		t.Errorf("expected MinParticles=50, got %d", filter.MinParticles)
	}

	if filter.MaxParticles != 500 {
		t.Errorf("expected MaxParticles=500, got %d", filter.MaxParticles)
	}

	var sumWeight float64
	for _, p := range filter.Particles {
		sumWeight += p.Weight
	}

	if math.Abs(sumWeight-1.0) > 1e-10 {
		t.Errorf("initial particle weights should sum to 1.0, got %.6f", sumWeight)
	}
}

func TestTermiteParticleFilter_PredictSync_Basic(t *testing.T) {
	filter := newTestFilter()

	output := filter.PredictSync(100.0)

	if output.CurrentActivity <= 0 {
		t.Errorf("expected positive current activity, got %.2f", output.CurrentActivity)
	}

	if output.PredictedPeak <= 0 {
		t.Errorf("expected positive predicted peak, got %.2f", output.PredictedPeak)
	}

	if output.Confidence <= 0 || output.Confidence > 1.0 {
		t.Errorf("confidence should be in (0, 1], got %.4f", output.Confidence)
	}

	if output.PredictedPeakTime.Before(time.Now()) {
		t.Errorf("predicted peak time should be in the future")
	}

	if output.OptimalReleaseTime.After(output.PredictedPeakTime) {
		t.Errorf("optimal release time should be before peak time")
	}
}

func TestEffectiveSampleSize(t *testing.T) {
	particles := []Particle{
		{Weight: 0.5},
		{Weight: 0.5},
	}
	ess := EffectiveSampleSize(particles)
	expected := 2.0
	if math.Abs(ess-expected) > 1e-10 {
		t.Errorf("ESS for two equal weights = %.4f, want %.4f", ess, expected)
	}

	particles = []Particle{
		{Weight: 1.0},
		{Weight: 0.0},
	}
	ess = EffectiveSampleSize(particles)
	expected = 1.0
	if math.Abs(ess-expected) > 1e-10 {
		t.Errorf("ESS for degenerate case = %.4f, want %.4f", ess, expected)
	}

	particles = make([]Particle, 100)
	for i := range particles {
		particles[i].Weight = 0.01
	}
	ess = EffectiveSampleSize(particles)
	expected = 100.0
	if math.Abs(ess-expected) > 0.1 {
		t.Errorf("ESS for uniform weights = %.4f, want %.4f", ess, expected)
	}
}

func TestSystematicResample(t *testing.T) {
	particles := []Particle{
		{ActivityLevel: 10.0, Trend: 0.5, Weight: 0.5},
		{ActivityLevel: 20.0, Trend: 1.0, Weight: 0.5},
	}

	resampled := SystematicResample(particles)

	if len(resampled) != 2 {
		t.Fatalf("expected 2 resampled particles, got %d", len(resampled))
	}

	for _, p := range resampled {
		if math.Abs(p.Weight-0.5) > 1e-10 {
			t.Errorf("resampled weight should be 0.5, got %.4f", p.Weight)
		}
	}
}

func TestTermiteParticleFilter_IncreaseParticles(t *testing.T) {
	filter := newTestFilter()

	originalCount := filter.ParticleCount
	targetCount := originalCount * 2

	filter.increaseParticles(targetCount)

	if filter.ParticleCount != targetCount {
		t.Errorf("particle count = %d, want %d", filter.ParticleCount, targetCount)
	}

	if len(filter.Particles) != targetCount {
		t.Errorf("particle slice length = %d, want %d", len(filter.Particles), targetCount)
	}

	var sumWeight float64
	for _, p := range filter.Particles {
		sumWeight += p.Weight
	}

	if math.Abs(sumWeight-1.0) > 1e-6 {
		t.Errorf("weights after increase should sum to 1.0, got %.6f", sumWeight)
	}
}

func TestTermiteParticleFilter_DecreaseParticles(t *testing.T) {
	filter := NewTermiteParticleFilter(
		200, 50, 500,
		0.3, 0.7,
		2.0, 5.0, 0.5,
		2*time.Hour, 8*time.Hour,
	)

	originalCount := filter.ParticleCount
	targetCount := originalCount / 2

	filter.decreaseParticles(targetCount)

	if filter.ParticleCount != targetCount {
		t.Errorf("particle count = %d, want %d", filter.ParticleCount, targetCount)
	}

	if len(filter.Particles) != targetCount {
		t.Errorf("particle slice length = %d, want %d", len(filter.Particles), targetCount)
	}

	var sumWeight float64
	for _, p := range filter.Particles {
		sumWeight += p.Weight
	}

	if math.Abs(sumWeight-1.0) > 1e-6 {
		t.Errorf("weights after decrease should sum to 1.0, got %.6f", sumWeight)
	}
}

func TestTermiteParticleFilter_AdaptParticleCount_Increase(t *testing.T) {
	filter := newTestFilter()
	originalCount := filter.ParticleCount

	filter.adaptParticleCount(0.1)

	if filter.ParticleCount <= originalCount {
		t.Errorf("particle count should increase when ESS ratio is low, got %d (was %d)",
			filter.ParticleCount, originalCount)
	}
}

func TestTermiteParticleFilter_AdaptParticleCount_Decrease(t *testing.T) {
	filter := NewTermiteParticleFilter(
		200, 50, 500,
		0.3, 0.7,
		2.0, 5.0, 0.5,
		2*time.Hour, 8*time.Hour,
	)
	originalCount := filter.ParticleCount

	filter.adaptParticleCount(0.9)

	if filter.ParticleCount >= originalCount {
		t.Errorf("particle count should decrease when ESS ratio is high, got %d (was %d)",
			filter.ParticleCount, originalCount)
	}
}

func TestTermiteParticleFilter_AdaptParticleCount_Boundaries(t *testing.T) {
	filter := NewTermiteParticleFilter(
		50, 50, 500,
		0.3, 0.7,
		2.0, 5.0, 0.5,
		2*time.Hour, 8*time.Hour,
	)

	filter.adaptParticleCount(0.9)
	if filter.ParticleCount != 50 {
		t.Errorf("should not decrease below MinParticles=50, got %d", filter.ParticleCount)
	}

	filter2 := NewTermiteParticleFilter(
		500, 50, 500,
		0.3, 0.7,
		2.0, 5.0, 0.5,
		2*time.Hour, 8*time.Hour,
	)

	filter2.adaptParticleCount(0.1)
	if filter2.ParticleCount != 500 {
		t.Errorf("should not increase above MaxParticles=500, got %d", filter2.ParticleCount)
	}
}

func TestTermiteParticleFilter_AdaptParticleCount_NoChange(t *testing.T) {
	filter := newTestFilter()
	originalCount := filter.ParticleCount

	filter.adaptParticleCount(0.5)

	if filter.ParticleCount != originalCount {
		t.Errorf("particle count should not change for middle ESS ratio, got %d (was %d)",
			filter.ParticleCount, originalCount)
	}
}

func TestTermiteParticleFilter_Goroutine_StartStop(t *testing.T) {
	filter := newTestFilter()

	ctx := context.Background()
	filter.Start(ctx)

	if !filter.running {
		t.Error("filter should be running after Start()")
	}

	filter.Stop()

	if filter.running {
		t.Error("filter should not be running after Stop()")
	}
}

func TestTermiteParticleFilter_Goroutine_PredictAsync(t *testing.T) {
	filter := newTestFilter()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filter.Start(ctx)
	defer filter.Stop()

	req := PredictRequest{
		Building:    "应县木塔",
		Observation: 100.0,
		Timestamp:   time.Now(),
	}

	err := filter.PredictAsync(req)
	if err != nil {
		t.Fatalf("PredictAsync failed: %v", err)
	}

	select {
	case resp, ok := <-filter.Response():
		if !ok {
			t.Fatal("response channel closed")
		}
		if resp.Err != nil {
			t.Errorf("got error in response: %v", resp.Err)
		}
		if resp.Output.Building != "应县木塔" {
			t.Errorf("building = %s, want 应县木塔", resp.Output.Building)
		}
		if resp.Output.CurrentActivity <= 0 {
			t.Errorf("expected positive current activity, got %.2f", resp.Output.CurrentActivity)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for response")
	}
}

func TestTermiteParticleFilter_ConsecutivePredictions(t *testing.T) {
	filter := newTestFilter()

	observations := []float64{50, 55, 60, 65, 70, 75, 80}

	for i, obs := range observations {
		output := filter.PredictSync(obs)
		t.Logf("Step %d: obs=%.0f, current=%.2f, particles=%d, confidence=%.3f",
			i, obs, output.CurrentActivity, filter.ParticleCount, output.Confidence)

		if output.CurrentActivity <= 0 {
			t.Errorf("step %d: current activity should be positive, got %.2f", i, output.CurrentActivity)
		}
	}

	if filter.ParticleCount < filter.MinParticles || filter.ParticleCount > filter.MaxParticles {
		t.Errorf("final particle count %d out of bounds [%d, %d]",
			filter.ParticleCount, filter.MinParticles, filter.MaxParticles)
	}
}

func TestTermiteParticleFilter_ShouldReleaseLogic(t *testing.T) {
	filter := newTestFilter()

	output := filter.PredictSync(100.0)

	now := time.Now()
	if output.OptimalReleaseTime.Before(now) && !output.ShouldReleaseNow {
		t.Error("if optimal release time has passed, ShouldReleaseNow should be true")
	}

	if output.OptimalReleaseTime.After(now) && output.OptimalReleaseTime.Sub(now) <= 30*time.Minute {
		if !output.ShouldReleaseNow {
			t.Error("if optimal release time is within 30 minutes, ShouldReleaseNow should be true")
		}
	}

	t.Logf("Optimal release: %v (in %v), ShouldReleaseNow: %v",
		output.OptimalReleaseTime, output.OptimalReleaseTime.Sub(now), output.ShouldReleaseNow)
}
