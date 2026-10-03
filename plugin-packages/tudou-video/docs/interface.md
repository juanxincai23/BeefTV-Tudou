# Tudou Video 接口字段

## 协议身份

- 插件 ID：`tudou-video`。
- Provider：`tudou-video-sora2`、`tudou-video-veo3`、`tudou-video-kling`、`tudou-video-pixverse`、`tudou-video-seedance`、`tudou-video-grok`、`tudou-video-wan`。
- 能力：`video`。
- 默认 Base URL：无。土豆网关地址因部署而异，必须由渠道显式配置。
- 鉴权驱动：`bearer`（`Authorization: Bearer <apiKey>`）。
- 提交：`POST /v1/videos/generations`，JSON body。
- 轮询：`GET /v1/tasks/{task_id}`，与土豆异步图片共用同一任务接口。
- 提交响应若已携带结果地址则直接当作完成，否则进入轮询。
- 同步与异步由路径决定，协议层不按模型名切换流程；模型对照表由渠道配置。

## 配置字段

| 字段 | 类型 | 必填 | 含义 |
| --- | --- | --- | --- |
| `apiKey` | secret | 是 | API Key |

## 统一字段映射

六个 provider 共用同一套统一字段，仅 body 形状按家族不同。

| 统一字段 | 类型 | 必填 | 上游映射 | 说明 |
| --- | --- | --- | --- | --- |
| `model` | string | 是 | `model` | 视频模型 ID，使用网关登记的规范名（如 seedance-2.0/-2.0-face/-2.5/-mini、minimax-h3-2k）。 |
| `prompt` | string | 是 | `prompt` | 视频提示词。 |
| `images` | media[] | 否 | `images` / `first_frame_image` / `last_frame_image` / `img_references` / `extra.reference_images` | 参考图，role=first_frame/last_frame 参与首尾帧语义，role=mask 不参与发送。 |
| `videos` | media[] | 否 | `videos` | 参考视频，仅 seedance 系发送。 |
| `audios` | media[] | 否 | `audios` | 参考音频，仅在已有图像或视频参考时随请求发送。 |
| `duration` | integer | 否 | `duration` | 时长（秒）；合法档位由模型能力配置约束，缺省值也由能力配置给定额定档位。 |
| `aspectRatio` | string | 否 | `aspect_ratio`（pixverse 为 `size`） | 画面比例，如 16:9；grok 系允许 1:1。 |
| `resolution` | string | 否 | `resolution` | 分辨率档位，缺省 720p；sora2/kling/grok 系不发送。 |
| `generateAudio` | boolean | 否 | `generate_audio` | 是否生成音频；seedance 系仅在为真时发送，pixverse 的 `audio` 是请求布尔字段（由界面音频开关 `generateAudio` 映射）。 |
| `providerOptions` | object | 否 | 插件命名空间扩展字段 | `body`/`extra_body` 整包合并进提交 body。 |

## 家族请求形状

