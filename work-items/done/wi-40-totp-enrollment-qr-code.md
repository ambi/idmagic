---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-06-21
---

# 認証アプリ登録時に otpauth URI の QR コードを表示する

## Motivation
[[wi-21-end-user-account-portal]] の security ステージで TOTP の self-service
登録を追加したが、`/account/security` の登録画面は otpauth URI とセットアップ
キーを文字列で提示するだけで、認証アプリへの登録は手動入力に頼る。実運用の
IdP (Keycloak / Okta / Google) はいずれも QR コードを第一手段として提示し、
手動キーは「スキャンできない場合」のフォールバックに置く。QR を出すことで
スマートフォンの認証アプリへワンスキャンで登録でき、手入力由来の enroll
失敗を減らせる。

## Scope
- **decision**:
  - QR は **クライアント側で生成**する。otpauth URI と secret は既に TOTPEnrollmentStart でブラウザに返っており、サーバで画像化しても新たな secret 露出面を増やすだけなので、サーバ側エンドポイントや Go 依存は 追加しない。SPA に小さな QR ライブラリ (`qrcode.react`、MIT、型同梱、 ランタイム依存は react のみ) を 1 つ加え、`enrollment.otpauth_uri` を SVG として描画する。SVG は CSP 親和的で高 DPI / 印刷でも崩れない。
- **scl**:
- **go**:
- **ui**:
  - AccountSecurityPage の登録ステップに QR (SVG) を追加し、セットアップキーと otpauth URI 文字列は「スキャンできない場合」のフォールバックとして残す。
  - QR の図形には代替テキストを与え、視覚に頼れない利用者にはセットアップキー 手動入力の導線を維持する (アクセシビリティの後退を起こさない)。
- **documentation**:
  - README の End-user account portal セクションのセキュリティ項に QR 表示を追記。

## Out of Scope
- サーバ側 QR 生成 / 画像エンドポイント。
- WebAuthn / SMS OTP / step-up auth (wi-21 後続ステージのまま)。
- QR のロゴ埋め込み・ブランドカラー化。

## Verification
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: /account/security で「認証アプリを設定」→ QR が表示され、認証アプリで スキャンするとアカウントが追加される。表示された 6 桁コードで登録を完了でき、 ログアウト→再ログインで TOTP チャレンジを通過できる。
- 手動: QR を使わずセットアップキー手動入力でも従来どおり登録できる (フォールバック維持)。

## Risk Notes
UI 表示のみの追加で contract / Go / 永続化に変更はない。リスクは新規 npm 依存
追加に伴うサプライチェーンのみで、`qrcode.react` は MIT・広く使われる小さな
依存。secret は既存挙動と同じくクライアント側に閉じる。

## Completion
- **Completed At**: 2026-06-21
- **Summary**:
  AccountSecurityPage の TOTP 登録ステップに qrcode.react の QRCodeSVG で
  otpauth URI の QR を表示。セットアップキー / otpauth URI 文字列はスキャン
  できない場合のフォールバックとして併置。SCL / Go / 永続化は無変更で、
  contract (/api/account/mfa/totp/enroll/*) も不変。
- **Verification Results**:
  - `bun --cwd idmagic/ui typecheck`
    - result: passed
  - `bun --cwd idmagic/ui lint`
    - result: passed
  - `bun --cwd idmagic/ui build`
    - result: passed
- **Affected Guarantees State**:
  - secret 露出面: QR はクライアント側生成のままで、サーバへ secret を送らない。 後退なし。
  - アクセシビリティ: セットアップキー手動入力を QR と併置。登録手段の後退なし。
