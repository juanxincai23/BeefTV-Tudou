package protocol

import (
	"context"
	"testing"
)

// 土豆网关的尺寸两套互斥：比例串可叠加 resolution，像素串必须省略 resolution。
// 同步/异步由路径决定，协议层不按模型名切换流程；取图路径固定为
// data.result.images[].url[0]，url 本身是数组。
func TestTudouImageAsyncSubmitBody(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "灰色圆点", AspectRatio: "3:4", Resolution: "2K", Quality: "High", ImageCount: 2,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if create.Method != "POST" || create.Path != "/v1/images/generations/async" {
		t.Fatalf("create = %#v", create)
	}
	body := manifestTestBody(t, create)
	if body["model"] != "flare" || body["prompt"] != "灰色圆点" {
		t.Fatalf("body = %#v", body)
	}
	if body["n"] != float64(2) {
		t.Fatalf("n = %#v, 期望 2", body["n"])
	}
	if body["size"] != "3:4" {
		t.Fatalf("size = %#v, 期望比例串 3:4", body["size"])
	}
	// 分辨率档位大小写不敏感；异步路由不接受 quality 字段，档位只进 resolution。
	if body["resolution"] != "2k" {
		t.Fatalf("resolution = %#v, 期望 2k", body["resolution"])
	}
	// 官方文档 quality 必填：合法档位透传（大小写不敏感）。
	if body["quality"] != "high" {
		t.Fatalf("quality = %#v, 期望 high", body["quality"])
	}
	if _, exists := body["images"]; exists {
		t.Fatalf("文生图不应携带 images = %#v", body["images"])
	}
	if _, exists := body["image_urls"]; exists {
		t.Fatalf("文生图不应携带已弃用的 image_urls = %#v", body["image_urls"])
	}
}

func TestTudouImagePixelSizeOmitsResolution(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "雪夜", AspectRatio: "1024X1024", Resolution: "2k",
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["size"] != "1024x1024" {
		t.Fatalf("size = %#v, 期望像素串 1024x1024", body["size"])
	}
	// 异步路由恒发送合法档位：请求档位合法时即使像素串也保留（对齐补丁 tudou_async_resolution）。
	if body["resolution"] != "2k" {
		t.Fatalf("resolution = %#v, 期望 2k", body["resolution"])
	}

	sync := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-sync")
	syncCreate, err := sync.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "雪夜", AspectRatio: "1024x1024", Resolution: "2k",
	}})
	if err != nil {
		t.Fatal(err)
	}
	syncBody := manifestTestBody(t, syncCreate)
	// 同步 generations/edits 像素串与 resolution 互斥，同时发送会被网关 400。
	if _, exists := syncBody["resolution"]; exists {
		t.Fatalf("同步像素串必须省略 resolution = %#v", syncBody["resolution"])
	}
}

// 画布图片任务会携带视频清晰度（vquality，如 720p），协议层不得把非法档位透传给网关：
// 先认 1k/2k/4k，再认模型名后缀（gpt-image-2-2k），否则缺省 1k。
func TestTudouImageResolutionFallbacks(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点", Resolution: "720p",
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["resolution"] != "1k" {
		t.Fatalf("resolution = %#v, 非法档位 720p 必须兜底为 1k", body["resolution"])
	}
	if body["size"] != "1:1" {
		t.Fatalf("size = %#v, 缺省应兜底 1:1", body["size"])
	}

	suffixed, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-2k", Prompt: "灰色圆点", Resolution: "720p",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if manifestTestBody(t, suffixed)["resolution"] != "2k" {
		t.Fatalf("resolution = %#v, 期望模型后缀 -2k 生效", manifestTestBody(t, suffixed)["resolution"])
	}

	explicit, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点", Resolution: "4K",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if manifestTestBody(t, explicit)["resolution"] != "4k" {
		t.Fatalf("resolution = %#v, 期望大小写不敏感透传 4k", manifestTestBody(t, explicit)["resolution"])
	}

	// 质量与分辨率彻底独立：质量值（即使撞名档位）不影响分辨率推导。
	viaQuality, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点", Quality: "2k",
	}})
	if err != nil {
		t.Fatal(err)
	}
	viaQualityBody := manifestTestBody(t, viaQuality)
	if viaQualityBody["resolution"] != "1k" {
		t.Fatalf("resolution = %#v, 质量不应影响分辨率（缺省 1k）", viaQualityBody["resolution"])
	}

	// 同步路由：质量独立于分辨率，五档透传。
	syncAdapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-sync")
	syncQuality, err := syncAdapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点", Quality: "high",
	}})
	if err != nil {
		t.Fatal(err)
	}
	syncQualityBody := manifestTestBody(t, syncQuality)
	if syncQualityBody["quality"] != "high" {
		t.Fatalf("同步 quality = %#v, 期望保留 high", syncQualityBody["quality"])
	}
	if syncQualityBody["resolution"] != "1k" {
		t.Fatalf("同步 resolution = %#v, 无尺寸信号时应缺省 1k", syncQualityBody["resolution"])
	}

	// 同步协议质量同为五档：xhigh 透传且分辨率走缺省 1k（无别名）。
	xhigh, err := syncAdapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-sunburst", Prompt: "灰色圆点", Quality: "xhigh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	xhighBody := manifestTestBody(t, xhigh)
	if xhighBody["quality"] != "xhigh" || xhighBody["resolution"] != "1k" {
		t.Fatalf("同步 xhigh = %#v/%#v, 期望 quality 透传 + 分辨率缺省 1k", xhighBody["quality"], xhighBody["resolution"])
	}
}

