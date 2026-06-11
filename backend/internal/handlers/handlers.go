package handlers

import (
	"ancient-wood-monitor/internal/algorithms"
	"ancient-wood-monitor/internal/algorithms/lstm"
	"ancient-wood-monitor/internal/models"
	pipe "ancient-wood-monitor/internal/pipeline"
	"ancient-wood-monitor/internal/services"
	lorasvc "ancient-wood-monitor/internal/services/lora"
	birddet "ancient-wood-monitor/internal/services/bird_deterrent"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	influxDB       *services.InfluxDBService
	alertService   *services.AlertService
	sensorService  *services.SensorService
	dedupService   *lorasvc.PacketDeduplicator
	acousticEWMA   *lstm.EWMASmoother
	moistureEWMA   *lstm.EWMASmoother
	pipeline       *pipe.ServicePipeline
	birdDeterrent  *birddet.BirdDeterrentService
}

func NewHandler(influxDB *services.InfluxDBService, alertService *services.AlertService, sensorService *services.SensorService, pipeline *pipe.ServicePipeline, birdDeterrent *birddet.BirdDeterrentService) *Handler {
	return &Handler{
		influxDB:      influxDB,
		alertService:  alertService,
		sensorService: sensorService,
		dedupService:  lorasvc.NewPacketDeduplicator(100000, 24*time.Hour, 50000),
		acousticEWMA:  lstm.NewEWMASmoother(0.3, 48),
		moistureEWMA:  lstm.NewEWMASmoother(0.25, 48),
		pipeline:      pipeline,
		birdDeterrent: birdDeterrent,
	}
}

func (h *Handler) ReceiveLoRaData(c *gin.Context) {
	var packet models.LoRaDataPacket
	if err := c.ShouldBindJSON(&packet); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dedupResult := h.dedupService.CheckPacket(packet.PacketID, packet.DeviceID, packet.Timestamp)
	if dedupResult.IsDuplicate {
		c.JSON(http.StatusConflict, gin.H{
			"status":     "duplicate_dropped",
			"packet_id":  dedupResult.PacketID,
			"reason":     dedupResult.Reason,
		})
		return
	}

	switch packet.DeviceType {
	case "acoustic_emission":
		data := &models.AcousticEmissionData{
			SensorID:      packet.DeviceID,
			Building:      packet.Data["building"].(string),
			Location:      packet.Data["location"].(string),
			Timestamp:     packet.Timestamp,
			EventCount:    int(packet.Data["event_count"].(float64)),
			Energy:        packet.Data["energy"].(float64),
			Amplitude:     packet.Data["amplitude"].(float64),
			Duration:      packet.Data["duration"].(float64),
			RiseTime:      packet.Data["rise_time"].(float64),
			Counts:        int(packet.Data["counts"].(float64)),
			FrequencyPeak: packet.Data["frequency_peak"].(float64),
		}

		if err := h.influxDB.WriteAcousticEmission(data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rawEventRate := float64(data.EventCount)
		smoothedEventRate := h.acousticEWMA.Smooth(data.SensorID, rawEventRate)

		h.sendToPipeline(packet)

		if alert, err := h.alertService.CheckAcousticAlert(data.SensorID, data.Building, data.Location, smoothedEventRate); err == nil && alert != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":          "received",
				"packet_id":       dedupResult.PacketID,
				"alert_triggered": true,
				"alert":           alert,
				"smoothed_rate":   smoothedEventRate,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          "received",
			"packet_id":       dedupResult.PacketID,
			"alert_triggered": false,
			"smoothed_rate":   smoothedEventRate,
		})
		return

	case "wood_moisture":
		data := &models.WoodMoistureData{
			SensorID:    packet.DeviceID,
			Building:    packet.Data["building"].(string),
			Location:    packet.Data["location"].(string),
			Timestamp:   packet.Timestamp,
			Moisture:    packet.Data["moisture"].(float64),
			Temperature: packet.Data["temperature"].(float64),
		}

		if err := h.influxDB.WriteWoodMoisture(data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		smoothedMoisture := h.moistureEWMA.Smooth(data.SensorID, data.Moisture)

		h.sendToPipeline(packet)

		if alert, err := h.alertService.CheckMoistureAlert(data.SensorID, data.Building, data.Location, smoothedMoisture); err == nil && alert != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":          "received",
				"packet_id":       dedupResult.PacketID,
				"alert_triggered": true,
				"alert":           alert,
				"smoothed_moisture": smoothedMoisture,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":            "received",
			"packet_id":        dedupResult.PacketID,
			"alert_triggered":  false,
			"smoothed_moisture": smoothedMoisture,
		})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown device type"})
		return
	}
}

