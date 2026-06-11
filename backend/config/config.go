package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

var AppConfig *Config

type Config struct {
	Server             ServerConfig             `mapstructure:"server"`
	InfluxDB           InfluxDBConfig           `mapstructure:"influxdb"`
	LoRa               LoRaConfig               `mapstructure:"lora"`
	Alert              AlertConfig              `mapstructure:"alert"`
	Fumigation         FumigationConfig         `mapstructure:"fumigation"`
	Model              ModelConfig              `mapstructure:"model"`
	Pipeline           PipelineConfig           `mapstructure:"pipeline"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type InfluxDBConfig struct {
	Addr            string `mapstructure:"addr"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	Precision       string `mapstructure:"precision"`
	WriteQueueSize  int    `mapstructure:"write_queue_size"`
	WriteMaxRetries int    `mapstructure:"write_max_retries"`
}

type LoRaConfig struct {
	DataEndpoint string `mapstructure:"data_endpoint"`
	DeviceCount  int    `mapstructure:"device_count"`
	UDPAddr      string `mapstructure:"udp_addr"`
}

type AlertConfig struct {
	AcousticEventThreshold float64 `mapstructure:"acoustic_event_threshold"`
	MoistureThreshold      float64 `mapstructure:"moisture_threshold"`
	WechatWebhookURL       string  `mapstructure:"wechat_webhook_url"`
	SmsAPIURL              string  `mapstructure:"sms_api_url"`
	SmsAPIKey              string  `mapstructure:"sms_api_key"`
}

type FumigationConfig struct {
	DefaultReleaseRate float64 `mapstructure:"default_release_rate"`
	WindSpeed          float64 `mapstructure:"wind_speed"`
	WindDirection      float64 `mapstructure:"wind_direction"`
	StabilityClass     string  `mapstructure:"stability_class"`
}

type ModelConfig struct {
	LstmPath       string `mapstructure:"lstm_path"`
	LstmInputSize  int    `mapstructure:"lstm_input_size"`
	LstmHiddenSize int    `mapstructure:"lstm_hidden_size"`
	LstmOutputSize int    `mapstructure:"lstm_output_size"`
}

type PipelineConfig struct {
	BufferSize        int                `mapstructure:"buffer_size"`
	LoRaIngest        LoRaIngestConfig   `mapstructure:"lora_ingest"`
	TermiteLSTM       TermiteLSTMConfig  `mapstructure:"termite_lstm"`
	FumigantDiffusion FumigantConfig     `mapstructure:"fumigant_diffusion"`
	Alerter           AlerterConfig      `mapstructure:"alerter"`
}

type LoRaIngestConfig struct {
	ExpectedItems     uint64        `mapstructure:"expected_items"`
	FalsePositiveRate float64       `mapstructure:"false_positive_rate"`
	CacheTTL          time.Duration `mapstructure:"cache_ttl"`
	MaxCacheSize      int           `mapstructure:"max_cache_size"`
}

type TermiteLSTMConfig struct {
	EWMAAcousticAlpha   float64 `mapstructure:"ewma_acoustic_alpha"`
	EWMAMoistureAlpha   float64 `mapstructure:"ewma_moisture_alpha"`
	EWMAMaxHistory      int     `mapstructure:"ewma_max_history"`
	SpikeThresholdSigma float64 `mapstructure:"spike_threshold_sigma"`
	ConsecutiveConfirm  int     `mapstructure:"consecutive_confirm"`
	PredictionHours     int     `mapstructure:"prediction_hours"`
}

type FumigantConfig struct {
	DefaultReleaseRate float64 `mapstructure:"default_release_rate"`
	DefaultWindSpeed   float64 `mapstructure:"default_wind_speed"`
	DefaultWindDir     float64 `mapstructure:"default_wind_direction"`
	StabilityClass     string  `mapstructure:"stability_class"`
	GridResolution     float64 `mapstructure:"grid_resolution"`
	GridSizeX          int     `mapstructure:"grid_size_x"`
	GridSizeY          int     `mapstructure:"grid_size_y"`
	GridSizeZ          int     `mapstructure:"grid_size_z"`
	ExposureTimeHours  float64 `mapstructure:"exposure_time_hours"`
}

type AlerterConfig struct {
	AcousticThreshold float64       `mapstructure:"acoustic_threshold"`
	MoistureThreshold float64       `mapstructure:"moisture_threshold"`
	CooldownPeriod    time.Duration `mapstructure:"cooldown_period"`
	EnableWeChat      bool          `mapstructure:"enable_wechat"`
	EnableSMS         bool          `mapstructure:"enable_sms"`
}

func LoadConfig(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("influxdb.write_queue_size", 4096)
	v.SetDefault("influxdb.write_max_retries", 3)
	v.SetDefault("pipeline.buffer_size", 4096)

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	AppConfig = &Config{}
	if err := v.Unmarshal(AppConfig); err != nil {
		return err
	}

	log.Printf("Config loaded from %s", path)
	return nil
}
