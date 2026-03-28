# 赛博视奸-Linux版

一个面向 Linux 桌面环境的轻量状态采集项目。

当前提供两部分：

- 客户端 `cmd/linux-stalk-client`
  采集当前桌面系统状态、AT-SPI 可访问性信息、媒体/蓝牙/Wi-Fi/电源信息，并可在窗口切换时按节流规则上报到服务端。
- 服务端 `cmd/linux-stalk-server`
  提供最小 ingest 接口和只读查询接口，使用 SQLite 落库存储。

## 目前支持

- 枚举 accessibility 树中的应用
- 获取焦点相关 AT-SPI 事件 （Gnome 可用，KDE 暂不支持）
- 过滤窗口切换和有效焦点事件
- 采集系统状态
  - 电源
  - Wi-Fi SSID
  - 蓝牙已连接设备
  - MPRIS 媒体会话
- 客户端配置文件化
  - `server_url`
  - `api_key`（设备 key）
  - `device_id`
- 服务端配置文件化
  - `devices`（设备列表，id+key 一一对应）
  - `admin_keys`（管理 key 列表）
- 服务端 SQLite 入库
- 服务端查询 API
  - `GET /healthz`
  - `GET /devices`
  - `GET /events/latest?device_id=...`
  - `GET /events?device_id=...&limit=...`

## 配置

示例配置：

- `configs/client.example.json`
- `configs/server.example.json`

服务端配置示例（关键字段）：

```json
{
  "listen_addr": ":8080",
  "db_path": "data/linux-stalk.db",
  "devices": [
    { "id": "device-id-1", "key": "key-12345678" },
    { "id": "device-id-2", "key": "key-abcdefg" }
  ],
  "admin_keys": ["sk-admin-123456", "sk-admin-abcdef"]
}
```

客户端仍然使用 `api_key` 字段，这个 key 就是设备 key。

服务端鉴权规则：

- `POST /ingest` 仅接受设备 key，并且 key 必须与 payload 里的 `device_id` 对应。
- `GET /devices` / `GET /events/latest` / `GET /events` 仅接受 `admin_keys`。

本地运行时可复制为：

- `configs/client.json`
- `configs/server.json`

## 运行

客户端快照：

```bash
go run ./cmd/linux-stalk-client --snapshot
```

客户端监听：

```bash
go run ./cmd/linux-stalk-client
```

客户端推送：

```bash
go run ./cmd/linux-stalk-client --push --config configs/client.json
```

服务端：

```bash
go run ./cmd/linux-stalk-server --config configs/server.json
```

## Web Dashboard

`web/` 提供一个独立的 Web 观察台，用来查看设备状态、最近动态和事件记录。
前端基于 React + Vite + Tailwind + HeroUI 构建，界面默认使用中文。

### 开发模式

```bash
cd web
pnpm install
pnpm dev
```

开发服务器启动后访问 http://localhost:5173。

默认情况下，Vite 会把 `/api` 请求代理到 `http://localhost:8080`，因此本地开发时通常只需要先启动服务端，再启动 `web`。

### 生产构建

```bash
cd web
pnpm build
```

构建产物位于 `web/dist` 目录，可直接用于静态部署。

### 环境变量

- `VITE_API_BASE_URL`：API 基础路径，默认 `/api`

### 使用方式

访问观察台后，输入服务端配置中的 `admin_keys` 之一即可进入。

当前包含 3 个主要页面：

- **总览**：查看整体活跃度、最近更新情况和最活跃设备
- **设备**：按搜索、活跃度和排序方式浏览全部设备
- **设备详情**：查看单台设备的最近状态、连接信息和事件历史

访问密钥只会保存在当前浏览器本地，用于后续只读查询。

### API 端点

观察台使用以下 API 端点（均需 `Authorization: Bearer <admin_key>` 认证）：

- `GET /devices`：获取设备列表
- `GET /events/latest?device_id=...`：获取指定设备最新事件
- `GET /events?device_id=...&limit=...`：获取指定设备事件列表
