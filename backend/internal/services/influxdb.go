package services

import (
	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/models"
	"ancient-wood-monitor/internal/services/influx"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

type InfluxDBService struct {
	client influxdb.Client
	writer *influx.AsyncWriter
}

func NewInfluxDBService() (*InfluxDBService, error) {
	cfg := config.AppConfig.InfluxDB
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create influxdb client: %w", err)
	}

	_, _, err = client.Ping(5 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to ping influxdb: %w", err)
	}

	queueSize := 4096
	maxRetries := 3
	if cfg.WriteQueueSize > 0 {
		queueSize = cfg.WriteQueueSize
	}
	if cfg.WriteMaxRetries > 0 {
		maxRetries = cfg.WriteMaxRetries
	}
	writer := influx.NewAsyncWriter(client, queueSize, maxRetries)

	return &InfluxDBService{client: client, writer: writer}, nil
}

func (s *InfluxDBService) Close() error {
	if s.writer != nil {
		s.writer.Close()
	}
	return s.client.Close()
}

func (s *InfluxDBService) WriteAcousticEmission(data *models.AcousticEmissionData) error {
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  config.AppConfig.InfluxDB.Database,
		Precision: config.AppConfig.InfluxDB.Precision,
	})
	if err != nil {
		return err
	}

	tags := map[string]string{
		"sensor_id": data.SensorID,
		"building":  data.Building,
		"location":  data.Location,
	}

	fields := map[string]interface{}{
		"event_count":    data.EventCount,
		"energy":         data.Energy,
		"amplitude":      data.Amplitude,
		"duration":       data.Duration,
		"rise_time":      data.RiseTime,
		"counts":         data.Counts,
		"frequency_peak": data.FrequencyPeak,
	}

	pt, err := influxdb.NewPoint("acoustic_emission", tags, fields, data.Timestamp)
	if err != nil {
		return err
	}

	bp.AddPoint(pt)
	if s.writer != nil {
		s.writer.Write(bp)
		return nil
	}
	return s.client.Write(bp)
}

func (s *InfluxDBService) WriteWoodMoisture(data *models.WoodMoistureData) error {
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  config.AppConfig.InfluxDB.Database,
		Precision: config.AppConfig.InfluxDB.Precision,
	})
	if err != nil {
		return err
	}

	tags := map[string]string{
		"sensor_id": data.SensorID,
		"building":  data.Building,
		"location":  data.Location,
	}

	fields := map[string]interface{}{
		"moisture":    data.Moisture,
		"temperature": data.Temperature,
	}

	pt, err := influxdb.NewPoint("wood_moisture", tags, fields, data.Timestamp)
	if err != nil {
		return err
	}

	bp.AddPoint(pt)
	if s.writer != nil {
		s.writer.Write(bp)
		return nil
	}
	return s.client.Write(bp)
}

func (s *InfluxDBService) QueryAcousticEvents(building string, start, end time.Time) ([]models.AcousticEmissionData, error) {
	query := fmt.Sprintf(`
		SELECT event_count, energy, amplitude, duration, rise_time, counts, frequency_peak
		FROM acoustic_emission
		WHERE building = '%s' AND time >= '%s' AND time <= '%s'
		ORDER BY time DESC
	`, building, start.Format(time.RFC3339), end.Format(time.RFC3339))

	q := influxdb.Query{
		Command:  query,
		Database: config.AppConfig.InfluxDB.Database,
	}

	resp, err := s.client.Query(q)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	var results []models.AcousticEmissionData
	for _, r := range resp.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				t, _ := time.Parse(time.RFC3339, v[0].(string))
				data := models.AcousticEmissionData{
					SensorID:    s.Tags["sensor_id"],
					Building:    s.Tags["building"],
					Location:    s.Tags["location"],
					Timestamp:   t,
					EventCount:  int(v[1].(float64)),
					Energy:      v[2].(float64),
					Amplitude:   v[3].(float64),
					Duration:    v[4].(float64),
					RiseTime:    v[5].(float64),
					Counts:      int(v[6].(float64)),
					FrequencyPeak: v[7].(float64),
				}
				results = append(results, data)
			}
		}
	}

	return results, nil
}

