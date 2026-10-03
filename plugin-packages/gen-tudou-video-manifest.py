# -*- coding: utf-8 -*-
"""Generate plugin-packages/tudou-video/manifest.json from the patch-derived protocol."""
import json
import os

OUT = os.path.join(os.path.dirname(__file__), "tudou-video", "manifest.json")


def ref(path):
    return {"$ref": path}


def omit_empty(value):
    return {"$omitEmpty": value}


def lower_trim(path):
    return {"$lower": {"$trim": ref(path)}}


def non_mask_images():
    return {
        "$filter": {
            "from": ref("request.images"),
            "as": "media",
            "where": {"$ne": [ref("media.role"), "mask"]},
        }
    }


def media_values(source):
    return {"$map": {"from": source, "as": "media", "in": ref("media.value")}}


def resolution_default_720p():
    return {
        "$omitEmpty": {
            "$switch": {
                "cases": [
                    {"when": {"$in": [lower_trim("request.resolution"), ["", "auto"]]}, "then": "720p"}
                ],
                "default": lower_trim("request.resolution"),
            }
        }
    }


def duration_passthrough():
    return omit_empty(ref("request.duration"))


def aspect_ratio_passthrough():
    return omit_empty(ref("request.aspectRatio"))


def generate_audio_always():
    return ref("request.generateAudio")


def generate_audio_only_when_true():
    return omit_empty({"$if": {"condition": ref("request.generateAudio"), "then": True, "else": None}})


def images_values():
    return omit_empty(media_values(non_mask_images()))


def videos_values():
    return omit_empty(media_values(ref("request.videos")))


def role_filter(role):
    return {
        "$filter": {
            "from": ref("request.images"),
            "as": "media",
            "where": {"$eq": [lower_trim("media.role"), role]},
        }
    }


def first_frame_value(role):
    return omit_empty(
        {"$first": media_values(role_filter(role))}
    )


def video_response_mapping():
    return {
        "taskId": {
            "$coalesce": [
                ref("response.data.id"),
                ref("response.data.task_id"),
                ref("response.id"),
                ref("response.task_id"),
                ref("taskId"),
            ]
        },
        "status": {
            "$coalesce": [
                ref("response.data.status"),
                ref("response.data.task_status"),
                ref("response.status"),
                ref("response.task_status"),
                "pending",
            ]
        },
        "message": {
            "$coalesce": [
                ref("response.data.error.message"),
                ref("response.error.message"),
                ref("response.data.message"),
                ref("response.data.fail_reason"),
                ref("response.message"),
                ref("response.fail_reason"),
            ]
        },
        "videos": {
            "$coalesce": [
                ref("response.data.result.video_url"),
                ref("response.data.result.url"),
                {
                    "$map": {
                        "from": ref("response.data.result.videos"),
                        "as": "item",
                        "in": {
                            "$coalesce": [
                                {"$first": ref("item.url")},
                                ref("item.url"),
                                ref("item.video_url"),
                                ref("item"),
                            ]
                        },
                    }
                },
                {
                    "$map": {
                        "from": ref("response.data.result.outputs"),
                        "as": "item",
                        "in": {
                            "$coalesce": [
                                {"$first": ref("item.url")},
                                ref("item.url"),
                                ref("item.video_url"),
                                ref("item"),
                            ]
                        },
                    }
                },
                ref("response.data.metadata.url"),
                ref("response.data.metadata.video_url"),
            ]
        },
        "errorPaths": ["error.code", "data.error.code"],
        "messagePaths": ["data.error.message", "error.message", "message", "msg"],
        "resultEphemeral": True,
    }


def base_params(extra):
    params = [
        {"name": "model", "type": "string", "required": True, "mapping": "model", "description": "视频模型 ID，由渠道模型列表配置，协议层不写死对照表。"},
        {"name": "prompt", "type": "string", "required": True, "mapping": "prompt", "description": "视频提示词。"},
    ]
    return params + extra + [
        {"name": "providerOptions", "type": "object", "required": False, "mapping": "provider-specific fields", "description": "插件命名空间内的扩展字段（body/extra_body 整包合并）。"},
    ]


