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
