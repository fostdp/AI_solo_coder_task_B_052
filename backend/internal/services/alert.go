package services

import (
	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/algorithms/lstm"
	"ancient-wood-monitor/internal/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func generateUUID() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uint32(time.Now().Unix()),
		uint16(time.Now().UnixNano()%0xFFFF),
		uint16(0x4000|(time.Now().UnixNano()%0x0FFF)),
		uint16(0x8000|(time.Now().UnixNano()%0x3FFF)),
		uint32(time.Now().UnixNano()%0xFFFFFFFF))
}

type AlertService struct {
	influxDB          *InfluxDBService
	acousticSmoother  *lstm.EWMASmoother
	moistureSmoother  *lstm.EWMASmoother
	consecutiveAlerts map[string]int
	alertCooldown     map[string]time.Time
	mu                sync.RWMutex
}

func NewAlertService(influxDB *InfluxDBService) *AlertService {
	return &AlertService{
		influxDB:          influxDB,
		acousticSmoother:  lstm.NewEWMASmoother(0.25, 96),
		moistureSmoother:  lstm.NewEWMASmoother(0.2, 96),
		consecutiveAlerts: make(map[string]int),
		alertCooldown:     make(map[string]time.Time),
	}
}

func (s *AlertService) CheckAcousticAlert(sensorID, building, location string, eventRate float64) (*models.Alert, error) {
	threshold := config.AppConfig.Alert.AcousticEventThreshold

	smoothedRate := s.acousticSmoother.Smooth(sensorID, eventRate)

	isSpike := s.acousticSmoother.IsSpike(sensorID, eventRate, 2.5)
	if isSpike {
		return nil, nil
	}

	trend, hasTrend := s.acousticSmoother.GetTrend(sensorID)
	if hasTrend && trend < 0 && smoothedRate < threshold*1.2 {
		return nil, nil
	}

	history := s.acousticSmoother.GetHistory(sensorID)
	consecutiveOver := 0
	for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
		if history[i] > threshold {
			consecutiveOver++
		} else {
			break
		}
	}

	cooldownKey := "acoustic:" + sensorID
	s.mu.RLock()
	if cd, ok := s.alertCooldown[cooldownKey]; ok && time.Since(cd) < 10*time.Minute {
		s.mu.RUnlock()
		return nil, nil
	}
	s.mu.RUnlock()

	shouldAlert := smoothedRate > threshold*1.1 && consecutiveOver >= 2
	if !shouldAlert && smoothedRate > threshold {
		s.mu.Lock()
		s.consecutiveAlerts[sensorID]++
		count := s.consecutiveAlerts[sensorID]
		s.mu.Unlock()

		if count >= 2 {
			shouldAlert = true
		}
	}

	if shouldAlert {
		s.mu.Lock()
		s.alertCooldown[cooldownKey] = time.Now()
		s.consecutiveAlerts[sensorID] = 0
		s.mu.Unlock()

		alert := &models.Alert{
			ID:           generateUUID(),
			Type:         "acoustic_emission",
			Severity:     getSeverity(smoothedRate, threshold),
			SensorID:     sensorID,
			Building:     building,
			Location:     location,
			Value:        smoothedRate,
			Threshold:    threshold,
			Message:      fmt.Sprintf("声发射事件率 %.1f 次/小时（平滑后），超过阈值 %.1f 次/小时，疑似白蚁活动", smoothedRate, threshold),
			Timestamp:    time.Now(),
			Acknowledged: false,
		}

		if s.influxDB != nil {
			if err := s.influxDB.WriteAlert(alert); err != nil {
				return nil, err
			}
		}

		go s.sendNotifications(alert)

		return alert, nil
	}

	return nil, nil
}

