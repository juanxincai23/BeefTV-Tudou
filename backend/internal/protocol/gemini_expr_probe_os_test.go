package protocol

import "os"

func osReadTudouManifest() ([]byte, error) {
	return os.ReadFile("../../../plugin-packages/tudou-image/manifest.json")
}
