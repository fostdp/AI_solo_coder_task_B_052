package strength_calc

import "time"

type WoodStrengthAssessment struct {
	SensorID              string    `json:"sensor_id"`
	Building              string    `json:"building"`
	Location              string    `json:"location"`
	WoodType              string    `json:"wood_type"`
	CumulativeEnergy      float64   `json:"cumulative_energy"`
	WoodDensity           float64   `json:"wood_density"`
	DamageIndex           float64   `json:"damage_index"`
	ResidualStrengthIndex float64   `json:"residual_strength_index"`
	SafetyFactor          float64   `json:"safety_factor"`
	StrengthLevel         string    `json:"strength_level"`
	Timestamp             time.Time `json:"timestamp"`
}

type SensorStrengthInput struct {
	SensorID         string
	Building         string
	Location         string
	WoodType         string
	CumulativeEnergy float64
	WoodDensity      float64
	DepthRatio       float64
}
