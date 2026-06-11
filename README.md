# 古代木结构建筑虫蛀监测与熏蒸剂智能释放系统

## 项目概述

本系统针对古建筑群（应县木塔、佛光寺）的木结构虫蛀监测与防治，构建了一套完整的智能监测与熏蒸剂释放系统。

### 核心功能

- **声发射监测**：50台声发射传感器，实时监测白蚁活动
- **含水率监测**：40台木材含水率传感器，监测木材湿度
- **LoRa无线传输**：每小时上报一次传感器数据
- **智能分析**：
  - 基于小波包能量谱的声发射信号特征提取
  - 基于LSTM的白蚁活动强度预测
  - 高斯烟羽模型的熏蒸剂扩散浓度场模拟
- **三维可视化**：基于Three.js的木结构三维模型与风险体素标注
- **智能告警**：企业微信、短信推送告警

## 项目结构

```
.
├── backend/                 # Go 后端服务
│   ├── cmd/server/         # 服务入口
│   ├── internal/
│   │   ├── handlers/       # API 处理器
│   │   ├── services/       # 业务逻辑服务
│   │   ├── algorithms/     # 核心算法
│   │   └── models/         # 数据模型
│   └── config/             # 配置文件
├── frontend/               # 前端应用
│   └── public/             # 静态资源
│       ├── index.html      # 主页面
│       └── app.js          # 三维可视化逻辑
├── influxdb/               # InfluxDB 配置
└── lora-simulator/         # LoRa 传感器模拟器
```

## 技术栈

- **后端**：Go 1.21 + Gin Framework
- **数据库**：InfluxDB 1.8 (时序数据库)
- **前端**：HTML5 Canvas + Three.js + Chart.js
- **通信协议**：LoRaWAN (模拟)
- **核心算法**：
  - 小波包变换 (Daubechies 4小波基)
  - LSTM 时间序列预测
  - 高斯烟羽模型 (Pasquill-Gifford扩散参数)

## 快速开始

### 1. 启动 InfluxDB

```bash
cd influxdb
docker-compose up -d
```

### 2. 启动后端服务

```bash
cd backend
go mod tidy
go run cmd/server/main.go
```

服务将在 http://localhost:8080 启动

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

或直接使用任何静态文件服务器访问 `frontend/public` 目录。

### 4. 启动 LoRa 模拟器

```bash
cd lora-simulator
go run main.go
```

模拟器将每3秒发送一批传感器数据到后端API。

## API 接口

### LoRa数据接收
- `POST /api/v1/lora/data` - 接收LoRa传感器数据

### 传感器管理
- `GET /api/v1/sensors` - 获取传感器列表
- `GET /api/v1/sensors/:id` - 获取单个传感器详情
- `GET /api/v1/buildings` - 获取建筑列表

### 数据查询
- `GET /api/v1/data/acoustic` - 查询声发射数据
- `GET /api/v1/data/moisture` - 查询含水率数据

### 告警
- `GET /api/v1/alerts` - 获取活动告警列表

### 风险分析
- `GET /api/v1/risk-zones` - 获取风险区域分布
- `GET /api/v1/predict/termite` - 白蚁活动预测

### 熏蒸模拟
- `POST /api/v1/simulate/fumigation` - 熏蒸剂扩散模拟

### 信号分析
- `GET /api/v1/analysis/wavelet` - 小波包能量谱分析

## 告警规则

1. **声发射告警**：事件率 > 100次/小时 触发
2. **含水率告警**：含水率 > 25% 触发

告警级别：
- 严重 (Critical)
- 高 (High)  
- 中 (Medium)
- 低 (Low)

## 建筑信息

### 应县木塔
- 全称：佛宫寺释迦塔
- 位置：山西省朔州市应县
- 建成：辽清宁二年 (1056年)
- 高度：67.31米
- 层数：5层（明五暗四）
- 传感器：声发射30台 + 含水率25台

### 佛光寺
- 位置：山西省五台县
- 建成：唐大中十一年 (857年)
- 高度：约20米
- 传感器：声发射25台 + 含水率20台

## 核心算法说明

### 小波包能量谱
- 采用Daubechies 4 (db4) 小波基
- 5层分解，得到32个频带
- 提取特征：总能量、各频带能量比、谱熵、谱质心等

### LSTM预测
- 输入：历史24小时声发射特征
- 输出：未来24小时白蚁活动强度预测
- 隐藏层：32个LSTM单元

### 高斯烟羽模型
- 基于Pasquill-Gifford扩散参数
- 考虑风速、风向、大气稳定度
- 输出三维浓度场分布

## 告警推送

支持两种告警推送方式：
1. **企业微信机器人** - Webhook方式
2. **短信通知** - HTTP API方式

请在 `backend/config/config.yaml` 中配置相应的API地址和密钥。

## 注意事项

1. 本系统为演示版本，部分数据为模拟生成
2. LSTM模型权重为随机初始化，实际使用需训练
3. 熏蒸剂模拟基于简化的高斯烟羽模型，实际应用需结合CFD
4. LoRa模拟器用于开发测试，实际部署需替换为真实LoRa网关