func (s *InfluxDBService) QueryHourlyAcousticEventRate(sensorID string, hours int) (float64, error) {
	query := fmt.Sprintf(`
		SELECT sum(event_count) as total_events
		FROM acoustic_emission
		WHERE sensor_id = '%s' AND time > now() - %dh
		GROUP BY time(1h)
	`, sensorID, hours)

	q := influxdb.Query{
		Command:  query,
		Database: config.AppConfig.InfluxDB.Database,
	}

	resp, err := s.client.Query(q)
	if err != nil {
		return 0, err
	}
	if resp.Error() != nil {
		return 0, resp.Error()
	}

	totalEvents := 0.0
	count := 0
	for _, r := range resp.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				if v[1] != nil {
					totalEvents += v[1].(float64)
					count++
				}
			}
		}
	}

	if count == 0 {
		return 0, nil
	}

	return totalEvents / float64(count), nil
}

func (s *InfluxDBService) QueryLatestMoisture(sensorID string) (float64, error) {
	query := fmt.Sprintf(`
		SELECT last(moisture) FROM wood_moisture WHERE sensor_id = '%s'
	`, sensorID)

	q := influxdb.Query{
		Command:  query,
		Database: config.AppConfig.InfluxDB.Database,
	}

	resp, err := s.client.Query(q)
	if err != nil {
		return 0, err
	}
	if resp.Error() != nil {
		return 0, resp.Error()
	}

	for _, r := range resp.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				if v[1] != nil {
					return v[1].(float64), nil
				}
			}
		}
	}

	return 0, nil
}

func (s *InfluxDBService) QueryMoistureHistory(building string, start, end time.Time) ([]models.WoodMoistureData, error) {
	query := fmt.Sprintf(`
		SELECT moisture, temperature
		FROM wood_moisture
		WHERE building = '%s' AND time >= '%s' AND time <= '%s'
		ORDER BY time DESC
	`, building, start.Format(time.RFC3339), end.Format(time.RFC3339))

	q := influxdb.Query{
		Command:  query,
		Database: config.AppConfig.InfluxDB.Database,
	}

	resp, err := s.client.Query(q)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	var results []models.WoodMoistureData
	for _, r := range resp.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				t, _ := time.Parse(time.RFC3339, v[0].(string))
				data := models.WoodMoistureData{
					SensorID:    s.Tags["sensor_id"],
					Building:    s.Tags["building"],
					Location:    s.Tags["location"],
					Timestamp:   t,
					Moisture:    v[1].(float64),
					Temperature: v[2].(float64),
				}
				results = append(results, data)
			}
		}
	}

	return results, nil
}

func (s *InfluxDBService) WriteAlert(alert *models.Alert) error {
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  config.AppConfig.InfluxDB.Database,
		Precision: config.AppConfig.InfluxDB.Precision,
	})
	if err != nil {
		return err
	}

	tags := map[string]string{
		"alert_id":   alert.ID,
		"type":       alert.Type,
		"severity":   alert.Severity,
		"sensor_id":  alert.SensorID,
		"building":   alert.Building,
		"location":   alert.Location,
	}

	fields := map[string]interface{}{
		"value":        alert.Value,
		"threshold":    alert.Threshold,
		"message":      alert.Message,
		"acknowledged": alert.Acknowledged,
	}

	pt, err := influxdb.NewPoint("alerts", tags, fields, alert.Timestamp)
	if err != nil {
		return err
	}

	bp.AddPoint(pt)
	if s.writer != nil {
		s.writer.Write(bp)
		return nil
	}
	return s.client.Write(bp)
}

func (s *InfluxDBService) QueryActiveAlerts(building string) ([]models.Alert, error) {
	query := fmt.Sprintf(`
		SELECT value, threshold, message, acknowledged
		FROM alerts
		WHERE building = '%s' AND time > now() - 24h
		ORDER BY time DESC
	`, building)

	q := influxdb.Query{
		Command:  query,
		Database: config.AppConfig.InfluxDB.Database,
	}

	resp, err := s.client.Query(q)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	var alerts []models.Alert
	for _, r := range resp.Results {
		for _, s := range r.Series {
			for _, v := range s.Values {
				t, _ := time.Parse(time.RFC3339, v[0].(string))
				ack := false
				if v[4] != nil {
					ack = v[4].(bool)
				}
				alert := models.Alert{
					ID:           s.Tags["alert_id"],
					Type:         s.Tags["type"],
					Severity:     s.Tags["severity"],
					SensorID:     s.Tags["sensor_id"],
					Building:     s.Tags["building"],
					Location:     s.Tags["location"],
					Value:        v[1].(float64),
					Threshold:    v[2].(float64),
					Message:      v[3].(string),
					Timestamp:    t,
					Acknowledged: ack,
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts, nil
}