func (h *Handler) GetSensors(c *gin.Context) {
	building := c.Query("building")
	sensorType := c.Query("type")

	var sensors []*models.SensorInfo

	if building != "" {
		sensors = h.sensorService.GetSensorsByBuilding(building)
	} else if sensorType != "" {
		sensors = h.sensorService.GetSensorsByType(sensorType)
	} else {
		sensors = h.sensorService.GetAllSensors()
	}

	c.JSON(http.StatusOK, gin.H{"sensors": sensors, "count": len(sensors)})
}

func (h *Handler) GetSensor(c *gin.Context) {
	sensorID := c.Param("id")
	sensor, exists := h.sensorService.GetSensorByID(sensorID)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "sensor not found"})
		return
	}

	c.JSON(http.StatusOK, sensor)
}

func (h *Handler) GetBuildings(c *gin.Context) {
	buildings := h.sensorService.GetBuildings()
	buildingInfos := make([]map[string]interface{}, 0, len(buildings))

	for _, building := range buildings {
		info := h.sensorService.GetBuildingInfo(building)
		buildingInfos = append(buildingInfos, info)
	}

	c.JSON(http.StatusOK, gin.H{"buildings": buildingInfos})
}

func (h *Handler) GetAcousticData(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}

	data, err := h.influxDB.QueryAcousticEvents(building, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "count": len(data)})
}

func (h *Handler) GetMoistureData(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}

	data, err := h.influxDB.QueryMoistureHistory(building, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "count": len(data)})
}

func (h *Handler) GetAlerts(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	alerts, err := h.alertService.GetActiveAlerts(building)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

func (h *Handler) GetRiskZones(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	acousticSensors := h.sensorService.GetSensorsByBuilding(building)
	eventRates := make(map[string]float64)

	for _, sensor := range acousticSensors {
		if sensor.Type == "acoustic_emission" {
			rate, _ := h.influxDB.QueryHourlyAcousticEventRate(sensor.SensorID, 1)
			eventRates[sensor.SensorID] = rate
		}
	}

	zones := h.sensorService.GetRiskZones(building, eventRates)
	c.JSON(http.StatusOK, gin.H{"risk_zones": zones, "count": len(zones)})
}

func (h *Handler) PredictTermiteActivity(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil {
		hours = 24
	}

	historicalData := make([]map[string]float64, 0)
	for i := 24; i >= 1; i-- {
		data := map[string]float64{
			"event_count":   30 + 40*float64(i)/24 + 20*float64(i%4),
			"energy":        500 + 300*float64(i)/24,
			"amplitude":     60 + 20*float64(i%6),
			"duration":      5 + 3*float64(i%5),
			"peak_freq":     3000 + 500*float64(i%8),
		}
		historicalData = append(historicalData, data)
	}

	predictions, err := algorithms.PredictTermiteActivity(historicalData, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"building":     building,
		"predictions":  predictions,
		"hours_ahead":  hours,
	})
}

