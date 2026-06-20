// Package stats 輕量遊戲數據上報：背景 POST 到共用 game-stats 收集服務。
// 失敗一律忽略，絕不影響遊戲本身。
package stats

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var statsURL = getenv("STATS_URL", "https://admin-games.ricky-nova.com/api/event")
var gameName = getenv("STATS_GAME", "duogame")

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Track 背景上報一筆事件。fire-and-forget，2 秒逾時，任何錯誤吞掉。
func Track(event string, data map[string]interface{}) {
	go func() {
		defer func() { _ = recover() }()
		payload := map[string]interface{}{"game": gameName, "event": event}
		for k, v := range data {
			payload[k] = v
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return
		}
		req, err := http.NewRequest("POST", statsURL, bytes.NewReader(b))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()
}
