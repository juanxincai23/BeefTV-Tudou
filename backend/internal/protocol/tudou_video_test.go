package protocol

import (
	"context"
	"testing"
)

// 土豆视频统一走 POST /v1/videos/generations 提交 + GET /v1/tasks/{id} 轮询，
// 六个家族只差 body 形状。状态大小写不敏感（补丁上游返回大写 COMPLETED 等），
// 取视频按 data.result 内 video_url/url/videos/outputs 顺序提取。
func tudouVideoAdapter(t *testing.T, providerID string) Adapter {
	t.Helper()
	return officialPackageAdapter(t, "tudou-video.beeftv-plugin", providerID)
}

func TestTudouVideoSora2SubmitBody(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-sora2")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "sora2", Prompt: "城市夜景", Duration: 8, AspectRatio: "16:9",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if create.Method != "POST" || create.Path != "/v1/videos/generations" {
		t.Fatalf("create = %#v", create)
	}
	body := manifestTestBody(t, create)
	if body["model"] != "sora2" || body["prompt"] != "城市夜景" {
		t.Fatalf("body = %#v", body)
	}
	// sora2 的 generate_audio 是恒发送的布尔：未开启也必须显式发 false。
	if body["generate_audio"] != false {
		t.Fatalf("generate_audio = %#v, 期望显式 false", body["generate_audio"])
	}
	if body["duration"] != float64(8) || body["aspect_ratio"] != "16:9" {
		t.Fatalf("duration/aspect = %#v/%#v", body["duration"], body["aspect_ratio"])
	}
	if _, exists := body["images"]; exists {
		t.Fatalf("无参考图时不应携带 images = %#v", body["images"])
	}
}

func TestTudouVideoVeo3ReferenceMode(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-veo3")
	base := GenerationRequest{Model: "veo3.1", Prompt: "雪夜城市", Resolution: "1080p"}

	none, err := adapter.BuildCreate(context.Background(), RequestContext{Request: base})
	if err != nil {
		t.Fatal(err)
	}
	noneBody := manifestTestBody(t, none)
	if noneBody["resolution"] != "1080p" {
		t.Fatalf("resolution = %#v, 期望透传 1080p", noneBody["resolution"])
	}
	if _, exists := noneBody["reference_mode"]; exists {
		t.Fatalf("无参考图时不应携带 reference_mode = %#v", noneBody["reference_mode"])
	}

	withImages := base
	withImages.Images = []MediaReference{
		{URL: "https://cdn.example/a.png", Role: "reference_image", Order: 1},
		{URL: "https://cdn.example/b.png", Role: "reference_image", Order: 2},
		{URL: "https://cdn.example/c.png", Role: "reference_image", Order: 3},
	}
	three, err := adapter.BuildCreate(context.Background(), RequestContext{Request: withImages})
	if err != nil {
		t.Fatal(err)
	}
	threeBody := manifestTestBody(t, three)
	if threeBody["reference_mode"] != "image" {
		t.Fatalf("reference_mode = %#v, 期望超过 2 张时为 image", threeBody["reference_mode"])
	}

	withImages.Images = withImages.Images[:1]
	one, err := adapter.BuildCreate(context.Background(), RequestContext{Request: withImages})
	if err != nil {
		t.Fatal(err)
	}
	oneBody := manifestTestBody(t, one)
	if oneBody["reference_mode"] != "frame" {
		t.Fatalf("reference_mode = %#v, 期望不超过 2 张时为 frame", oneBody["reference_mode"])
	}
}

