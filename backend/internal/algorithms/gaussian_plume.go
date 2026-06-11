package algorithms

import (
	"math"
	"time"
)

type GaussianPlumeModel struct {
	ReleaseRate      float64
	WindSpeed        float64
	WindDirection    float64
	StackHeight      float64
	StabilityClass   string
}

type ConcentrationPoint struct {
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Z             float64 `json:"z"`
	Concentration float64 `json:"concentration"`
}

type FumigationSimulationResult struct {
	GridSizeX    int                   `json:"grid_size_x"`
	GridSizeY    int                   `json:"grid_size_y"`
	GridSizeZ    int                   `json:"grid_size_z"`
	CellSize     float64               `json:"cell_size"`
	Points       []ConcentrationPoint  `json:"points"`
	MaxConc      float64               `json:"max_concentration"`
	AvgConc      float64               `json:"avg_concentration"`
	Timestamp    time.Time             `json:"timestamp"`
	Duration     float64               `json:"duration"`
	EffectiveVolume float64            `json:"effective_volume"`
}

func NewGaussianPlumeModel(releaseRate, windSpeed, windDirection, stackHeight float64, stabilityClass string) *GaussianPlumeModel {
	return &GaussianPlumeModel{
		ReleaseRate:    releaseRate,
		WindSpeed:      windSpeed,
		WindDirection:  windDirection,
		StackHeight:    stackHeight,
		StabilityClass: stabilityClass,
	}
}

func pasquillGiffordSigmaY(distance float64, stabilityClass string) float64 {
	switch stabilityClass {
	case "A":
		return 0.22 * distance * math.Sqrt(1+0.0001*distance)
	case "B":
		return 0.16 * distance * math.Sqrt(1+0.0001*distance)
	case "C":
		return 0.11 * distance / math.Sqrt(1+0.0001*distance)
	case "D":
		return 0.08 * distance / math.Sqrt(1+0.0001*distance)
	case "E":
		return 0.06 * distance / math.Sqrt(1+0.0001*distance)
	case "F":
		return 0.04 * distance / math.Sqrt(1+0.0001*distance)
	default:
		return 0.08 * distance / math.Sqrt(1+0.0001*distance)
	}
}

func pasquillGiffordSigmaZ(distance float64, stabilityClass string) float64 {
	switch stabilityClass {
	case "A":
		return 0.2 * distance
	case "B":
		return 0.12 * distance
	case "C":
		return 0.08 * distance
	case "D":
		return 0.06 * distance
	case "E":
		return 0.03 * distance
	case "F":
		return 0.016 * distance
	default:
		return 0.06 * distance
	}
}

func (m *GaussianPlumeModel) CalculateConcentration(x, y, z float64) float64 {
	if x <= 0 || m.WindSpeed <= 0 {
		return 0
	}

	sigmaY := pasquillGiffordSigmaY(x, m.StabilityClass)
	sigmaZ := pasquillGiffordSigmaZ(x, m.StabilityClass)

	term1 := m.ReleaseRate / (2 * math.Pi * m.WindSpeed * sigmaY * sigmaZ)
	term2 := math.Exp(-math.Pow(y, 2) / (2 * math.Pow(sigmaY, 2)))
	term3 := math.Exp(-math.Pow(z-m.StackHeight, 2)/(2*math.Pow(sigmaZ, 2))) +
		math.Exp(-math.Pow(z+m.StackHeight, 2)/(2*math.Pow(sigmaZ, 2)))

	return term1 * term2 * term3
}

