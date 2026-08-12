// Package bootstrap は idmagic プロセスの起動・DI を司る。
// main.go はここを呼ぶだけで、エンドポイント追加・永続層差し替えは本パッケージ内で完結する。
//
// EnvDefault/EnvInt/EnvDuration は idmagic-worker の worker 固有設定 (JOB_*,
// WORKER_ID 等、未移行) のみが使う。idmagic (API) プロセスと、API/worker が
// 共有する設定は ConfigLoader (config.go) + SharedConfig/APIConfig
// (sharedconfig.go, apiconfig.go) 経由で読み、不正値は起動を fail-fast させる
// (wi-103)。EnvInt/EnvDuration は不正値や負値を静かに fallback へ戻す旧来の
// 挙動を保持しており、worker 固有設定の移行 (wi-103 の後続タスク) が終わるまで
// 残す。
package bootstrap

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func EnvDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func EnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
