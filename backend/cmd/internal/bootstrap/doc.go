// Package bootstrap は idmagic プロセスの起動・DI を司る。
// main.go はここを呼ぶだけで、エンドポイント追加・永続層差し替えは本パッケージ内で完結する。
//
// 起動時設定はこのパッケージだけが環境変数から読む。各プロセスは ConfigLoader
// (config.go) を1つ作り、必要な Load*Config (sharedconfig.go, apiconfig.go,
// workerconfig.go, seeding_config.go) をすべて呼んでから loader.Err() を1回
// 確認する。値の型・範囲・相互依存はそこで検証し、不正なら副作用のある初期化
// (Assemble、listener 起動、seed 適用) より前に集約エラーで停止する
// (REQ-SYSTEM-016)。設定リファレンス (CONFIGURATION.md) は同じ Load*Config が
// 記録する ConfigField から生成する (REQ-SYSTEM-017)。
package bootstrap
