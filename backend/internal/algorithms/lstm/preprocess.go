package lstm

import (
	"math"
	"sync"
)

type EWMASmoother struct {
	alpha     float64
	lastValue map[string]float64
	history   map[string][]float64
	maxHistory int
	mu        sync.RWMutex
}

func NewEWMASmoother(alpha float64, maxHistory int) *EWMASmoother {
	if alpha <= 0 {
		alpha = 0.3
	}
	if alpha >= 1 {
		alpha = 0.3
	}
	if maxHistory <= 0 {
		maxHistory = 24
	}

	return &EWMASmoother{
		alpha:      alpha,
		lastValue:  make(map[string]float64),
		history:    make(map[string][]float64),
		maxHistory: maxHistory,
	}
}

func (e *EWMASmoother) Smooth(key string, value float64) float64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	last, exists := e.lastValue[key]
	if !exists {
		e.lastValue[key] = value
		e.history[key] = append(e.history[key], value)
		return value
	}

	smoothed := e.alpha*value + (1-e.alpha)*last
	e.lastValue[key] = smoothed

	e.history[key] = append(e.history[key], smoothed)
	if len(e.history[key]) > e.maxHistory {
		e.history[key] = e.history[key][len(e.history[key])-e.maxHistory:]
	}

	return smoothed
}

func (e *EWMASmoother) SmoothSeries(key string, values []float64) []float64 {
	smoothed := make([]float64, len(values))

	for i, v := range values {
		smoothed[i] = e.Smooth(key, v)
	}

	return smoothed
}

func (e *EWMASmoother) SmoothMapSlice(data []map[string]float64, fields []string) []map[string]float64 {
	if len(data) == 0 {
		return data
	}

	result := make([]map[string]float64, len(data))
	fieldEWMA := make(map[string]*EWMASmoother)

	for _, field := range fields {
		fieldEWMA[field] = NewEWMASmoother(e.alpha, e.maxHistory)
	}

	for i, item := range data {
		result[i] = make(map[string]float64)

		for k, v := range item {
			if smoother, ok := fieldEWMA[k]; ok {
				result[i][k] = smoother.Smooth(k, v)
			} else {
				result[i][k] = v
			}
		}
	}

	return result
}

func (e *EWMASmoother) GetSmoothedValue(key string) (float64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	v, ok := e.lastValue[key]
	return v, ok
}

func (e *EWMASmoother) GetHistory(key string) []float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	h := e.history[key]
	result := make([]float64, len(h))
	copy(result, h)
	return result
}

func (e *EWMASmoother) GetTrend(key string) (float64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	h := e.history[key]
	if len(h) < 2 {
		return 0, false
	}

	n := len(h)
	var sumX, sumY, sumXY, sumX2 float64

	for i := 0; i < n; i++ {
		x := float64(i)
		y := h[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)
	return slope, true
}

func (e *EWMASmoother) GetStandardDeviation(key string) (float64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	h := e.history[key]
	if len(h) < 2 {
		return 0, false
	}

	var sum, mean, variance float64
	n := float64(len(h))

	for _, v := range h {
		sum += v
	}
	mean = sum / n

	for _, v := range h {
		variance += (v - mean) * (v - mean)
	}
	variance /= n

	return math.Sqrt(variance), true
}

func (e *EWMASmoother) IsSpike(key string, value float64, thresholdSigma float64) bool {
	if thresholdSigma <= 0 {
		thresholdSigma = 2.0
	}

	std, ok := e.GetStandardDeviation(key)
	if !ok {
		return false
	}

	mean, ok := e.GetSmoothedValue(key)
	if !ok {
		return false
	}

	deviation := math.Abs(value - mean)
	return deviation > thresholdSigma*std
}

func (e *EWMASmoother) Reset(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.lastValue, key)
	delete(e.history, key)
}

func (e *EWMASmoother) ResetAll() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastValue = make(map[string]float64)
	e.history = make(map[string][]float64)
}

func SmoothAcousticData(eventCounts []float64, alpha float64) []float64 {
	if alpha <= 0 {
		alpha = 0.3
	}

	result := make([]float64, len(eventCounts))
	if len(eventCounts) == 0 {
		return result
	}

	result[0] = eventCounts[0]
	for i := 1; i < len(eventCounts); i++ {
		result[i] = alpha*eventCounts[i] + (1-alpha)*result[i-1]
	}

	return result
}

func DoubleExponentialSmoothing(values []float64, alpha, beta float64) []float64 {
	if len(values) == 0 {
		return values
	}
	if alpha <= 0 || alpha > 1 {
		alpha = 0.3
	}
	if beta <= 0 || beta > 1 {
		beta = 0.1
	}

	n := len(values)
	result := make([]float64, n)

	level := values[0]
	trend := 0.0
	if n >= 2 {
		trend = values[1] - values[0]
	}

	result[0] = values[0]

	for i := 1; i < n; i++ {
		lastLevel := level
		level = alpha*values[i] + (1-alpha)*(level+trend)
		trend = beta*(level-lastLevel) + (1-beta)*trend
		result[i] = level + trend
	}

	return result
}

func RemoveOutliers(values []float64, k float64) []float64 {
	if len(values) < 3 {
		return values
	}
	if k <= 0 {
		k = 1.5
	}

	n := len(values)
	sorted := make([]float64, n)
	copy(sorted, values)

	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	q1 := sorted[n/4]
	q3 := sorted[3*n/4]
	iqr := q3 - q1
	lower := q1 - k*iqr
	upper := q3 + k*iqr

	result := make([]float64, 0, n)
	for _, v := range values {
		if v >= lower && v <= upper {
			result = append(result, v)
		} else {
			if len(result) > 0 {
				result = append(result, result[len(result)-1])
			} else {
				result = append(result, (q1+q3)/2)
			}
		}
	}

	return result
}
