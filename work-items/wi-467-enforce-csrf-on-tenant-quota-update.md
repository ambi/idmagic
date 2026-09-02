---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-012 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
---

# テナントクォータ更新で Origin と CSRF トークンを検証する

## Motivation

Tenancy の決定は、状態を変える管理リクエストについて、セッション認証に加えて `Origin` と CSRF トークンの検証を要求している。

テナントの作成、属性更新、停止、再開、正規ロケーション切替は、ハンドラーの先頭で `VerifyBrowserRequest` を呼ぶ。

しかし `UpdateTenantQuota` だけはこの検証を行わず、認証と `system_admin` の確認後にクォータを保存する。

フロントエンドは `X-Csrf-Token` を送信しているが、サーバーが値を検証しないため、画面の実装だけが安全性を示している状態になっている。

## Scope

- `UpdateTenantQuota` の副作用より前に `VerifyBrowserRequest` を実行する。
- Cookie セッションによる要求は、正しい `Origin`、テナントに対応する CSRF Cookie、`X-Csrf-Token` がすべて一致する場合だけ許可する。
- `Authorization` ヘッダーによる正規の Bearer または DPoP 資格情報は、既存の `VerifyBrowserRequest` 契約どおり Cookie CSRF 検査の対象外とし、認証、スコープ、制御面主体の検査は維持する。
- CSRF 拒否時にクォータと利用量のいずれも変わらないことを HTTP 受け入れテストで確認する。
- Tenancy のほかの制御面変更ハンドラーを棚卸しし、同じ欠落がないことを回帰テストで固定する。

## Out of Scope

- `VerifyBrowserRequest` の double-submit Cookie 方式を別方式へ置き換えること。
- API アクセストークンから `UpdateTenantQuota` を呼べなくすること。
  [[wi-461-control-plane-credential-boundary]] はクォータ更新の自動化を維持する。
- システムコンソールに要求する認証強度または再認証時刻。
  [[wi-465-system-console-privileged-session-assurance]] が扱う。
- クォータ値の妥当性、同時更新、監査イベントを変更すること。

## Design

`VerifyBrowserRequest` は資格情報の種類を見て、Cookie を周囲資格情報として使う要求だけに Origin と CSRF の検査を適用する。

そのため `UpdateTenantQuota` の冒頭で同じ関数を呼んでも、`tenants:write` を持つ非ブラウザーの Bearer または DPoP クライアントは従来どおり呼び出せる。

拒否検査は `tenant_id` の解決、要求本文のデコード、`QuotaRepo.SetQuota` より前に置く。

認可後に置くと、CSRF 要求が対象の存在や本文の妥当性を観測できるため採らない。

ルーターへ個別の CSRF ミドルウェアを追加する案も採らない。

同じルート群には読出しと書込みが混在し、Bearer と DPoP の例外判断を既存関数と二重に実装することになるためである。

## Plan

1. CSRF トークンを持たない Cookie セッションが `UpdateTenantQuota` を成功させる現在の挙動を HTTP 境界で観測し、RED を確認する。
2. `VerifyBrowserRequest` を副作用より前へ追加する。
3. CSRF 拒否で `QuotaRepo.SetQuota` が呼ばれず、保存済みクォータが変わらないことを確認する。
4. 正しい CSRF を持つシステムコンソールと、正規の Bearer または DPoP クライアントの成功を確認する。
5. 制御面変更ハンドラーの棚卸し結果をテスト名または作業項目の完了記録へ残す。

## Tasks

- [ ] T001 [Acceptance] CSRF のない Cookie セッションでクォータが変更される現在の挙動を観測し、RED を確認する。
- [ ] T002 [App] `UpdateTenantQuota` の冒頭へ `VerifyBrowserRequest` を追加する。
- [ ] T003 [Unit] 拒否時に保存ポートが呼ばれないことを確認する。
- [ ] T004 [Acceptance] 不正 Origin、CSRF 欠落、不一致を拒否し、正しい Cookie セッションと非周囲資格情報を許可することを確認する。
- [ ] T005 [Inventory] 制御面変更ハンドラーの Origin と CSRF 検査を棚卸しする。
- [ ] T006 [Verify] 仕様とセキュリティ制御の検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-security-controls`
- `mise run report-security-test-gaps`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

拒否応答だけを確認しても、応答を書いた後に保存処理が進む誤実装を検出できない。

受け入れテストは HTTP 403 に加え、既存クォータが変わらず保存ポートも呼ばれないことを確認する。

Bearer または DPoP まで一律に CSRF 必須へすると、周囲資格情報ではない正規の自動化を壊す。

資格情報別の許可側テストを同じ変更に含める。

新しい公開記号や規範 ID を割り当てず、既存のセキュリティ決定へ実装を一致させる修正であるため、`reversibility` は reversible とする。
