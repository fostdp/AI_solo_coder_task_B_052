package particle_filter_timing

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

type TermiteParticleFilter struct {
	mu                  sync.Mutex
	Particles           []Particle
	ParticleCount       int
	MinParticles        int
	MaxParticles        int
	ProcessNoise        float64
	MeasurementNoise    float64
	ResampleThreshold   float64
	ESSIncreaseThreshold float64
	ESSDecreaseThreshold float64
	ReleaseLeadTime     time.Duration
	PredictionHorizon   time.Duration
	randSource          *rand.Rand

	requestChan  chan PredictRequest
	responseChan chan PredictResponse
	cancelFunc   context.CancelFunc
	wg           sync.WaitGroup
	running      bool
}

func NewTermiteParticleFilter(initialCount, minCount, maxCount int, essIncreaseThreshold, essDecreaseThreshold, processNoise, measurementNoise, resampleThreshold float64, releaseLeadTime, predictionHorizon time.Duration) *TermiteParticleFilter {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	particles := make([]Particle, initialCount)
	for i := 0; i < initialCount; i++ {
		particles[i] = Particle{
			ActivityLevel: src.Float64() * 150.0,
			Trend:         (src.Float64() - 0.5) * 2.0,
			Weight:        1.0 / float64(initialCount),
		}
	}
	return &TermiteParticleFilter{
		Particles:            particles,
		ParticleCount:        initialCount,
		MinParticles:         minCount,
		MaxParticles:         maxCount,
		ProcessNoise:         processNoise,
		MeasurementNoise:     measurementNoise,
		ResampleThreshold:    resampleThreshold,
		ESSIncreaseThreshold: essIncreaseThreshold,
		ESSDecreaseThreshold: essDecreaseThreshold,
		ReleaseLeadTime:      releaseLeadTime,
		PredictionHorizon:    predictionHorizon,
		randSource:           src,
	}
}

func (tpf *TermiteParticleFilter) Start(ctx context.Context) {
	tpf.mu.Lock()
	if tpf.running {
		tpf.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	tpf.cancelFunc = cancel
	tpf.requestChan = make(chan PredictRequest, 100)
	tpf.responseChan = make(chan PredictResponse, 100)
	tpf.running = true
	tpf.mu.Unlock()

	tpf.wg.Add(1)
	go func() {
		defer tpf.wg.Done()
		for {
			select {
			case <-ctx.Done():
				close(tpf.responseChan)
				return
			case req, ok := <-tpf.requestChan:
				if !ok {
					return
				}
				output := tpf.PredictSync(req.Observation)
				output.Building = req.Building
				tpf.responseChan <- PredictResponse{
					Output: output,
					Err:    nil,
				}
			}
		}
	}()
}

func (tpf *TermiteParticleFilter) Stop() {
	tpf.mu.Lock()
	defer tpf.mu.Unlock()

	if !tpf.running {
		return
	}

	if tpf.cancelFunc != nil {
		tpf.cancelFunc()
	}
	tpf.wg.Wait()
	close(tpf.requestChan)
	tpf.running = false
}

func (tpf *TermiteParticleFilter) PredictAsync(req PredictRequest) error {
	tpf.mu.Lock()
	defer tpf.mu.Unlock()

	if !tpf.running {
		return nil
	}

	select {
	case tpf.requestChan <- req:
		return nil
	default:
		return nil
	}
}

func (tpf *TermiteParticleFilter) Response() <-chan PredictResponse {
	tpf.mu.Lock()
	defer tpf.mu.Unlock()
	return tpf.responseChan
}

func (tpf *TermiteParticleFilter) PredictSync(observation float64) ParticleFilterOutput {
	tpf.mu.Lock()
	defer tpf.mu.Unlock()

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
		ess = float64(tpf.ParticleCount)
	}

	essRatio := ess / float64(tpf.ParticleCount)
	tpf.adaptParticleCount(essRatio)

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

	particleStates := make([]ParticleState, len(tpf.Particles))
	for i, p := range tpf.Particles {
		particleStates[i] = ParticleState{
			ActivityLevel: p.ActivityLevel,
			Trend:         p.Trend,
			Weight:        p.Weight,
			Timestamp:     now,
		}
	}

	return ParticleFilterOutput{
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

func (tpf *TermiteParticleFilter) adaptParticleCount(essRatio float64) {
	if essRatio < tpf.ESSIncreaseThreshold && tpf.ParticleCount < tpf.MaxParticles {
		targetCount := tpf.ParticleCount * 2
		if targetCount > tpf.MaxParticles {
			targetCount = tpf.MaxParticles
		}
		tpf.increaseParticles(targetCount)
	} else if essRatio > tpf.ESSDecreaseThreshold && tpf.ParticleCount > tpf.MinParticles {
		targetCount := tpf.ParticleCount / 2
		if targetCount < tpf.MinParticles {
			targetCount = tpf.MinParticles
		}
		tpf.decreaseParticles(targetCount)
	}
}

func (tpf *TermiteParticleFilter) increaseParticles(targetCount int) {
	if targetCount <= tpf.ParticleCount {
		return
	}

	newParticles := make([]Particle, targetCount)
	copy(newParticles, tpf.Particles)

	weight := 1.0 / float64(targetCount)
	for i := tpf.ParticleCount; i < targetCount; i++ {
		srcIdx := tpf.randSource.Intn(tpf.ParticleCount)
		src := tpf.Particles[srcIdx]
		newParticles[i] = Particle{
			ActivityLevel: src.ActivityLevel + tpf.ProcessNoise*tpf.randSource.NormFloat64()*0.5,
			Trend:         src.Trend + 0.01*tpf.randSource.NormFloat64()*0.5,
			Weight:        weight,
		}
	}

	for i := 0; i < tpf.ParticleCount; i++ {
		newParticles[i].Weight = weight
	}

	tpf.Particles = newParticles
	tpf.ParticleCount = targetCount
}

func (tpf *TermiteParticleFilter) decreaseParticles(targetCount int) {
	if targetCount >= tpf.ParticleCount || targetCount <= 0 {
		return
	}

	resampled := SystematicResample(tpf.Particles)
	step := len(resampled) / targetCount

	newParticles := make([]Particle, targetCount)
	weight := 1.0 / float64(targetCount)
	for i := 0; i < targetCount; i++ {
		srcIdx := i * step
		if srcIdx >= len(resampled) {
			srcIdx = len(resampled) - 1
		}
		newParticles[i] = Particle{
			ActivityLevel: resampled[srcIdx].ActivityLevel,
			Trend:         resampled[srcIdx].Trend,
			Weight:        weight,
		}
	}

	tpf.Particles = newParticles
	tpf.ParticleCount = targetCount
}
