import { http } from "@/services/api/request";
import type { ModelProtocolDefinition, ProtocolCapability } from "@/lib/model-protocols";
import { workspaceCapabilities } from "@/services/workspace-mode";

type PluginProviderCatalogItem = {
    id: string;
    version: string;
    name: string;
    vendor: string;
    categories: string[];
    scopes: string[];
    create?: string;
    poll?: string;
    contentType?: string;
    enabled: boolean;
    unavailableReason?: string;
    baseUrl?: string;
    workflows?: Array<{
        id: string;
        label: string;
        providerId: string;
        capability: ProtocolCapability;
        parameters: Array<{ name: string; type: string; required?: boolean; description?: string; values?: string[]; mapping?: string }>;
        defaults?: Record<string, string | number | boolean>;
    }>;
};

export async function fetchPluginProviderCatalog(scope: string, capability?: ProtocolCapability) {
    if (workspaceCapabilities().local && scope === "user.custom-channel") {
        return BUILTIN_OPENAI_PROTOCOLS.filter((item) => !capability || item.capability === capability);
    }
    try {
        const result = await http.get<{ providers: PluginProviderCatalogItem[] }>("/plugins/catalog", { params: { scope, capability } });
        return result.providers.filter((item) => item.enabled && !item.unavailableReason).map(toProviderDefinition);
    } catch (error) {
        // The local desktop profile can run without the optional plugin center.
        // Keep the built-in OpenAI-compatible protocols available so a custom
        // channel remains usable even when protocol metadata is unavailable.
        if (scope === "user.custom-channel") {
            const fallback = BUILTIN_OPENAI_PROTOCOLS.filter((item) => !capability || item.capability === capability);
            if (fallback.length) return fallback;
        }
        throw error;
    }
}

const BUILTIN_OPENAI_PROTOCOLS: ModelProtocolDefinition[] = [
    { value: "chat-completion", label: "OpenAI Chat Completions", vendor: "OpenAI", capability: "text", create: "POST /v1/chat/completions", contentType: "application/json", media: "内置协议", enabled: true },
    { value: "openai-response", label: "OpenAI Responses", vendor: "OpenAI", capability: "text", create: "POST /v1/responses", contentType: "application/json", media: "内置协议", enabled: true },
    { value: "openai-image", label: "OpenAI Images", vendor: "OpenAI", capability: "image", create: "POST /v1/images/generations", contentType: "application/json", media: "内置协议", enabled: true },
    // 土豆协议由官方插件包提供执行，本地目录仅补登记，保证自定义渠道在无插件中心的部署里也能选中。
    { value: "tudou-image", label: "Tudou Image", vendor: "Tudou", capability: "image", create: "POST /v1/images/generations/async", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-image-sync", label: "Tudou Image Sync", vendor: "Tudou", capability: "image", create: "POST /v1/images/generations", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-image-gemini", label: "Tudou Image Gemini", vendor: "Tudou", capability: "image", create: "POST /v1beta/models/{model}:generateContent", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-sora2", label: "Tudou Video Sora2", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-veo3", label: "Tudou Video Veo3", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-kling", label: "Tudou Video Kling", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-pixverse", label: "Tudou Video PixVerse", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-seedance", label: "Tudou Video Seedance", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-grok", label: "Tudou Video Grok", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "tudou-video-wan", label: "Tudou Video Wan", vendor: "Tudou", capability: "video", create: "POST /v1/videos/generations", poll: "GET /v1/tasks/{task_id}", contentType: "application/json", media: "官方插件", enabled: true },
    { value: "newapi", label: "OpenAI Videos", vendor: "OpenAI compatible", capability: "video", create: "POST /v1/videos", poll: "GET /v1/videos/{task_id}", contentType: "multipart/form-data", media: "内置协议", enabled: true },
];

function toProviderDefinition(item: PluginProviderCatalogItem): ModelProtocolDefinition {
    return {
        value: item.id,
        label: item.name,
        vendor: item.vendor,
        capability: (item.categories[0] || "text") as ProtocolCapability,
        create: item.create || "",
        poll: item.poll,
        contentType: item.contentType || "application/json",
        media: `${item.vendor} · ${item.version}`,
        enabled: item.enabled && !item.unavailableReason,
        baseUrl: item.baseUrl,
        workflows: item.workflows || [],
    };
}
