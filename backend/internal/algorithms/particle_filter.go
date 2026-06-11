package algorithms

import (
	"math"
	"math/rand"
	"time"

	"ancient-wood-monitor/internal/models"
)

type Particle struct {
	ActivityLevel float64
	Trend         float64
	Weight        float64
}

type TermiteParticleFilter struct {
	Particles         []Particle
	ParticleCount     int
	ProcessNoise      float64
	MeasurementNoise  float64
	ResampleThreshold float64
	ReleaseLeadTime   time.Duration
	PredictionHorizon time.Duration
	randSource        *rand.Rand
}

func NewTermiteParticleFilter(particleCount int, processNoise, measurementNoise, resampleThreshold float64, releaseLeadTime, predictionHorizon time.Duration) *TermiteParticleFilter {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	particles := make([]Particle, particleCount)
	for i := 0; i < particleCount; i++ {
		particles[i] = Particle{
			ActivityLevel: src.Float64() * 150.0,
			Trend:         (src.Float64() - 0.5) * 2.0,
			Weight:        1.0 / float64(particleCount),
		}
	}
	return &TermiteParticleFilter{
		Particles:         particles,
		ParticleCount:     particleCount,
		ProcessNoise:      processNoise,
		MeasurementNoise:  measurementNoise,
		ResampleThreshold: resampleThreshold,
		ReleaseLeadTime:   releaseLeadTime,
		PredictionHorizon: predictionHorizon,
		randSource:        src,
	}
}

func (tpf *TermiteParticleFilter) Predict(observation float64) models.ParticleFilterOutput {
	for i := range tpf.Particles {
		tpf.Particles[i].ActivityLevel += tpf.Particles[i].Trend + tpf.ProcessNoise*tpf.randSource.NormFloat64()
		tpf.Particles[i].Trend += 0.01 * tpf.randSource.NormFloat64()
	}

	for i := range tpf.Particles {
		diff := tpf.Particles[i].ActivityLevel - observation
		tpf.Particles[i].Weight = math.Exp(-0.5 * math.Pow(diff/tpf.MeasurementNoise, 2))
	}

	var weightSum float64
	for i := range tpf.Particles {
		weightSum += tpf.Particles[i].Weight
	}
	if weightSum > 0 {
		for i := range tpf.Particles {
			tpf.Particles[i].Weight /= weightSum
		}
	}

	ess := EffectiveSampleSize(tpf.Particles)
	if ess < float64(tpf.ParticleCount)*tpf.ResampleThreshold {
		tpf.Particles = SystematicResample(tpf.Particles)
	}

	var meanActivity, meanTrend float64
	for i := range tpf.Particles {
		meanActivity += tpf.Particles[i].Weight * tpf.Particles[i].ActivityLevel
		meanTrend += tpf.Particles[i].Weight * tpf.Particles[i].Trend
	}

	stepDuration := 1 * time.Minute
	steps := int(tpf.PredictionHorizon / stepDuration)

	simActivity := meanActivity
	simTrend := meanTrend
	peakActivity := simActivity
	peakStep := 0
	declineCount := 0
	prevActivity := simActivity

	for step := 1; step <= steps; step++ {
		simActivity += simTrend + tpf.ProcessNoise*tpf.randSource.NormFloat64()*0.1
		simTrend += 0.01 * tpf.randSource.NormFloat64() * 0.1

		if simActivity < prevActivity {
			declineCount++
		} else {
			declineCount = 0
		}

		if simActivity > peakActivity {
			peakActivity = simActivity
			peakStep = step
		}

		if declineCount >= 3 {
			break
		}

		prevActivity = simActivity
	}

	if peakStep == 0 {
		peakStep = steps
	}

	now := time.Now()
	predictedPeakTime := now.Add(time.Duration(peakStep) * stepDuration)
	optimalReleaseTime := predictedPeakTime.Add(-tpf.ReleaseLeadTime)

	shouldRelease := false
	if optimalReleaseTime.After(now) && optimalReleaseTime.Sub(now) <= 30*time.Minute {
		shouldRelease = true
	}
	if !optimalReleaseTime.After(now) {
		shouldRelease = true
	}

	confidence := ess / float64(tpf.ParticleCount)
	if confidence > 1.0 {
		confidence = 1.0
	}

	particleStates := make([]models.ParticleState, len(tpf.Particles))
	for i, p := range tpf.Particles {
		particleStates[i] = models.ParticleState{
			ActivityLevel: p.ActivityLevel,
			Trend:         p.Trend,
			Weight:        p.Weight,
			Timestamp:     now,
		}
	}

	return models.ParticleFilterOutput{
		Particles:          particleStates,
		PredictedPeakTime:  predictedPeakTime,
		OptimalReleaseTime: optimalReleaseTime,
		CurrentActivity:    meanActivity,
		PredictedPeak:      peakActivity,
		Confidence:         confidence,
		ShouldReleaseNow:   shouldRelease,
	}
}

func SystematicResample(particles []Particle) []Particle {
	n := len(particles)
	cdf := make([]float64, n)
	cdf[0] = particles[0].Weight
	for i := 1; i < n; i++ {
		cdf[i] = cdf[i-1] + particles[i].Weight
	}

	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	u0 := src.Float64() / float64(n)

	resampled := make([]Particle, n)
	for i := 0; i < n; i++ {
		u := u0 + float64(i)/float64(n)
		idx := 0
		for idx < n-1 && cdf[idx] < u {
			idx++
		}
		resampled[i] = Particle{
			ActivityLevel: particles[idx].ActivityLevel,
			Trend:         particles[idx].Trend,
			Weight:        1.0 / float64(n),
		}
	}

	return resampled
}

func EffectiveSampleSize(particles []Particle) float64 {
	var sumSq float64
	for _, p := range particles {
		sumSq += p.Weight * p.Weight
	}
	if sumSq == 0 {
		return 0
	}
	return 1.0 / sumSq
}
