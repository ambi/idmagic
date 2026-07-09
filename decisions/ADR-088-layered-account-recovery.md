---
status: suggested
authors: [tn]
created_at: 2026-07-09
---

# ADR-088: アカウント復旧を階層化し、recovery code を最終手段に再位置づける

## コンテキスト

現状 idmagic の TOTP / WebAuthn 喪失時の復旧経路は、ユーザー自身が管理する
backup **recovery code のみ**である（[ADR-087](file:///Users/tn/src/idmagic/decisions/ADR-087-webauthn-phishing-resistant-mfa.md)、
`spec/contexts/authentication.yaml` の `RecoveryCode` / `RecoveryCodePolicy`）。
recovery code 実装自体は hash-only・single-use・set 全置換・step-up 必須で堅実だが、
これが**唯一のセルフサービス安全網**であることが構造的な単一障害点になっている。
recovery code をユーザーが正しく保管しないケースは多く、静的共有秘密ゆえソーシャル
エンジニアリングにも弱い。管理者による認証器リセットや緊急ロック解除の導線は存在しない
（Explore 調査で確認）。

代表的な IdP はこの問題をユーザー管理コード**単独**では扱わず、多層で対処している。

- **Okta**: 既定は管理者リセット（Reset Authenticators → 再登録）。org 設定で
  self-service account recovery（Email / SMS / セキュリティ質問 / Okta Verify Push）。
  ユーザー管理の使い捨てコードは主役ではなく、本人確認（ID プルーフィング）ベースの
  復旧へ移行しつつある。
- **Microsoft Entra ID**: 既定で MFA の self-service リセットは無く、SSPR は「最低 1 手段の
  保持」を前提とする。TAP（管理者発行の時限パスコード）で再オンボード。全手段喪失時は
  SSAR（政府発行 ID + 顔ライブネス検証）。combined registration で**複数手段の登録を推奨**し、
  ロックアウト自体を予防する。
- **Keycloak**: Recovery Authentication Codes（idmagic とほぼ同型）を持つ一方、管理者が
  OTP credential を削除して required action で再登録させる導線と、forgot-password メールで
  OTP を再設定する導線を併設する。
- **Google（消費者）**: **手段の冗長化**が本体（別端末 / バックアップ番号 / セキュリティキー /
  別端末の passkey / 信頼済み端末）。バックアップコードは de-emphasize され、最終的には
  リスクスコアリング付きの本人確認回復（数日）に委ねる。

共通する潮流は「ユーザー管理のリカバリコードを主役から外し、(1) 手段の冗長化で
ロックアウトを予防し、(2) 管理者リセットを緊急 backstop に置き、(3) recovery code は
最終手段のバックアップとして残す」ことである。idmagic は Keycloak と同じセルフホスト型
IdP の位置づけであり、recovery code の廃止ではなく**上位 2 層の追加**が本質的なギャップと判断する。

## 決定

アカウント復旧を次の 3 層構造として定義し、recovery code をその最終層に再位置づける。

### 1. 第 1 層: 手段の冗長化（予防）

2 個目の認証器（2 個目の passkey、または TOTP + passkey）の登録を推奨・任意強制できる
ようにし、「1 台紛失 = ロックアウト」を予防する。idmagic は既に `WebAuthnCredential` を
複数保持でき、同期 passkey を示す `backup_eligible` / `backup_state` を保存済みであり、
この層に自然に乗る。MFA 登録オンボーディング（[wi-127](file:///Users/tn/src/idmagic/work-items/wi-127-mfa-enrollment-onboarding-and-enforcement.md)）
と連携し、登録直後に 2 個目の認証器登録を促す。詳細は別 work item で仕様化する。

### 2. 第 2 層: 管理者による認証器リセット（緊急 backstop）

管理者が対象ユーザーの認証器（TOTP / WebAuthn / recovery code）をリセットし、次回ログイン
時に再登録を強制できる導線を新設する。Okta / Entra / Keycloak の企業向けベースラインに
相当し、idmagic に現状欠けている層である。監査イベントを必須とし、リセットは既存の管理者
操作（[wi-93](file:///Users/tn/src/idmagic/work-items/wi-93-admin-user-impersonation.md) 等）と同じ権限モデル・監査枠組みに揃える。詳細は別 work item で仕様化する。

### 3. 第 3 層: recovery code（最終手段のバックアップ）

recovery code は ADR-087 の設計（hash-only・single-use・set 全置換・step-up 必須・
`mfa_enrolled` に数えない）を維持する。ただし UI 上の位置づけを「第二要素の一手段」から
**「最終手段のバックアップ」**へ改め、残数警告・生成/ダウンロードの明確化など UX を整える
（本 ADR では方針のみ。実装可否は別途）。

本人確認（ID プルーフィング）ベースの復旧（Entra SSAR / Okta+Nametag 型）は idmagic の
現段階では過剰と判断し、本 ADR では採用しない（Out of Scope、将来の再検討候補）。

## 却下した代替案

- **recovery code を廃止して管理者リセットのみにする**: セルフホスト小規模運用では管理者
  対応が常時可能とは限らず、セルフサービスの最終手段を失う。Keycloak も両方を併設する。
- **メール検証による自動復旧を主経路にする（Keycloak forgot-password 型）**: 実装は容易だが
  「メールアカウント = 復旧のルート」となり、認証強度がメールと同値まで低下する。第 1・第 2 層を
  先に埋める方が費用対効果が高い。secondary/recovery email 自体は [wi-41](file:///Users/tn/src/idmagic/work-items/wi-41-secondary-and-recovery-email.md) の範疇に留める。
- **本人確認ベース復旧（SSAR 相当）を今導入する**: ID / ライブネス検証の設計面積が大きく、
  セルフホスト IdP の初期段階には過剰。将来検討に回す。
- **recovery code を主要素（`mfa_enrolled` に数える）に昇格させる**: ADR-087 で却下済み。
  backup は backup に留める方針を維持する。

## 影響

- **SCL**: 本 ADR は方針決定であり、SCL への反映は派生 work item（管理者リセット、
  手段冗長化推進）で `spec/contexts/authentication.yaml`（復旧イベント、管理者リセット
  interface、第二認証器推奨状態）および `spec/contexts/application.yaml`（管理 UI / ポリシー）
  として行う。derived artifacts は各 work item で再生成する。
- **既存資産の再利用**: `WebAuthnCredential` の複数登録・`backup_eligible`/`backup_state`、
  `RecoveryCode` 一式、step-up（[ADR-043](file:///Users/tn/src/idmagic/decisions/ADR-043-account-portal-csrf-and-step-up.md)）、
  監査イベント基盤、管理者操作の権限モデルをそのまま使う。
- **関連 work item**: 第 1 層は wi-127 と連携する新 work item、第 2 層は管理者リセットの
  新 work item として起票する。第 3 層 UX は第 2 層 work item に含めるか従属タスクとする。
- **非目標**: 本人確認ベース復旧・SMS/voice factor・メールルート復旧は本 ADR の対象外。