func TestTudouVideoPixverseFrameFields(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-pixverse")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "pixverse-v6", Prompt: "换景", Duration: 5, AspectRatio: "9:16",
		Images: []MediaReference{
			{URL: "https://cdn.example/first.png", Role: "first_frame", Order: 1},
			{URL: "https://cdn.example/mid.png", Role: "reference_image", Order: 2},
			{DataURL: "data:image/png;base64,QUJD", Role: "mask", Order: 3},
			{URL: "https://cdn.example/last.png", Role: "last_frame", Order: 4},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["first_frame_image"] != "https://cdn.example/first.png" {
		t.Fatalf("first_frame_image = %#v", body["first_frame_image"])
	}
	if body["last_frame_image"] != "https://cdn.example/last.png" {
		t.Fatalf("last_frame_image = %#v", body["last_frame_image"])
	}
	refs, _ := body["img_references"].([]any)
	if len(refs) != 1 || refs[0] != "https://cdn.example/mid.png" {
		t.Fatalf("img_references = %#v, 期望只含普通参考图且剔除首尾帧与蒙版", body["img_references"])
	}
	if _, exists := body["generate_audio"]; exists {
		t.Fatalf("pixverse 的 audio 由模型名决定，不应发送 generate_audio = %#v", body["generate_audio"])
	}
}

func TestTudouVideoSeedanceOrderingAndGatedAudios(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-seedance")
	images := []MediaReference{
		{URL: "https://cdn.example/mid.png", Role: "reference_image", Order: 1},
		{URL: "https://cdn.example/last.png", Role: "last_frame", Order: 2},
		{URL: "https://cdn.example/first.png", Role: "first_frame", Order: 3},
	}
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "seedance-2.5", Prompt: "运镜", Duration: 10, GenerateAudio: true,
		Images: images, Audios: []MediaReference{{URL: "https://cdn.example/voice.mp3"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	ordered, _ := body["images"].([]any)
	if len(ordered) != 3 || ordered[0] != "https://cdn.example/first.png" || ordered[2] != "https://cdn.example/last.png" {
		t.Fatalf("images = %#v, 期望按 首帧→其余→末帧 排序", body["images"])
	}
	audios, _ := body["audios"].([]any)
	if len(audios) != 1 || audios[0] != "https://cdn.example/voice.mp3" {
		t.Fatalf("audios = %#v, 带图像参考时应发送", body["audios"])
	}
	if body["generate_audio"] != true {
		t.Fatalf("generate_audio = %#v, 期望 true", body["generate_audio"])
	}

	// 无图像/视频参考时 audios 不发送；generate_audio 为 false 时省略。
	bare, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "seedance-2.0", Prompt: "文生视频",
	}})
	if err != nil {
		t.Fatal(err)
	}
	bareBody := manifestTestBody(t, bare)
	if _, exists := bareBody["audios"]; exists {
		t.Fatalf("无参考时不应携带 audios = %#v", bareBody["audios"])
	}
	if _, exists := bareBody["generate_audio"]; exists {
		t.Fatalf("seedance 仅在开启时发送 generate_audio = %#v", bareBody["generate_audio"])
	}
	if bareBody["resolution"] != "720p" {
		t.Fatalf("resolution = %#v, 缺省应为 720p", bareBody["resolution"])
	}
}

