package main

// 首启预置：数据目录还没有本地模型配置时，写入内嵌的默认渠道预设
// （土豆API 渠道，baseUrl https://api.ai-tudou.net，密钥留空，由用户自行填写）。
// 用户已有任何配置（文件已存在）时绝不覆盖；预设写入失败也不阻断启动，
// 用户仍可在设置里手动添加渠道。

import (
	_ "embed"
	"errors"
	"log"
	"os"
	"path/filepath"
)

//go:embed presets/local-model-config.default.json
var defaultModelConfig []byte

func seedDefaultModelConfig(dataDir string) {
	path := filepath.Join(dataDir, "local-model-config.json")
	if _, err := os.Stat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		return
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return
	}
	if err := os.WriteFile(path, defaultModelConfig, 0o600); err != nil {
		log.Printf("写入默认渠道预设失败（可稍后在设置中手动添加）: %v", err)
	}
}
