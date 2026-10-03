# Tudou Image

土豆（Tudou / ai-tudou）出图协议插件，适配「同步 generations/edits + 异步任务轮询」的土豆网关形状。

本插件由 BeefTV Contributors 维护，用于适配土豆协议网关接口；不表示 BeefTV 与该服务商存在隶属、授权或合作关系。网关地址与模型列表由渠道配置提供，协议层不绑定具体域名，也不写死模型对照表。

- `tudou-image`：推荐。异步提交 `POST /v1/images/generations/async`，轮询 `GET /v1/tasks/{task_id}`；由后端任务持久化驱动，重启后仍可继续取图。覆盖 gpt-image-2 / gpt-image-2.5 异步系列，质量档位 low/medium/high/xhigh/max。
- `tudou-image-sync`：同步直出。文生图 `POST /v1/images/generations`，图生图 `POST /v1/images/edits`（JSON body 携带 `images` 参考图数组）；适合未实现异步接口的网关。
- `tudou-image-gemini`：Gemini 兼容路线。`POST /v1beta/models/{model}:generateContent`，参考图 inlineData 内联，响应取 candidates 的 inlineData；适配网关的 Gemini 形状模型（如 gemini-3 系图片模型）。

完整接口见 [docs/interface.md](docs/interface.md)。
