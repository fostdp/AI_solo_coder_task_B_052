package services

import (
	"ancient-wood-monitor/internal/models"
	"fmt"
	"math"
	"math/rand"
	"sync"
)

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

type SensorService struct {
	sensors map[string]*models.SensorInfo
	mu      sync.RWMutex
}

func NewSensorService() *SensorService {
	s := &SensorService{
		sensors: make(map[string]*models.SensorInfo),
	}
	s.initializeSensors()
	return s
}

func (s *SensorService) initializeSensors() {
	buildings := []string{"应县木塔", "佛光寺"}

	for _, building := range buildings {
		acousticCount := 25
		moistureCount := 20

		if building == "应县木塔" {
			acousticCount = 30
			moistureCount = 25
		}

		for i := 0; i < acousticCount; i++ {
			sensorID := s.generateSensorID("AC", building, i+1)
			sensor := &models.SensorInfo{
				SensorID: sensorID,
				Type:     "acoustic_emission",
				Building: building,
				Location: s.getLocationDescription(building, i, acousticCount),
				PosX:     randFloat(-10, 10),
				PosY:     randFloat(0, 20),
				PosZ:     randFloat(0, 60),
				Status:   "online",
			}
			s.sensors[sensorID] = sensor
		}

		for i := 0; i < moistureCount; i++ {
			sensorID := s.generateSensorID("MS", building, i+1)
			sensor := &models.SensorInfo{
				SensorID: sensorID,
				Type:     "wood_moisture",
				Building: building,
				Location: s.getLocationDescription(building, i, moistureCount),
				PosX:     randFloat(-8, 8),
				PosY:     randFloat(0, 15),
				PosZ:     randFloat(0, 50),
				Status:   "online",
			}
			s.sensors[sensorID] = sensor
		}
	}
}

func (s *SensorService) generateSensorID(prefix, building string, num int) string {
	buildingCode := "YMT"
	if building == "佛光寺" {
		buildingCode = "FGS"
	}
	return prefix + "-" + buildingCode + "-" + fmt.Sprintf("%03d", num)
}

func (s *SensorService) getLocationDescription(building string, index, total int) string {
	locations := []string{
		"一层斗拱", "二层斗拱", "三层斗拱", "四层斗拱", "五层斗拱",
		"东立柱", "西立柱", "南立柱", "北立柱",
		"主梁", "次梁", "横梁",
		"东侧墙", "西侧墙", "南侧墙", "北侧墙",
		"塔顶", "塔基",
	}

	if building == "佛光寺" {
		locations = []string{
			"东大殿斗拱", "东大殿立柱", "东大殿主梁",
			"文殊殿斗拱", "文殊殿立柱", "文殊殿主梁",
			"山门", "钟楼", "鼓楼",
			"配殿", "藏经楼",
		}
	}

	return locations[index%len(locations)]
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (s *SensorService) GetAllSensors() []*models.SensorInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sensors := make([]*models.SensorInfo, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		sensors = append(sensors, sensor)
	}
	return sensors
}

func (s *SensorService) GetSensorsByBuilding(building string) []*models.SensorInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sensors []*models.SensorInfo
	for _, sensor := range s.sensors {
		if sensor.Building == building {
			sensors = append(sensors, sensor)
		}
	}
	return sensors
}

func (s *SensorService) GetSensorsByType(sensorType string) []*models.SensorInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sensors []*models.SensorInfo
	for _, sensor := range s.sensors {
		if sensor.Type == sensorType {
			sensors = append(sensors, sensor)
		}
	}
	return sensors
}

func (s *SensorService) GetSensorByID(sensorID string) (*models.SensorInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sensor, exists := s.sensors[sensorID]
	return sensor, exists
}

func (s *SensorService) UpdateSensorStatus(sensorID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sensor, exists := s.sensors[sensorID]; exists {
		sensor.Status = status
	}
}

func (s *SensorService) GetSensorCount() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acousticCount := 0
	moistureCount := 0

	for _, sensor := range s.sensors {
		if sensor.Type == "acoustic_emission" {
			acousticCount++
		} else if sensor.Type == "wood_moisture" {
			moistureCount++
		}
	}

	return acousticCount, moistureCount
}

func (s *SensorService) GetBuildings() []string {
	return []string{"应县木塔", "佛光寺"}
}

func (s *SensorService) GetBuildingInfo(building string) map[string]interface{} {
	sensors := s.GetSensorsByBuilding(building)

	acousticCount := 0
	moistureCount := 0
	for _, sensor := range sensors {
		if sensor.Type == "acoustic_emission" {
			acousticCount++
		} else {
			moistureCount++
		}
	}

	var height float64
	var floors int

	if building == "应县木塔" {
		height = 67.31
		floors = 5
	} else {
		height = 20.0
		floors = 1
	}

	return map[string]interface{}{
		"name":           building,
		"height":         height,
		"floors":         floors,
		"acoustic_sensors": acousticCount,
		"moisture_sensors": moistureCount,
		"total_sensors":  len(sensors),
		"description":    getBuildingDescription(building),
	}
}

func getBuildingDescription(building string) string {
	if building == "应县木塔" {
		return "应县木塔，全称佛宫寺释迦塔，位于山西省朔州市应县城西北佛宫寺内，建于辽清宁二年（1056年），是中国现存最高最古的一座木构塔式建筑。"
	}
	return "佛光寺，位于山西省五台县，建于唐大中十一年（857年），是中国现存排名第三早的木结构建筑。"
}

func (s *SensorService) GetRiskZones(building string, eventRates map[string]float64) []map[string]interface{} {
	sensors := s.GetSensorsByBuilding(building)
	var zones []map[string]interface{}

	for _, sensor := range sensors {
		if sensor.Type != "acoustic_emission" {
			continue
		}

		eventRate, ok := eventRates[sensor.SensorID]
		if !ok {
			eventRate = 0
		}

		riskLevel := getRiskLevel(eventRate)

		zone := map[string]interface{}{
			"sensor_id":  sensor.SensorID,
			"building":   building,
			"location":   sensor.Location,
			"pos_x":      sensor.PosX,
			"pos_y":      sensor.PosY,
			"pos_z":      sensor.PosZ,
			"radius":     1.0 + eventRate/50.0,
			"risk_level": riskLevel,
			"event_rate": eventRate,
			"intensity":  math.Min(1.0, eventRate/150.0),
		}

		zones = append(zones, zone)
	}

	return zones
}
