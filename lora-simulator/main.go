package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type LoRaDataPacket struct {
	PacketID        string                 `json:"packet_id"`
	DeviceType      string                 `json:"device_type"`
	DeviceID        string                 `json:"device_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Sequence        uint64                 `json:"sequence"`
	Data            map[string]interface{} `json:"data"`
	RSSI            float64                `json:"rssi"`
	SNR             float64                `json:"snr"`
	SpreadingFactor int                    `json:"spreading_factor"`
}

type SensorConfig struct {
	ID        string
	Type      string
	Building  string
	Location  string
	BaseValue float64
	Variance  float64
}

type TermitePulse struct {
	SensorIDs    []string  `json:"sensor_ids,omitempty"`
	Building     string    `json:"building,omitempty"`
	Duration     time.Duration `json:"duration"`
	Multiplier   float64   `json:"multiplier"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Active       bool      `json:"active"`
}

type SimulatorConfig struct {
	APIURL         string
	DeviceCount    int
	ReportInterval time.Duration
	SimSpeed       float64
	HTTPPort       string
}

var (
	globalSequence uint64

	acousticSensors []SensorConfig
	moistureSensors []SensorConfig
	allSensors      []SensorConfig

	termitePulses     []*TermitePulse
	termitePulsesMu   sync.RWMutex

	cfg SimulatorConfig

	metricsSentPackets = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lora_simulator_packets_sent_total",
		Help: "Total number of LoRa packets sent",
	}, []string{"device_type", "status"})

	metricsActivePulses = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lora_simulator_active_termite_pulses",
		Help: "Number of active termite attack pulses",
	})

	metricsSensorCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lora_simulator_sensors_total",
		Help: "Total number of simulated sensors",
	}, []string{"type"})
)

func generatePacketID(deviceID string, timestamp time.Time, sequence uint64) string {
	if sequence == 0 {
		sequence = uint64(time.Now().UnixNano())
	}
	h := fnv.New64a()
	h.Write([]byte(deviceID))
	h.Write([]byte(fmt.Sprintf("%d", timestamp.UnixNano())))
	h.Write([]byte(fmt.Sprintf("%d", sequence)))
	return fmt.Sprintf("%s-%d-%x", deviceID, timestamp.Unix(), h.Sum64())
}

