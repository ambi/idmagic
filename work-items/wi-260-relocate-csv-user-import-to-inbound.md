---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-19
depends_on: [wi-258-inbound-integration-taxonomy]
---

# CSV user import を IdManagement から inbound の適所へ移設する

## Motivation

CSV user import (`backend/idmanagement/usecases/user_import.go` と
`backend/idmanagement/adapters/http/admin_user_import_handler.go`) は現状 IdManagement context の
管理者一括操作として実装されているが、これは実質 **inbound provisioning の upload / batch 型**である。
ADR-128 §影響 が「適所でない、別 WI で然るべき場所へリファクタすべき」と明記し、
[[wi-258-inbound-integration-taxonomy]] が upload 型の帰属先を確定する。本 WI はその確定先へ CSV
import を移設し、IdManagement を identity principal の record-of-truth へ痩せさせる。

## Scope
- `idmanagement/usecases/user_import.go` / `admin_user_import_handler.go` を
  [[wi-258-inbound-integration-taxonomy]] 確定の inbound context / feature へ `git mv` する。
- SCL の CSV import 関連要素 (UserImport 系の interface / model / scenario) の context 帰属を
  確定先へ移す。`just yaml-check` を通す。
- Go import path の一括置換と、routes 配線 (admin CSV import ハンドラ登録) の移設。
- UI の CSV import route / 画面が参照する API path に変更が生じるかを確認し、必要なら追随する。

## Out of Scope
- CSV import の振る舞い (検証ルール・上限・dry-run / apply・エラーコード) の変更 (純移設)。
- SCIM server の rename ([[wi-259-rename-scim-inbound-server-context]])。
- CSV **export** ([[wi-148-admin-resource-csv-export]]) — これは別方向 (outbound read) の機能で対象外。

## Plan
- [[wi-258-inbound-integration-taxonomy]] の確定構造を待ってから着手する。
- wi-258 が upload を「統一 inbound context 内の feature slice」に置く場合、その context を作る
  [[wi-259-rename-scim-inbound-server-context]] の完了を前提とする可能性がある。依存の要否は wi-258
  の決定で確定するため、着手時に depends_on を見直す。
- import 系 SCL 要素の context 帰属変更は canonical ref の namespace を動かすので、SCL 参照・backend・
  生成物・UI の API path を一括で追随させる。

## Tasks
- [ ] T001 [SCL] UserImport 系要素を確定 inbound context へ移し `just yaml-check` を通す。
- [ ] T002 [Go] `user_import.go` / `admin_user_import_handler.go` を `git mv` し、import path・
      routes 配線を修正する。
- [ ] T003 [UI] CSV import 画面の API path 追随を確認・修正する。
- [ ] T004 [Verify] 下記 Verification を緑にする。

## Verification
- `just verify-go` / `just build-go` / `just test-go` — import path 解決とテスト緑。
- `just verify-ui` — UI の typecheck / lint / build 緑 (API path 変更時)。
- `just yaml-check` / `just check-ids` — context 帰属・canonical ref・Architecture 整合。
- `git log --follow` で履歴保持、旧 context への参照残存ゼロを grep で確認。
- 手動: 管理者が CSV を dry-run / apply して従来通りユーザーを検証・作成できることを確認する。

## Risk Notes
純移設で振る舞いは変えないが、context 帰属変更で canonical ref・API path が動くため、backend /
UI / 生成物の追随を検証ゲートで担保する。IdManagement から import ロジックを抜くとき、共有ヘルパ
(属性スキーマ検証等) への依存が残らないかを確認する。
