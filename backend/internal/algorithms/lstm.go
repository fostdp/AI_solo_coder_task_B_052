package algorithms

import (
	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/algorithms/lstm"
	"math"
	"sync"
	"time"
)

type LSTMPredictor struct {
	InputSize   int       `json:"input_size"`
	HiddenSize  int       `json:"hidden_size"`
	OutputSize  int       `json:"output_size"`
	WeightsIH   [][]float64 `json:"weights_ih"`
	WeightsHH   [][]float64 `json:"weights_hh"`
	Biases      []float64   `json:"biases"`
	WeightsHO   [][]float64 `json:"weights_ho"`
	BiasesO     []float64   `json:"biases_o"`
	HiddenState []float64   `json:"-"`
	CellState   []float64   `json:"-"`
}

type TermitePredictionResult struct {
	Timestamp     time.Time `json:"timestamp"`
	ActivityLevel float64   `json:"activity_level"`
	RiskLevel     string    `json:"risk_level"`
	Confidence    float64   `json:"confidence"`
	Trend         string    `json:"trend"`
}

func NewLSTMPredictor(inputSize, hiddenSize, outputSize int) *LSTMPredictor {
	p := &LSTMPredictor{
		InputSize:   inputSize,
		HiddenSize:  hiddenSize,
		OutputSize:  outputSize,
		HiddenState: make([]float64, hiddenSize),
		CellState:   make([]float64, hiddenSize),
	}

	p.WeightsIH = make([][]float64, 4*hiddenSize)
	for i := range p.WeightsIH {
		p.WeightsIH[i] = make([]float64, inputSize)
		for j := range p.WeightsIH[i] {
			p.WeightsIH[i][j] = randomWeight(inputSize)
		}
	}

	p.WeightsHH = make([][]float64, 4*hiddenSize)
	for i := range p.WeightsHH {
		p.WeightsHH[i] = make([]float64, hiddenSize)
		for j := range p.WeightsHH[i] {
			p.WeightsHH[i][j] = randomWeight(hiddenSize)
		}
	}

	p.Biases = make([]float64, 4*hiddenSize)

	p.WeightsHO = make([][]float64, outputSize)
	for i := range p.WeightsHO {
		p.WeightsHO[i] = make([]float64, hiddenSize)
		for j := range p.WeightsHO[i] {
			p.WeightsHO[i][j] = randomWeight(hiddenSize)
		}
	}

	p.BiasesO = make([]float64, outputSize)

	return p
}

func randomWeight(size int) float64 {
	scale := 1.0 / math.Sqrt(float64(size))
	return (randFloat()*2 - 1) * scale
}