IMAGES_PARAM = {"name": "images", "type": "media[]", "required": False, "mapping": "images", "description": "参考图（图生视频/首尾帧），role=first_frame/last_frame 参与排序或单独字段，role=mask 不参与发送。"}
VIDEOS_PARAM = {"name": "videos", "type": "media[]", "required": False, "mapping": "videos", "description": "参考视频，公网 URL 或 data URL。"}
AUDIOS_PARAM = {"name": "audios", "type": "media[]", "required": False, "mapping": "audios", "description": "参考音频，仅在已有图像或视频参考时随请求发送。"}
DURATION_PARAM = {"name": "duration", "type": "integer", "required": False, "mapping": "duration", "description": "时长（秒），合法档位由模型能力配置约束，缺省按网关规则。"}
ASPECT_PARAM = {"name": "aspectRatio", "type": "string", "required": False, "mapping": "aspect_ratio/size", "description": "画面比例，如 16:9。"}
RESOLUTION_PARAM = {"name": "resolution", "type": "string", "required": False, "mapping": "resolution", "description": "分辨率档位，缺省 720p。"}
AUDIO_TOGGLE_PARAM = {"name": "generateAudio", "type": "boolean", "required": False, "mapping": "generate_audio", "description": "是否生成音频。"}

PROVIDER_COMMON = {
    "capabilities": ["video"],
    "scopes": ["admin.system-channel", "user.custom-channel", "canvas", "creation", "agent"],
    "requiresPublicMediaUrls": False,
    "auth": {"type": "bearer", "field": "apiKey"},
}

CREATE_COMMON = {
    "method": "POST",
    "path": "/v1/videos/generations",
    "contentType": "application/json",
}


def merge_body(core, provider_id):
    return {
        "$merge": [
            core,
            {
                "$coalesce": [
                    ref(f"request.providerOptions.{provider_id}.body"),
                    ref(f"request.providerOptions.{provider_id}.extra_body"),
                    {},
                ]
            },
        ]
    }


def provider(pid, label, params, core):
    p = dict(PROVIDER_COMMON)
    p.update({
        "id": pid,
        "label": label,
        "parameters": params,
        "create": dict(CREATE_COMMON, body=merge_body(core, pid)),
        "poll": {"method": "GET", "path": "/v1/tasks/{{taskId}}"},
        "response": video_response_mapping(),
    })
    return p


