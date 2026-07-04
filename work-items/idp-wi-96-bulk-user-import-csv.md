---
id: idp-wi-96-bulk-user-import-csv
title: "管理者向けの CSV ユーザ一括インポートを導入する"
created_at: 2026-07-03
authors: ["tn"]
status: pending
risk: medium
---

# Motivation
現状ユーザ作成は 1 件ずつの CreateAdminUser のみで、初期移行や一括登録の導線が
無い。代表的な IdP は CSV による一括インポートを提供する (Okta / Entra / Google
の bulk import)。継続同期は SCIM ([[wi-31-scim2-provisioning]]) が担うが、単発の
移行 / 初期投入には SCIM は重い。

本 WI は管理者向けに、検証付きの CSV 一括インポート (dry-run で検証プレビュー →
適用) を追加し、行単位のエラーを集約して安全に取り込めるようにする。

# Scope
- **decision**:
  - 新規 ADR: CSV フォーマット (列 = 組み込み属性 + custom key)、検証方針 (行単位の部分成功 vs 全体 rollback)、既存ユーザ / 重複の扱い (skip / update)、 同期 vs 非同期 (初期は同期 + サイズ上限) を記録する。
- **scl**:
  - §3.3 interfaces: ImportUsers (dry-run / commit) とジョブ結果取得を 追加する。
  - §3.2 models: UserImportJob / UserImportRowError を追加する。
  - §3.4 states/events: UsersImported を追加する。
  - §3.5 invariants: dry-run は副作用なし、行検証 (email 形式 / 一意 / 属性 schema 準拠) を通すことを明示する。
- **go**:
  - CSV パーサ / 検証を追加し、既存 CreateUser usecase を再利用して行単位エラーを 集約する。サイズ上限を設ける。
- **http**:
  - admin の CSV upload / dry-run 結果 / commit エンドポイントを追加する。
- **ui**:
  - AdminUsers に「一括インポート」ウィザード (アップロード → 検証プレビュー → 適用) を追加する。
- **documentation**:
  - README に CSV フォーマットとインポート手順を追記する。

# Out of Scope
- 継続同期 ([[wi-31-scim2-provisioning]] / [[wi-45-outbound-scim-provisioning]])。
- export (既存の account data export とは別物)。
- 巨大ファイルの非同期 / ストリーミング処理、スケジュール実行。

# Verification
- `go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: CSV をアップロード → dry-run で行エラーがプレビューされる (副作用なし) → 適用で有効行のみ取り込まれ、無効行はエラーとして残ることを確認する。

# Risk Notes
一括書き込みのため、検証漏れによる不正データ投入・部分適用時の不整合・大容量
入力による負荷がリスク。dry-run を副作用なしに保ち、行検証 (形式 / 一意 /
schema) とサイズ上限をテストで担保する。既存 CreateUser のバリデーションを
再利用して二重管理を避ける。
