package tdoa_strength

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"ancient-wood-monitor/internal/algorithms"
	"ancient-wood-monitor/internal/models"
	"ancient-wood-monitor/internal/pipeline"

	tdoa "github.com/ancient-wood/tdoa_locator"
	strength "github.com/ancient-wood/strength_calc"
	pf "github.com/ancient-wood/particle_filter_timing"
)

type Config struct {
	SoundSpeedWood       float64       `yaml:"sound_speed_wood"`
	MinSensors           int           `yaml:"min_sensors"`
	NodeMergeDistance     float64       `yaml:"node_merge_distance"`
	EdgeMaxDistance       float64       `yaml:"edge_max_distance"`
	MaxNodes             int           `yaml:"max_nodes"`
	DefaultWoodType      string        `yaml:"default_wood_type"`
	ReferenceDensity     float64       `yaml:"reference_density"`
	CriticalEnergy       float64       `yaml:"critical_energy"`
	RequiredSafetyFactor float64       `yaml:"required_safety_factor"`
	DepthRatioDefault    float64       `yaml:"depth_ratio_default"`
	MinParticles         int           `yaml:"min_particles"`
	MaxParticles         int           `yaml:"max_particles"`
	InitialParticles     int           `yaml:"initial_particles"`
	ProcessNoise         float64       `yaml:"process_noise"`
	MeasurementNoise     float64       `yaml:"measurement_noise"`
	ResampleThreshold    float64       `yaml:"resample_threshold"`
	ReleaseLeadTime      time.Duration `yaml:"release_lead_time"`
	PredictionHorizon    time.Duration `yaml:"prediction_horizon"`
	ESSIncreaseThreshold float64       `yaml:"ess_increase_threshold"`
	ESSDecreaseThreshold float64       `yaml:"ess_decrease_threshold"`
}

type TDOAStrengthService struct {
	cfg            Config
	locator        *tdoa.TDOALocator
	evaluator      *strength.WoodStrengthEvaluator
	particleFilter *pf.TermiteParticleFilter
	tunnelNetworks map[string]*models.TunnelNetwork
	cumulativeEnergy map[string]float64
	mu             sync.RWMutex
	name           string
}

func NewService(cfg Config) *TDOAStrengthService {
	if cfg.DefaultWoodType == "" {
		cfg.DefaultWoodType = "pine"
	}
	if cfg.MinParticles == 0 {
		cfg.MinParticles = 50
	}
	if cfg.MaxParticles == 0 {
		cfg.MaxParticles = 500
	}
	if cfg.InitialParticles == 0 {
		cfg.InitialParticles = 100
	}
	if cfg.ESSIncreaseThreshold == 0 {
		cfg.ESSIncreaseThreshold = 0.5
	}
	if cfg.ESSDecreaseThreshold == 0 {
		cfg.ESSDecreaseThreshold = 0.9
	}
	return &TDOAStrengthService{
		cfg:            cfg,
		locator:        tdoa.NewTDOALocator(cfg.SoundSpeedWood, cfg.MinSensors, cfg.NodeMergeDistance, cfg.EdgeMaxDistance, cfg.MaxNodes),
		evaluator:      strength.NewWoodStrengthEvaluator(cfg.ReferenceDensity, cfg.CriticalEnergy, cfg.RequiredSafetyFactor, cfg.DepthRatioDefault),
		particleFilter: pf.NewTermiteParticleFilter(cfg.InitialParticles, cfg.MinParticles, cfg.MaxParticles, cfg.ESSIncreaseThreshold, cfg.ESSDecreaseThreshold, cfg.ProcessNoise, cfg.MeasurementNoise, cfg.ResampleThreshold, cfg.ReleaseLeadTime, cfg.PredictionHorizon),
		tunnelNetworks: make(map[string]*models.TunnelNetwork),
		cumulativeEnergy: make(map[string]float64),
		name:           "tdoa_strength",
	}
}

func (s *TDOAStrengthService) Name() string {
	return s.name
}

func (s *TDOAStrengthService) Start(ctx context.Context, in <-chan pipeline.PipelineMessage, out chan<- pipeline.PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}

			if msg.Type != pipeline.MsgTypeTermitePrediction {
				out <- msg
				continue
			}

			processed, err := s.process(ctx, &msg)
			if err != nil {
				msg.Err = err
				out <- msg
				continue
			}

			if processed != nil {
				out <- *processed
			}
		}
	}
}

