package bird_drive

import "time"

type BirdRadarData struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	BirdCount     int       `json:"bird_count"`
	BirdType      string    `json:"bird_type"`
	Direction     float64   `json:"direction"`
	Distance      float64   `json:"distance"`
	Altitude      float64   `json:"altitude"`
	Speed         float64   `json:"speed"`
	ActivityLevel string    `json:"activity_level"`
}

type DeterrentAction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Building  string    `json:"building"`
	StartTime time.Time `json:"start_time"`
	Duration  float64   `json:"duration"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
	BirdCount int       `json:"bird_count"`
	BirdType  string    `json:"bird_type"`
}

type DeterrentStatus struct {
	ActiveDeterrents []*DeterrentAction `json:"active_deterrents"`
	RecentBirdCount  int                `json:"recent_bird_count"`
	WoodpeckerCount  int                `json:"woodpecker_count"`
	ActivityLevel    string             `json:"activity_level"`
}