func SimulateFumigation(
	centerX, centerY, centerZ float64,
	releaseRate float64,
	windSpeed float64,
	windDirection float64,
	gridSizeX, gridSizeY, gridSizeZ int,
	cellSize float64,
	stabilityClass string,
	duration float64,
) *FumigationSimulationResult {
	model := NewGaussianPlumeModel(releaseRate, windSpeed, windDirection, centerZ, stabilityClass)

	points := make([]ConcentrationPoint, 0, gridSizeX*gridSizeY*gridSizeZ)
	var totalConc float64
	var maxConc float64
	var effectiveVolume float64

	windRad := windDirection * math.Pi / 180.0

	for i := 0; i < gridSizeX; i++ {
		for j := 0; j < gridSizeY; j++ {
			for k := 0; k < gridSizeZ; k++ {
				dx := (float64(i) - float64(gridSizeX)/2) * cellSize
				dy := (float64(j) - float64(gridSizeY)/2) * cellSize
				dz := (float64(k) - float64(gridSizeZ)/2) * cellSize

				rotX := dx*math.Cos(windRad) + dy*math.Sin(windRad)
				rotY := -dx*math.Sin(windRad) + dy*math.Cos(windRad)

				conc := model.CalculateConcentration(rotX+1.0, rotY, dz+centerZ)

				conc *= 1.0 - math.Exp(-duration/30.0)

				if conc > maxConc {
					maxConc = conc
				}
				totalConc += conc

				if conc > 0.01 {
					effectiveVolume += cellSize * cellSize * cellSize
				}

				points = append(points, ConcentrationPoint{
					X:             centerX + dx,
					Y:             centerY + dy,
					Z:             centerZ + dz,
					Concentration: conc,
				})
			}
		}
	}

	avgConc := totalConc / float64(len(points))

	return &FumigationSimulationResult{
		GridSizeX:       gridSizeX,
		GridSizeY:       gridSizeY,
		GridSizeZ:       gridSizeZ,
		CellSize:        cellSize,
		Points:          points,
		MaxConc:         maxConc,
		AvgConc:         avgConc,
		Timestamp:       time.Now(),
		Duration:        duration,
		EffectiveVolume: effectiveVolume,
	}
}

func CalculateOptimalReleasePoints(
	riskZones []map[string]interface{},
	buildingBounds map[string]float64,
	numPoints int,
) []map[string]interface{} {
	var releasePoints []map[string]interface{}

	highRiskPoints := make([]map[string]interface{}, 0)
	for _, zone := range riskZones {
		if zone["risk_level"] == "critical" || zone["risk_level"] == "high" {
			highRiskPoints = append(highRiskPoints, zone)
		}
	}

	if len(highRiskPoints) == 0 {
		for _, zone := range riskZones {
			if zone["risk_level"] == "medium" {
				highRiskPoints = append(highRiskPoints, zone)
			}
		}
	}

	numToSelect := numPoints
	if len(highRiskPoints) < numPoints {
		numToSelect = len(highRiskPoints)
	}

	for i := 0; i < numToSelect; i++ {
		zone := highRiskPoints[i]
		releasePoint := map[string]interface{}{
			"pos_x":          zone["pos_x"],
			"pos_y":          zone["pos_y"],
			"pos_z":          zone["pos_z"],
			"release_rate":   5.0,
			"duration":       120.0,
			"target_zone":    zone["sensor_id"],
			"estimated_max_conc": 0.5,
		}
		releasePoints = append(releasePoints, releasePoint)
	}

	return releasePoints
}

func GetFumigationStatus(simulation *FumigationSimulationResult, threshold float64) map[string]interface{} {
	status := make(map[string]interface{})

	aboveThreshold := 0
	totalPoints := len(simulation.Points)

	for _, p := range simulation.Points {
		if p.Concentration >= threshold {
			aboveThreshold++
		}
	}

	coverage := float64(aboveThreshold) / float64(totalPoints) * 100

	status["coverage_percent"] = coverage
	status["max_concentration"] = simulation.MaxConc
	status["avg_concentration"] = simulation.AvgConc
	status["effective_volume"] = simulation.EffectiveVolume
	status["is_effective"] = coverage > 60.0 && simulation.MaxConc > threshold

	var effectLevel string
	switch {
	case coverage > 80 && simulation.MaxConc > threshold*2:
		effectLevel = "excellent"
	case coverage > 60 && simulation.MaxConc > threshold:
		effectLevel = "good"
	case coverage > 40:
		effectLevel = "moderate"
	default:
		effectLevel = "poor"
	}
	status["effectiveness_level"] = effectLevel

	return status
}