func loadConfig() {
	cfg = SimulatorConfig{
		APIURL:         getEnv("API_URL", "http://localhost:8080/api/v1/lora/data"),
		DeviceCount:    getEnvInt("DEVICE_COUNT", 50),
		ReportInterval: getEnvDuration("REPORT_INTERVAL", 1*time.Hour),
		SimSpeed:       getEnvFloat("SIMULATION_SPEED", 1.0),
		HTTPPort:       getEnv("HTTP_PORT", "8081"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func initSensors() {
	acousticCount := cfg.DeviceCount * 50 / 90
	moistureCount := cfg.DeviceCount - acousticCount

	buildings := []string{"应县木塔", "佛光寺"}

	for _, building := range buildings {
		bAcousticCount := acousticCount / 2
		bMoistureCount := moistureCount / 2
		if building == "应县木塔" {
			bAcousticCount = acousticCount - bAcousticCount
			bMoistureCount = moistureCount - bMoistureCount
		}

		for i := 0; i < bAcousticCount; i++ {
			sensorID := generateSensorID("AC", building, i+1)
			sc := SensorConfig{
				ID:        sensorID,
				Type:      "acoustic_emission",
				Building:  building,
				Location:  getLocation(building, i),
				BaseValue: 30 + rand.Float64()*50,
				Variance:  20,
			}
			acousticSensors = append(acousticSensors, sc)
			allSensors = append(allSensors, sc)
		}

		for i := 0; i < bMoistureCount; i++ {
			sensorID := generateSensorID("MS", building, i+1)
			sc := SensorConfig{
				ID:        sensorID,
				Type:      "wood_moisture",
				Building:  building,
				Location:  getLocation(building, i),
				BaseValue: 15 + rand.Float64()*8,
				Variance:  3,
			}
			moistureSensors = append(moistureSensors, sc)
			allSensors = append(allSensors, sc)
		}
	}

	metricsSensorCount.WithLabelValues("acoustic").Set(float64(len(acousticSensors)))
	metricsSensorCount.WithLabelValues("moisture").Set(float64(len(moistureSensors)))
}

func generateSensorID(prefix, building string, num int) string {
	buildingCode := "YMT"
	if building == "佛光寺" {
		buildingCode = "FGS"
	}
	return fmt.Sprintf("%s-%s-%03d", prefix, buildingCode, num)
}

func getLocation(building string, index int) string {
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

func getTermiteMultiplier(sensorID string, building string, t time.Time) float64 {
	termitePulsesMu.RLock()
	defer termitePulsesMu.RUnlock()

	mult := 1.0
	activeCount := 0

	for _, pulse := range termitePulses {
		if !pulse.Active {
			continue
		}
		if t.Before(pulse.StartTime) || t.After(pulse.EndTime) {
			continue
		}
		if pulse.Building != "" && pulse.Building != building {
			continue
		}
		if len(pulse.SensorIDs) > 0 {
			found := false
			for _, sid := range pulse.SensorIDs {
				if sid == sensorID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		mult *= pulse.Multiplier
		activeCount++
	}

	metricsActivePulses.Set(float64(activeCount))
	return mult
}

func generateAcousticData(sensor SensorConfig, hour int, t time.Time) map[string]interface{} {
	hourFactor := 1.0 + 0.5*math.Sin(float64(hour)*math.Pi/12)
	eventCount := int(sensor.BaseValue*hourFactor + rand.Float64()*sensor.Variance)

	pulseMult := getTermiteMultiplier(sensor.ID, sensor.Building, t)
	eventCount = int(float64(eventCount) * pulseMult)

	data := map[string]interface{}{
		"building":       sensor.Building,
		"location":       sensor.Location,
		"event_count":    eventCount,
		"energy":         100 + rand.Float64()*900,
		"amplitude":      40 + rand.Float64()*60,
		"duration":       1 + rand.Float64()*10,
		"rise_time":      0.1 + rand.Float64()*2,
		"counts":         10 + rand.Intn(100),
		"frequency_peak": 1000 + rand.Float64()*8000,
	}
	return data
}

func generateMoistureData(sensor SensorConfig, hour int, t time.Time) map[string]interface{} {
	seasonFactor := 1.0 + 0.3*math.Sin(float64(time.Now().Month())*math.Pi/6)
	diurnalFactor := 1.0 + 0.1*math.Sin(float64(hour)*math.Pi/12)

	moisture := sensor.BaseValue * seasonFactor * diurnalFactor
	moisture += (rand.Float64() - 0.5) * sensor.Variance

	pulseMult := getTermiteMultiplier(sensor.ID, sensor.Building, t)
	if pulseMult > 1.0 {
		moisture *= 1.1
	}

	data := map[string]interface{}{
		"building":    sensor.Building,
		"location":    sensor.Location,
		"moisture":    math.Max(5, math.Min(40, moisture)),
		"temperature": 15 + rand.Float64()*15,
	}
	return data
}

func sendPacket(packet LoRaDataPacket) error {
	jsonData, err := json.Marshal(packet)
	if err != nil {
		metricsSentPackets.WithLabelValues(packet.DeviceType, "error").Inc()
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.APIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		metricsSentPackets.WithLabelValues(packet.DeviceType, "error").Inc()
		return fmt.Errorf("failed to send packet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		metricsSentPackets.WithLabelValues(packet.DeviceType, "duplicate").Inc()
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		metricsSentPackets.WithLabelValues(packet.DeviceType, "error").Inc()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	metricsSentPackets.WithLabelValues(packet.DeviceType, "success").Inc()
	return nil
}

func runSimulation(ctx context.Context) {
	fmt.Printf("Starting LoRa sensor simulator...\n")
	fmt.Printf("API URL: %s\n", cfg.APIURL)
	fmt.Printf("Device count: %d (acoustic: %d, moisture: %d)\n",
		cfg.DeviceCount, len(acousticSensors), len(moistureSensors))
	fmt.Printf("Report interval: %v (sim speed: %.1fx)\n", cfg.ReportInterval, cfg.SimSpeed)
	fmt.Println("Press Ctrl+C to stop")

	actualInterval := time.Duration(float64(cfg.ReportInterval) / cfg.SimSpeed)
	if actualInterval < 1*time.Second {
		actualInterval = 1 * time.Second
	}

	simStartTime := time.Now().Truncate(cfg.ReportInterval)
	simHourOffset := 0

	ticker := time.NewTicker(actualInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Simulation stopped")
			return
		case <-ticker.C:
			reportTime := simStartTime.Add(time.Duration(simHourOffset) * cfg.ReportInterval)
			hour := reportTime.Hour()

			fmt.Printf("\n[%s] Sending batch (%d sensors)...\n",
				reportTime.Format("2006-01-02 15:04"), len(allSensors))

			var wg sync.WaitGroup
			sem := make(chan struct{}, 10)

			for _, sensor := range acousticSensors {
				wg.Add(1)
				sem <- struct{}{}
				go func(s SensorConfig) {
					defer wg.Done()
					defer func() { <-sem }()

					data := generateAcousticData(s, hour, reportTime)
					seq := atomic.AddUint64(&globalSequence, 1)

					packet := LoRaDataPacket{
						PacketID:        generatePacketID(s.ID, reportTime, seq),
						DeviceType:      s.Type,
						DeviceID:        s.ID,
						Timestamp:       reportTime,
						Sequence:        seq,
						Data:            data,
						RSSI:            -70 - rand.Float64()*40,
						SNR:             5 + rand.Float64()*15,
						SpreadingFactor: 7 + rand.Intn(5),
					}

					if err := sendPacket(packet); err != nil {
						fmt.Printf("  [ERROR] %s: %v\n", s.ID, err)
					} else {
						fmt.Printf("  [AC] %s: events=%d\n", s.ID, data["event_count"])
					}

					if rand.Float64() < 0.15 {
						time.Sleep(50 * time.Millisecond)
						sendPacket(packet)
					}
				}(sensor)
			}

			for _, sensor := range moistureSensors {
				wg.Add(1)
				sem <- struct{}{}
				go func(s SensorConfig) {
					defer wg.Done()
					defer func() { <-sem }()

					data := generateMoistureData(s, hour, reportTime)
					seq := atomic.AddUint64(&globalSequence, 1)

					packet := LoRaDataPacket{
						PacketID:        generatePacketID(s.ID, reportTime, seq),
						DeviceType:      s.Type,
						DeviceID:        s.ID,
						Timestamp:       reportTime,
						Sequence:        seq,
						Data:            data,
						RSSI:            -65 - rand.Float64()*35,
						SNR:             8 + rand.Float64()*12,
						SpreadingFactor: 7 + rand.Intn(5),
					}

					if err := sendPacket(packet); err != nil {
						fmt.Printf("  [ERROR] %s: %v\n", s.ID, err)
					} else {
						fmt.Printf("  [MS] %s: moisture=%.1f%%\n", s.ID, data["moisture"])
					}

					if rand.Float64() < 0.1 {
						time.Sleep(30 * time.Millisecond)
						sendPacket(packet)
					}
				}(sensor)
			}

			wg.Wait()
			simHourOffset = (simHourOffset + 1) % 24
		}
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"devices":   len(allSensors),
		"acoustic":  len(acousticSensors),
		"moisture":  len(moistureSensors),
		"api_url":   cfg.APIURL,
		"interval":  cfg.ReportInterval.String(),
		"sim_speed": cfg.SimSpeed,
	})
}

func handleListSensors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"acoustic": acousticSensors,
		"moisture": moistureSensors,
	})
}

func handleListPulses(w http.ResponseWriter, r *http.Request) {
	termitePulsesMu.RLock()
	defer termitePulsesMu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(termitePulses)
}

func handleCreatePulse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SensorIDs  []string `json:"sensor_ids"`
		Building   string   `json:"building"`
		Duration   string   `json:"duration"`
		Multiplier float64  `json:"multiplier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Multiplier < 1.0 {
		req.Multiplier = 3.0
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		duration = 4 * time.Hour
	}

	now := time.Now()
	pulse := &TermitePulse{
		SensorIDs:  req.SensorIDs,
		Building:   req.Building,
		Duration:   duration,
		Multiplier: req.Multiplier,
		StartTime:  now,
		EndTime:    now.Add(duration),
		Active:     true,
	}

	termitePulsesMu.Lock()
	termitePulses = append(termitePulses, pulse)
	termitePulsesMu.Unlock()

	metricsActivePulses.Inc()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "created",
		"message": fmt.Sprintf("白蚁脉冲已注入，持续 %v，倍率 %.1fx", duration, req.Multiplier),
		"pulse":   pulse,
	})
}

func handleClearPulses(w http.ResponseWriter, r *http.Request) {
	termitePulsesMu.Lock()
	for _, p := range termitePulses {
		p.Active = false
	}
	termitePulsesMu.Unlock()

	metricsActivePulses.Set(0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "所有白蚁脉冲已清除",
	})
}

func startHTTPServer(ctx context.Context) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/sensors", handleListSensors)
	mux.HandleFunc("/pulses", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListPulses(w, r)
		case http.MethodPost:
			handleCreatePulse(w, r)
		case http.MethodDelete:
			handleClearPulses(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":" + cfg.HTTPPort
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		fmt.Printf("HTTP API listening on %s\n", addr)
		fmt.Println("  POST /pulses  - 注入白蚁脉冲")
		fmt.Println("  GET  /pulses  - 列出活跃脉冲")
		fmt.Println("  DELETE /pulses - 清除所有脉冲")
		fmt.Println("  GET  /sensors - 列出传感器")
		fmt.Println("  GET  /health  - 健康检查")
		fmt.Println("  GET  /metrics - Prometheus指标")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	return srv
}

func cleanupExpiredPulses(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			activeCount := 0
			termitePulsesMu.Lock()
			for _, p := range termitePulses {
				if p.Active && now.After(p.EndTime) {
					p.Active = false
				}
				if p.Active {
					activeCount++
				}
			}
			termitePulsesMu.Unlock()
			metricsActivePulses.Set(float64(activeCount))
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	loadConfig()
	initSensors()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived shutdown signal...")
		cancel()
	}()

	startHTTPServer(ctx)
	go cleanupExpiredPulses(ctx)
	runSimulation(ctx)
}