func (h *Handler) SimulateFumigation(c *gin.Context) {
	var req models.FumigationSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Building == "" {
		req.Building = "应县木塔"
	}
	if req.ReleaseRate == 0 {
		req.ReleaseRate = 5.0
	}
	if req.WindSpeed == 0 {
		req.WindSpeed = 2.0
	}
	if req.Duration == 0 {
		req.Duration = 120.0
	}

	gridSizeX := 20
	gridSizeY := 20
	gridSizeZ := 15
	cellSize := 1.0

	result := algorithms.SimulateFumigation(
		req.ReleasePointX,
		req.ReleasePointY,
		req.ReleasePointZ,
		req.ReleaseRate,
		req.WindSpeed,
		req.WindDirection,
		gridSizeX,
		gridSizeY,
		gridSizeZ,
		cellSize,
		"D",
		req.Duration,
	)

	c.JSON(http.StatusOK, gin.H{
		"result":  result,
		"request": req,
	})
}

func (h *Handler) GetWaveletAnalysis(c *gin.Context) {
	sensorID := c.Query("sensor_id")

	signal := generateTestSignal(1024, 10000)
	features := algorithms.ExtractWaveletFeatures(signal, 10000)

	wp := algorithms.NewWaveletPacket(signal, 5, 10000)
	spectrum := wp.GetEnergySpectrum()
	freqRanges := wp.GetFrequencyRanges()

	waveletData := make([]models.WaveletPacketEnergy, len(spectrum))
	for i, energy := range spectrum {
		waveletData[i] = models.WaveletPacketEnergy{
			Level:          5,
			NodeIndex:      i,
			Energy:         energy,
			FrequencyRange: freqRanges[i],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sensor_id": sensorID,
		"features":  features,
		"spectrum":  waveletData,
	})
}

func generateTestSignal(length int, samplingRate float64) []float64 {
	signal := make([]float64, length)
	for i := 0; i < length; i++ {
		t := float64(i) / samplingRate
		signal[i] = 1.0*math.Sin(2*math.Pi*500*t) +
			0.5*math.Sin(2*math.Pi*1500*t) +
			0.3*math.Sin(2*math.Pi*3000*t) +
			0.1*rand.Float64()
	}
	return signal
}

func (h *Handler) sendToPipeline(packet models.LoRaDataPacket) {
	if h.pipeline == nil || h.pipeline.Input == nil {
		return
	}

	msg := pipe.PipelineMessage{
		Type: pipe.MsgTypeRawLoRa,
		Metadata: pipe.Metadata{
			Timestamp: time.Now(),
			Source:    "http_handler",
			TraceID:   packet.PacketID,
		},
		Data: packet,
	}

	select {
	case h.pipeline.Input <- msg:
	default:
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	status := "ok"
	pipelineStatus := "running"

	if h.pipeline == nil {
		pipelineStatus = "not_initialized"
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"version":   "2.0.0",
		"timestamp": time.Now().UTC(),
		"services": gin.H{
			"influxdb":       h.influxDB != nil,
			"pipeline":       pipelineStatus,
			"alerter":        h.alertService != nil,
			"sensors":        h.sensorService != nil,
			"bird_deterrent": h.birdDeterrent != nil,
		},
	})
}

