# Tudou Image 接口字段

## 协议身份

- 插件 ID：`tudou-image`。
- Provider：`tudou-image`（异步任务）、`tudou-image-sync`(同步直出)、`tudou-image-gemini`（Gemini 兼容路线）。
- 能力：`image`。
- 默认 Base URL：无。土豆网关地址因部署而异，必须由渠道显式配置。
- 鉴权驱动：`bearer`（`Authorization: Bearer <apiKey>`）。
- 异步创建：`POST /v1/images/generations/async`，立刻返回任务 ID，不阻塞出图。
- 异步轮询：`GET /v1/tasks/{task_id}`，取图路径固定为 `data.result.images[0].url[0]`，`url` 本身是数组。
- 同步文生图：`POST /v1/images/generations`，OpenAI 形 `{created, model, data:[{url}]}`。
- 同步图生图：`POST /v1/images/edits`，JSON body 携带 `images` 参考图数组（data URL 或公网 URL）；网关明确拒绝 multipart，参考图必须走 JSON。
- 同步与异步由路径决定，与模型名无关；不要按模型名后缀切换流程。

## 配置字段

| 字段 | 类型 | 必填 | 含义 |
| --- | --- | --- | --- |
| `apiKey` | secret | 是 | API Key |

## 统一字段映射

两个 provider 共用同一套统一字段。

| 统一字段 | 类型 | 必填 | 上游映射 | 说明 |
| --- | --- | --- | --- | --- |
| `model` | string | 是 | `model` | 图片模型 ID，由渠道模型列表配置，协议层不写死对照表。 |
| `prompt` | string | 是 | `prompt` | 图片提示词。 |
| `images` | media[] | 否 | `images`（JSON 数组，异步与同步一致） | 参考图或编辑源图，公网 URL 或 data URL，最多 9 张；role=mask 的蒙版不参与发送。 |
| `imageCount` | integer | 否 | `n` | 输出数量，缺省 1。 |
| `aspectRatio` | string | 否 | `size` | 比例串（`1:1`）或精确像素串（`1024x1024`）。 |
| `resolution` | string | 否 | `resolution` | 分辨率档位 `1k`/`2k`/`4k`，缺省 `1k`。 |
| `quality` | string | 否 | `quality` | 质量档位 `low`/`medium`/`high`/`xhigh`/`max`（异步与同步均为五档，2.5 系支持 xhigh/max）；恒发送，缺省 `medium`，与分辨率独立选择。 |
| `providerOptions` | object | 否 | 插件命名空间扩展字段 | 见「Provider 扩展键」。 |

## Gemini 兼容路线（tudou-image-gemini）

- 提交：`POST /v1beta/models/{{model}}:generateContent`，模型 ID 进入路径。
- 文生图 `contents[].parts` 只有 `{ "text": prompt }`；图生图 parts 前面是 `inlineData` 参考图，最后一条 `text` 是 prompt。
- `generationConfig`：`responseModalities: ["TEXT","IMAGE"]`；`imageConfig.aspectRatio` 缺省 `1:1`，`imageConfig.imageSize` 为 `1K/2K/4K`（大小写不敏感，发送前转大写），档位兜底顺序与异步一致：请求 resolution → quality 档位 → 模型名后缀 → 缺省 1k。
- 响应只走 `candidates[].content.parts[].inlineData`：取带图像数据的部件拼回 data URL，文本部件忽略；错误用 Gemini 形 `error.{code,message,status}`。
- 参考图限制：`inlineData` 需要 base64 内联（data URL）；公网 URL 参考图请改用 `tudou-image` 异步路线。
- 图生图必须有参考图，否则网关 400。

## 尺寸规则

`size` 两套形态：比例串（`1:1`，可叠加 `resolution`）与像素串（`1024x1024`）。异步 provider 恒发送 `size`（空或 `auto` 兜底 `1:1`）；同步 provider 保持像素串与 `resolution` 互斥，同时传会被上游拒绝。

`resolution` 只接受 `1k` / `2k` / `4k`，按多级兜底推导，绝不透传非法档位（画布会携带视频清晰度值如 `720p`，不能原样发给网关）：