| Provider | 专属字段与规则 |
| --- | --- |
| `tudou-video-sora2` | `generate_audio` 恒发送（布尔）；参考图语义上仅 1 张；时长档位 4/8/12，缺省 8。 |
| `tudou-video-veo3` | `resolution` 缺省 720p；时长档位 4/6/8，缺省 8；参考图 ≤3，多于 2 张发 `reference_mode: "image"`，否则 `"frame"`。 |
| `tudou-video-kling` | 仅图生视频，至少 1 张参考图（kling-v3 ≤2 张）；时长 kling-v3 为 5/10/15 缺省 10，其余（如 omni）为 5/15 缺省 15。 |
| `tudou-video-pixverse` | `size` 承载比例（16:9/4:3/1:1/3:4/9:16/2:3/3:2/21:9）；role=first_frame/last_frame 分别进 `first_frame_image`/`last_frame_image`，其余参考图进 `img_references`；时长 1..15 缺省 5；`audio` 由模型名 `-audio` 后缀决定，可用 `providerOptions.tudou-video-pixverse.audio` 覆盖。 |
| `tudou-video-seedance` | 图像按 首帧→其余→末帧 排序，2.5 系最多 30 张、其余 9 张；`videos` ≤3；`audios` ≤3 且仅在有图像/视频参考时发送；时长 2.5 系 4..30 缺省 10，其余 4..15 缺省 4；face 系固定 720p。 |
| `tudou-video-grok` | 参考图进 `extra.reference_images`（`{url, role}`，role 为 first_frame/last_frame/reference_image），≤7 张且需公网可访问；`resolution` 恒 720p；时长档位 6/10/15，缺省 6。 |
| `tudou-video-wan` | 万相 All-in-One：`duration` 必填（2–30，有参考视频 2–15，禁止 -1，缺省兜底 5）；`aspect_ratio` 只允许 auto/16:9/9:16/1:1（无默认，兜底 16:9）；`resolution` 仅 720p/1080p 缺省 720p；`extra` 镜像 aspect_ratio 与 resolution；参考图 ≤10，role=first_frame/last_frame 进 `first_frame_image`/`last_frame_image`；`videos` 仅 wan3.0-video / wan3.0-video-fast 支持；素材必须公网 URL。 |

## 上游请求模板逐字段清单

六个 provider 的提交均为：

| 上游位置 | 值或转换表达式 |
| --- | --- |
| `create.method` | `"POST"` |
| `create.path` | `"/v1/videos/generations"` |
| `create.contentType` | `"application/json"` |
| `create.body.model` | `{"$ref":"request.model"}` |
| `create.body.prompt` | `{"$ref":"request.prompt"}` |
| `create.body.*` | 其余字段按上表家族规则映射 |
| `create.body.providerOptions.*.body` | 整包合并进提交 body 的开放 schema |
| `poll.method` | `"GET"` |
| `poll.path` | `"/v1/tasks/{{taskId}}"` |

## Provider 扩展键

- `providerOptions.tudou-video-sora2.body`（或 `extra_body`）
- `providerOptions.tudou-video-veo3.body`（或 `extra_body`）
- `providerOptions.tudou-video-kling.body`（或 `extra_body`）
- `providerOptions.tudou-video-pixverse.body`、`.audio`
- `providerOptions.tudou-video-seedance.body`（或 `extra_body`）
- `providerOptions.tudou-video-grok.body`（或 `extra_body`）

## 响应映射逐字段清单

| 映射位置 | 上游路径或转换表达式 |
| --- | --- |
| `response.taskId` | `data.id` → `data.task_id` → `id` → `task_id` → 既有 taskId |
| `response.status` | `data.status` → `data.task_status` → `status` → `task_status`，再回退 pending |
| `response.message` | `data.error.message` → `error.message` → `data.message` → `data.fail_reason` → `message` → `fail_reason` |
| `response.videos` | `data.result.video_url` → `data.result.url` → 遍历 `data.result.videos`/`data.result.outputs`（条目取 `url[0]`/`url`/`video_url`/字符串本身）→ `data.metadata.url`/`data.metadata.video_url` |
| `response.errorPaths` | `error.code`、`data.error.code` |
| `response.messagePaths` | `data.error.message`、`error.message`、`message`、`msg` |
| `response.resultEphemeral` | `true`，临时视频地址由宿主立即下载持久化 |

## 响应与错误

插件把上游 task/status/media 映射为统一结果。取视频按 `data.result` 内的 `video_url`/`url`/`videos`/`outputs` 顺序提取，条目里的 `url` 数组取第一项。状态大小写不敏感：`submitted` 等进入等待，`processing` 等进入处理中，`completed`（及 `complete/done/success` 等等价词）归一为成功，`failed/error/expired/cancelled` 归一为失败；失败信息取 `data.error.{code,message,type}`。提交返回 HTTP 200 但缺少任务 ID 时按失败处理，不会伪造成功。任务由后端持久化，重启后终态结果仍可继续轮询取回。

## 兼容边界

