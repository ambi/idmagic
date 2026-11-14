---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-09
priority: p3
group: authentication
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/totp.md, requirement: REQ-AUTHENTICATION-011 }
  - { path: docs/modules/authentication/features/mfa.md, requirement: REQ-AUTHENTICATION-022 }
---

# 二つ目の認証器の登録を勧め、テナントの設定で必須にできるようにする

## 動機

TOTP や WebAuthn の認証器を失ったときの復旧を、復旧コードだけに頼ると、コードを失えば復旧の道が絶たれる（[アカウントの復旧](../../docs/modules/authentication/features/recovery.md)）。
Google と Microsoft Entra が示すとおり、最も費用対効果の高い予防は手段の冗長化であり、認証器を二つ以上登録させれば一台の紛失は締め出しにならない。
IdMagic は一人の User に複数の WebAuthn の資格情報を登録でき、同期される passkey を示す `backup_eligible` と `backup_state` も保存しているので、この予防に乗せやすい。
MFA の登録の案内（[wi-245](../done/wi-245-mfa-enrollment-onboarding-and-enforcement.md)）は完了しており、初めの登録の直後に二つ目を促す位置がある。

既存の機能の延長だが、新しい推奨の状態とテナントの設定を足すので、優先度は p3 とする。

## 対象範囲

- 復旧の手段が一つだけの状態（認証器が一つだけ、同期されない passkey だけ、復旧コードだけ）を判定し、アカウントのセキュリティの画面で警告して二つ目の登録へ案内する。
- テナントの設定で、二つ目の認証器を推奨にするか必須にするかを選べるようにする。
- MFA の初めの登録の直後に、二つ目の登録を促す。

## 対象外

- 管理者による認証器のリセット。既に[アカウントの復旧](../../docs/modules/authentication/features/recovery.md)にある。
- 復旧コードの設計の変更（ハッシュだけの保存、単回の使用、全件の置き換え）。
- 新しい要素の種類（SMS、音声）。wi-295 が扱う。
- 本人確認に基づく復旧。

## 検証

- 復旧の手段が一つだけの状態の判定を固定するテスト。
- 必須にしたテナントで、二つ目を登録するまで登録の案内が終わらないことを観測する受け入れのテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

必須にすると、認証器を一つしか持てない利用者が登録を終えられない。
必須の設定には、二つ目に復旧コードを数えるかどうかを含めて決める。