1. 请求里的 `resolution` 是 `1k`/`2k`/`4k` 时原样使用（大小写不敏感）。
2. `size` 为像素串时按面积推导：面积 ≥ 4,000,000 → `4k`、≥ 2,000,000 → `2k`、否则 `1k`（对官方 15 比例 × 3 档的全部 45 个像素组合逐一校验通过）。
3. 统一字段的 `quality` 是 `1k`/`2k`/`4k` 档位时使用该档位。
4. `quality` 为 `low`/`medium`/`high` 时按别名映射 1k/2k/4k（与质量字段独立五档共存，`xhigh`/`max` 无别名按缺省 1k）。
5. 模型名以 `-1k`/`-2k`/`-4k` 结尾时使用该档位（如 `gpt-image-2-2k`）。
6. 否则缺省 `1k`。

分辨率与质量是两个独立维度：尺寸预设携带官方精确像素（如 1:1 · 4K = 2880x2880），质量五档（low→max）独立选择，任意组合合法。

异步路由不接受 `quality` 字段，档位只进 `resolution`；同步路由的 `quality` 与异步同为五档透传，缺省 medium。

插件不发明 `aspect_ratio` 入参，比例字段名就是 `size`。

## 上游请求模板逐字段清单

### tudou-image（异步提交）

| 上游位置 | 值或转换表达式 |
| --- | --- |
| `create.method` | `"POST"` |
| `create.path` | `"/v1/images/generations/async"` |
| `create.contentType` | `"application/json"` |
| `create.body.model` | `{"$ref":"request.model"}` |
| `create.body.prompt` | `{"$ref":"request.prompt"}` |
| `create.body.n` | imageCount > 0 时取 imageCount，否则 1 |
| `create.body.size` | aspectRatio 去空白小写后为空或 auto 时省略，否则原样发送 |
| `create.body.resolution` | aspectRatio 含 `x`（像素串）时省略；resolution 为空或 auto 时发 `1k`；否则小写发送 |
| `create.body.quality` | quality 为空或 auto 时省略；否则小写发送（上游缺省 medium） |
| `create.body.images` | request.images 中 role ≠ mask 的媒体值数组（URL 或 data URL）；旧字段 image_urls 已被网关弃用，会被静默忽略 |
| `create.body.response_format` | `providerOptions.tudou-image.response_format`，缺省省略（上游默认 url） |
| `poll.method` | `"GET"` |
| `poll.path` | `"/v1/tasks/{{taskId}}"` |

### tudou-image-sync（同步直出）

| 上游位置 | 值或转换表达式 |
| --- | --- |
| `create.path` | 有参考图走 `/v1/images/edits`，否则 `/v1/images/generations` |
| `create.contentType` | 恒为 `application/json`（网关拒绝 multipart） |
| `create.body.*` | model/prompt/n/size/resolution/quality/images 规则与异步一致 |

## Provider 扩展键

- `providerOptions.tudou-image.response_format`
- `providerOptions.tudou-image.body`（或 `extra_body`）：整包合并进异步提交 body 的开放 schema。
- `providerOptions.tudou-image-sync.response_format`
- `providerOptions.tudou-image-sync.body`（或 `extra_body`）：整包合并进同步 body 的开放 schema。

## 响应映射逐字段清单

### tudou-image（提交与轮询共用一套映射）

| 映射位置 | 上游路径或转换表达式 |
| --- | --- |
| `response.taskId` | `data.id`，回退 `id`/`task_id`/既有 taskId |
| `response.status` | `data.status`，回退 `status`，再回退 pending |
| `response.message` | `data.error.message` → `error.message` → `message` → `msg` |
| `response.images` | 遍历 `data.result.images`，每项取 `url[0]`（`url` 为数组；字符串形态也能兼容） |
| `response.errorPaths` | `error.code`、`data.error.code` |
| `response.messagePaths` | `data.error.message`、`error.message`、`message`、`msg` |
| `response.resultEphemeral` | `true`，临时媒体 URL 由宿主立即下载持久化 |

状态机：`submitted` → `processing` → `completed` | `failed`；没有 `succeeded`。宿主把 `completed` 归一为成功、`failed` 归一为失败，失败信息取 `data.error.{code,message,type}`。异步任务表由后端持久化，重启后终态结果仍可继续轮询取图。

### tudou-image-sync