- role=mask 的蒙版媒体不参与任何家族的请求；蒙版编辑不是视频协议语义。
- `tudou-video-grok` 的参考图要求公网可访问；本地素材会先由宿主下载并以上传后的可访问地址引用。
- kling-v3 系没有文生视频：不提供参考图时网关会拒绝，能力配置层已把该协议的默认操作限定为图生视频。
- pixverse 的 `audio` 是请求布尔字段；seedance 的 `generate_audio` 仅在开启时发送。
- 模型名只用于路由到渠道配置的模型；seedance 系请使用网关登记的规范名，协议层不做改名映射。

<!-- BEEFTV_PLUGIN_MANIFEST_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "beeftv.plugin/v2",
  "id": "tudou-video",
  "name": "Tudou Video",
  "version": "1.0.0",
  "author": "BeefTV Contributors",
  "description": "土豆（ai-tudou）视频协议插件：统一 /v1/videos/generations 提交与 /v1/tasks 轮询，按模型家族拆分请求形状。",
  "permissions": [
    "generation.run",
    "media.read"
  ],
  "configuration": {
    "fields": [
      {
        "name": "apiKey",
        "type": "secret",
        "label": "API Key",
        "required": true
      }
    ]
  },
  "contributes": {
    "providers": [
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-sora2",
        "label": "Tudou Video Sora2",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "generateAudio",
            "type": "boolean",
            "required": false,
            "mapping": "generate_audio",
            "description": "是否生成音频。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "aspect_ratio": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "generate_audio": {
                  "$ref": "request.generateAudio"
                },
                "images": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "media",
                          "where": {
                            "$ne": [
                              {
                                "$ref": "media.role"
                              },
                              "mask"
                            ]
                          }
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-sora2.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-sora2.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-veo3",
        "label": "Tudou Video Veo3",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位，缺省 720p。"
          },
          {
            "name": "generateAudio",
            "type": "boolean",
            "required": false,
            "mapping": "generate_audio",
            "description": "是否生成音频。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "resolution": {
                  "$omitEmpty": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$in": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.resolution"
                                  }
                                }
                              },
                              [
                                "",
                                "auto"
                              ]
                            ]
                          },
                          "then": "720p"
                        }
                      ],
                      "default": {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.resolution"
                          }
                        }
                      }
                    }
                  }
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "aspect_ratio": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "generate_audio": {
                  "$ref": "request.generateAudio"
                },
                "images": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "media",
                          "where": {
                            "$ne": [
                              {
                                "$ref": "media.role"
                              },
                              "mask"
                            ]
                          }
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "reference_mode": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$gt": [
                          {
                            "$len": {
                              "$ref": "request.images"
                            }
                          },
                          0
                        ]
                      },
                      "then": {
                        "$if": {
                          "condition": {
                            "$gt": [
                              {
                                "$len": {
                                  "$ref": "request.images"
                                }
                              },
                              2
                            ]
                          },
                          "then": "image",
                          "else": "frame"
                        }
                      },
                      "else": null
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-veo3.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-veo3.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-kling",
        "label": "Tudou Video Kling",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "generateAudio",
            "type": "boolean",
            "required": false,
            "mapping": "generate_audio",
            "description": "是否生成音频。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "aspect_ratio": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "generate_audio": {
                  "$ref": "request.generateAudio"
                },
                "images": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "media",
                          "where": {
                            "$ne": [
                              {
                                "$ref": "media.role"
                              },
                              "mask"
                            ]
                          }
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-kling.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-kling.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-pixverse",
        "label": "Tudou Video PixVerse",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位，缺省 720p。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "resolution": {
                  "$omitEmpty": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$in": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.resolution"
                                  }
                                }
                              },
                              [
                                "",
                                "auto"
                              ]
                            ]
                          },
                          "then": "720p"
                        }
                      ],
                      "default": {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.resolution"
                          }
                        }
                      }
                    }
                  }
                },
                "size": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "first_frame_image": {
                  "$omitEmpty": {
                    "$first": {
                      "$map": {
                        "from": {
                          "$filter": {
                            "from": {
                              "$ref": "request.images"
                            },
                            "as": "media",
                            "where": {
                              "$eq": [
                                {
                                  "$lower": {
                                    "$trim": {
                                      "$ref": "media.role"
                                    }
                                  }
                                },
                                "first_frame"
                              ]
                            }
                          }
                        },
                        "as": "media",
                        "in": {
                          "$ref": "media.value"
                        }
                      }
                    }
                  }
                },
                "last_frame_image": {
                  "$omitEmpty": {
                    "$first": {
                      "$map": {
                        "from": {
                          "$filter": {
                            "from": {
                              "$ref": "request.images"
                            },
                            "as": "media",
                            "where": {
                              "$eq": [
                                {
                                  "$lower": {
                                    "$trim": {
                                      "$ref": "media.role"
                                    }
                                  }
                                },
                                "last_frame"
                              ]
                            }
                          }
                        },
                        "as": "media",
                        "in": {
                          "$ref": "media.value"
                        }
                      }
                    }
                  }
                },
                "img_references": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "media",
                          "where": {
                            "$and": [
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "first_frame"
                                ]
                              },
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "last_frame"
                                ]
                              },
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "mask"
                                ]
                              }
                            ]
                          }
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "audio": {
                  "$ref": "request.generateAudio"
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-pixverse.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-pixverse.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-seedance",
        "label": "Tudou Video Seedance",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "videos",
            "type": "media[]",
            "required": false,
            "mapping": "videos",
            "description": "参考视频，公网 URL 或 data URL。"
          },
          {
            "name": "audios",
            "type": "media[]",
            "required": false,
            "mapping": "audios",
            "description": "参考音频，仅在已有图像或视频参考时随请求发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位，缺省 720p。"
          },
          {
            "name": "generateAudio",
            "type": "boolean",
            "required": false,
            "mapping": "generate_audio",
            "description": "是否生成音频。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "aspect_ratio": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "resolution": {
                  "$omitEmpty": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$in": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.resolution"
                                  }
                                }
                              },
                              [
                                "",
                                "auto"
                              ]
                            ]
                          },
                          "then": "720p"
                        }
                      ],
                      "default": {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.resolution"
                          }
                        }
                      }
                    }
                  }
                },
                "generate_audio": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$ref": "request.generateAudio"
                      },
                      "then": true,
                      "else": null
                    }
                  }
                },
                "images": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$concatArrays": [
                          {
                            "$filter": {
                              "from": {
                                "$ref": "request.images"
                              },
                              "as": "media",
                              "where": {
                                "$eq": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "first_frame"
                                ]
                              }
                            }
                          },
                          {
                            "$filter": {
                              "from": {
                                "$ref": "request.images"
                              },
                              "as": "media",
                              "where": {
                                "$and": [
                                  {
                                    "$ne": [
                                      {
                                        "$lower": {
                                          "$trim": {
                                            "$ref": "media.role"
                                          }
                                        }
                                      },
                                      "first_frame"
                                    ]
                                  },
                                  {
                                    "$ne": [
                                      {
                                        "$lower": {
                                          "$trim": {
                                            "$ref": "media.role"
                                          }
                                        }
                                      },
                                      "last_frame"
                                    ]
                                  },
                                  {
                                    "$ne": [
                                      {
                                        "$lower": {
                                          "$trim": {
                                            "$ref": "media.role"
                                          }
                                        }
                                      },
                                      "mask"
                                    ]
                                  }
                                ]
                              }
                            }
                          },
                          {
                            "$filter": {
                              "from": {
                                "$ref": "request.images"
                              },
                              "as": "media",
                              "where": {
                                "$eq": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "last_frame"
                                ]
                              }
                            }
                          }
                        ]
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "videos": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$ref": "request.videos"
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "audios": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$or": [
                          {
                            "$gt": [
                              {
                                "$len": {
                                  "$ref": "request.images"
                                }
                              },
                              0
                            ]
                          },
                          {
                            "$gt": [
                              {
                                "$len": {
                                  "$ref": "request.videos"
                                }
                              },
                              0
                            ]
                          }
                        ]
                      },
                      "then": {
                        "$map": {
                          "from": {
                            "$ref": "request.audios"
                          },
                          "as": "media",
                          "in": {
                            "$ref": "media.value"
                          }
                        }
                      },
                      "else": null
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-seedance.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-seedance.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-grok",
        "label": "Tudou Video Grok",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": false,
            "mapping": "duration",
            "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "aspect_ratio/size",
            "description": "画面比例，如 16:9。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "duration": {
                  "$omitEmpty": {
                    "$ref": "request.duration"
                  }
                },
                "aspect_ratio": {
                  "$omitEmpty": {
                    "$ref": "request.aspectRatio"
                  }
                },
                "resolution": "720p",
                "extra": {
                  "aspect_ratio": {
                    "$omitEmpty": {
                      "$ref": "request.aspectRatio"
                    }
                  },
                  "resolution": "720p",
                  "reference_images": {
                    "$omitEmpty": {
                      "$map": {
                        "from": {
                          "$filter": {
                            "from": {
                              "$ref": "request.images"
                            },
                            "as": "media",
                            "where": {
                              "$ne": [
                                {
                                  "$ref": "media.role"
                                },
                                "mask"
                              ]
                            }
                          }
                        },
                        "as": "media",
                        "in": {
                          "url": {
                            "$ref": "media.value"
                          },
                          "role": {
                            "$switch": {
                              "cases": [
                                {
                                  "when": {
                                    "$in": [
                                      {
                                        "$lower": {
                                          "$trim": {
                                            "$ref": "media.role"
                                          }
                                        }
                                      },
                                      [
                                        "first_frame",
                                        "last_frame"
                                      ]
                                    ]
                                  },
                                  "then": {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  }
                                }
                              ],
                              "default": "reference_image"
                            }
                          }
                        }
                      }
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-grok.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-grok.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "capabilities": [
          "video"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "requiresPublicMediaUrls": false,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "id": "tudou-video-wan",
        "label": "Tudou Video Wan",
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "wan3.0 / wan3.0-fast / wan3.0-video / wan3.0-video-fast。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "视频描述，可用「图1」「视频1」指代素材顺序。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images / first_frame_image / last_frame_image",
            "description": "参考图（公网 URL），最多 10 张；role=first_frame/last_frame 进首尾帧字段，role=mask 不发送。"
          },
          {
            "name": "videos",
            "type": "media[]",
            "required": false,
            "mapping": "videos",
            "description": "参考视频，仅 wan3.0-video / wan3.0-video-fast 支持。"
          },
          {
            "name": "duration",
            "type": "integer",
            "required": true,
            "mapping": "duration",
            "description": "时长秒数，2–30（有参考视频 2–15），禁止 -1。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": true,
            "mapping": "aspect_ratio",
            "description": "只允许 auto / 16:9 / 9:16 / 1:1，无默认值。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "仅 720p / 1080p，缺省 720p。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/videos/generations",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "model": {
                  "$ref": "request.model"
                },
                "prompt": {
                  "$ref": "request.prompt"
                },
                "duration": {
                  "$if": {
                    "condition": {
                      "$gt": [
                        {
                          "$ref": "request.duration"
                        },
                        0
                      ]
                    },
                    "then": {
                      "$ref": "request.duration"
                    },
                    "else": 5
                  }
                },
                "aspect_ratio": {
                  "$switch": {
                    "cases": [
                      {
                        "when": {
                          "$in": [
                            {
                              "$lower": {
                                "$trim": {
                                  "$ref": "request.aspectRatio"
                                }
                              }
                            },
                            [
                              "auto",
                              "16:9",
                              "9:16",
                              "1:1"
                            ]
                          ]
                        },
                        "then": {
                          "$lower": {
                            "$trim": {
                              "$ref": "request.aspectRatio"
                            }
                          }
                        }
                      }
                    ],
                    "default": "16:9"
                  }
                },
                "resolution": {
                  "$switch": {
                    "cases": [
                      {
                        "when": {
                          "$in": [
                            {
                              "$lower": {
                                "$trim": {
                                  "$ref": "request.resolution"
                                }
                              }
                            },
                            [
                              "720p",
                              "1080p"
                            ]
                          ]
                        },
                        "then": {
                          "$lower": {
                            "$trim": {
                              "$ref": "request.resolution"
                            }
                          }
                        }
                      }
                    ],
                    "default": "720p"
                  }
                },
                "images": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$filter": {
                          "from": {
                            "$ref": "request.images"
                          },
                          "as": "media",
                          "where": {
                            "$and": [
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "first_frame"
                                ]
                              },
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "last_frame"
                                ]
                              },
                              {
                                "$ne": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "media.role"
                                      }
                                    }
                                  },
                                  "mask"
                                ]
                              }
                            ]
                          }
                        }
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "videos": {
                  "$omitEmpty": {
                    "$map": {
                      "from": {
                        "$ref": "request.videos"
                      },
                      "as": "media",
                      "in": {
                        "$ref": "media.value"
                      }
                    }
                  }
                },
                "first_frame_image": {
                  "$omitEmpty": {
                    "$first": {
                      "$map": {
                        "from": {
                          "$filter": {
                            "from": {
                              "$ref": "request.images"
                            },
                            "as": "media",
                            "where": {
                              "$eq": [
                                {
                                  "$lower": {
                                    "$trim": {
                                      "$ref": "media.role"
                                    }
                                  }
                                },
                                "first_frame"
                              ]
                            }
                          }
                        },
                        "as": "media",
                        "in": {
                          "$ref": "media.value"
                        }
                      }
                    }
                  }
                },
                "last_frame_image": {
                  "$omitEmpty": {
                    "$first": {
                      "$map": {
                        "from": {
                          "$filter": {
                            "from": {
                              "$ref": "request.images"
                            },
                            "as": "media",
                            "where": {
                              "$eq": [
                                {
                                  "$lower": {
                                    "$trim": {
                                      "$ref": "media.role"
                                    }
                                  }
                                },
                                "last_frame"
                              ]
                            }
                          }
                        },
                        "as": "media",
                        "in": {
                          "$ref": "media.value"
                        }
                      }
                    }
                  }
                },
                "extra": {
                  "aspect_ratio": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$in": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.aspectRatio"
                                  }
                                }
                              },
                              [
                                "auto",
                                "16:9",
                                "9:16",
                                "1:1"
                              ]
                            ]
                          },
                          "then": {
                            "$lower": {
                              "$trim": {
                                "$ref": "request.aspectRatio"
                              }
                            }
                          }
                        }
                      ],
                      "default": "16:9"
                    }
                  },
                  "resolution": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$in": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.resolution"
                                  }
                                }
                              },
                              [
                                "720p",
                                "1080p"
                              ]
                            ]
                          },
                          "then": {
                            "$lower": {
                              "$trim": {
                                "$ref": "request.resolution"
                              }
                            }
                          }
                        }
                      ],
                      "default": "720p"
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-video-wan.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-video-wan.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.data.task_status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.task_status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.data.error.message"
              },
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.data.message"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              },
              {
                "$ref": "response.fail_reason"
              }
            ]
          },
          "videos": {
            "$coalesce": [
              {
                "$ref": "response.data.result.video_url"
              },
              {
                "$ref": "response.data.result.url"
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.videos"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$map": {
                  "from": {
                    "$ref": "response.data.result.outputs"
                  },
                  "as": "item",
                  "in": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      },
                      {
                        "$ref": "item.video_url"
                      },
                      {
                        "$ref": "item"
                      }
                    ]
                  }
                }
              },
              {
                "$ref": "response.data.metadata.url"
              },
              {
                "$ref": "response.data.metadata.video_url"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "data.error.message",
            "error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      }
    ]
  },
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>"
}
```
<!-- BEEFTV_PLUGIN_MANIFEST_END -->
