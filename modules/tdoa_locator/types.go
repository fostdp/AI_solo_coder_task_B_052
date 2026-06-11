package tdoa_locator

import "time"

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

type LocatorResult struct {
	X          float64
	Y          float64
	Z          float64
	Confidence float64
}