func (h *Handler) GetTunnelNetwork(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	if h.pipeline == nil || h.pipeline.TDOAStrength == nil {
		network := h.generateMockTunnelNetwork(building)
		c.JSON(http.StatusOK, gin.H{"tunnel_network": network, "source": "mock"})
		return
	}

	network := h.pipeline.TDOAStrength.GetTunnelNetwork(building)
	if network == nil || len(network.Nodes) == 0 {
		network = h.generateMockTunnelNetwork(building)
		c.JSON(http.StatusOK, gin.H{"tunnel_network": network, "source": "mock"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tunnel_network": network, "source": "live"})
}

func (h *Handler) GetStrengthAssessment(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	if h.pipeline == nil || h.pipeline.TDOAStrength == nil {
		assessments := h.generateMockStrengthAssessments(building)
		c.JSON(http.StatusOK, gin.H{"assessments": assessments, "count": len(assessments), "source": "mock"})
		return
	}

	assessments := h.pipeline.TDOAStrength.GetStrengthAssessments(building)
	c.JSON(http.StatusOK, gin.H{"assessments": assessments, "count": len(assessments), "source": "live"})
}

func (h *Handler) GetFumigationTiming(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	if h.pipeline == nil || h.pipeline.TDOAStrength == nil {
		output := h.generateMockParticleFilterOutput(building)
		c.JSON(http.StatusOK, gin.H{"particle_filter": output, "source": "mock"})
		return
	}

	output := h.pipeline.TDOAStrength.GetParticleFilterOutput(building)
	c.JSON(http.StatusOK, gin.H{"particle_filter": output, "source": "live"})
}

func (h *Handler) GetBirdRadar(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	if h.birdDeterrent == nil {
		scanData := h.generateMockBirdRadarData(building)
		c.JSON(http.StatusOK, gin.H{"scan_data": scanData, "count": len(scanData), "source": "mock"})
		return
	}

	scanData := h.birdDeterrent.ScanBuilding(building)
	c.JSON(http.StatusOK, gin.H{"scan_data": scanData, "count": len(scanData), "source": "live"})
}

func (h *Handler) GetBirdDeterrentStatus(c *gin.Context) {
	building := c.Query("building")
	if building == "" {
		building = "应县木塔"
	}

	if h.birdDeterrent == nil {
		c.JSON(http.StatusOK, gin.H{
			"active_deterrents":  []interface{}{},
			"recent_bird_count":  0,
			"woodpecker_count":   0,
			"activity_level":     "low",
			"source":             "mock",
		})
		return
	}

	status := h.birdDeterrent.GetDeterrentStatus(building)
	c.JSON(http.StatusOK, status)
}

func (h *Handler) TriggerBirdDeterrent(c *gin.Context) {
	var req struct {
		Building       string `json:"building"`
		DeterrentType  string `json:"deterrent_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Building == "" {
		req.Building = "应县木塔"
	}
	if req.DeterrentType == "" {
		req.DeterrentType = "ultrasonic"
	}

	if h.birdDeterrent == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "simulated",
			"message":   "鸟情驱赶已启动（模拟模式）",
			"type":      req.DeterrentType,
			"building":  req.Building,
		})
		return
	}

	action := h.birdDeterrent.TriggerDeterrent(req.Building, req.DeterrentType)
	c.JSON(http.StatusOK, gin.H{
		"status":   "activated",
		"action":   action,
		"building": req.Building,
	})
}

func (h *Handler) generateMockTunnelNetwork(building string) *models.TunnelNetwork {
	nodes := make([]models.TunnelNode, 0)
	edges := make([]models.TunnelEdge, 0)
	now := time.Now()

	numNodes := 8 + rand.Intn(5)
	for i := 0; i < numNodes; i++ {
		var x, y, z float64
		if building == "应县木塔" {
			angle := rand.Float64() * 2 * math.Pi
			radius := 2 + rand.Float64()*6
			x = math.Cos(angle) * radius
			z = math.Sin(angle) * radius
			y = 3 + rand.Float64()*25
		} else {
			x = -12 + rand.Float64()*24
			z = -6 + rand.Float64()*12
			y = 2 + rand.Float64()*8
		}
		nodes = append(nodes, models.TunnelNode{
			ID:         "TN-" + strconv.Itoa(i+1),
			PositionX:  x,
			PositionY:  y,
			PositionZ:  z,
			Building:   building,
			Confidence: 0.5 + rand.Float64()*0.5,
			FirstSeen:  now.Add(-time.Duration(rand.Intn(48)) * time.Hour),
			LastSeen:   now.Add(-time.Duration(rand.Intn(2)) * time.Hour),
			Active:     rand.Float64() > 0.2,
		})
	}

	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			dx := nodes[i].PositionX - nodes[j].PositionX
			dy := nodes[i].PositionY - nodes[j].PositionY
			dz := nodes[i].PositionZ - nodes[j].PositionZ
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if dist < 3.0 {
				edges = append(edges, models.TunnelEdge{
					FromNodeID: nodes[i].ID,
					ToNodeID:   nodes[j].ID,
					Length:     dist,
					Strength:   1.0 - dist/3.0,
				})
			}
		}
	}

	return &models.TunnelNetwork{
		Building:  building,
		Nodes:     nodes,
		Edges:     edges,
		UpdatedAt: now,
	}
}

func (h *Handler) generateMockStrengthAssessments(building string) []models.WoodStrengthAssessment {
	sensors := h.sensorService.GetSensorsByBuilding(building)
	assessments := make([]models.WoodStrengthAssessment, 0)
	now := time.Now()

	for _, sensor := range sensors {
		if sensor.Type != "acoustic_emission" {
			continue
		}
		cumEnergy := 5000 + rand.Float64()*30000
		density := algorithms.SimulateWoodDensity(450, 968, 12+rand.Float64()*15)
		depthRatio := 0.1 + rand.Float64()*0.4

		damageIndex := 1.0 - cumEnergy/50000.0
		if damageIndex < 0 {
			damageIndex = 0
		}
		rsi := (density / 450.0) * damageIndex * (1.0 - depthRatio)
		sf := rsi * 3.0

		var level string
		switch {
		case sf >= 2.0:
			level = "safe"
		case sf >= 1.5:
			level = "caution"
		case sf >= 1.0:
			level = "warning"
		case sf >= 0.5:
			level = "danger"
		default:
			level = "critical"
		}

		assessments = append(assessments, models.WoodStrengthAssessment{
			SensorID:              sensor.SensorID,
			Building:              building,
			Location:              sensor.Location,
			CumulativeEnergy:      cumEnergy,
			WoodDensity:           density,
			DamageIndex:           damageIndex,
			ResidualStrengthIndex: rsi,
			SafetyFactor:          sf,
			StrengthLevel:         level,
			Timestamp:             now,
		})
	}

	return assessments
}

func (h *Handler) generateMockParticleFilterOutput(building string) models.ParticleFilterOutput {
	now := time.Now()
	peakTime := now.Add(4 * time.Hour)
	releaseTime := peakTime.Add(-1 * time.Hour)
	currentActivity := 40 + rand.Float64()*60

	particles := make([]models.ParticleState, 20)
	for i := range particles {
		particles[i] = models.ParticleState{
			ActivityLevel: currentActivity + (rand.Float64()-0.5)*30,
			Trend:         (rand.Float64() - 0.3) * 5,
			Weight:        1.0 / 20.0,
			Timestamp:     now,
		}
	}

	return models.ParticleFilterOutput{
		Building:           building,
		Particles:          particles,
		PredictedPeakTime:  peakTime,
		OptimalReleaseTime: releaseTime,
		CurrentActivity:    currentActivity,
		PredictedPeak:      currentActivity * 1.5,
		Confidence:         0.6 + rand.Float64()*0.3,
		ShouldReleaseNow:   releaseTime.Before(now.Add(30*time.Minute)),
	}
}

func (h *Handler) generateMockBirdRadarData(building string) []models.BirdRadarData {
	now := time.Now()
	count := rand.Intn(6)
	data := make([]models.BirdRadarData, count)

	birdTypes := []string{"woodpecker", "sparrow", "swallow", "crow"}
	for i := 0; i < count; i++ {
		birdType := birdTypes[rand.Intn(len(birdTypes))]
		if rand.Float64() > 0.3 {
			birdType = birdTypes[rand.Intn(len(birdTypes)-1)+1]
		}
		activityLevel := "low"
		if count >= 6 {
			activityLevel = "intense"
		} else if count >= 4 {
			activityLevel = "high"
		} else if count >= 2 {
			activityLevel = "moderate"
		}

		data[i] = models.BirdRadarData{
			ID:            "BIRD-" + building + "-" + strconv.Itoa(i+1),
			Timestamp:     now,
			BirdCount:     1,
			BirdType:      birdType,
			Direction:     rand.Float64() * 360,
			Distance:      10 + rand.Float64()*90,
			Altitude:      2 + rand.Float64()*28,
			Speed:         5 + rand.Float64()*20,
			ActivityLevel: activityLevel,
		}
	}

	return data
}
