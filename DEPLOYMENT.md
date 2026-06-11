# 古代木结构建筑虫蛀监测系统 - 工程化部署指南

## 一、架构总览

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  LoRa Simulator │────▶│   Go Backend    │────▶│   InfluxDB 1.8  │
│  (50 devices)   │     │  (4 microsvc)   │     │  (CQ + RP)      │
│  :8081 /metrics │     │  :8080 /metrics │     │  :8086          │
└─────────────────┘     └─────────────────┘     └─────────────────┘
          │                        │                        │
          ▼                        ▼                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Prometheus (:9090)                          │
│                  + pprof (:6060 for Go)                         │
└──────────────────────────────────────────────────────────────────┘
```

## 二、快速启动（Docker Compose）

### 前置要求
- Docker 20.10+
- Docker Compose v2+
- 至少 4GB 可用内存

### 一键启动

```bash
# 1. 克隆项目后进入根目录
cd AI_solo_coder_task_A_052

# 2. 启动所有服务
docker-compose up -d --build

# 3. 查看服务状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f backend
docker-compose logs -f simulator
docker-compose logs -f influxdb
```

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Go Backend | 8080 | API + 前端静态资源 |
| LoRa Simulator | 8081 | 模拟设备 + 脉冲注入 API |
| InfluxDB | 8086 | 时序数据库 |
| Prometheus | 9090 | 指标监控 |
| pprof | 6060 | Go 性能剖析 |

### 访问地址

```
前端页面:      http://localhost:8080/
API 健康检查:  http://localhost:8080/health
Prometheus:   http://localhost:9090/
pprof:        http://localhost:6060/debug/pprof/
Simulator:    http://localhost:8081/health
```

### 停止服务

```bash
# 停止并保留数据
docker-compose down

# 停止并清除所有数据
docker-compose down -v
```

## 三、白蚁脉冲注入（模拟白蚁爆发）

### 1. 对整个建筑注入脉冲

```bash
# 给应县木塔注入3x强度的白蚁活动，持续4小时
curl -X POST http://localhost:8081/pulses \
  -H "Content-Type: application/json" \
  -d '{
    "building": "应县木塔",
    "duration": "4h",
    "multiplier": 3.0
  }'
```

### 2. 对指定传感器注入脉冲

```bash
# 给特定3个传感器注入5x强度，持续2小时
curl -X POST http://localhost:8081/pulses \
  -H "Content-Type: application/json" \
  -d '{
    "sensor_ids": ["AC-YMT-001", "AC-YMT-002", "AC-YMT-003"],
    "duration": "2h",
    "multiplier": 5.0
  }'
```

### 3. 查看活跃脉冲

```bash
curl http://localhost:8081/pulses
```

### 4. 清除所有脉冲

```bash
curl -X DELETE http://localhost:8081/pulses
```

## 四、InfluxDB 连续查询（CQ）配置

系统自动创建以下聚合策略，数据每1分钟重采样：

### 保留策略（RP）

| 名称 | 保留时长 | 用途 |
|------|----------|------|
| rp_1year | 365天 | 原始数据 |
| rp_30day | 30天 | 中间聚合（预留） |
| rp_forever | 永久 | 聚合数据 |

### 连续查询（CQ）列表

```
# 声发射 - 小时级（每传感器）
cq_acoustic_hourly
  ├── event_count, total_events
  ├── avg_energy, max_energy
  ├── avg_amplitude, avg_frequency_peak
  └── GROUP BY time(1h), sensor_id, building, location

# 声发射 - 小时级（按建筑）
cq_acoustic_hourly_building
  ├── total_events, sensor_count
  ├── max_energy, avg_energy
  └── GROUP BY time(1h), building

# 声发射 - 天级（每传感器）
cq_acoustic_daily
  └── 从 hourly 聚合而来

# 含水率 - 小时级（每传感器）
cq_moisture_hourly
  ├── avg_moisture, max_moisture, min_moisture
  ├── avg_temperature, sample_count
  └── GROUP BY time(1h), sensor_id, building, location