| 映射位置 | 上游路径或转换表达式 |
| --- | --- |
| `response.status` | `"succeeded"` |
| `response.images` | 遍历 `data`，`url` 原样取用；`b64_json` 拼接为 `data:image/png;base64,...` 的 dataUrl |
| `response.usage` | `usage` 原样透传 |
| `response.errorPaths` | `error.code`、`data.error.code` |
| `response.messagePaths` | `error.message`、`data.error.message`、`message`、`msg` |
| `response.resultEphemeral` | `true`，带过期时间的临时 URL 由宿主立即下载持久化 |

## 响应与错误

插件把上游 task/status/media/usage 映射为统一结果。取图只认 `data.result.images[].url`（`url` 是数组，取第一项），不把扁平的 `data.image_url`/`data.url` 当唯一信封。HTTP 错误、业务 code 和 error object 保持失败语义，不包装成成功；错误判断用 `errorPaths`，不解析展示文案。提交阶段若返回 HTTP 200 但缺少 `data.id`，宿主按缺少任务 ID 处理，不会伪造成功。

## 兼容边界

- 参考图上限 9 张；公网 `http(s)` URL 原样交给网关拉取，data URL 直接内联；不会把会 403 的签名 CDN 长链当参考图塞给上游。
- 蒙版编辑不受支持：role=mask 的媒体不参与发送，需要蒙版编辑时请改用 OpenAI Images 协议。
- 同步 `edits` 端点只接受 JSON：实测网关对 multipart 直接报 `multipart/form-data is not supported`，参考图统一放 `images` 数组；字段名 `image_urls` 在该端点不被识别。
- 模型名只用于路由到渠道配置的模型，不携带「某模型 = 某上游版本」的永恒对照。
- `quality` 缺省 `medium`，插件层不擅自把 high 降级为 medium，由网关按自身配额决定。
- 网关若只实现同步接口，请选 `tudou-image-sync`；只实现异步接口则选 `tudou-image`。两套 provider 不会按模型名自动切换。

