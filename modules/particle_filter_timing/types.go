package particle_filter_timing

import "time"

type Particle struct {
	ActivityLevel float64
	Trend         float64
	Weight        float64
}

type ParticleState struct {
	ActivityLevel float64   `json:"activity_level"`
	Trend         float64   `json:"trend"`
	Weight        float64   `json:"weight"`
	Timestamp     time.Time `json:"timestamp"`
}

type ParticleFilterOutput struct {
	Building           string          `json:"building"`
	Particles          []ParticleState `json:"particles"`
	PredictedPeakTime  time.Time       `json:"predicted_peak_time"`
	OptimalReleaseTime time.Time       `json:"optimal_release_time"`
	CurrentActivity    float64         `json:"current_activity"`
	PredictedPeak      float64         `json:"predicted_peak"`
	Confidence         float64         `json:"confidence"`
	ShouldReleaseNow   bool            `json:"should_release_now"`
}

type PredictRequest struct {
	Building    string
	Observation float64
	Timestamp   time.Time
}

type PredictResponse struct {
	Output ParticleFilterOutput
	Err    error
}