func (s *AlertService) CheckMoistureAlert(sensorID, building, location string, moisture float64) (*models.Alert, error) {
	threshold := config.AppConfig.Alert.MoistureThreshold

	smoothedMoisture := s.moistureSmoother.Smooth(sensorID, moisture)

	isSpike := s.moistureSmoother.IsSpike(sensorID, moisture, 2.0)
	if isSpike {
		return nil, nil
	}

	history := s.moistureSmoother.GetHistory(sensorID)
	consecutiveOver := 0
	for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
		if history[i] > threshold {
			consecutiveOver++
		} else {
			break
		}
	}

	cooldownKey := "moisture:" + sensorID
	s.mu.RLock()
	if cd, ok := s.alertCooldown[cooldownKey]; ok && time.Since(cd) < 30*time.Minute {
		s.mu.RUnlock()
		return nil, nil
	}
	s.mu.RUnlock()

	shouldAlert := smoothedMoisture > threshold*1.05 && consecutiveOver >= 2

	if shouldAlert {
		s.mu.Lock()
		s.alertCooldown[cooldownKey] = time.Now()
		s.mu.Unlock()

		alert := &models.Alert{
			ID:           generateUUID(),
			Type:         "wood_moisture",
			Severity:     getMoistureSeverity(smoothedMoisture, threshold),
			SensorID:     sensorID,
			Building:     building,
			Location:     location,
			Value:        smoothedMoisture,
			Threshold:    threshold,
			Message:      fmt.Sprintf("木材含水率 %.1f%%（平滑后），超过阈值 %.1f%%，存在虫蛀风险", smoothedMoisture, threshold),
			Timestamp:    time.Now(),
			Acknowledged: false,
		}

		if s.influxDB != nil {
			if err := s.influxDB.WriteAlert(alert); err != nil {
				return nil, err
			}
		}

		go s.sendNotifications(alert)

		return alert, nil
	}

	return nil, nil
}

func getSeverity(value, threshold float64) string {
	ratio := value / threshold
	switch {
	case ratio >= 3:
		return "critical"
	case ratio >= 2:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

func getMoistureSeverity(value, threshold float64) string {
	diff := value - threshold
	switch {
	case diff >= 15:
		return "critical"
	case diff >= 10:
		return "high"
	case diff >= 5:
		return "medium"
	default:
		return "low"
	}
}

func (s *AlertService) sendNotifications(alert *models.Alert) {
	s.sendWechatNotification(alert)
	s.sendSMSNotification(alert)
}

type WechatMessage struct {
	MsgType string            `json:"msgtype"`
	Text    WechatTextContent `json:"text"`
}

type WechatTextContent struct {
	Content string `json:"content"`
}

func (s *AlertService) sendWechatNotification(alert *models.Alert) error {
	webhookURL := config.AppConfig.Alert.WechatWebhookURL
	if webhookURL == "" || webhookURL == "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-key" {
		return nil
	}

	content := fmt.Sprintf(`【古建筑虫蛀告警】
告警类型: %s
告警级别: %s
建筑名称: %s
位置: %s
传感器ID: %s
当前值: %.2f
阈值: %.2f
告警信息: %s
时间: %s`,
		alert.Type, alert.Severity, alert.Building, alert.Location,
		alert.SensorID, alert.Value, alert.Threshold,
		alert.Message, alert.Timestamp.Format("2006-01-02 15:04:05"))

	msg := WechatMessage{
		MsgType: "text",
		Text: WechatTextContent{
			Content: content,
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *AlertService) sendSMSNotification(alert *models.Alert) error {
	smsAPIURL := config.AppConfig.Alert.SmsAPIURL
	apiKey := config.AppConfig.Alert.SmsAPIKey

	if smsAPIURL == "" || apiKey == "" || apiKey == "your-sms-api-key" {
		return nil
	}

	message := fmt.Sprintf("【古建筑监测告警】%s %s %s, %s",
		alert.Building, alert.Location, alert.Severity, alert.Message)

	payload := map[string]interface{}{
		"api_key": apiKey,
		"message": message,
		"alert_id": alert.ID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", smsAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *AlertService) GetActiveAlerts(building string) ([]models.Alert, error) {
	return s.influxDB.QueryActiveAlerts(building)
}

func (s *AlertService) AcknowledgeAlert(alertID string) error {
	return nil
}