# 含水率 - 小时级（按建筑）
cq_moisture_hourly_building
  └── GROUP BY time(1h), building

# 含水率 - 天级
cq_moisture_daily
  └── 从 hourly 聚合而来

# 告警 - 小时级
cq_alerts_hourly
  ├── alert_count, critical_count, high_count, medium_count
  └── GROUP BY time(1h), building, level, channel

# 告警触发 - 小时级
cq_alert_triggers_hourly
  ├── trigger_count, avg_event_rate, avg_moisture
  ├── affected_sensors
  └── GROUP BY time(1h), building, trigger_type

# LoRa 报文 - 小时级
cq_lora_packets_hourly
  ├── total_packets, duplicate_count, active_devices
  ├── avg_rssi, avg_snr
  └── GROUP BY time(1h), device_type, building
```

### 查询聚合数据示例

```sql
-- 查询应县木塔过去24小时每小时的声发射事件总数
SELECT time, total_events
FROM "rp_forever"."acoustic_emission_hourly_by_building"
WHERE building = '应县木塔'
AND time > now() - 24h
ORDER BY time DESC

-- 查询所有传感器30天内的含水率最大值
SELECT time, sensor_id, max_moisture
FROM "rp_forever"."moisture_daily"
WHERE time > now() - 30d
```

## 五、Prometheus 监控指标

### Go 后端指标 (:8080/metrics)

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `http_request_duration_seconds` | Histogram | HTTP 请求延迟分布 |
| `http_requests_total` | Counter | HTTP 请求总数（按 method/path/status） |
| `influxdb_writes_total` | Counter | InfluxDB 写入总数 |
| `influxdb_write_errors_total` | Counter | InfluxDB 写入错误数 |
| `influxdb_write_queue_size` | Gauge | 写入队列当前大小 |
| `alerts_total` | Counter | 告警触发总数（按 level/channel） |
| `lora_packets_received_total` | Counter | LoRa 报文接收总数 |
| `lora_packets_duplicate_total` | Counter | 去重丢弃的重复报文数 |
| `termite_predictions_total` | Counter | LSTM 预测总数（按风险等级） |
| `pipeline_messages_total` | Counter | Pipeline 消息处理数（按 stage/type） |
| `pipeline_errors_total` | Counter | Pipeline 错误数（按 stage） |

### 模拟器指标 (:8081/metrics)

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `lora_simulator_packets_sent_total` | Counter | 发送报文总数（按 device_type/status） |
| `lora_simulator_active_termite_pulses` | Gauge | 活跃白蚁脉冲数 |
| `lora_simulator_sensors_total` | Gauge | 传感器总数（按 type） |

### 常用 PromQL 查询

```promql
# 后端5xx错误率
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# 后端API平均延迟（p95）
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))

# InfluxDB写入错误率
rate(influxdb_write_errors_total[5m]) / rate(influxdb_writes_total[5m])

# 活跃白蚁脉冲数
lora_simulator_active_termite_pulses

# 每小时声发射事件数（模拟）
sum by (device_type) (rate(lora_simulator_packets_sent_total{status="success"}[1h]))
```

## 六、pprof 性能剖析

Go 后端启动独立的 pprof 服务器在 `:6060`：

### 常用剖析命令

```bash
# 30秒 CPU 剖析
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 内存剖析（当前）
go tool pprof http://localhost:6060/debug/pprof/heap

# 内存剖析（分配）
go tool pprof http://localhost:6060/debug/pprof/allocs

# 协程剖析
go tool pprof http://localhost:6060/debug/ppprof/goroutine

# 阻塞剖析
go tool pprof http://localhost:6060/debug/pprof/block

# 锁竞争剖析
go tool pprof http://localhost:6060/debug/pprof/mutex

