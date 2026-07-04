---
id: repo-wi-1-rename-go-module-to-github-import-path
title: "Go module path を github.com/ambi/idmagic に変更する"
created_at: 2026-07-04
authors: [tn]
status: pending
risk: medium
---

# Motivation
現在の Go module path は `idmagic` であり、外部リポジトリや GitHub 上の canonical import path として扱えない。
module path を `github.com/ambi/idmagic` に揃えることで、公開リポジトリ名、依存解決、import path、将来の外部利用の前提を一貫させる。

# Scope
- `go.mod` の module 宣言を `github.com/ambi/idmagic` に変更する。
- リポジトリ内の Go import path を新しい module path に合わせて更新する。
- 必要に応じてビルド、テスト、静的検証、開発用スクリプトの参照を更新する。
- `spec/scl.yaml` の仕様意味は変更しない想定。実装時にユーザー可視の機能・振る舞いが変わる場合は、SCL-first で対象セクションを更新する。

# Out of Scope
- パッケージ構成やディレクトリ構成の再設計。
- GitHub リポジトリの作成、移管、リモート設定変更。
- 機能追加や API 振る舞いの変更。

# Verification
- `go test ./...`
- `go build ./...`
- `just yaml-check-work-items`
- `just check-ids`

# Risk Notes
import path 変更は全 Go パッケージに波及し、更新漏れがあるとビルド不能になる。機械的な一括更新後に `go test ./...` と `go build ./...` で漏れを検出する。
