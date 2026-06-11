package alerter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/models"
	"ancient-wood-monitor/internal/pipeline"
)

type Config struct {
	AcousticThreshold float64       `yaml:"acoustic_threshold"`
	MoistureThreshold float64       `yaml:"moisture_threshold"`
	CooldownPeriod    time.Duration `yaml:"cooldown_period"`
	EnableWeChat      bool          `yaml:"enable_wechat"`
	EnableSMS         bool          `yaml:"enable_sms"`
	WeChatWebhookURL  string        `yaml:"wechat_webhook_url"`
	SmsAPIURL         string        `yaml:"sms_api_url"`
	SmsAPIKey         string        `yaml:"sms_api_key"`
}

type AlertService struct {
	cfg              Config
	cooldownTracker  map[string]time.Time
	consecutiveCount map[string]int
	mu               sync.RWMutex
	name             string
}

func NewService(cfg Config) *AlertService {
	return &AlertService{
		cfg:              cfg,
		cooldownTracker:  make(map[string]time.Time),
		consecutiveCount: make(map[string]int),
		name:             "alerter",
	}
}

func (s *AlertService) Name() string {
	return s.name
}

func (s *AlertService) Start(ctx context.Context, in <-chan pipeline.PipelineMessage, out chan<- pipeline.PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}

			if msg.Type != pipeline.MsgTypeTermitePrediction && msg.Type != pipeline.MsgTypeFumigantDiffusion {
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

func (s *AlertService) process(ctx context.Context, msg *pipeline.PipelineMessage) (*pipeline.PipelineMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var termiteData pipeline.TermiteOutput
	var ok bool

	if msg.Type == pipeline.MsgTypeTermitePrediction {
		termiteData, ok = msg.Data.(pipeline.TermiteOutput)
		if !ok {
			return msg, nil
		}
	} else {
		return nil, nil
	}

	alert, err := s.checkAlert(ctx, termiteData)
	if err != nil {
		return msg, err
	}

	if alert == nil {
		return nil, nil
	}

	channels := s.sendNotifications(ctx, alert)

	output := pipeline.AlertOutput{
		Alert:    *alert,
		Channels: channels,
	}

	return &pipeline.PipelineMessage{
		Type: pipeline.MsgTypeAlert,
		Metadata: pipeline.Metadata{
			MessageID: alert.ID,
			Timestamp: time.Now(),
			Source:    s.name,
			TraceID:   msg.Metadata.TraceID,
			Retries:   msg.Metadata.Retries,
		},
		Data: output,
	}, nil
}

func (s *AlertService) checkAlert(ctx context.Context, data pipeline.TermiteOutput) (*models.Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cooldownKey := data.SensorID

	if cd, ok := s.cooldownTracker[cooldownKey]; ok {
		if time.Since(cd) < s.cfg.CooldownPeriod {
			return nil, nil
		}
	}

	threshold := s.cfg.AcousticThreshold
	alertType := "acoustic_emission"
	value := data.SmoothedRate

	if data.SensorID[:2] == "MS" {
		threshold = s.cfg.MoistureThreshold
		alertType = "wood_moisture"
	}

	if value <= threshold*1.05 {
		return nil, nil
	}

	s.consecutiveCount[cooldownKey]++
	if s.consecutiveCount[cooldownKey] < 2 {
		return nil, nil
	}

	s.consecutiveCount[cooldownKey] = 0
	s.cooldownTracker[cooldownKey] = time.Now()

	severity := s.getSeverity(value, threshold)
	message := s.buildMessage(data, alertType, value, threshold, severity)

	alert := &models.Alert{
		ID:           generateUUID(),
		Type:         alertType,
		Severity:     severity,
		SensorID:     data.SensorID,
		Building:     data.Building,
		Location:     data.Location,
		Value:        value,
		Threshold:    threshold,
		Message:      message,
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	return alert, nil
}

func (s *AlertService) getSeverity(value, threshold float64) string {
	ratio := value / threshold
	switch {
	case ratio > 1.5:
		return "critical"
	case ratio > 1.2:
		return "high"
	case ratio > 1.05:
		return "medium"
	default:
		return "low"
	}
}

func (s *AlertService) buildMessage(data pipeline.TermiteOutput, alertType string, value, threshold float64, severity string) string {
	unit := "次/小时"
	if alertType == "wood_moisture" {
		unit = "%"
	}

	return fmt.Sprintf("[%s] %s - %s: %.1f%s (阈值%.1f%s), 趋势:%s, 风险等级:%s",
		severity, data.Building, data.Location, value, unit, threshold, unit, data.Trend, data.RiskLevel)
}

func (s *AlertService) sendNotifications(ctx context.Context, alert *models.Alert) []string {
	channels := make([]string, 0)

	if s.cfg.EnableWeChat && s.cfg.WeChatWebhookURL != "" {
		if err := s.sendWeChat(ctx, alert); err == nil {
			channels = append(channels, "wechat")
		} else {
			log.Printf("[%s] wechat send failed: %v", s.name, err)
		}
	}

	if s.cfg.EnableSMS && s.cfg.SmsAPIURL != "" && s.cfg.SmsAPIKey != "" {
		if err := s.sendSMS(ctx, alert); err == nil {
			channels = append(channels, "sms")
		} else {
			log.Printf("[%s] sms send failed: %v", s.name, err)
		}
	}

	return channels
}

func (s *AlertService) sendWeChat(ctx context.Context, alert *models.Alert) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf(`## 古建筑虫蛀告警

**级别**: <font color="warning">%s</font>
**建筑**: %s
**位置**: %s
**传感器**: %s
**类型**: %s
**数值**: %.2f
**阈值**: %.2f
**时间**: %s
**信息**: %s`,
				alert.Severity, alert.Building, alert.Location,
				alert.SensorID, alert.Type, alert.Value, alert.Threshold,
				alert.Timestamp.Format("2006-01-02 15:04:05"), alert.Message),
		},
	}

	return s.postJSON(ctx, s.cfg.WeChatWebhookURL, payload)
}

func (s *AlertService) sendSMS(ctx context.Context, alert *models.Alert) error {
	payload := map[string]interface{}{
		"api_key": s.cfg.SmsAPIKey,
		"mobile":  config.AppConfig.Alert.SmsAPIKey,
		"content": fmt.Sprintf("[古建筑监测] %s告警: %s-%s, 数值%.1f, 请及时处理",
			alert.Severity, alert.Building, alert.Location, alert.Value),
	}

	return s.postJSON(ctx, s.cfg.SmsAPIURL, payload)
}

func (s *AlertService) postJSON(ctx context.Context, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	return nil
}

func generateUUID() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uint32(time.Now().Unix()),
		uint16(time.Now().UnixNano()%0xFFFF),
		uint16(0x4000|(time.Now().UnixNano()%0x0FFF)),
		uint16(0x8000|(time.Now().UnixNano()%0x3FFF)),
		uint32(time.Now().UnixNano()%0xFFFFFFFF))
}

func (s *AlertService) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cooldown_count": len(s.cooldownTracker),
		"thresholds": map[string]float64{
			"acoustic": s.cfg.AcousticThreshold,
			"moisture": s.cfg.MoistureThreshold,
		},
	}
}
