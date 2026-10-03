package app

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math/rand"
	"strings"
	"testing"
)

// 官方 1x1 WebP（无损）样例。
const testWebPBase64 = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func makeTestDataURL(mime, base64Payload string) string {
	return "data:image/" + mime + ";base64," + base64Payload
}

func decodeDataURLPayload(t *testing.T, dataURL string) []byte {
	t.Helper()
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		t.Fatalf("data URL 缺少载荷：%.40s", dataURL)
	}
	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("data URL 载荷无法解码：%v", err)
	}
	return payload
}

func TestNormalizeTudouReferenceDataURLPassThrough(t *testing.T) {
	// 网关实测可读的 PNG/JPEG/WebP 在预算内必须原样透传：
	// 转码只会把体积吹大（2.1MB WebP → 9MB PNG），把图推向单图体积上限。
	webp := makeTestDataURL("webp", testWebPBase64)
	if got := normalizeTudouReferenceDataURL(webp); got != webp {
		t.Fatalf("webp 应原样透传，得到前缀 %.40s", got)
	}
	pngPayload := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	if got := normalizeTudouReferenceDataURL(makeTestDataURL("png", pngPayload)); got != makeTestDataURL("png", pngPayload) {
		t.Fatalf("png 应原样透传")
	}
	jpegPayload := "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFAABAQAAAAAAAAAAAAAAAAAAAAn/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFAEBAAAAAAAAAAAAAAAAAAAAAP/aAAwDAQACEQMRAD8AmAA//9k="
	if got := normalizeTudouReferenceDataURL(makeTestDataURL("jpeg", jpegPayload)); got != makeTestDataURL("jpeg", jpegPayload) {
		t.Fatalf("jpeg 应原样透传")
	}

	// 公网 URL 与非 data URL 不处理。
	if got := normalizeTudouReferenceDataURL("https://cdn.example/ref.webp"); got != "https://cdn.example/ref.webp" {
		t.Fatalf("公网 URL 应原样透传，得到 %.40s", got)
	}
}

func TestNormalizeTudouReferenceDataURLConvertsGif(t *testing.T) {
	// GIF 是网关未验证格式：统一重编码（本夹具不透明，按策略走 JPEG）。
	palette := color.Palette{color.RGBA{R: 200, G: 30, B: 30, A: 255}, color.RGBA{B: 200, A: 255}}
	src := image.NewPaletted(image.Rect(0, 0, 4, 4), palette)
	for i := range src.Pix {
		src.Pix[i] = uint8(i % 2)
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, src, nil); err != nil {
		t.Fatal(err)
	}
	converted := normalizeTudouReferenceDataURL(makeTestDataURL("gif", base64.StdEncoding.EncodeToString(buf.Bytes())))
	if !strings.HasPrefix(converted, "data:image/jpeg;base64,") && !strings.HasPrefix(converted, "data:image/png;base64,") {
		t.Fatalf("gif 必须重编码为 jpeg/png，得到前缀 %.40s", converted)
	}
	decoded, _, err := image.Decode(bytes.NewReader(decodeDataURLPayload(t, converted)))
	if err != nil {
		t.Fatalf("转换结果无法解码：%v", err)
	}
	if decoded.Bounds().Dx() != 4 || decoded.Bounds().Dy() != 4 {
		t.Fatalf("尺寸 = %dx%d, 期望 4x4", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func noisyImage(t *testing.T, width, height int, opaque bool) image.Image {
	t.Helper()
	rnd := rand.New(rand.NewSource(42))
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			alpha := uint8(255)
			if !opaque && x < width/4 {
				alpha = 128
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(rnd.Intn(256)), G: uint8(rnd.Intn(256)), B: uint8(rnd.Intn(256)), A: alpha})
		}
	}
	return img
}

func pngDataURL(t *testing.T, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return makeTestDataURL("png", base64.StdEncoding.EncodeToString(buf.Bytes()))
}

func TestNormalizeTudouReferenceDataURLShrinksOversizedImage(t *testing.T) {
	// 超预算的无透明通道图：降采样 + JPEG，体积必须落回预算内。
	budget := tudouReferenceByteBudget
	tudouReferenceByteBudget = 64 << 10
	defer func() { tudouReferenceByteBudget = budget }()

	source := pngDataURL(t, noisyImage(t, 512, 512, true))
	if len(decodeDataURLPayload(t, source)) <= tudouReferenceByteBudget {
		t.Fatal("测试源图必须超过预算")
	}
	normalized := normalizeTudouReferenceDataURL(source)
	if !strings.HasPrefix(normalized, "data:image/jpeg;base64,") {
		t.Fatalf("无透明通道超预算图应编码为 jpeg，得到前缀 %.40s", normalized)
	}
	payload := decodeDataURLPayload(t, normalized)
	if len(payload) > tudouReferenceByteBudget {
		t.Fatalf("归一化后体积 %d 仍超过预算 %d", len(payload), tudouReferenceByteBudget)
	}
	img, _, err := image.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("归一化结果无法解码：%v", err)
	}
	if img.Bounds().Dx() >= 512 {
		t.Fatalf("超预算图必须降采样，得到 %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestNormalizeTudouReferenceDataURLKeepsAlphaOnShrink(t *testing.T) {
	// 带透明通道的超预算图：保留 PNG（JPEG 会丢 alpha）。
	budget := tudouReferenceByteBudget
	tudouReferenceByteBudget = 64 << 10
	defer func() { tudouReferenceByteBudget = budget }()

	source := pngDataURL(t, noisyImage(t, 512, 512, false))
	normalized := normalizeTudouReferenceDataURL(source)
	if !strings.HasPrefix(normalized, "data:image/png;base64,") {
		t.Fatalf("带透明通道的超预算图应保持 png，得到前缀 %.40s", normalized)
	}
	payload := decodeDataURLPayload(t, normalized)
	if len(payload) > tudouReferenceByteBudget {
		t.Fatalf("归一化后体积 %d 仍超过预算 %d", len(payload), tudouReferenceByteBudget)
	}
}