func (s *TDOAStrengthService) process(ctx context.Context, msg *pipeline.PipelineMessage) (*pipeline.PipelineMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	termiteOutput, ok := msg.Data.(pipeline.TermiteOutput)
	if !ok {
		return msg, nil
	}

	if len(termiteOutput.Predictions) == 0 {
		return nil, nil
	}

	now := time.Now()
	building := termiteOutput.Building
	sensorID := termiteOutput.SensorID
	location := termiteOutput.Location

	measurements := s.generateSyntheticTDOA(termiteOutput, now)

	locResult, err := s.locator.LocateSource(measurements)
	if err != nil {
		log.Printf("[%s] TDOA localization failed for %s: %v", s.name, sensorID, err)
		return nil, nil
	}

	newNode := models.TunnelNode{
		ID:         fmt.Sprintf("node-%s-%d", building, now.UnixMilli()),
		PositionX:  locResult.X,
		PositionY:  locResult.Y,
		PositionZ:  locResult.Z,
		Building:   building,
		Confidence: locResult.Confidence,
		FirstSeen:  now,
		LastSeen:   now,
		Active:     true,
	}

	s.mu.Lock()
	network, exists := s.tunnelNetworks[building]
	if !exists {
		network = &models.TunnelNetwork{
			Building:  building,
			Nodes:     []models.TunnelNode{},
			Edges:     []models.TunnelEdge{},
			UpdatedAt: now,
		}
		s.tunnelNetworks[building] = network
	}

	mergedNodes, wasMerged := algorithms.MergeNode(network.Nodes, newNode, s.cfg.NodeMergeDistance)
	network.Nodes = mergedNodes
	if len(network.Nodes) > s.cfg.MaxNodes {
		network.Nodes = network.Nodes[len(network.Nodes)-s.cfg.MaxNodes:]
	}

	network.Edges = algorithms.BuildTunnelNetwork(network.Nodes, s.cfg.EdgeMaxDistance)
	network.UpdatedAt = now

	if wasMerged {
		log.Printf("[%s] merged node into existing for building %s", s.name, building)
	}

	energyKey := building + "-" + sensorID
	s.cumulativeEnergy[energyKey] += termiteOutput.SmoothedRate

	woodDensity := algorithms.SimulateWoodDensity(450, 968, 20)

	woodType := s.getWoodTypeForBuilding(building)

	strengthResultMod := s.evaluator.AssessStrength(
		sensorID,
		building,
		location,
		woodType,
		s.cumulativeEnergy[energyKey],
		woodDensity,
		s.cfg.DepthRatioDefault,
	)
	strengthResult := toModelStrengthAssessment(strengthResultMod)
	s.mu.Unlock()

	pfOutputMod := s.particleFilter.PredictSync(termiteOutput.SmoothedRate)
	pfOutputMod.Building = building
	pfOutput := toModelParticleFilterOutput(pfOutputMod)

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeTDOAStrength,
		Metadata: pipeline.Metadata{
			MessageID: msg.Metadata.MessageID,
			Timestamp: now,
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: pipeline.TDOAStrengthOutput{
			TunnelNetwork:   *network,
			StrengthResults: []models.WoodStrengthAssessment{strengthResult},
			SensorID:        sensorID,
			Building:        building,
			Location:        location,
			RiskLevel:       termiteOutput.RiskLevel,
			ParticleFilter:  &pfOutput,
		},
	}, nil
}

func (s *TDOAStrengthService) generateSyntheticTDOA(output pipeline.TermiteOutput, baseTime time.Time) []tdoa.TDOAMeasurement {
	sensorPositions := [][3]float64{
		{0, 0, 0},
		{5, 0, 2},
		{0, 5, 1},
		{5, 5, 3},
		{2.5, 2.5, 0},
	}

	n := 4 + int(math.Mod(float64(baseTime.Unix()), 3))
	if n > len(sensorPositions) {
		n = len(sensorPositions)
	}

	measurements := make([]tdoa.TDOAMeasurement, n)
	for i := 0; i < n; i++ {
		measurements[i] = tdoa.TDOAMeasurement{
			SensorID:  fmt.Sprintf("synth-tdoa-%d", i),
			Timestamp: baseTime.Add(time.Duration(i) * time.Microsecond),
			PosX:      sensorPositions[i][0],
			PosY:      sensorPositions[i][1],
			PosZ:      sensorPositions[i][2],
			Amplitude: output.SmoothedRate * float64(n-i),
		}
	}

	return measurements
}

