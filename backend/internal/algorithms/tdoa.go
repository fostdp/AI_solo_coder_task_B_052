package algorithms

import (
	"errors"
	"math"
	"sort"

	"ancient-wood-monitor/internal/models"
)

type TDOALocator struct {
	SoundSpeed       float64
	MinSensors       int
	NodeMergeDistance float64
	EdgeMaxDistance   float64
	MaxNodes         int
}

func NewTDOALocator(soundSpeed float64, minSensors int, mergeDist float64, edgeMaxDist float64, maxNodes int) *TDOALocator {
	return &TDOALocator{
		SoundSpeed:       soundSpeed,
		MinSensors:       minSensors,
		NodeMergeDistance: mergeDist,
		EdgeMaxDistance:   edgeMaxDist,
		MaxNodes:         maxNodes,
	}
}

func (l *TDOALocator) LocateSource(measurements []models.TDOAMeasurement) (x, y, z float64, confidence float64, err error) {
	if len(measurements) < l.MinSensors {
		return 0, 0, 0, 0, errors.New("insufficient sensors for TDOA localization")
	}

	ref := measurements[0]
	n := len(measurements) - 1

	p0x, p0y, p0z := ref.PosX, ref.PosY, ref.PosZ
	p0dot := p0x*p0x + p0y*p0y + p0z*p0z

	A := make([][]float64, n)
	b := make([]float64, n)

	for i := 0; i < n; i++ {
		m := measurements[i+1]
		timeDiff := m.Timestamp.Sub(ref.Timestamp).Seconds()
		di := timeDiff * l.SoundSpeed

		dx := m.PosX - p0x
		dy := m.PosY - p0y
		dz := m.PosZ - p0z
		pidot := m.PosX*m.PosX + m.PosY*m.PosY + m.PosZ*m.PosZ

		A[i] = []float64{2 * dx, 2 * dy, 2 * dz, 2 * di}
		b[i] = (pidot - p0dot) - di*di
	}

	ATA := make([][]float64, 4)
	for i := range ATA {
		ATA[i] = make([]float64, 4)
	}
	ATb := make([]float64, 4)

	for i := 0; i < n; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				ATA[j][k] += A[i][j] * A[i][k]
			}
			ATb[j] += A[i][j] * b[i]
		}
	}

	sol, solveErr := solve4x4(ATA, ATb)
	if solveErr != nil {
		return 0, 0, 0, 0, errors.New("singular matrix in TDOA least squares")
	}

	var residual float64
	for i := 0; i < n; i++ {
		pred := A[i][0]*sol[0] + A[i][1]*sol[1] + A[i][2]*sol[2] + A[i][3]*sol[3]
		diff := pred - b[i]
		residual += diff * diff
	}
	rmsResidual := math.Sqrt(residual / float64(n))
	confidence = 1.0 / (1.0 + rmsResidual)

	return sol[0], sol[1], sol[2], confidence, nil
}

func solve4x4(mat [][]float64, rhs []float64) ([]float64, error) {
	aug := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		aug[i] = make([]float64, 5)
		copy(aug[i][:4], mat[i])
		aug[i][4] = rhs[i]
	}

	const eps = 1e-12

	for col := 0; col < 4; col++ {
		maxRow := col
		maxVal := math.Abs(aug[col][col])
		for row := col + 1; row < 4; row++ {
			if math.Abs(aug[row][col]) > maxVal {
				maxVal = math.Abs(aug[row][col])
				maxRow = row
			}
		}
		if maxVal < eps {
			return nil, errors.New("singular matrix")
		}
		aug[col], aug[maxRow] = aug[maxRow], aug[col]

		pivot := aug[col][col]
		for j := col; j < 5; j++ {
			aug[col][j] /= pivot
		}

		for row := 0; row < 4; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for j := col; j < 5; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	return []float64{aug[0][4], aug[1][4], aug[2][4], aug[3][4]}, nil
}

func BuildTunnelNetwork(nodes []models.TunnelNode, edgeMaxDist float64) []models.TunnelEdge {
	var edges []models.TunnelEdge

	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			dx := nodes[i].PositionX - nodes[j].PositionX
			dy := nodes[i].PositionY - nodes[j].PositionY
			dz := nodes[i].PositionZ - nodes[j].PositionZ
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

			if dist <= edgeMaxDist {
				strength := 1.0 - dist/edgeMaxDist
				edges = append(edges, models.TunnelEdge{
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

func MergeNode(existingNodes []models.TunnelNode, newNode models.TunnelNode, mergeDist float64) ([]models.TunnelNode, bool) {
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
