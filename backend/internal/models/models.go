package models

import "time"

type AcousticEmissionData struct {
	SensorID    string    `json:"sensor_id"`
	Building    string    `json:"building"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
	EventCount  int       `json:"event_count"`
	Energy      float64   `json:"energy"`
	Amplitude   float64   `json:"amplitude"`
	Duration    float64   `json:"duration"`
	RiseTime    float64   `json:"rise_time"`
	Counts      int       `json:"counts"`
	FrequencyPeak float64 `json:"frequency_peak"`
}

type WoodMoistureData struct {
	SensorID    string    `json:"sensor_id"`
	Building    string    `json:"building"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
	Moisture    float64   `json:"moisture"`
	Temperature float64   `json:"temperature"`
}

type LoRaDataPacket struct {
	PacketID      string                 `json:"packet_id"`
	DeviceType    string                 `json:"device_type"`
	DeviceID      string                 `json:"device_id"`
	Timestamp     time.Time              `json:"timestamp"`
	Sequence      uint64                 `json:"sequence"`
	Data          map[string]interface{} `json:"data"`
	RSSI          float64                `json:"rssi"`
	SNR           float64                `json:"snr"`
	SpreadingFactor int                  `json:"spreading_factor"`
}

type SensorInfo struct {
	SensorID string  `json:"sensor_id"`
	Type     string  `json:"type"`
	Building string  `json:"building"`
	Location string  `json:"location"`
	PosX     float64 `json:"pos_x"`
	PosY     float64 `json:"pos_y"`
	PosZ     float64 `json:"pos_z"`
	Status   string  `json:"status"`
}

type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	SensorID    string    `json:"sensor_id"`
	Building    string    `json:"building"`
	Location    string    `json:"location"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Acknowledged bool     `json:"acknowledged"`
}

type TermiteActivityPrediction struct {
	Building      string    `json:"building"`
	Location      string    `json:"location"`
	Timestamp     time.Time `json:"timestamp"`
	ActivityLevel float64   `json:"activity_level"`
	RiskLevel     string    `json:"risk_level"`
	Confidence    float64   `json:"confidence"`
}

type TermitePredictionResult struct {
	Timestamp     time.Time `json:"timestamp"`
	ActivityLevel float64   `json:"activity_level"`
	RiskLevel     string    `json:"risk_level"`
	Confidence    float64   `json:"confidence"`
	Trend         string    `json:"trend"`
}

type FumigationSimulationRequest struct {
	Building      string    `json:"building"`
	ReleasePointX float64   `json:"release_point_x"`
	ReleasePointY float64   `json:"release_point_y"`
	ReleasePointZ float64   `json:"release_point_z"`
	ReleaseRate   float64   `json:"release_rate"`
	WindSpeed     float64   `json:"wind_speed"`
	WindDirection float64   `json:"wind_direction"`
	Duration      float64   `json:"duration"`
}

type FumigationResult struct {
	GridX     int       `json:"grid_x"`
	GridY     int       `json:"grid_y"`
	GridZ     int       `json:"grid_z"`
	PosX      float64   `json:"pos_x"`
	PosY      float64   `json:"pos_y"`
	PosZ      float64   `json:"pos_z"`
	Concentration float64 `json:"concentration"`
	Timestamp time.Time `json:"timestamp"`
}

type WaveletPacketEnergy struct {
	Level     int       `json:"level"`
	NodeIndex int       `json:"node_index"`
	Energy    float64   `json:"energy"`
	FrequencyRange string `json:"frequency_range"`
}

type TDOAMeasurement struct {
	SensorID  string    `json:"sensor_id"`
	Timestamp time.Time `json:"timestamp"`
	PosX      float64   `json:"pos_x"`
	PosY      float64   `json:"pos_y"`
	PosZ      float64   `json:"pos_z"`
	Amplitude float64   `json:"amplitude"`
}

type TunnelNode struct {
	ID         string    `json:"id"`
	PositionX  float64   `json:"position_x"`
	PositionY  float64   `json:"position_y"`
	PositionZ  float64   `json:"position_z"`
	Building   string    `json:"building"`
	Confidence float64   `json:"confidence"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	Active     bool      `json:"active"`
}

type TunnelEdge struct {
	FromNodeID string  `json:"from_node_id"`
	ToNodeID   string  `json:"to_node_id"`
	Length     float64 `json:"length"`
	Strength   float64 `json:"strength"`
}

type TunnelNetwork struct {
	Building  string       `json:"building"`
	Nodes     []TunnelNode `json:"nodes"`
	Edges     []TunnelEdge `json:"edges"`
	UpdatedAt time.Time    `json:"updated_at"`
}

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

type ParticleState struct {
	ActivityLevel float64   `json:"activity_level"`
	Trend         float64   `json:"trend"`
	Weight        float64   `json:"weight"`
	Timestamp     time.Time `json:"timestamp"`
}

type ParticleFilterOutput struct {
	Building           string          `json:"building"`
	Particles          []ParticleState `json:"particles"`
	PredictedPeakTime  time.Time       `json:"predicted_peak_time"`
	OptimalReleaseTime time.Time       `json:"optimal_release_time"`
	CurrentActivity    float64         `json:"current_activity"`
	PredictedPeak      float64         `json:"predicted_peak"`
	Confidence         float64         `json:"confidence"`
	ShouldReleaseNow   bool            `json:"should_release_now"`
}

type BirdRadarData struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	BirdCount    int       `json:"bird_count"`
	BirdType     string    `json:"bird_type"`
	Direction    float64   `json:"direction"`
	Distance     float64   `json:"distance"`
	Altitude     float64   `json:"altitude"`
	Speed        float64   `json:"speed"`
	ActivityLevel string  `json:"activity_level"`
}

type DeterrentAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Building    string    `json:"building"`
	StartTime   time.Time `json:"start_time"`
	Duration    float64   `json:"duration"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	BirdCount   int       `json:"bird_count"`
	BirdType    string    `json:"bird_type"`
}
