---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-09
---

# 復旧目的の 2 個目認証器登録を推奨し手段冗長化でロックアウトを予防する

## Motivation

TOTP / WebAuthn 喪失時の復旧を backup recovery code **単独**に依存させると、コード紛失で
復旧経路が絶たれる（[ADR-088](file:///Users/tn/src/idmagic/decisions/ADR-088-layered-account-recovery.md)）。
Google / Entra ID（combined registration）が示すとおり、最も費用対効果の高い予防策は
**手段の冗長化**——認証器を 2 個以上登録させ、1 台紛失をロックアウトにしないことである。

idmagic は既に 1 ユーザーが複数 `WebAuthnCredential` を登録でき、同期 passkey を示す
`backup_eligible` / `backup_state` も保存済みで、この層に自然に乗る。ADR-088 の第 1 層として、
2 個目認証器（2 個目の passkey、または TOTP + passkey）の登録を推奨・任意強制できる仕組みを
仕様化する。MFA 登録オンボーディング（[wi-127](file:///Users/tn/src/idmagic/work-items/wi-127-mfa-enrollment-onboarding-and-enforcement.md)）と連携し、初回登録直後に 2 個目を促す。

## Scope

- `spec/contexts/authentication.yaml`:
  - 「復旧手段が単一（single point of failure）」を表す派生状態（例: 認証器 1 個のみ / 同期不可
    passkey のみ / recovery code のみ）と、その account security への提示。
  - 2 個目認証器登録の推奨状態と、任意強制ポリシー（推奨 / 必須）の表現。
- `spec/contexts/application.yaml` および管理 UI: テナント既定 / アプリ単位で「2 個目認証器を
  推奨するか必須にするか」を設定する項目。
- Account UI（`AccountSecurityPage`）: 復旧手段が単一のときの警告と、2 個目登録への導線。
- 登録オンボーディング（wi-127）: 初回 MFA 登録完了直後に 2 個目認証器登録を促すステップ。

## Out of Scope

- 管理者による認証器リセット（ADR-088 第 2 層。[wi-143](file:///Users/tn/src/idmagic/work-items/wi-143-admin-authenticator-reset-and-account-recovery.md)）。
- recovery code の設計変更（ADR-087 の hash-only / single-use / 全置換は維持）。
- 新しい factor 種別（SMS / voice）の追加。
- 本人確認ベース復旧（SSAR 相当）。

## Plan

- 方針:
  - 既存の `WebAuthnCredential` 複数登録・`backup_eligible`/`backup_state`・TOTP を再利用し、
    新しい factor は増やさない。
  - 「復旧手段が単一か」を派生状態として計算し、account security とオンボーディングで提示する。
    同期 passkey（`backup_state=true`）は復旧耐性が高い扱いとし、単一警告の判定に反映する。
  - 2 個目登録の強制はテナント / アプリのオプトインポリシーに留め、既定は推奨（非強制）とする。
    強制する場合は wi-127 の enrollment-required flow を再利用し、fail-closed で扱う。
- 参考にする外部パターン: Entra ID combined registration、Google の複数手段・同期 passkey。
- 却下する代替案:
  - 常に 2 個目を必須化: 小規模運用の導入摩擦が大きい。既定は推奨に留めオプトインで必須化。
  - recovery code を廃止して冗長化のみに依存: セルフサービスの最終手段を失う（ADR-088 で却下済み）。
- 未決定事項: 「復旧手段が単一」の厳密な判定基準（同期 passkey 1 個を単一とみなすか）、
  推奨導線を wi-127 に統合するか独立ステップにするか。

## Tasks

- [ ] T001 [SCL] 復旧手段の単一性を表す派生状態、2 個目推奨状態、任意強制ポリシー、提示 UX を仕様化する。
- [ ] T002 [Domain] 認証器数・同期 passkey 有無から「復旧手段が単一か」を判定する関数を追加する。
- [ ] T003 [UseCase] 2 個目登録推奨状態の算出と、強制ポリシー時の enrollment-required flow 接続を追加する。
- [ ] T004 [Account/UI] 単一時の警告と 2 個目登録導線を account security に追加する。
- [ ] T005 [Admin/UI] テナント / アプリの「2 個目認証器 推奨 / 必須」設定を追加する。
- [ ] T006 [Verify] E2E で、単一警告の表示、2 個目登録での警告解消、必須ポリシー時の登録誘導を固定する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動確認:
  - 認証器 1 個のみのユーザーに account security で復旧単一の警告が出る。
  - 2 個目の passkey / TOTP を登録すると警告が消える。
  - 必須ポリシー下では初回登録後に 2 個目登録が促される。

## Risk Notes

リスクは中程度。強制ポリシーを誤ると登録摩擦やロックアウトを招くため、既定は推奨（非強制）
とし、強制は明示オプトイン + wi-127 の fail-closed な enrollment-required flow に限定する。
「復旧手段が単一か」の判定は保守的に倒し、過剰警告より見落としを避ける。
