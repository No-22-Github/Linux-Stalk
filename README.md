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
  - `api_key`
  - `device_id`
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
