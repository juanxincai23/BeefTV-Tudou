# Tudou Video

土豆（Tudou / ai-tudou）视频协议插件，适配「统一 `POST /v1/videos/generations` 提交 + `GET /v1/tasks/{task_id}` 轮询、按模型家族区分请求形状」的土豆网关视频接口。

本插件由 BeefTV Contributors 维护，用于适配土豆协议网关接口；不表示 BeefTV 与该服务商存在隶属、授权或合作关系。网关地址与模型列表由渠道配置提供，协议层不绑定具体域名，也不写死模型对照表。

六个 provider 对应补丁里的六个请求形状，按模型家族在渠道里为模型指定：

- `tudou-video-sora2`：sora2 系。
- `tudou-video-veo3`：veo3.1 系。
- `tudou-video-kling`：kling-v3 系（仅图生视频）。
- `tudou-video-pixverse`：pixverse-v6 系（含首尾帧）。
- `tudou-video-seedance`：seedance 系与 minimax-h3 系（多模态参考）。
- `tudou-video-grok`：grok-imagine-video 系（参考图进 extra）。
- `tudou-video-wan`：wan3.0 系（万相 All-in-One：文生/图生/首尾帧/多参考图/参考视频）。
- `tudou-video-wan`：wan3.0 系（万相 All-in-One：文生/图生/首尾帧/多参考图/参考视频）。

完整接口见 [docs/interface.md](docs/interface.md)。