func TestTudouVideoGrokExtraReferences(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-grok")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "grok-imagine-video-1.5", Prompt: "镜头推进", Duration: 6,
		Images: []MediaReference{
			{URL: "https://cdn.example/first.png", Role: "first_frame", Order: 1},
			{URL: "https://cdn.example/ref.png", Role: "edit_source", Order: 2},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["resolution"] != "720p" {
		t.Fatalf("resolution = %#v, grok 恒为 720p", body["resolution"])
	}
	extra, _ := body["extra"].(map[string]any)
	if extra == nil {
		t.Fatalf("extra = %#v, grok 必须携带 extra 对象", body["extra"])
	}
	refs, _ := extra["reference_images"].([]any)
	if len(refs) != 2 {
		t.Fatalf("reference_images = %#v, 期望 2 条", extra["reference_images"])
	}
	first, _ := refs[0].(map[string]any)
	if first["url"] != "https://cdn.example/first.png" || first["role"] != "first_frame" {
		t.Fatalf("首条参考 = %#v, 期望保留 first_frame 角色", first)
	}
	second, _ := refs[1].(map[string]any)
	if second["role"] != "reference_image" {
		t.Fatalf("普通参考角色 = %#v, 期望归一为 reference_image", second["role"])
	}
}

func TestTudouVideoPollLifecycle(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-seedance")
	created, err := adapter.ParseCreate(context.Background(), []byte(`{"code":200,"data":{"id":"task_v1","status":"SUBMITTED"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if created.TaskID != "task_v1" {
		t.Fatalf("taskId = %#v", created.TaskID)
	}
	if created.Status != StatusPending {
		t.Fatalf("提交态 = %#v, 期望 pending", created.Status)
	}
	poll, err := adapter.BuildPoll(context.Background(), PollContext{TaskID: created.TaskID})
	if err != nil {
		t.Fatal(err)
	}
	if poll.Method != "GET" || poll.Path != "/v1/tasks/task_v1" {
		t.Fatalf("poll = %#v", poll)
	}

	completed, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: created.TaskID}, []byte(`{"code":200,"data":{"id":"task_v1","status":"COMPLETED","result":{"video_url":"https://cdn.example/clip.mp4"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != StatusSucceeded {
		t.Fatalf("大写 COMPLETED 必须归一为成功，得到 %#v", completed.Status)
	}
	if completed.Result == nil || len(completed.Result.Videos) != 1 || completed.Result.Videos[0].URL != "https://cdn.example/clip.mp4" {
		t.Fatalf("videos = %#v", completed.Result)
	}

	nested, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: created.TaskID}, []byte(`{"code":200,"data":{"id":"task_v1","status":"completed","result":{"videos":[{"url":["https://cdn.example/nested.mp4"]}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if nested.Result == nil || len(nested.Result.Videos) != 1 || nested.Result.Videos[0].URL != "https://cdn.example/nested.mp4" {
		t.Fatalf("嵌套 videos = %#v, 期望取 url 数组第一项", nested.Result)
	}

	failed, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: created.TaskID}, []byte(`{"code":200,"data":{"id":"task_v1","status":"FAILED","error":{"code":"quota","message":"配额不足"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != StatusFailed || failed.Message != "配额不足" {
		t.Fatalf("失败态 = %#v/%#v", failed.Status, failed.Message)
	}
}

// 万相 All-in-One：duration 必填（缺省兜底 5），比例只允许 auto/16:9/9:16/1:1（兜底 16:9），
// resolution 仅 720p/1080p 缺省 720p，extra 镜像比例与分辨率，首尾帧进独立字段。
func TestTudouVideoWanBody(t *testing.T) {
	adapter := tudouVideoAdapter(t, "tudou-video-wan")
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "wan3.0-video", Prompt: "海边日落",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if create.Path != "/v1/videos/generations" {
		t.Fatalf("create = %#v", create)
	}
	body := manifestTestBody(t, create)
	if body["duration"] != float64(5) || body["aspect_ratio"] != "16:9" || body["resolution"] != "720p" {
		t.Fatalf("duration/aspect/resolution = %#v/%#v/%#v", body["duration"], body["aspect_ratio"], body["resolution"])
	}
	extra, _ := body["extra"].(map[string]any)
	if extra == nil || extra["aspect_ratio"] != "16:9" || extra["resolution"] != "720p" {
		t.Fatalf("extra = %#v, 期望镜像 aspect_ratio 与 resolution", body["extra"])
	}
	if _, exists := body["images"]; exists {
		t.Fatalf("文生视频不应携带 images = %#v", body["images"])
	}

	withRefs, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "wan3.0-video", Prompt: "让图1动起来", Duration: 8, AspectRatio: "auto", Resolution: "1080p",
		Images: []MediaReference{
			{URL: "https://cdn.example/first.png", Role: "first_frame", Order: 1},
			{URL: "https://cdn.example/ref.png", Role: "reference_image", Order: 2},
			{URL: "https://cdn.example/last.png", Role: "last_frame", Order: 3},
			{DataURL: "data:image/png;base64,QUJD", Role: "mask", Order: 4},
		},
		Videos: []MediaReference{{URL: "https://cdn.example/clip.mp4"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	refBody := manifestTestBody(t, withRefs)
	if refBody["duration"] != float64(8) || refBody["aspect_ratio"] != "auto" || refBody["resolution"] != "1080p" {
		t.Fatalf("透传 = %#v/%#v/%#v", refBody["duration"], refBody["aspect_ratio"], refBody["resolution"])
	}
	if refBody["first_frame_image"] != "https://cdn.example/first.png" || refBody["last_frame_image"] != "https://cdn.example/last.png" {
		t.Fatalf("首尾帧 = %#v/%#v", refBody["first_frame_image"], refBody["last_frame_image"])
	}
	images, _ := refBody["images"].([]any)
	if len(images) != 1 || images[0] != "https://cdn.example/ref.png" {
		t.Fatalf("images = %#v, 期望普通参考图且剔除首尾帧与蒙版", refBody["images"])
	}
	videos, _ := refBody["videos"].([]any)
	if len(videos) != 1 || videos[0] != "https://cdn.example/clip.mp4" {
		t.Fatalf("videos = %#v", refBody["videos"])
	}
}
