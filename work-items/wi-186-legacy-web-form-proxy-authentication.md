---
status: pending
authors: ["tn"]
risk: critical
created_at: 2026-07-11
---

# レガシーWebフォームへの代理認証を安全な限定互換機能として導入する

## Motivation
OIDC、SAML、WS-Fed のいずれにも対応しないレガシーWebサービスでは、利用者が
個別の ID/パスワードを毎回入力しなければならない。標準フェデレーションへ移行できない
サービスに限り、IdMagic が暗号化保管した資格情報をブラウザ拡張で正規ログイン画面へ
入力することで、利用者体験とアクセス統制を改善する。

これは真のプロトコルSSOではなく、対象ページDOMへ秘密情報を渡す高リスクな互換機能である。
標準連携を常に優先し、最小権限、MFA、厳格なURL照合、監査、fail-closed を満たす場合だけ
提供する。

## Scope
- **dependencies**:
  - `wi-97-envelope-encryption-at-rest` の tenant-scoped envelope encryption を、可逆な
    代理認証資格情報の保管・復号に必須とする。
  - `wi-151-managed-device-inventory-and-posture-access-conditions` 完了後に、検証済み
    管理端末向けの認証freshness緩和を利用する。完了前は全端末を未管理として扱う。
- **scl**:
  - `spec/scl.yaml` の context map に `LegacyAccess` を追加する。
  - `spec/contexts/legacy-access.yaml` の glossary、models、interfaces、states、invariants、
    scenarios、permissions、objectives、user_experience を追加する。
  - `spec/contexts/application.yaml` に `legacy_form` protocol binding と、代理認証を使う
    Application のポータル起動情報を追加する。
- **decision**:
  - 新規 ADR で、標準フェデレーションを優先する境界、秘密情報の保管/発行境界、
    browser extension の信頼境界、auto-submit の例外運用、共有資格情報の監査を記録する。
- **go**:
  - `backend/legacyaccess/` に domain、usecases、ports、HTTP/persistence adapters、module を追加する。
  - 本人用 credential と共有 credential/grant、origin/path/selector 検証、credential 発行、
    監査、application assignment / sign-in policy gate を実装する。
  - 暗号文だけを保存し、平文の credential を DB・ログ・監査イベント・通常 API 応答に残さない。
- **http / extension**:
  - account/admin の credential と legacy binding 管理 API、extension 専用の一回限り
    autofill credential 発行 API を追加する。
  - Chrome、Edge、Firefox 向け WebExtension を追加し、OAuth Authorization Code + PKCE、
    メモリ内の一回限りメッセージング、最小 host permission で入力する。
- **ui**:
  - admin application 管理に legacy form binding、共有資格情報、auto-submit 承認状態を追加する。
  - account portal に本人用資格情報の登録/置換/削除と、拡張導入状態を追加する。
- **architecture / documentation**:
  - `ARCHITECTURE.md` に LegacyAccess context と依存方向を同期する。
  - README に対象範囲、Vault設定、拡張配布、対応外ログイン形態、運用上の残余リスクを追加する。

## Out of Scope
- OIDC/SAML/WS-Fed 化できるサービスへの代理認証適用。
- パスワード平文の再表示・エクスポート・監査ログ記録。
- 初期版での CAPTCHA、WebAuthn、任意の多段ログイン、パスワード変更の自動追従。
- `wi-151` より先の端末証明、MDM connector、OS-level attestation。
- 対象サービスの認証リクエストを IdMagic から直接送信するプロキシ。

## Plan
- `LegacyFormBinding` は完全一致の許可origin/path規則、入力/送信セレクタ、submit mode、
  検証状態を持つ。既定は `manual_submit` とし、`auto_submit` は管理者が検証済みにした
  binding に限る。
- `PersonalCredential` を優先し、無い場合だけ user/group grant が一致する
  `SharedCredential` を使う。共有候補が複数一致する設定は拒否する。
- credential 発行には MFA を必須にする。device posture を信頼できない間は1時間の MFA
  freshness を既定とし、`wi-151` 完了後のみ管理端末の本人用 credential に8時間の既定を適用する。
- credential 発行は extension client、IdP session、MFA freshness、Application assignment、
  binding 状態、正規origin/path をすべて検証し、いずれかが欠ければ fail-closed で拒否する。
- extension は資格情報を永続化せず、background から対象tabの content script へ一度だけ
  渡す。ページやcontent scriptからのメッセージはすべて非信頼入力として検証する。

## Tasks
- [ ] T001 [Dependency] wi-97 の暗号化基盤を完了し、可逆秘密を安全に保存できる状態にする。
- [ ] T002 [SCL] LegacyAccess と Application binding の規範仕様、scenario、不変条件、権限、UX を追加する。
- [ ] T003 [Decision] 代理認証の安全境界と運用例外を ADR に記録する。
- [ ] T004 [Domain] credential、grant、binding、発行判定を実装し、unit test を追加する。
- [ ] T005 [Adapter] memory/postgres/Vault persistence、HTTP API、監査、DI/route を実装する。
- [ ] T006 [UI] admin/account UI と Chrome/Edge/Firefox 拡張を実装する。
- [ ] T007 [Dependency] wi-151 完了後に端末posture連動の freshness policy を有効化する。
- [ ] T008 [Verify] SCL、Go、UI、各ブラウザE2E、Vault障害と拒否シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just verify`
- 手動: 本人用資格情報を登録し、割当済みの正規originでだけ拡張が入力し、別originでは拒否する。
- 手動: 本人用が無いときだけ単一の共有grantを使い、複数共有grantが一致する設定は保存できない。
- 手動: MFA freshness不足、未割当、binding未検証、Vault障害では credential が発行されない。
- 手動: auto-submit未承認bindingは入力後に利用者が送信し、承認済みbindingだけが自動送信する。

## Risk Notes
代理認証は集中保管した秘密情報を対象ページDOMへ渡すため、対象サービスのXSS、端末侵害、
誤ったURL/セレクタ設定による漏洩の影響が大きい。暗号化基盤の未実装時に平文保管へ
フォールバックしてはならない。標準プロトコルを優先し、origin/pathを厳格照合し、MFA、
最小権限、単回発行、監査、Vault障害時fail-closed を必須とする。
