package app

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// tudou 系图片路线（异步 / 同步 / Gemini）的参考图出站归一化：
//   - PNG / JPEG / WebP 在体积预算内原样透传——网关实测三种格式都能读，
//     统一转 PNG 只会把体积吹大 3~4 倍（2.1MB WebP → 9MB PNG），把图推向单图体积上限；
//   - 超过预算的内联图降采样重编码：无透明通道用 JPEG、有透明通道用 PNG，直到落回预算内；
//   - GIF 等网关未验证格式统一重编码（无透明通道用 JPEG、有透明通道用 PNG）。
//
// 网关对单张内联参考图存在体积上限且超限时静默丢弃（实测 9.1MB 通过、15.8MB 被忽略，
// 表现为模型按纯文生图生成、无视用户上传的图），所以出站前必须压回安全体积。
func isTudouImageInterface(interfaceType string) bool {
	switch strings.TrimSpace(interfaceType) {
	case "tudou-image", "tudou-image-sync", "tudou-image-gemini":
		return true
	}
	return false
}

// 单张内联参考图的出站体积预算。网关真实上限在 9.1MB（通过）与 15.8MB（被丢）之间
// 且没有错误提示，6MiB 留足余量；base64 后约 8MB，远低于已验证可用的 12.1MB。
// 测试可下调该值以构造小体积用例。
var tudouReferenceByteBudget = 6 << 20

const (
	// 降采样前的单边像素上限，避免超大图在编码阶段就吃掉内存。
	tudouReferenceMaxSide = 4096
	// 体积仍超预算时的最大重编码轮数。
	tudouReferenceShrinkRounds = 4
)

func normalizeTudouReferenceDataURL(dataURL string) string {
	comma := strings.Index(dataURL, ",")
	if comma < 0 || !strings.HasPrefix(strings.ToLower(dataURL), "data:image/") {
		return dataURL
	}
	meta := strings.ToLower(dataURL[len("data:image/"):comma])
	raw, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil || len(raw) == 0 {
		return dataURL
	}
	// 网关实测可读的三种格式在预算内原样透传，不做无谓转码。
	knownFormat := strings.Contains(meta, "png") || strings.Contains(meta, "jpeg") || strings.Contains(meta, "jpg") || strings.Contains(meta, "webp")
	if knownFormat && len(raw) <= tudouReferenceByteBudget {
		return dataURL
	}
	var img image.Image
	switch {
	case strings.Contains(meta, "webp"):
		img, err = webp.Decode(bytes.NewReader(raw))
	case strings.Contains(meta, "gif"):
		img, err = gif.Decode(bytes.NewReader(raw))
	default:
		img, _, err = image.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		// 解码失败不伪装成功：保留原样，交由上游对非法格式报错。
		return dataURL
	}
	mime, payload := shrinkTudouReferenceImage(img)
	if len(payload) == 0 {
		return dataURL
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(payload)
}

// shrinkTudouReferenceImage 把超预算的参考图压回预算内：先按最长边限幅，
// 再按体积比例迭代降采样；无透明通道编码为 JPEG，有透明通道保留 PNG。
func shrinkTudouReferenceImage(img image.Image) (string, []byte) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return "", nil
	}
	scale := 1.0
	if longest := max(width, height); longest > tudouReferenceMaxSide {
		scale = float64(tudouReferenceMaxSide) / float64(longest)
	}
	var lastMime string
	var lastPayload []byte
	for round := 0; round < tudouReferenceShrinkRounds; round++ {
		targetWidth := max(1, int(math.Round(float64(width)*scale)))
		targetHeight := max(1, int(math.Round(float64(height)*scale)))
		resized := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		if targetWidth == width && targetHeight == height {
			draw.Draw(resized, resized.Bounds(), img, bounds.Min, draw.Src)
		} else {
			draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Src, nil)
		}
		var out bytes.Buffer
		mime := "image/png"
		if resized.Opaque() {
			mime = "image/jpeg"
			if err := jpeg.Encode(&out, resized, &jpeg.Options{Quality: 90}); err != nil {
				return "", nil
			}
		} else if err := png.Encode(&out, resized); err != nil {
			return "", nil
		}
		lastMime, lastPayload = mime, out.Bytes()
		if out.Len() <= tudouReferenceByteBudget {
			return mime, out.Bytes()
		}
		// 仍超预算：按剩余体积比例继续收缩；比例异常时至少收缩 25%，保证收敛。
		next := scale * math.Sqrt(float64(tudouReferenceByteBudget)/float64(out.Len()))
		if next >= scale {
			next = scale * 0.75
		}
		scale = next
		if scale < 0.05 {
			break
		}
	}
	if len(lastPayload) == 0 {
		return "", nil
	}
	return lastMime, lastPayload
}
