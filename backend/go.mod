module ancient-wood-monitor

go 1.21

require (
	github.com/ancient-wood/bird_drive v0.0.0
	github.com/ancient-wood/particle_filter_timing v0.0.0
	github.com/ancient-wood/strength_calc v0.0.0
	github.com/ancient-wood/tdoa_locator v0.0.0
	github.com/gin-contrib/gzip v1.0.1
	github.com/gin-gonic/gin v1.9.1
	github.com/influxdata/influxdb1-client v0.0.0-20220302092344-a9ab560c08ae
	github.com/prometheus/client_golang v1.19.0
	github.com/spf13/viper v1.18.2
	gonum.org/v1/gonum v0.14.0
)

require (
	github.com/bytedance/sonic v1.10.2 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3afa95e0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.16.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gonum.org/v1/netlib v0.0.0-20231025120521-1d4d3ba37d3e // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/ancient-wood/bird_drive => ../modules/bird_drive
	github.com/ancient-wood/particle_filter_timing => ../modules/particle_filter_timing
	github.com/ancient-wood/strength_calc => ../modules/strength_calc
	github.com/ancient-wood/tdoa_locator => ../modules/tdoa_locator
)