func randFloat() float64 {
	return float64(uint32(time.Now().UnixNano()%100000)) / 100000.0
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func tanh(x float64) float64 {
	return math.Tanh(x)
}

func (p *LSTMPredictor) Forward(input []float64) []float64 {
	gates := make([]float64, 4*p.HiddenSize)

	for i := 0; i < 4*p.HiddenSize; i++ {
		sum := p.Biases[i]
		for j := 0; j < p.InputSize; j++ {
			sum += p.WeightsIH[i][j] * input[j]
		}
		for j := 0; j < p.HiddenSize; j++ {
			sum += p.WeightsHH[i][j] * p.HiddenState[j]
		}
		gates[i] = sum
	}

	newCellState := make([]float64, p.HiddenSize)
	newHiddenState := make([]float64, p.HiddenSize)

	for i := 0; i < p.HiddenSize; i++ {
		f := sigmoid(gates[i])
		iGate := sigmoid(gates[p.HiddenSize+i])
		cTilde := tanh(gates[2*p.HiddenSize+i])
		oGate := sigmoid(gates[3*p.HiddenSize+i])

		newCellState[i] = f*p.CellState[i] + iGate*cTilde
		newHiddenState[i] = oGate * tanh(newCellState[i])
	}

	p.CellState = newCellState
	p.HiddenState = newHiddenState

	output := make([]float64, p.OutputSize)
	for i := 0; i < p.OutputSize; i++ {
		sum := p.BiasesO[i]
		for j := 0; j < p.HiddenSize; j++ {
			sum += p.WeightsHO[i][j] * p.HiddenState[j]
		}
		output[i] = sigmoid(sum)
	}

	return output
}

func (p *LSTMPredictor) Reset() {
	p.HiddenState = make([]float64, p.HiddenSize)
	p.CellState = make([]float64, p.HiddenSize)
}

var modelService *lstm.ModelService
var modelServiceOnce sync.Once

func getModelService() *lstm.ModelService {
	modelServiceOnce.Do(func() {
		modelPath := ""
		if config.AppConfig != nil && config.AppConfig.Model.LstmPath != "" {
			modelPath = config.AppConfig.Model.LstmPath
		}
		modelService = lstm.GetModelService(modelPath)
	})
	return modelService
}

func PredictTermiteActivity(historicalData []map[string]float64, hoursAhead int) ([]TermitePredictionResult, error) {
	ms := getModelService()
	ms.ResetState()

	sequenceLen := len(historicalData)
	if sequenceLen == 0 {
		return nil, nil
	}

	smoother := lstm.NewEWMASmoother(0.3, 48)
	smoothedFields := []string{"event_count", "energy", "amplitude", "duration", "peak_freq"}
	processedData := smoother.SmoothMapSlice(historicalData, smoothedFields)

	eventCounts := make([]float64, len(processedData))
	for i, d := range processedData {
		eventCounts[i] = d["event_count"]
	}
	eventCounts = lstm.RemoveOutliers(eventCounts, 1.5)
	eventCounts = lstm.DoubleExponentialSmoothing(eventCounts, 0.3, 0.1)

	for i, d := range processedData {
		d["event_count"] = eventCounts[i]
	}

	var lastActivity float64
	var avgEnergy float64
	var avgEvents float64
	var trend float64

	for i, d := range processedData {
		lastActivity = d["event_count"] / 100.0
		avgEnergy += d["energy"]
		avgEvents += d["event_count"]
		if i > 0 {
			trend += (d["event_count"] - processedData[i-1]["event_count"]) / 100.0
		}
	}
	avgEnergy /= float64(sequenceLen)
	avgEvents /= float64(sequenceLen)
	if sequenceLen > 1 {
		trend /= float64(sequenceLen - 1)
	}

	results := make([]TermitePredictionResult, hoursAhead)
	now := time.Now()

	baseActivity := avgEvents / 100.0
	currentActivity := lastActivity

	var lastPredicted float64
	smoothedPrediction := 0.0

	for i := 0; i < hoursAhead; i++ {
		input := []float64{
			currentActivity,
			baseActivity,
			avgEnergy / 1000.0,
			trend,
			0.5 + 0.3*math.Sin(float64(i)*math.Pi/12),
			0.5,
			float64(i) / float64(hoursAhead),
			0.3 + 0.2*math.Sin(float64(i)*math.Pi/24),
		}

		output := ms.Predict(input)

		activityLevel := output[0] * 150.0

		activityLevel += trend * float64(i) * 5.0
		activityLevel += 20.0 * math.Sin(float64(now.Hour()+i)*math.Pi/12)
		activityLevel = math.Max(0, math.Min(200, activityLevel))

		if i > 0 {
			alpha := 0.4
			smoothedPrediction = alpha*activityLevel + (1-alpha)*lastPredicted
			activityLevel = smoothedPrediction
		} else {
			smoothedPrediction = activityLevel
		}
		lastPredicted = activityLevel

		riskLevel := getRiskLevel(activityLevel)
		confidence := 0.6 + 0.3*math.Exp(-float64(i)/24.0)

		trendDirection := "stable"
		if i > 0 && results[i-1].ActivityLevel > 0 {
			change := (activityLevel - results[i-1].ActivityLevel) / results[i-1].ActivityLevel
			if change > 0.1 {
				trendDirection = "rising"
			} else if change < -0.1 {
				trendDirection = "falling"
			}
		}

		results[i] = TermitePredictionResult{
			Timestamp:     now.Add(time.Duration(i+1) * time.Hour),
			ActivityLevel: activityLevel,
			RiskLevel:     riskLevel,
			Confidence:    confidence,
			Trend:         trendDirection,
		}

		currentActivity = activityLevel / 150.0
	}

	return results, nil
}

func getRiskLevel(activityLevel float64) string {
	switch {
	case activityLevel >= 100:
		return "critical"
	case activityLevel >= 70:
		return "high"
	case activityLevel >= 40:
		return "medium"
	case activityLevel >= 20:
		return "low"
	default:
		return "very_low"
	}
}

func GenerateRiskMap(building string, sensors []map[string]interface{}) []map[string]interface{} {
	var riskZones []map[string]interface{}

	for _, sensor := range sensors {
		sensorID := sensor["id"].(string)
		posX := sensor["pos_x"].(float64)
		posY := sensor["pos_y"].(float64)
		posZ := sensor["pos_z"].(float64)
		eventRate := sensor["event_rate"].(float64)

		riskLevel := getRiskLevel(eventRate)

		zone := map[string]interface{}{
			"sensor_id":   sensorID,
			"building":    building,
			"pos_x":       posX,
			"pos_y":       posY,
			"pos_z":       posZ,
			"radius":      1.0 + eventRate/50.0,
			"risk_level":  riskLevel,
			"event_rate":  eventRate,
			"intensity":   math.Min(1.0, eventRate/150.0),
		}

		riskZones = append(riskZones, zone)
	}

	return riskZones
}