func TestTudouImageAsyncSubmitDefaultsAndReferences(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "换背景",
		Images: []MediaReference{
			{URL: "https://cdn.example/anchor.png", Role: "edit_source", Order: 1},
			{DataURL: "data:image/png;base64,QUJD", Role: "mask", Order: 2},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["n"] != float64(1) {
		t.Fatalf("n = %#v, 缺省应为 1", body["n"])
	}
	// 异步路由恒发送 size：缺省兜底 1:1（对齐补丁 tudou_async_size）。
	if body["size"] != "1:1" {
		t.Fatalf("size = %#v, 缺省应兜底 1:1", body["size"])
	}
	// size 缺省时仍按 1k 发送 resolution（土豆缺省档位）。
	if body["resolution"] != "1k" {
		t.Fatalf("resolution = %#v, 缺省应为 1k", body["resolution"])
	}
	// 官方文档 quality 必填：缺省发送 medium。
	if body["quality"] != "medium" {
		t.Fatalf("quality = %#v, 缺省应为 medium", body["quality"])
	}
	// 蒙版不属于土豆参考图语义：role=mask 必须被剔除，避免把蒙版当编辑源图。
	// 网关只认 images 字段；旧 image_urls 会被静默丢弃（图生图退化为纯文生图）。
	urls, _ := body["images"].([]any)
	if len(urls) != 1 || urls[0] != "https://cdn.example/anchor.png" {
		t.Fatalf("images = %#v, 期望只含参考图且剔除蒙版", body["images"])
	}
	if _, exists := body["image_urls"]; exists {
		t.Fatalf("异步路由不应再携带已弃用的 image_urls = %#v", body["image_urls"])
	}
}

func TestTudouImagePollParsesNestedResult(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	created, err := adapter.ParseCreate(context.Background(), []byte(`{"code":200,"data":{"id":"task_1","status":"submitted","progress":0,"created":1726300000,"estimated_time":90}}`))
	if err != nil {
		t.Fatal(err)
	}
	if created.TaskID != "task_1" {
		t.Fatalf("taskId = %#v, 期望 task_1", created.TaskID)
	}
	if created.Status != StatusPending {
		t.Fatalf("提交态 status = %#v, 期望 pending", created.Status)
	}
	poll, err := adapter.BuildPoll(context.Background(), PollContext{TaskID: created.TaskID})
	if err != nil {
		t.Fatal(err)
	}
	if poll.Method != "GET" || poll.Path != "/v1/tasks/task_1" {
		t.Fatalf("poll = %#v", poll)
	}
	completed, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: created.TaskID}, []byte(`{"code":200,"data":{"id":"task_1","status":"completed","progress":100,"result":{"images":[{"url":["https://cdn.example/x.png"],"expires_at":1792022400,"size":"1920x1920"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != StatusSucceeded {
		t.Fatalf("completed 必须归一为成功，得到 %#v", completed.Status)
	}
	if completed.Result == nil || len(completed.Result.Images) != 1 || completed.Result.Images[0].URL != "https://cdn.example/x.png" {
		t.Fatalf("images = %#v, 期望取 url 数组第一项", completed.Result)
	}
	if !completed.Result.Images[0].Ephemeral {
		t.Fatal("土豆产物 URL 带过期时间，必须标记 ephemeral 由宿主立即下载")
	}
}

func TestTudouImagePollFailureCarriesDataErrorMessage(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	failed, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: "task_1"}, []byte(`{"code":200,"data":{"id":"task_1","status":"failed","error":{"code":"quota_exhausted","message":"配额不足","type":"insufficient_quota"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != StatusFailed {
		t.Fatalf("failed 必须归一为失败，得到 %#v", failed.Status)
	}
	if failed.Message != "配额不足" {
		t.Fatalf("message = %#v, 期望取 data.error.message", failed.Message)
	}
}

func TestTudouImageSyncSwitchesBetweenGenerationsAndEdits(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-sync")
	generate, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "灰色圆点",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if generate.Path != "/v1/images/generations" || generate.ContentType != "application/json" {
		t.Fatalf("文生图 create = %#v", generate)
	}
	created, err := adapter.ParseCreate(context.Background(), []byte(`{"created":1726300000,"model":"flare","data":[{"url":"https://cdn.example/a.png"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != StatusSucceeded || created.Result == nil || len(created.Result.Images) != 1 || created.Result.Images[0].URL != "https://cdn.example/a.png" {
		t.Fatalf("同步结果 = %#v", created)
	}

	edit, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "flare", Prompt: "换装",
		Images: []MediaReference{
			{URL: "https://cdn.example/anchor.png", Role: "edit_source", Order: 1},
			{DataURL: "data:image/png;base64,QUJD", Role: "mask", Order: 2},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if edit.Path != "/v1/images/edits" || edit.ContentType != "application/json" {
		t.Fatalf("图生图 create = %#v", edit)
	}
	// 网关实测拒绝 multipart（"multipart/form-data is not supported"），
	// 参考图必须走 JSON images 数组；字段名 image_urls 在该端点不被识别。
	if len(edit.Files) != 0 {
		t.Fatalf("edits 不得携带 multipart 文件部件，得到 %#v", edit.Files)
	}
	editBody := manifestTestBody(t, edit)
	images, _ := editBody["images"].([]any)
	if len(images) != 1 || images[0] != "https://cdn.example/anchor.png" {
		t.Fatalf("edits images = %#v, 期望只含参考图且剔除蒙版", editBody["images"])
	}
	if _, exists := editBody["image_urls"]; exists {
		t.Fatalf("edits 不应携带 image_urls = %#v", editBody["image_urls"])
	}
}

// 土豆网关的 Gemini 兼容路线：模型 ID 进路径，parts 前面是 inlineData 参考图、
// 最后一条 text 是 prompt；档位走 generationConfig.imageConfig（缺省 1:1 + 1K，发送前转大写），
// 响应只取 candidates[].content.parts[].inlineData。
func TestTudouImageGeminiTextToImageBody(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-gemini")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gemini-3-pro-image-preview", Prompt: "灰色圆点",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if create.Method != "POST" || create.Path != "/v1beta/models/gemini-3-pro-image-preview:generateContent" {
		t.Fatalf("create = %#v", create)
	}
	body := manifestTestBody(t, create)
	contents, _ := body["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("contents = %#v", body["contents"])
	}
	first, _ := contents[0].(map[string]any)
	parts, _ := first["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("文生图 parts = %#v, 期望只有一条 text", parts)
	}
	textPart, _ := parts[0].(map[string]any)
	if textPart["text"] != "灰色圆点" {
		t.Fatalf("text = %#v", textPart)
	}
	config, _ := body["generationConfig"].(map[string]any)
	if config == nil {
		t.Fatalf("generationConfig 缺失 = %#v", body)
	}
	modalities, _ := config["responseModalities"].([]any)
	if len(modalities) != 2 || modalities[0] != "TEXT" || modalities[1] != "IMAGE" {
		t.Fatalf("responseModalities = %#v", config["responseModalities"])
	}
	imageConfig, _ := config["imageConfig"].(map[string]any)
	if imageConfig["aspectRatio"] != "1:1" || imageConfig["imageSize"] != "1K" {
		t.Fatalf("imageConfig = %#v, 期望缺省 1:1 + 1K", imageConfig)
	}
}

func TestTudouImageGeminiInlineReferencesAndTiers(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-gemini")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gemini-3-pro-image-preview", Prompt: "把背景换成雪夜", Quality: "2k", AspectRatio: "9:16",
		Images: []MediaReference{
			{DataURL: "data:image/jpeg;base64,QUJD", Role: "edit_source", Order: 1},
			{URL: "https://cdn.example/mask.png", Role: "mask", Order: 2},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	contents, _ := body["contents"].([]any)
	first, _ := contents[0].(map[string]any)
	parts, _ := first["parts"].([]any)
	if len(parts) != 2 {
		t.Fatalf("图生图 parts = %#v, 期望 1 条 inlineData + 1 条 text（蒙版剔除）", parts)
	}
	inline, _ := parts[0].(map[string]any)["inlineData"].(map[string]any)
	if inline["mimeType"] != "image/jpeg" || inline["data"] != "QUJD" {
		t.Fatalf("inlineData = %#v", inline)
	}
	textPart, _ := parts[1].(map[string]any)
	if textPart["text"] != "把背景换成雪夜" {
		t.Fatalf("prompt 必须在最后一条 text = %#v", textPart)
	}
	config, _ := body["generationConfig"].(map[string]any)
	imageConfig, _ := config["imageConfig"].(map[string]any)
	if imageConfig["imageSize"] != "2K" || imageConfig["aspectRatio"] != "9:16" {
		t.Fatalf("imageConfig = %#v, 期望 quality 档位 2K + 9:16", imageConfig)
	}
}

func TestTudouImageGeminiResponseAndError(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image-gemini")
	created, err := adapter.ParseCreate(context.Background(), []byte(`{"candidates":[{"content":{"parts":[{"text":"已生成"},{"inlineData":{"mimeType":"image/png","data":"QUJD"}}]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != StatusSucceeded {
		t.Fatalf("status = %#v", created.Status)
	}
	if created.Result == nil || len(created.Result.Images) != 1 {
		t.Fatalf("images = %#v, 期望只取 inlineData 部件", created.Result)
	}
	if created.Result.Images[0].DataURL != "data:image/png;base64,QUJD" {
		t.Fatalf("dataUrl = %#v", created.Result.Images[0].DataURL)
	}

	failed, err := adapter.ParseCreate(context.Background(), []byte(`{"error":{"code":400,"message":"模型不支持该档位","status":"INVALID_ARGUMENT"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != StatusFailed || failed.Message != "模型不支持该档位" {
		t.Fatalf("失败态 = %#v/%#v", failed.Status, failed.Message)
	}
}

// 官方文档：异步 quality 为必填档位（2.0: low/medium/high，2.5 增加 xhigh/max），
// 缺省发 medium；UI 的 1k/2k/4k 清晰度档位走 resolution，不串到 quality。
func TestTudouImageAsyncQualityField(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-flare-async", Prompt: "星夜城堡", Quality: "xhigh", Resolution: "4k",
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["quality"] != "xhigh" || body["resolution"] != "4k" {
		t.Fatalf("quality/resolution = %#v/%#v, 期望 xhigh + 4k", body["quality"], body["resolution"])
	}

	// UI 清晰度承载档位（2k）时：档位进 resolution，quality 发默认 medium。
	tier, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点", Quality: "medium",
	}})
	if err != nil {
		t.Fatal(err)
	}
	tierBody := manifestTestBody(t, tier)
	// 质量与分辨率独立：medium 只进 quality，分辨率不受影响（缺省 1k）。
	if tierBody["quality"] != "medium" || tierBody["resolution"] != "1k" {
		t.Fatalf("quality/resolution = %#v/%#v, 期望 medium + 1k", tierBody["quality"], tierBody["resolution"])
	}

	// 无质量输入时同样缺省 medium（文档标注必填，不能省略）。
	bare, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2-all", Prompt: "灰色圆点",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if manifestTestBody(t, bare)["quality"] != "medium" {
		t.Fatalf("quality = %#v, 缺省应为 medium", manifestTestBody(t, bare)["quality"])
	}
}

// 分辨率与质量解耦：尺寸走官方像素对照表（按面积推导档位，阈值 4M/2M 对 45 个官方组合逐一校验），
// 质量独立五档，任意组合合法（如 4K + low、1K + max）。
func TestTudouImageAsyncPixelSizeDerivation(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	cases := []struct {
		size       string
		resolution string
	}{
		{"2880x2880", "4k"},   // 1:1 4K
		{"1536x864", "1k"},    // 16:9 1K
		{"2048x1152", "2k"},   // 16:9 2K
		{"3840x2160", "4k"},   // 16:9 4K
		{"1881x836", "1k"},    // 3:1 1K（宽幅按面积判定）
		{"864x2016", "1k"},    // 9:21 1K
		{"1024x3072", "2k"},   // 1:3 2K
		{"1280x3840", "4k"},   // 1:3 4K
		{"768x1024", "1k"},    // 3:4 1K
	}
	for _, tc := range cases {
		create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
			Model: "gpt-image-2.5-flare-async", Prompt: "星夜", AspectRatio: tc.size,
		}})
		if err != nil {
			t.Fatal(err)
		}
		body := manifestTestBody(t, create)
		if body["size"] != tc.size || body["resolution"] != tc.resolution {
			t.Fatalf("size %s: size/resolution = %#v/%#v, 期望 %s/%s", tc.size, body["size"], body["resolution"], tc.size, tc.resolution)
		}
	}

	// 解耦证明：4K 像素 + low 质量、1K 像素 + max 质量都能成立。
	low, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-flare-async", Prompt: "星夜", AspectRatio: "3840x2160", Quality: "low",
	}})
	if err != nil {
		t.Fatal(err)
	}
	lowBody := manifestTestBody(t, low)
	if lowBody["resolution"] != "4k" || lowBody["quality"] != "low" {
		t.Fatalf("4K+low = %#v/%#v", lowBody["resolution"], lowBody["quality"])
	}
	max, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-flare-async", Prompt: "星夜", AspectRatio: "1024x1024", Quality: "max",
	}})
	if err != nil {
		t.Fatal(err)
	}
	maxBody := manifestTestBody(t, max)
	if maxBody["resolution"] != "1k" || maxBody["quality"] != "max" {
		t.Fatalf("1K+max = %#v/%#v", maxBody["resolution"], maxBody["quality"])
	}
}

// seedream 系没有质量维度：模型名首段为 seedream 时整个 quality 字段省略，
// 避免 UI 残留的质量值（如切换模型前选过的档位）被发给不认识该字段的路线。
func TestTudouImageSeedreamOmitsQuality(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "seedream-5.0-pro", Prompt: "青绿山水", Quality: "xhigh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if _, exists := body["quality"]; exists {
		t.Fatalf("seedream 不应携带 quality = %#v", body["quality"])
	}
	if body["resolution"] != "1k" {
		t.Fatalf("resolution = %#v, seedream 质量值不应影响分辨率", body["resolution"])
	}

	// 非 seedream 模型不受守卫影响，quality 照常透传。
	other, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-flare-async", Prompt: "星夜", Quality: "xhigh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if manifestTestBody(t, other)["quality"] != "xhigh" {
		t.Fatalf("quality = %#v, 非 seedream 应保留五档透传", manifestTestBody(t, other)["quality"])
	}
}

// 复现宿主链路的 DataURL-only 参考图（无 URL、role=edit_source）在出站 body 中的形态。
func TestTudouImageDataUrlReferencesShape(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "gpt-image-2.5-flare-async", Prompt: "画一只大金毛",
		Images: []MediaReference{
			{DataURL: "data:image/png;base64,QUJD", Role: "edit_source", Order: 0},
			{DataURL: "data:image/png;base64,QUJD", Role: "edit_source", Order: 1},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	images, _ := body["images"].([]any)
	t.Logf("images = %#v", body["images"])
	if len(images) != 2 {
		t.Fatalf("images = %#v, 期望 2 张", body["images"])
	}
}

// 复现宿主 protocolRequestFromInput 产出的完整请求形状，定位 images 丢失的触发字段。
func TestTudouImageFromInputShapeBisect(t *testing.T) {
	adapter := officialPackageAdapter(t, "tudou-image.beeftv-plugin", "tudou-image")
	metadata := map[string]any{"bytes": 0, "durationMs": 0, "height": 0, "storageKey": "", "width": 0}

	build := func(name string, request GenerationRequest) {
		create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: request})
		if err != nil {
			t.Fatalf("%s: BuildCreate error = %v", name, err)
		}
		body := manifestTestBody(t, create)
		images, _ := body["images"].([]any)
		t.Logf("%-14s images = %d", name, len(images))
	}

	// 完整 fromInput 形状。
	full := GenerationRequest{
		Capability: "image", Model: "gpt-image-2.5-flare-async", Prompt: "画一只大金毛",
		Inputs: []MediaReference{
			{DataURL: "data:image/png;base64,QUJD", Kind: "image", Role: "edit_source", Metadata: metadata},
			{DataURL: "data:image/png;base64,QUJD", Kind: "image", Role: "edit_source", Order: 1, Metadata: metadata},
		},
		Images: []MediaReference{
			{DataURL: "data:image/png;base64,QUJD", Kind: "image", Role: "edit_source", Metadata: metadata},
			{DataURL: "data:image/png;base64,QUJD", Kind: "image", Role: "edit_source", Order: 1, Metadata: metadata},
		},
		Output: OutputOptions{},
		Extra:  map[string]any{"audioFormat": "mp3", "audioSpeed": "1", "audioVoice": "alloy", "count": "", "videoSeconds": ""},
	}
	build("full", full)

	noInputs := full
	noInputs.Inputs = nil
	build("no-inputs", noInputs)

	noExtra := full
	noExtra.Extra = nil
	build("no-extra", noExtra)
}
