package tdoa_locator

import (
	"errors"
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
)

type TDOALocator struct {
	SoundSpeed        float64
	MinSensors        int
	NodeMergeDistance float64
	EdgeMaxDistance   float64
	MaxNodes          int
}

func NewTDOALocator(soundSpeed float64, minSensors int, mergeDist float64, edgeMaxDist float64, maxNodes int) *TDOALocator {
	return &TDOALocator{
		SoundSpeed:        soundSpeed,
		MinSensors:        minSensors,
		NodeMergeDistance: mergeDist,
		EdgeMaxDistance:   edgeMaxDist,
		MaxNodes:          maxNodes,
	}
}

func (l *TDOALocator) LocateSource(measurements []TDOAMeasurement) (*LocatorResult, error) {
	if len(measurements) < l.MinSensors {
		return nil, errors.New("insufficient sensors for TDOA localization")
	}

	ref := measurements[0]
	n := len(measurements) - 1

	p0x, p0y, p0z := ref.PosX, ref.PosY, ref.PosZ
	p0dot := p0x*p0x + p0y*p0y + p0z*p0z

	A := make([]float64, n*4)
	b := make([]float64, n)
	weights := make([]float64, n)

	for i := 0; i < n; i++ {
		m := measurements[i+1]
		timeDiff := m.Timestamp.Sub(ref.Timestamp).Seconds()
		di := timeDiff * l.SoundSpeed

		dx := m.PosX - p0x
		dy := m.PosY - p0y
		dz := m.PosZ - p0z
		distFromRef := math.Sqrt(dx*dx + dy*dy + dz*dz)
		pidot := m.PosX*m.PosX + m.PosY*m.PosY + m.PosZ*m.PosZ

		A[i*4+0] = 2 * dx
		A[i*4+1] = 2 * dy
		A[i*4+2] = 2 * dz
		A[i*4+3] = 2 * di
		b[i] = (pidot - p0dot) - di*di

		amplitude := math.Max(1.0, m.Amplitude)
		weights[i] = amplitude / (1.0 + distFromRef*distFromRef)
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight > 0 {
		for i := range weights {
			weights[i] /= totalWeight
			weights[i] *= float64(n)
		}
	}

	ATA := mat.NewSymDense(4, nil)
	ATb := mat.NewVecDense(4, nil)

	for i := 0; i < n; i++ {
		w := weights[i]
		for j := 0; j < 4; j++ {
			Aij := A[i*4+j]
			ATb.SetVec(j, ATb.AtVec(j)+Aij*w*b[i])
			for k := j; k < 4; k++ {
				Aik := A[i*4+k]
				val := ATA.At(j, k) + Aij*w*Aik
				ATA.SetSym(j, k, val)
			}
		}
	}

	var sol mat.VecDense
	if err := sol.SolveVec(ATA, ATb); err != nil {
		return nil, errors.New("singular matrix in TDOA weighted least squares")
	}

	var residual float64
	for i := 0; i < n; i++ {
		pred := A[i*4+0]*sol.AtVec(0) + A[i*4+1]*sol.AtVec(1) + A[i*4+2]*sol.AtVec(2) + A[i*4+3]*sol.AtVec(3)
		diff := pred - b[i]
		residual += weights[i] * diff * diff
	}
	rmsResidual := math.Sqrt(residual / float64(n))
	confidence := 1.0 / (1.0 + rmsResidual)

	return &LocatorResult{
		X:          sol.AtVec(0),
		Y:          sol.AtVec(1),
		Z:          sol.AtVec(2),
		Confidence: confidence,
	}, nil
}

func BuildTunnelNetwork(nodes []TunnelNode, edgeMaxDist float64) []TunnelEdge {
	var edges []TunnelEdge

	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			dx := nodes[i].PositionX - nodes[j].PositionX
			dy := nodes[i].PositionY - nodes[j].PositionY
			dz := nodes[i].PositionZ - nodes[j].PositionZ
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

			if dist <= edgeMaxDist {
				strength := 1.0 - dist/edgeMaxDist
				edges = append(edges, TunnelEdge{
					FromNodeID: nodes[i].ID,
					ToNodeID:   nodes[j].ID,
					Length:     dist,
					Strength:   strength,
				})
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Strength > edges[j].Strength
	})

	return edges
}

func MergeNode(existingNodes []TunnelNode, newNode TunnelNode, mergeDist float64) ([]TunnelNode, bool) {
	for i := range existingNodes {
		dx := existingNodes[i].PositionX - newNode.PositionX
		dy := existingNodes[i].PositionY - newNode.PositionY
		dz := existingNodes[i].PositionZ - newNode.PositionZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

		if dist <= mergeDist {
			existingNodes[i].PositionX = (existingNodes[i].PositionX + newNode.PositionX) / 2.0
			existingNodes[i].PositionY = (existingNodes[i].PositionY + newNode.PositionY) / 2.0
			existingNodes[i].PositionZ = (existingNodes[i].PositionZ + newNode.PositionZ) / 2.0
			if newNode.LastSeen.After(existingNodes[i].LastSeen) {
				existingNodes[i].LastSeen = newNode.LastSeen
			}
			return existingNodes, true
		}
	}

	return append(existingNodes, newNode), false
}
