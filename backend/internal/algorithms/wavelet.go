package algorithms

import (
	"fmt"
	"math"
)

type WaveletPacket struct {
	Level        int
	Data         []float64
	SamplingRate float64
}

type WaveletPacketNode struct {
	Level  int
	Index  int
	Data   []float64
	Energy float64
}

func NewWaveletPacket(data []float64, level int, samplingRate float64) *WaveletPacket {
	return &WaveletPacket{
		Level:        level,
		Data:         data,
		SamplingRate: samplingRate,
	}
}

func daubechies4(signal []float64) []float64 {
	n := len(signal)
	n2 := n / 2
	approx := make([]float64, n2)
	detail := make([]float64, n2)

	c0 := (1 + math.Sqrt(3)) / (4 * math.Sqrt(2))
	c1 := (3 + math.Sqrt(3)) / (4 * math.Sqrt(2))
	c2 := (3 - math.Sqrt(3)) / (4 * math.Sqrt(2))
	c3 := (1 - math.Sqrt(3)) / (4 * math.Sqrt(2))

	for i := 0; i < n2; i++ {
		idx0 := (2 * i) % n
		idx1 := (2*i + 1) % n
		idx2 := (2*i + 2) % n
		idx3 := (2*i + 3) % n

		approx[i] = c0*signal[idx0] + c1*signal[idx1] + c2*signal[idx2] + c3*signal[idx3]
		detail[i] = c3*signal[idx0] - c2*signal[idx1] + c1*signal[idx2] - c0*signal[idx3]
	}

	result := make([]float64, n)
	copy(result[:n2], approx)
	copy(result[n2:], detail)
	return result
}

func (wp *WaveletPacket) Decompose() []WaveletPacketNode {
	var allNodes []WaveletPacketNode

	currentLevelData := make([][]float64, 1)
	currentLevelData[0] = make([]float64, len(wp.Data))
	copy(currentLevelData[0], wp.Data)

	for level := 1; level <= wp.Level; level++ {
		nodesInLevel := int(math.Pow(2, float64(level)))
		nextLevelData := make([][]float64, nodesInLevel)

		prevNodes := nodesInLevel / 2
		for i := 0; i < prevNodes; i++ {
			decomposed := daubechies4(currentLevelData[i])
			n2 := len(decomposed) / 2
			nextLevelData[i*2] = decomposed[:n2]
			nextLevelData[i*2+1] = decomposed[n2:]
		}

		for i := 0; i < nodesInLevel; i++ {
			energy := calculateEnergy(nextLevelData[i])
			allNodes = append(allNodes, WaveletPacketNode{
				Level:  level,
				Index:  i,
				Data:   nextLevelData[i],
				Energy: energy,
			})
		}

		currentLevelData = nextLevelData
	}

	return allNodes
}

func calculateEnergy(data []float64) float64 {
	var energy float64
	for _, v := range data {
		energy += v * v
	}
	return energy
}

func (wp *WaveletPacket) GetEnergySpectrum() []float64 {
	allNodes := wp.Decompose()

	numBands := int(math.Pow(2, float64(wp.Level)))
	startIdx := len(allNodes) - numBands

	spectrum := make([]float64, numBands)
	for i := 0; i < numBands; i++ {
		spectrum[i] = allNodes[startIdx+i].Energy
	}

	return spectrum
}

func (wp *WaveletPacket) GetFrequencyRanges() []string {
	numNodes := int(math.Pow(2, float64(wp.Level)))
	nyquist := wp.SamplingRate / 2.0
	bandwidth := nyquist / float64(numNodes)

	ranges := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		low := float64(i) * bandwidth
		high := float64(i+1) * bandwidth
		ranges[i] = formatFreqRange(low, high)
	}

	return ranges
}

func formatFreqRange(low, high float64) string {
	if low < 1000 {
		return formatFloat(low) + "Hz-" + formatFloat(high) + "Hz"
	}
	return formatFloat(low/1000) + "kHz-" + formatFloat(high/1000) + "kHz"
}

func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%.1f", f)
}

func ExtractWaveletFeatures(signal []float64, samplingRate float64) map[string]float64 {
	wp := NewWaveletPacket(signal, 5, samplingRate)
	spectrum := wp.GetEnergySpectrum()

	totalEnergy := 0.0
	for _, e := range spectrum {
		totalEnergy += e
	}

	features := make(map[string]float64)
	features["total_energy"] = totalEnergy
	features["num_bands"] = float64(len(spectrum))

	lowFreqEnergy := 0.0
	midFreqEnergy := 0.0
	highFreqEnergy := 0.0

	n := len(spectrum)
	for i := 0; i < n/3; i++ {
		lowFreqEnergy += spectrum[i]
	}
	for i := n / 3; i < 2*n/3; i++ {
		midFreqEnergy += spectrum[i]
	}
	for i := 2 * n / 3; i < n; i++ {
		highFreqEnergy += spectrum[i]
	}

	if totalEnergy > 0 {
		features["low_freq_ratio"] = lowFreqEnergy / totalEnergy
		features["mid_freq_ratio"] = midFreqEnergy / totalEnergy
		features["high_freq_ratio"] = highFreqEnergy / totalEnergy
	}

	entropy := 0.0
	for _, e := range spectrum {
		if e > 0 && totalEnergy > 0 {
			p := e / totalEnergy
			entropy -= p * math.Log2(p)
		}
	}
	features["spectral_entropy"] = entropy

	maxEnergy := 0.0
	peakIndex := 0
	for i, e := range spectrum {
		if e > maxEnergy {
			maxEnergy = e
			peakIndex = i
		}
	}
	features["peak_energy"] = maxEnergy
	features["peak_band_index"] = float64(peakIndex)
	nyquist := samplingRate / 2.0
	bandwidth := nyquist / float64(n)
	features["peak_frequency"] = (float64(peakIndex) + 0.5) * bandwidth

	centroid := 0.0
	for i, e := range spectrum {
		freq := (float64(i) + 0.5) * bandwidth
		centroid += freq * e
	}
	if totalEnergy > 0 {
		features["spectral_centroid"] = centroid / totalEnergy
	}

	return features
}