func toModelStrengthAssessment(s strength.WoodStrengthAssessment) models.WoodStrengthAssessment {
	return models.WoodStrengthAssessment{
		SensorID:              s.SensorID,
		Building:              s.Building,
		Location:              s.Location,
		WoodType:              s.WoodType,
		CumulativeEnergy:      s.CumulativeEnergy,
		WoodDensity:           s.WoodDensity,
		DamageIndex:           s.DamageIndex,
		ResidualStrengthIndex: s.ResidualStrengthIndex,
		SafetyFactor:          s.SafetyFactor,
		StrengthLevel:         s.StrengthLevel,
		Timestamp:             s.Timestamp,
	}
}

func toModelParticleFilterOutput(p pf.ParticleFilterOutput) models.ParticleFilterOutput {
	particles := make([]models.ParticleState, len(p.Particles))
	for i, ps := range p.Particles {
		particles[i] = models.ParticleState{
			ActivityLevel: ps.ActivityLevel,
			Trend:         ps.Trend,
			Weight:        ps.Weight,
			Timestamp:     ps.Timestamp,
		}
	}
	return models.ParticleFilterOutput{
		Building:           p.Building,
		Particles:          particles,
		PredictedPeakTime:  p.PredictedPeakTime,
		OptimalReleaseTime: p.OptimalReleaseTime,
		CurrentActivity:    p.CurrentActivity,
		PredictedPeak:      p.PredictedPeak,
		Confidence:         p.Confidence,
		ShouldReleaseNow:   p.ShouldReleaseNow,
	}
}

func modelToTdoaMeasurement(m []models.TDOAMeasurement) []tdoa.TDOAMeasurement {
	result := make([]tdoa.TDOAMeasurement, len(m))
	for i, mm := range m {
		result[i] = tdoa.TDOAMeasurement{
			SensorID:  mm.SensorID,
			Timestamp: mm.Timestamp,
			PosX:      mm.PosX,
			PosY:      mm.PosY,
			PosZ:      mm.PosZ,
			Amplitude: mm.Amplitude,
		}
	}
	return result
}

func (s *TDOAStrengthService) GetTunnelNetwork(building string) *models.TunnelNetwork {
	s.mu.RLock()
	defer s.mu.RUnlock()

	network, exists := s.tunnelNetworks[building]
	if !exists {
		return nil
	}

	cp := *network
	cp.Nodes = make([]models.TunnelNode, len(network.Nodes))
	copy(cp.Nodes, network.Nodes)
	cp.Edges = make([]models.TunnelEdge, len(network.Edges))
	copy(cp.Edges, network.Edges)
	return &cp
}

func (s *TDOAStrengthService) GetStrengthAssessments(building string) []models.WoodStrengthAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	woodDensity := algorithms.SimulateWoodDensity(450, 968, 20)
	woodType := s.getWoodTypeForBuilding(building)

	var results []models.WoodStrengthAssessment
	for key, energy := range s.cumulativeEnergy {
		var sensorID string
		var loc string
		fmt.Sscanf(key, building+"-%s", &sensorID)
		modResult := s.evaluator.AssessStrength(
			sensorID,
			building,
			loc,
			woodType,
			energy,
			woodDensity,
			s.cfg.DepthRatioDefault,
		)
		results = append(results, toModelStrengthAssessment(modResult))
	}

	return results
}

func (s *TDOAStrengthService) getWoodTypeForBuilding(building string) string {
	switch building {
	case "应县木塔":
		return "pine"
	case "佛光寺":
		return "nanmu"
	default:
		if s.cfg.DefaultWoodType != "" {
			return s.cfg.DefaultWoodType
		}
		return "pine"
	}
}

func (s *TDOAStrengthService) GetParticleFilterOutput(building string) *models.ParticleFilterOutput {
	s.mu.RLock()
	var activity float64
	for key, energy := range s.cumulativeEnergy {
		if len(building) > 0 && len(key) > len(building)+1 && key[:len(building)] == building {
			activity += energy
			break
		}
	}
	s.mu.RUnlock()

	outputMod := s.particleFilter.PredictSync(activity)
	outputMod.Building = building
	output := toModelParticleFilterOutput(outputMod)
	return &output
}
