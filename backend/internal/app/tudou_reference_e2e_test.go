package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/protocol"
)

// 端到端回归（生产加载路径）：土豆异步出站 body 的 images 必须保留 WebP 原样
// （网关实测 PNG/JPEG/WebP 都能读，转码会把体积吹大数倍、逼近单图体积上限），
// 且不得再携带已弃用的 image_urls（网关只认 images）。
func TestTudouAsyncMultiImageReferencesPassThroughEndToEnd(t *testing.T) {
	zipData, err := os.ReadFile("../../../plugin-packages/tudou-image.beeftv-plugin")
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := protocol.ParsePluginPackage(zipData)
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := protocol.LoadInstalledProviders(pkg.ManifestRaw, nil)
	if err != nil {
		t.Fatal(err)
	}
	var adapter protocol.Adapter
	for _, item := range adapters {
		if item.Metadata().ID == "tudou-image" {
			adapter = item
			break
		}
	}
	if adapter == nil {
		t.Fatal("tudou-image adapter not found")
	}

	webpReference := "data:image/webp;base64," + testWebPBase64
	input := canvasGenerationInput{
		Mode:   "image",
		Prompt: "画一只大金毛",
		Config: providerConfig{
			InterfaceType: "tudou-image",
			BaseURL:       "https://api.ai-tudou.net",
			APIKey:        "test-key",
			Model:         "gpt-image-2.5-flare-async",
		},
		ReferenceImages: []providerMedia{
			{DataURL: webpReference},
			{DataURL: webpReference},
		},
	}
	request := protocolRequestFromInput(input)
	spec, err := adapter.BuildCreate(context.Background(), protocol.RequestContext{BaseURL: input.Config.BaseURL, Request: request})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(spec.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	images, ok := body["images"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("images missing: ok=%v len=%d body=%#v", ok, len(images), body)
	}
	for index, item := range images {
		if item != webpReference {
			t.Fatalf("images[%d] 应保留 WebP 原样透传，得到 %.50v", index, item)
		}
	}
	if _, exists := body["image_urls"]; exists {
		t.Fatalf("异步路由不应携带已弃用的 image_urls = %#v", body["image_urls"])
	}
	if body["model"] != "gpt-image-2.5-flare-async" || body["prompt"] == "" {
		t.Fatalf("model/prompt = %#v/%#v", body["model"], body["prompt"])
	}
	if !strings.Contains(webpReference, "data:image/webp") {
		t.Fatal("测试夹具必须是 webp data URL")
	}
}
