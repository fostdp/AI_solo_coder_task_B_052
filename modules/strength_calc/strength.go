package strength_calc

import (
	"math"
	"time"
)

type WoodStrengthEvaluator struct {
	ReferenceDensity     float64
	CriticalEnergy       float64
	RequiredSafetyFactor float64
	DefaultDepthRatio    float64
}

var woodTypeCorrection = map[string]float64{
	"pine":    0.85,
	"nanmu":   1.15,
	"fir":     0.90,
	"oak":     1.25,
	"default": 1.0,
}

func GetWoodTypeCorrection(woodType string) float64 {
	if corr, ok := woodTypeCorrection[woodType]; ok {
		return corr
	}
	return woodTypeCorrection["default"]
}

func NewWoodStrengthEvaluator(refDensity, criticalEnergy, requiredSF, depthRatio float64) *WoodStrengthEvaluator {
	return &WoodStrengthEvaluator{
		ReferenceDensity:     refDensity,
		CriticalEnergy:       criticalEnergy,
		RequiredSafetyFactor: requiredSF,
		DefaultDepthRatio:    depthRatio,
	}
}

func (e *WoodStrengthEvaluator) AssessStrength(sensorID, building, location, woodType string, cumulativeEnergy, woodDensity, depthRatio float64) WoodStrengthAssessment {
	damageIndex := 1.0 - (cumulativeEnergy / e.CriticalEnergy)
	damageIndex = math.Max(0, math.Min(1, damageIndex))

	woodCorrection := GetWoodTypeCorrection(woodType)

	residualStrengthIndex := (woodDensity / e.ReferenceDensity) * damageIndex * (1.0 - depthRatio) * woodCorrection

	safetyFactor := residualStrengthIndex * 3.0

	var strengthLevel string
	switch {
	case safetyFactor >= 2.0:
		strengthLevel = "safe"
	case safetyFactor >= 1.5:
		strengthLevel = "caution"
	case safetyFactor >= 1.0:
		strengthLevel = "warning"
	case safetyFactor >= 0.5:
		strengthLevel = "danger"
	default:
		strengthLevel = "critical"
	}

	return WoodStrengthAssessment{
		SensorID:              sensorID,
		Building:              building,
		Location:              location,
		WoodType:              woodType,
		CumulativeEnergy:      cumulativeEnergy,
		WoodDensity:           woodDensity,
		DamageIndex:           damageIndex,
		ResidualStrengthIndex: residualStrengthIndex,
		SafetyFactor:          safetyFactor,
		StrengthLevel:         strengthLevel,
		Timestamp:             time.Now(),
	}
}

func (e *WoodStrengthEvaluator) BatchAssess(sensors []SensorStrengthInput) []WoodStrengthAssessment {
	results := make([]WoodStrengthAssessment, len(sensors))
	for i, s := range sensors {
		results[i] = e.AssessStrength(s.SensorID, s.Building, s.Location, s.WoodType, s.CumulativeEnergy, s.WoodDensity, s.DepthRatio)
	}
	return results
}

func SimulateWoodDensity(baseDensity float64, age int, moistureContent float64) float64 {
	ageFactor := 1.0 - 0.001*float64(age/100)
	density := baseDensity * ageFactor
	density = density * (1 + 0.01*(moistureContent-12.0))
	density = math.Max(200, math.Min(800, density))
	return density
}