# 执行追踪 5秒
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out
go tool trace trace.out
```

### pprof 常用交互命令

```
(pprof) top                  # 查看top函数
(pprof) top -cum             # 按累计耗时排序
(pprof) list functionName    # 查看具体函数代码
(pprof) web                  # 生成火焰图（需graphviz）
(pprof) png                  # 输出PNG图
```

## 七、Gzip 压缩

后端自动启用 Gzip 压缩（`gin-contrib/gzip`），覆盖：
- 所有 API 响应（`/api/v1/*`）
- 静态文件（`/app.js`, `/TimberModel.js`, `/VoxelRisk.js` 等）
- `/metrics` 指标

压缩级别：DefaultCompression（6），根据 `Accept-Encoding` 头自动协商。

### 验证 Gzip

```bash
# 查看响应头，应包含 Content-Encoding: gzip
curl -I -H "Accept-Encoding: gzip" http://localhost:8080/api/v1/sensors
curl -I -H "Accept-Encoding: gzip" http://localhost:8080/app.js
```

## 八、环境变量配置

### backend 服务

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `CONFIG_PATH` | `/app/config/config.yaml` | 配置文件路径 |
| `INFLUXDB_ADDR` | `http://influxdb:8086` | InfluxDB 地址 |
| `SERVER_PORT` | `8080` | 服务端口 |
| `FRONTEND_PATH` | 自动探测 | 前端静态资源路径 |

### simulator 服务

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_URL` | `http://backend:8080/api/v1/lora/data` | 后端上报地址 |
| `DEVICE_COUNT` | `50` | 传感器总数 |
| `REPORT_INTERVAL` | `1h` | 真实上报间隔 |
| `SIMULATION_SPEED` | `1.0` | 模拟加速倍率（设为3600则1秒=1小时） |
| `HTTP_PORT` | `8081` | 模拟器API端口 |

### 快速模拟加速

```bash
# docker-compose.yml 中修改 simulator 环境变量
environment:
  - SIMULATION_SPEED=3600   # 1秒 = 1小时（用于快速测试）
```

## 九、数据持久化

Docker Compose 自动创建以下命名卷：

| 卷名 | 用途 | 默认位置 |
|------|------|----------|
| `influxdb_data` | InfluxDB 数据 | `/var/lib/docker/volumes/.../_data` |
| `prometheus_data` | Prometheus 数据 | `/var/lib/docker/volumes/.../_data` |

### 备份数据

```bash
# 备份 InfluxDB
docker exec ancient-wood-influxdb influxd backup -portable /backup/$(date +%Y%m%d)
docker cp ancient-wood-influxdb:/backup/$(date +%Y%m%d) ./influxdb-backup/

# 备份 Prometheus
docker cp ancient-wood-prometheus:/prometheus ./prometheus-backup/
```

## 十、故障排查

### 后端无法连接 InfluxDB

```bash
# 检查 InfluxDB 状态
docker-compose ps influxdb
docker-compose logs influxdb

# 手动连接测试
docker exec -it ancient-wood-influxdb influx -execute "SHOW DATABASES"
```

### 模拟器无法上报数据

```bash
# 检查后端健康状态
curl http://localhost:8080/health

# 检查模拟器日志
docker-compose logs simulator

# 检查模拟器健康
curl http://localhost:8081/health
```

### Prometheus 无数据

```bash
# 检查目标状态
open http://localhost:9090/targets

# 检查配置
docker exec ancient-wood-prometheus cat /etc/prometheus/prometheus.yml
```

### Pipeline 消息堆积

```bash
# 查看队列大小指标
curl -s http://localhost:8080/metrics | grep influxdb_write_queue_size

# 查看 pipeline 消息处理速率
curl -s http://localhost:8080/metrics | grep pipeline_messages_total
```

## 十一、生产环境建议

1. **启用 InfluxDB 认证**：修改 `influxdb.conf` 中 `auth-enabled = true`
2. **配置 HTTPS**：使用 nginx 反向代理，配置 TLS 证书
3. **资源限制**：在 docker-compose.yml 中添加 `mem_limit` 和 `cpus`
4. **告警通知**：配置 Prometheus Alertmanager 发送告警
5. **数据备份**：设置定时任务备份 InfluxDB 和 Prometheus
6. **日志聚合**：使用 ELK 或 Loki 收集容器日志
7. **滚动更新**：使用 `docker-compose up -d --no-deps backend` 零停机更新
