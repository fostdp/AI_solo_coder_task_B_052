package algorithms

import (
	"math"
	"time"

	"ancient-wood-monitor/internal/models"
)

type WoodStrengthEvaluator struct {
	ReferenceDensity     float64
	CriticalEnergy       float64
	RequiredSafetyFactor float64
	DefaultDepthRatio    float64
}

type SensorStrengthInput struct {
	SensorID        string
	Building        string
	Location        string
	CumulativeEnergy float64
	WoodDensity     float64
	DepthRatio      float64
}

func NewWoodStrengthEvaluator(refDensity, criticalEnergy, requiredSF, depthRatio float64) *WoodStrengthEvaluator {
	return &WoodStrengthEvaluator{
		ReferenceDensity:     refDensity,
		CriticalEnergy:       criticalEnergy,
		RequiredSafetyFactor: requiredSF,
		DefaultDepthRatio:    depthRatio,
	}
}

func (e *WoodStrengthEvaluator) AssessStrength(sensorID, building, location string, cumulativeEnergy, woodDensity, depthRatio float64) models.WoodStrengthAssessment {
	damageIndex := 1.0 - (cumulativeEnergy / e.CriticalEnergy)
	damageIndex = math.Max(0, math.Min(1, damageIndex))

	residualStrengthIndex := (woodDensity / e.ReferenceDensity) * damageIndex * (1.0 - depthRatio)

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

	return models.WoodStrengthAssessment{
		SensorID:              sensorID,
		Building:              building,
		Location:              location,
		CumulativeEnergy:      cumulativeEnergy,
		WoodDensity:           woodDensity,
		DamageIndex:           damageIndex,
		ResidualStrengthIndex: residualStrengthIndex,
		SafetyFactor:          safetyFactor,
		StrengthLevel:         strengthLevel,
		Timestamp:             time.Now(),
	}
}

func (e *WoodStrengthEvaluator) BatchAssess(sensors []SensorStrengthInput) []models.WoodStrengthAssessment {
	results := make([]models.WoodStrengthAssessment, len(sensors))
	for i, s := range sensors {
		results[i] = e.AssessStrength(s.SensorID, s.Building, s.Location, s.CumulativeEnergy, s.WoodDensity, s.DepthRatio)
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