providers = [
    # sora2：duration ∈ {4,8,12} 缺省 8；generate_audio 恒发送；参考图仅 1 张。
    provider(
        "tudou-video-sora2",
        "Tudou Video Sora2",
        base_params([IMAGES_PARAM, DURATION_PARAM, ASPECT_PARAM, AUDIO_TOGGLE_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "duration": duration_passthrough(),
            "aspect_ratio": aspect_ratio_passthrough(),
            "generate_audio": generate_audio_always(),
            "images": images_values(),
        },
    ),
    # veo3.1：resolution 缺省 720p；duration ∈ {4,6,8} 缺省 8；参考图 ≤3，>2 张用 image 模式否则 frame。
    provider(
        "tudou-video-veo3",
        "Tudou Video Veo3",
        base_params([IMAGES_PARAM, DURATION_PARAM, ASPECT_PARAM, RESOLUTION_PARAM, AUDIO_TOGGLE_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "resolution": resolution_default_720p(),
            "duration": duration_passthrough(),
            "aspect_ratio": aspect_ratio_passthrough(),
            "generate_audio": generate_audio_always(),
            "images": images_values(),
            "reference_mode": omit_empty(
                {
                    "$if": {
                        "condition": {"$gt": [{"$len": ref("request.images")}, 0]},
                        "then": {
                            "$if": {
                                "condition": {"$gt": [{"$len": ref("request.images")}, 2]},
                                "then": "image",
                                "else": "frame",
                            }
                        },
                        "else": None,
                    }
                }
            ),
        },
    ),
    # kling-v3 系：仅图生视频（至少 1 张参考图，能力配置强制）；duration kling-v3 ∈ {5,10,15} 缺省 10，omni ∈ {5,15} 缺省 15。
    provider(
        "tudou-video-kling",
        "Tudou Video Kling",
        base_params([IMAGES_PARAM, DURATION_PARAM, ASPECT_PARAM, AUDIO_TOGGLE_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "duration": duration_passthrough(),
            "aspect_ratio": aspect_ratio_passthrough(),
            "generate_audio": generate_audio_always(),
            "images": images_values(),
        },
    ),
    # pixverse-v6 系：resolution 缺省 720p；首尾帧进独立字段，其余进 img_references；duration 1..15 缺省 5；audio 由模型名（-audio 后缀）决定。
    provider(
        "tudou-video-pixverse",
        "Tudou Video PixVerse",
        base_params([IMAGES_PARAM, DURATION_PARAM, ASPECT_PARAM, RESOLUTION_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "resolution": resolution_default_720p(),
            "size": aspect_ratio_passthrough(),
            "duration": duration_passthrough(),
            "first_frame_image": first_frame_value("first_frame"),
            "last_frame_image": first_frame_value("last_frame"),
            "img_references": omit_empty(
                media_values(
                    {
                        "$filter": {
                            "from": ref("request.images"),
                            "as": "media",
                            "where": {
                                "$and": [
                                    {"$ne": [lower_trim("media.role"), "first_frame"]},
                                    {"$ne": [lower_trim("media.role"), "last_frame"]},
                                    {"$ne": [lower_trim("media.role"), "mask"]},
                                ]
                            },
                        }
                    }
                )
            ),
        },
    ),
    # seedance / minimax-h3 系：图像按 首帧→其余→末帧 排序；音画参考仅在带图像或视频时发送；
    # duration 2.5 系 4..30 缺省 10，其余 4..15 缺省 4；face 系固定 720p（能力配置约束）。
    provider(
        "tudou-video-seedance",
        "Tudou Video Seedance",
        base_params([IMAGES_PARAM, VIDEOS_PARAM, AUDIOS_PARAM, DURATION_PARAM, ASPECT_PARAM, RESOLUTION_PARAM, AUDIO_TOGGLE_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "duration": duration_passthrough(),
            "aspect_ratio": aspect_ratio_passthrough(),
            "resolution": resolution_default_720p(),
            "generate_audio": generate_audio_only_when_true(),
            "images": omit_empty(
                media_values(
                    {
                        "$concatArrays": [
                            role_filter("first_frame"),
                            {
                                "$filter": {
                                    "from": ref("request.images"),
                                    "as": "media",
                                    "where": {
                                        "$and": [
                                            {"$ne": [lower_trim("media.role"), "first_frame"]},
                                            {"$ne": [lower_trim("media.role"), "last_frame"]},
                                            {"$ne": [lower_trim("media.role"), "mask"]},
                                        ]
                                    },
                                }
                            },
                            role_filter("last_frame"),
                        ]
                    }
                )
            ),
            "videos": videos_values(),
            "audios": omit_empty(
                {
                    "$if": {
                        "condition": {
                            "$or": [
                                {"$gt": [{"$len": ref("request.images")}, 0]},
                                {"$gt": [{"$len": ref("request.videos")}, 0]},
                            ]
                        },
                        "then": media_values(ref("request.audios")),
                        "else": None,
                    }
                }
            ),
        },
    ),
    # grok-imagine-video 系：duration ∈ {6,10,15} 缺省 6；resolution 恒 720p；参考图进 extra.reference_images 并带角色。
    provider(
        "tudou-video-grok",
        "Tudou Video Grok",
        base_params([IMAGES_PARAM, DURATION_PARAM, ASPECT_PARAM]),
        {
            "model": ref("request.model"),
            "prompt": ref("request.prompt"),
            "duration": duration_passthrough(),
            "aspect_ratio": aspect_ratio_passthrough(),
            "resolution": "720p",
            "extra": {
                "aspect_ratio": aspect_ratio_passthrough(),
                "resolution": "720p",
                "reference_images": omit_empty(
                    {
                        "$map": {
                            "from": non_mask_images(),
                            "as": "media",
                            "in": {
                                "url": ref("media.value"),
                                "role": {
                                    "$switch": {
                                        "cases": [
                                            {
                                                "when": {
                                                    "$in": [
                                                        lower_trim("media.role"),
                                                        ["first_frame", "last_frame"],
                                                    ]
                                                },
                                                "then": lower_trim("media.role"),
                                            }
                                        ],
                                        "default": "reference_image",
                                    }
                                },
                            },
                        }
                    }
                ),
            },
        },
    ),
]

manifest = {
    "apiVersion": "beeftv.plugin/v2",
    "id": "tudou-video",
    "name": "Tudou Video",
    "version": "1.0.0",
    "author": "BeefTV Contributors",
    "description": "土豆（ai-tudou）视频协议插件：统一 /v1/videos/generations 提交与 /v1/tasks 轮询，按模型家族拆分请求形状。",
    "permissions": ["generation.run", "media.read"],
    "configuration": {
        "fields": [
            {"name": "apiKey", "type": "secret", "label": "API Key", "required": True}
        ]
    },
    "contributes": {"providers": providers},
}

os.makedirs(os.path.dirname(OUT), exist_ok=True)
with open(OUT, "w", encoding="utf-8") as f:
    json.dump(manifest, f, ensure_ascii=False, indent=2)
    f.write("\n")
print("written:", OUT, "providers:", [p["id"] for p in providers])
