---
status: accepted
authors: [tn]
created_at: 2026-07-12
---

# ADR-101: CSV ユーザーインポートは検証・適用を別ジョブとして部分成功で処理する

## コンテキスト
管理者が初期移行時に多数のユーザーを登録するには、単件作成 API では安全性と操作性が不足する。CSV は不正な行を含み得る一方、HTTP リクエスト内で全件を書き込むとタイムアウトと再試行時の重複を招く。

## 決定
UTF-8 (BOM 可) の CSV を API でサイズ・ヘッダー・行数を検証して `Jobs` の tenant-scoped job に格納する。`dry_run` と `apply` は別ジョブとし、どちらも行ごとの stable error code を結果へ記録する。apply は有効行だけを既存 `CreateUser` use case 経由で作成し、既存 username はエラーとして残す部分成功方式を採る。CSV は `preferred_username,email,name,roles` を必須ヘッダー順で受け付け、password/hash は受け付けない。

## 却下した代替案
- HTTP 内で全件を同期作成する: リクエストのタイムアウトとクライアント再試行による重複を安全に扱えない。
- 全行 rollback: 1 行の修正のために有効な初期移行まで止まり、利用者に行単位の復旧手段を与えられない。
- CSV に初期パスワードを含める: 平文資格情報の保管・ログ・履歴への漏えい面を増やす。

## 影響
- `spec/contexts/identity-management.yaml` の `UserImportJob` / `UserImportRowError`、Import/Result interfaces、invariant、permission、scenario を追加する。
- `Jobs` の `user_import_preview` / `user_import_apply` handler が IdentityManagement use case を呼ぶ。
- 管理 API は CSV を受け付けて job ref を返し、job result は tenant 所有者だけに公開する。
