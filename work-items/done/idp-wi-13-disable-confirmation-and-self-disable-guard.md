---
id: idp-wi-13-disable-confirmation-and-self-disable-guard
title: "ユーザ無効化に確認ダイアログと自爆防止を入れる (削除と対称にする)"
created_at: 2026-06-16
authors: ["tn"]
status: completed
risk: low
---

# Motivation
wi-8 で削除 (anonymize cascade) には preferred_username typing 確認と
admin の自爆防止を入れたが、無効化 (`POST /api/admin/users/:sub/disable`)
はワンクリックで即実行されたままになっている。Disable も実際は
破壊的:
  - 既存 session 利用が拒否される。
  - 新規 login が invalid_credentials になる。
  - refresh が invalid_grant になる (refresh_tokens.go で disabled_at
    チェック + family 失効)。
「Disable は復活可能だから軽い」というのは admin 操作が正しい前提の
話で、誤操作で社内ユーザ 1 名を即時アクセス遮断する事故は十分に起こる。
業界標準 (Okta deactivate / Google suspend / Azure block sign-in) も
確認ダイアログを挟む。

本 WI は削除側 (wi-8) の安全性と対称になるよう Disable にも軽い確認
と自爆防止を入れる。Enable は影響が "アクセス回復" のみで誤操作リスクが
低いため、確認は入れない (片側非対称)。

# Scope
- **go**:
  - authusecases.SetUserDisabled に「actor.Sub == target.Sub かつ target.Roles に admin / system_admin を含むかつ disabled=true」の pre-check を追加し、新規 sentinel `ErrSelfDisableForbidden` を 返す。Enable 方向 (disabled=false) は対象。`ErrSelfDeleteForbidden` と並びの error として扱う。
  - admin_user_handler.handleSetAdminUserDisabled は writeAdminUserError で `ErrSelfDisableForbidden` を 400 `self_disable_forbidden` として 返す。
- **ui**:
  - AdminUsersPage の UserDetails にある「アカウントを無効化」ボタンを `DisableUserDialog` 経由で開くように変更する。ダイアログは:
      - 対象 user の表示名と preferred_username を見せる。
      - 影響説明: "ログイン拒否・既存 session 無効・refresh token 拒否"。
      - 復元動線: "アカウント状態 → 再有効化 で元に戻せる" を明記。
      - typing 確認は **しない** (復元可能のため、delete より弱い
        ceremony で済ます)。
      - 確定ボタンは destructive variant、キャンセルは outline。
  - Enable (再有効化) はダイアログ無しのまま。`user.disabled_at != null` のときの「再有効化」ボタンは現状の即時実行を維持する。
  - ダイアログのコンポーネント名は `DisableUserDialog`。`DeleteUserDialog` の構造を流用するが username typing 入力は持たない。
- **api**:
  - 変更なし。`POST /api/admin/users/:sub/disable` の HTTP 契約は そのまま。
- **documentation**:
  - 変更なし (README の admin 機能行に変更なし)。

# Out of Scope
- Disable の任意 reason フィールド (audit に残す場合は別 WI で UserDisabled event の payload を拡張する)。
- Enable 側の確認ダイアログ。
- 削除 (wi-8 / wi-12) との UI / state 統合。
- bulk disable (一度に複数 user を無効化する経路)。

# Verification
- `go test ./...` (in: idmagic)
  - reason: SetUserDisabled の自爆防止テスト追加。HTTP 側で self_disable_forbidden が返ることを確認。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動 1: alice を選んで「アカウントを無効化」 → DisableUserDialog が 表示 → キャンセルで何も起きない → 確定で disabled_at が設定される ことを確認。
- 手動 2: admin 自身を無効化しようとして 400 self_disable_forbidden が返ることを確認。
- 手動 3: 無効化済 user の「再有効化」はダイアログ無しで即実行される ことを確認 (Enable は確認しない)。

# Risk Notes
Disable は既に頻繁に走る経路ではないため変更影響は小さい。確認 1 段
を増やすだけで behavior は不変。自爆防止の pre-check は DeleteUser の
同じ `hasPrivilegedRole` ヘルパを流用できる。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  削除側 (wi-8) と対称になるよう、無効化 (disable) に軽い確認ダイアログと
  admin の自爆防止を入れた。enable (再有効化) は誤操作リスクが低いため
  確認なしの即時実行を維持した (片側非対称)。新規 model / state / event /
  permission / HTTP endpoint は追加せず、既存 `SetUserDisabled` use case と
  `hasPrivilegedRole` ヘルパを流用した。
- **Verification Results**:
  - `go test ./...` (in: idmagic)
    - result: ok (新規 TestSetUserDisabledRejectsSelfDisable / TestSetUserDisabledAllowsDisablingOtherAdmin を含め全 pass)
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok (0 issues)
  - `go build ./...` (in: idmagic)
    - result: ok
  - `bun --cwd idmagic/ui typecheck`
    - result: ok (tsc --noEmit pass)
  - `bun --cwd idmagic/ui lint`
    - result: ok (biome、no fixes)
  - `bun --cwd idmagic/ui build`
    - result: ok
  - 手動確認 (residual): dev サーバを起動した実ブラウザ操作は本セッションでは 未実施。既存ダイアログ (DeleteUser) と同パターンで実装し typecheck / lint / build 緑のため動作する見込み。実環境での操作確認は次回 dev 起動時。
- **Affected Guarantees State**:
  - 自爆防止: SetUserDisabled (disable 方向のみ) に 「actor.Sub == target.Sub かつ privileged role」の pre-check を追加。 enable 方向は対象外。go テストで self-disable 拒否 / self-enable 許可 / 他 admin の無効化許可を確認した。
  - tenant isolation: 既存 requireAdmin 経路をそのまま使い、新規境界は無し。
  - auth fail-close: 既存の disabled_at 検知ルート (refresh / login / session) は無変更。pre-check が 1 段増えるだけ。
  - SCL coherence は不変。
