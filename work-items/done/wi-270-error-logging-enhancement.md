---
status: completed
authors: [Antigravity]
risk: medium
created_at: 2026-07-20
depends_on: []
completion:
  completed_at: 2026-07-20
  summary: "Implemented centralized logging middleware and enhanced error handler to log 400 validation errors as Warn, 500 errors as Error, and normal requests as Info."
---

# 汎用的なバックエンドログ充実化

## Motivation
API のエラー（500 だけでなく 4xx 系）や正常リクエスト時でも、どのパラメータがバリデーションに失敗したか、どの内部処理が走ったかなど、運用・トラブルシューティングに有用な情報をログに残す必要がある。現状はエラー時のスタックトレースが欠如し、正常処理の粒度も低いため、ログの一貫性と詳細度を向上させる。

## Scope
- `backend/` 全体の共通ロガー実装
- HTTP ハンドラ・ミドルウェアにおけるエラー・リクエスト情報の標準的出力
- バリデーションエラー（4xx）時のフィールド・バリデーション失敗情報の出力
- 正常リクエスト時の重要な処理ステップやパラメータのデバッグレベルログ

## Out of Scope
- ログフォーマットの大幅な変更（JSON 化等）は現行仕様を踏襲
- すべての内部ロジックにデバッグログを埋め込むことは行わない

## Plan
- 共通ロガーに `Info`, `Warn`, `Error` レベルを定義し、リクエスト開始・終了を `Info`、バリデーション失敗を `Warn`、内部例外を `Error` として出力。
- バリデーションミドルウェアで失敗したフィールド名と理由を構造化してログに含める。
- エラーハンドラで 4xx/5xx エラー時に詳細情報（スタックトレース、原因）を `Error` ログに出す。
- 正常レスポンス時にリクエスト URL, メソッド, ユーザーID (存在すれば), 主要パラメータを `Info` に記録。

## Tasks
- [x] T001 [App] 既存ロガーのレベルとフォーマットを確認・統一する。
- [x] T002 [App] バリデーションミドルウェアで失敗情報を `Warn` ログに出力する実装。
- [x] T003 [App] エラーハンドラで 4xx/5xx エラー時に詳細情報（スタックトレース、原因）を `Error` ログに出す。
- [x] T004 [App] 正常リクエスト時の `Info` ログ出力を追加する。
- [x] T005 [Verify] 各種エンドポイントで意図的に 4xx/5xx エラーを起こし、ログが期待通り出力されることを確認。
- [x] T006 [Verify] 正常リクエストでも期待する情報が `Info` ログに記録されることを確認。

## Verification
- `just dev-api` で対象エンドポイントをテストし、ログ出力を確認。
- `just test-go` でテストがパスすることを確認。

## Risk Notes
- ログ出力増加によるパフォーマンス影響は低いが、過剰なデバッグレベルは本番環境で無効化する設計とする。

## Notes
- **T001**: 既存の `backend/shared/logging` が `slog` を用いて JSON Lines で統一されていることを確認。
- **T002, T004**: 新たに `LoggingMiddleware` (`backend/shared/http/support_http/logging_middleware.go`) を追加し、リクエスト終了時に `Info` を、400 (Bad Request) エラー時に抽出したバリデーションメッセージと共に `Warn` ログを出力するようにした。その他の 4xx は client request error として `Warn` に記録。
- **T003**: `ErrorHandler` を拡張し、`code >= 400` の未処理エラーにスタックトレース情報を付与して `Error` レベルで出力するように変更。
- **検証**: `just dev-api` 環境で実際に `GET /health` (200 Info), 存在しないリソース (404 Warn & Error), 不正な JSON リクエスト (400 Warn) を発生させ、期待通りのログ出力を確認。`just test-go` で関連する HTTP テストが通過することを確認した。