<!-- BEEFTV_PLUGIN_MANIFEST_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "beeftv.plugin/v2",
  "id": "tudou-image",
  "name": "Tudou Image",
  "version": "1.0.2",
  "author": "BeefTV Contributors",
  "description": "土豆（ai-tudou）出图协议插件：异步任务提交轮询与同步 generations/edits 两种形状。",
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
        "id": "tudou-image",
        "label": "Tudou Image",
        "capabilities": [
          "image"
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
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "图片模型 ID，由渠道模型列表配置，不在协议层写死。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "图片提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图或编辑源图，公网 URL 或 data URL，最多 9 张；role=mask 的蒙版不参与发送。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "n",
            "description": "输出数量，缺省 1。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "size",
            "description": "比例串（如 1:1）或精确像素串（如 1024x1024），语义按协议说明。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位 1k/2k/4k，缺省 1k；size 为像素串时必须省略。"
          },
          {
            "name": "quality",
            "type": "string",
            "required": false,
            "mapping": "resolution 档位（1k/2k/4k）；同步协议仅在 low/medium/high 时发送 quality",
            "description": "清晰度即分辨率档位 1k/2k/4k，缺省 1k。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（tudou-image.response_format、tudou-image.body）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/images/generations/async",
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
                "n": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$gt": [
                          {
                            "$ref": "request.imageCount"
                          },
                          0
                        ]
                      },
                      "then": {
                        "$ref": "request.imageCount"
                      },
                      "else": 1
                    }
                  }
                },
                "size": {
                  "$if": {
                    "condition": {
                      "$in": [
                        {
                          "$lower": {
                            "$trim": {
                              "$ref": "request.aspectRatio"
                            }
                          }
                        },
                        [
                          "",
                          "auto"
                        ]
                      ]
                    },
                    "then": "1:1",
                    "else": {
                      "$lower": {
                        "$trim": {
                          "$ref": "request.aspectRatio"
                        }
                      }
                    }
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
                              "1k",
                              "2k",
                              "4k"
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
                      },
                      {
                        "when": {
                          "$gt": [
                            {
                              "$len": {
                                "$split": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.aspectRatio"
                                      }
                                    }
                                  },
                                  "x"
                                ]
                              }
                            },
                            1
                          ]
                        },
                        "then": {
                          "$switch": {
                            "cases": [
                              {
                                "when": {
                                  "$gte": [
                                    {
                                      "$multiply": [
                                        {
                                          "$toInt": {
                                            "$first": {
                                              "$split": [
                                                {
                                                  "$lower": {
                                                    "$trim": {
                                                      "$ref": "request.aspectRatio"
                                                    }
                                                  }
                                                },
                                                "x"
                                              ]
                                            }
                                          }
                                        },
                                        {
                                          "$toInt": {
                                            "$last": {
                                              "$split": [
                                                {
                                                  "$lower": {
                                                    "$trim": {
                                                      "$ref": "request.aspectRatio"
                                                    }
                                                  }
                                                },
                                                "x"
                                              ]
                                            }
                                          }
                                        }
                                      ]
                                    },
                                    4000000
                                  ]
                                },
                                "then": "4k"
                              },
                              {
                                "when": {
                                  "$gte": [
                                    {
                                      "$multiply": [
                                        {
                                          "$toInt": {
                                            "$first": {
                                              "$split": [
                                                {
                                                  "$lower": {
                                                    "$trim": {
                                                      "$ref": "request.aspectRatio"
                                                    }
                                                  }
                                                },
                                                "x"
                                              ]
                                            }
                                          }
                                        },
                                        {
                                          "$toInt": {
                                            "$last": {
                                              "$split": [
                                                {
                                                  "$lower": {
                                                    "$trim": {
                                                      "$ref": "request.aspectRatio"
                                                    }
                                                  }
                                                },
                                                "x"
                                              ]
                                            }
                                          }
                                        }
                                      ]
                                    },
                                    2000000
                                  ]
                                },
                                "then": "2k"
                              }
                            ],
                            "default": "1k"
                          }
                        }
                      },
                      {
                        "when": {
                          "$in": [
                            {
                              "$last": {
                                "$split": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.model"
                                      }
                                    }
                                  },
                                  "-"
                                ]
                              }
                            },
                            [
                              "1k",
                              "2k",
                              "4k"
                            ]
                          ]
                        },
                        "then": {
                          "$last": {
                            "$split": [
                              {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.model"
                                  }
                                }
                              },
                              "-"
                            ]
                          }
                        }
                      }
                    ],
                    "default": "1k"
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
                "response_format": {
                  "$omitEmpty": {
                    "$ref": "request.providerOptions.tudou-image.response_format"
                  }
                },
                "quality": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$not": {
                          "$in": [
                            {
                              "$first": {
                                "$split": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.model"
                                      }
                                    }
                                  },
                                  "-"
                                ]
                              }
                            },
                            [
                              "seedream"
                            ]
                          ]
                        }
                      },
                      "then": {
                        "$switch": {
                          "cases": [
                            {
                              "when": {
                                "$in": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.quality"
                                      }
                                    }
                                  },
                                  [
                                    "low",
                                    "medium",
                                    "high",
                                    "xhigh",
                                    "max"
                                  ]
                                ]
                              },
                              "then": {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.quality"
                                  }
                                }
                              }
                            }
                          ],
                          "default": "medium"
                        }
                      }
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-image.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-image.extra_body"
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
                "$ref": "response.status"
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
                "$ref": "response.message"
              },
              {
                "$ref": "response.msg"
              }
            ]
          },
          "images": {
            "$map": {
              "from": {
                "$ref": "response.data.result.images"
              },
              "as": "item",
              "in": {
                "url": {
                  "$omitEmpty": {
                    "$coalesce": [
                      {
                        "$first": {
                          "$ref": "item.url"
                        }
                      },
                      {
                        "$ref": "item.url"
                      }
                    ]
                  }
                }
              }
            }
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
        "id": "tudou-image-sync",
        "label": "Tudou Image Sync",
        "capabilities": [
          "image"
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
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "图片模型 ID，由渠道模型列表配置，不在协议层写死。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "图片提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "images",
            "description": "参考图或编辑源图，最多 9 张；role=mask 的蒙版不参与发送，协议不支持蒙版编辑。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "n",
            "description": "输出数量，缺省 1。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "size",
            "description": "比例串（如 1:1）或精确像素串（如 1024x1024），语义按协议说明。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位 1k/2k/4k，缺省 1k；size 为像素串时必须省略。"
          },
          {
            "name": "quality",
            "type": "string",
            "required": false,
            "mapping": "resolution 档位（1k/2k/4k）；同步协议仅在 low/medium/high 时发送 quality",
            "description": "清晰度即分辨率档位 1k/2k/4k，缺省 1k。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的扩展字段（tudou-image-sync.response_format、tudou-image-sync.body）。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/images/generations",
          "pathTemplate": {
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
              "then": "/v1/images/edits",
              "else": "/v1/images/generations"
            }
          },
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
                "n": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$gt": [
                          {
                            "$ref": "request.imageCount"
                          },
                          0
                        ]
                      },
                      "then": {
                        "$ref": "request.imageCount"
                      },
                      "else": 1
                    }
                  }
                },
                "size": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$in": [
                          {
                            "$lower": {
                              "$trim": {
                                "$ref": "request.aspectRatio"
                              }
                            }
                          },
                          [
                            "",
                            "auto"
                          ]
                        ]
                      },
                      "then": null,
                      "else": {
                        "$lower": {
                          "$trim": {
                            "$ref": "request.aspectRatio"
                          }
                        }
                      }
                    }
                  }
                },
                "resolution": {
                  "$omitEmpty": {
                    "$switch": {
                      "cases": [
                        {
                          "when": {
                            "$gt": [
                              {
                                "$len": {
                                  "$split": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.aspectRatio"
                                        }
                                      }
                                    },
                                    "x"
                                  ]
                                }
                              },
                              1
                            ]
                          },
                          "then": null
                        },
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
                                "1k",
                                "2k",
                                "4k"
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
                        },
                        {
                          "when": {
                            "$in": [
                              {
                                "$last": {
                                  "$split": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.model"
                                        }
                                      }
                                    },
                                    "-"
                                  ]
                                }
                              },
                              [
                                "1k",
                                "2k",
                                "4k"
                              ]
                            ]
                          },
                          "then": {
                            "$last": {
                              "$split": [
                                {
                                  "$lower": {
                                    "$trim": {
                                      "$ref": "request.model"
                                    }
                                  }
                                },
                                "-"
                              ]
                            }
                          }
                        }
                      ],
                      "default": "1k"
                    }
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
                "quality": {
                  "$omitEmpty": {
                    "$if": {
                      "condition": {
                        "$not": {
                          "$in": [
                            {
                              "$first": {
                                "$split": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.model"
                                      }
                                    }
                                  },
                                  "-"
                                ]
                              }
                            },
                            [
                              "seedream"
                            ]
                          ]
                        }
                      },
                      "then": {
                        "$switch": {
                          "cases": [
                            {
                              "when": {
                                "$in": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.quality"
                                      }
                                    }
                                  },
                                  [
                                    "low",
                                    "medium",
                                    "high",
                                    "xhigh",
                                    "max"
                                  ]
                                ]
                              },
                              "then": {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.quality"
                                  }
                                }
                              }
                            }
                          ],
                          "default": "medium"
                        }
                      }
                    }
                  }
                },
                "response_format": {
                  "$omitEmpty": {
                    "$ref": "request.providerOptions.tudou-image-sync.response_format"
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-image-sync.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-image-sync.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "response": {
          "status": "succeeded",
          "images": {
            "$map": {
              "from": {
                "$ref": "response.data"
              },
              "as": "item",
              "in": {
                "url": {
                  "$omitEmpty": {
                    "$ref": "item.url"
                  }
                },
                "dataUrl": {
                  "$if": {
                    "condition": {
                      "$ref": "item.b64_json"
                    },
                    "then": {
                      "$concat": [
                        "data:image/png;base64,",
                        {
                          "$ref": "item.b64_json"
                        }
                      ]
                    },
                    "else": null
                  }
                }
              }
            }
          },
          "usage": {
            "$ref": "response.usage"
          },
          "errorPaths": [
            "error.code",
            "data.error.code"
          ],
          "messagePaths": [
            "error.message",
            "data.error.message",
            "message",
            "msg"
          ],
          "resultEphemeral": true
        }
      },
      {
        "id": "tudou-image-gemini",
        "label": "Tudou Image Gemini",
        "capabilities": [
          "image"
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
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "URL path {model}",
            "description": "图片模型 ID，进入 /v1beta/models/{model}:generateContent 路径。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "contents[].parts[].text",
            "description": "图片提示词，位于 parts 最后一条 text。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "contents[].parts[].inlineData",
            "description": "参考图，需 data URL 形态（base64 内联）；role=mask 不参与发送。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "不适用",
            "description": "Gemini 路线单次返回一张，多张由业务层多次请求。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "generationConfig.imageConfig.aspectRatio",
            "description": "画面比例，缺省 1:1。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "generationConfig.imageConfig.imageSize",
            "description": "分辨率档位 1k/2k/4k，缺省 1k，发送前转大写。"
          },
          {
            "name": "quality",
            "type": "string",
            "required": false,
            "mapping": "generationConfig.imageConfig.imageSize",
            "description": "清晰度即分辨率档位，与 resolution 等效。"
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
          "path": "/v1beta/models/{{model}}:generateContent",
          "contentType": "application/json",
          "body": {
            "$merge": [
              {
                "contents": [
                  {
                    "role": "user",
                    "parts": {
                      "$concatArrays": [
                        {
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
                              "inlineData": {
                                "mimeType": {
                                  "$coalesce": [
                                    {
                                      "$dataMime": {
                                        "$ref": "media.value"
                                      }
                                    },
                                    "image/png"
                                  ]
                                },
                                "data": {
                                  "$dataPayload": {
                                    "$ref": "media.value"
                                  }
                                }
                              }
                            }
                          }
                        },
                        [
                          {
                            "text": {
                              "$ref": "request.prompt"
                            }
                          }
                        ]
                      ]
                    }
                  }
                ],
                "generationConfig": {
                  "responseModalities": [
                    "TEXT",
                    "IMAGE"
                  ],
                  "imageConfig": {
                    "aspectRatio": {
                      "$if": {
                        "condition": {
                          "$in": [
                            {
                              "$lower": {
                                "$trim": {
                                  "$ref": "request.aspectRatio"
                                }
                              }
                            },
                            [
                              "",
                              "auto"
                            ]
                          ]
                        },
                        "then": "1:1",
                        "else": {
                          "$lower": {
                            "$trim": {
                              "$ref": "request.aspectRatio"
                            }
                          }
                        }
                      }
                    },
                    "imageSize": {
                      "$upper": {
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
                                    "1k",
                                    "2k",
                                    "4k"
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
                            },
                            {
                              "when": {
                                "$in": [
                                  {
                                    "$lower": {
                                      "$trim": {
                                        "$ref": "request.quality"
                                      }
                                    }
                                  },
                                  [
                                    "1k",
                                    "2k",
                                    "4k"
                                  ]
                                ]
                              },
                              "then": {
                                "$lower": {
                                  "$trim": {
                                    "$ref": "request.quality"
                                  }
                                }
                              }
                            },
                            {
                              "when": {
                                "$in": [
                                  {
                                    "$last": {
                                      "$split": [
                                        {
                                          "$lower": {
                                            "$trim": {
                                              "$ref": "request.model"
                                            }
                                          }
                                        },
                                        "-"
                                      ]
                                    }
                                  },
                                  [
                                    "1k",
                                    "2k",
                                    "4k"
                                  ]
                                ]
                              },
                              "then": {
                                "$last": {
                                  "$split": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.model"
                                        }
                                      }
                                    },
                                    "-"
                                  ]
                                }
                              }
                            }
                          ],
                          "default": "1k"
                        }
                      }
                    }
                  }
                }
              },
              {
                "$coalesce": [
                  {
                    "$ref": "request.providerOptions.tudou-image-gemini.body"
                  },
                  {
                    "$ref": "request.providerOptions.tudou-image-gemini.extra_body"
                  },
                  {}
                ]
              }
            ]
          }
        },
        "response": {
          "status": "succeeded",
          "images": {
            "$map": {
              "from": {
                "$ref": "response.candidates"
              },
              "as": "candidate",
              "in": {
                "$map": {
                  "from": {
                    "$filter": {
                      "from": {
                        "$ref": "candidate.content.parts"
                      },
                      "as": "part",
                      "where": {
                        "$gt": [
                          {
                            "$len": {
                              "$coalesce": [
                                {
                                  "$ref": "part.inlineData.data"
                                },
                                ""
                              ]
                            }
                          },
                          0
                        ]
                      }
                    }
                  },
                  "as": "part",
                  "in": {
                    "dataUrl": {
                      "$concat": [
                        "data:",
                        {
                          "$coalesce": [
                            {
                              "$ref": "part.inlineData.mimeType"
                            },
                            "image/png"
                          ]
                        },
                        ";base64,",
                        {
                          "$ref": "part.inlineData.data"
                        }
                      ]
                    }
                  }
                }
              }
            }
          },
          "errorPaths": [
            "error.code"
          ],
          "messagePaths": [
            "error.message",
            "error.status"
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
