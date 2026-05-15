# LLM Probe

Claude API 网关检测服务 — 渠道真伪鉴别、智商评测、长期监控、告警。

## 功能

- **渠道检测** — 15 个探针覆盖指纹、结构、签名、行为、多模态五个维度，自动评分和修复建议
- **智商评测** — 基于 SWE-Atlas-QnA 等静态数据集，支持 effort/thinking 矩阵，按官方 key 基线做相对对比
- **长期监控** — 定时调度，channel 轻量先跑、异常升级 full；benchmark 高信号采样先跑、偏差超限升级采样
- **告警** — 规则引擎 + Webhook 推送，支持渠道分数、智商通过率、偏差等多维度触发
- **持久化** — SQLite 存储，重启不丢数据

## 支持模型

| 模型 | Thinking | Effort 级别 |
|------|----------|-------------|
| claude-opus-4-7 | adaptive | low · medium · high · xhigh · max |
| claude-opus-4-6 | adaptive | low · medium · high · max |
| claude-sonnet-4-6 | adaptive | low · medium · high · max |
| claude-opus-4-5 | enabled | low · medium · high |
| claude-haiku-4-5 | enabled | 不支持 |

## 快速开始

### Docker（推荐）

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml 填入 upstream API key（可选）

docker compose up -d
# 访问 http://localhost:8080
```

### 本地编译

```bash
go build -o detector-service .
cp config.example.yaml config.yaml
./detector-service -config config.yaml
```

### 直接部署到服务器

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o detector-service .
scp detector-service config.yaml user@server:/opt/llm-probe/
ssh user@server "cd /opt/llm-probe && ./detector-service -config config.yaml"
```

## 配置

```yaml
server:
  listen: ":8080"

upstream:
  base_url: "https://api.anthropic.com"   # 可选默认目标
  api_key: "${CHANNEL_TARGET_KEY:-}"
  timeout: 300

storage:
  path: "data/detector.db"

alert:
  enabled: true
  webhooks:
    - name: "feishu"
      url: "${ALERT_WEBHOOK_URL:-}"
```

完整示例见 [config.example.yaml](config.example.yaml)。

## API

### 渠道检测

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/channel/run` | 运行渠道检测（单模型/多模型） |
| POST | `/api/channel/run/stream` | SSE 运行渠道检测 |
| GET | `/api/channel/report` | 获取缓存的检测报告 |
| GET | `/api/channel/history` | 检测历史列表 |
| GET | `/api/channel/probes` | 获取探针元数据与文档路径 |
| GET | `/api/channel/checks` | 获取 check 元数据与文档路径 |

### 智商评测

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/intelligence/datasets` | 数据集列表 |
| POST | `/api/intelligence/datasets/{name}/run` | 运行评测 |
| POST | `/api/intelligence/datasets/{name}/stream` | SSE 流式运行 |
| GET | `/api/intelligence/history` | 评测历史 |

### 长期监控

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/api/monitor/targets` | 监控目标 CRUD |
| POST | `/api/monitor/targets/{id}/run` | 手动触发运行 |
| GET | `/api/monitor/runs` | 运行记录 |
| GET/POST | `/api/monitor/baselines` | 基线管理 |
| GET | `/api/monitor/status` | 健康状态总览 |

### 告警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alert/events` | 告警事件列表 |
| GET | `/api/alert/rules` | 告警规则列表 |

### 元数据

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/meta/models` | 获取模型能力、thinking/effort 与价格元数据 |

## 文档

- [整体架构](docs/architecture.md)
- [检测模型与评分语义](docs/detection-model.md)
- [Channel probe 文档索引](docs/channel-probes/README.md)
- [官方 key baseline 对比](docs/benchmark/official-baseline.md)
- [长期 staged monitoring](docs/monitor/staged-monitoring.md)
- [SQLite schema 与持久化边界](docs/db-schema.md)

## 项目结构

```
internal/
  channeltest/   渠道检测探针、评分、报告
  intelligence/  智商评测运行器、数据集、评估器
  monitor/       长期监控调度、基线、偏差计算
  alert/         告警规则、评估、Webhook
  persist/       SQLite 持久化层
  api/           HTTP API handlers
  config/        配置加载
web/static/      前端 SPA（纯 JS，无构建步骤）
```

## License

Private.
