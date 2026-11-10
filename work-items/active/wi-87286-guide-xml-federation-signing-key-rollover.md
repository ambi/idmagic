---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-11
priority: p2
depends_on: [wi-96973-guide-entra-federation-setup-and-verify-applied-settings]
change_kind: feature
affected_spec:
  - { path: docs/modules/signing-keys/features/separation.md, requirement: REQ-SIGNINGKEYS-006 }
  - { path: docs/modules/signing-keys/features/lifecycle.md, requirement: REQ-SIGNINGKEYS-001 }
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
---

# XML 署名鍵の準備から Entra への証明書登録と切り替えまでを管理画面で案内する

## 動機

XML 署名鍵の prepare/activate API と Entra 向け NextSigningCertificate の生成は既にある。
一方、一般の鍵画面の回転操作は通常の署名用途の操作であり、XML 鍵の二段階の更新を完遂できない。
管理者は API と複数のドメインの Graph 更新を手で組み合わせる必要がある。
Okta の[証明書更新手順](https://help.okta.com/oie/en-us/content/topics/apps/office365-deployment/rotate-signing-certificate-manual.htm)も参考に、事前登録、確認、切り替え、実 SSO の確認をつなぐ。

## 対象範囲

- Entra の詳細から共有 XML 鍵の管理へ移動し、用途、スコープ、Active/Prepared/Verifying の証明書とフィンガープリントを表示する。
- prepare、現在と次の証明書の再取得、対象全ドメインの Graph コマンドと登録値の確認、確認付き activate を操作できるようにする。
- 同じ XML 鍵を使う SAML の信頼先を含む影響範囲を表示し、Entra の確認だけを共有鍵全体の信頼更新の完了としない。
- 中断後の再開、準備済み鍵の取り消し、切り替え後の署名と実 SSO の確認、失敗時の復旧手順を提供する。

## 対象外

- 暗号アルゴリズムや既存の二段階 API の再設計、JWT 鍵の変更。
- 外部への証明書の自動配布、確認を省いた即時切り替え、旧鍵を再有効化する新しい API。

## 設計

鍵の状態と署名資格情報は SigningKeys が所有し、Entra 登録値は WsFederation が所有する既存の分担を維持する。
新しいモジュール間依存を加えず、画面が既存 API と導入確認 API の操作を組み合わせる。
prepare では現在の署名鍵を変えず、再訪問時も同じ Prepared の kid を表示する。
activate は現在の Prepared の kid と、信頼更新の明示的な確認を送る。
確認対象のドメイン、鍵、設定の版が変われば確認をやり直す。
Entra の照合には [[wi-96973-guide-entra-federation-setup-and-verify-applied-settings]] の手動観測を使い、同じ鍵の他の信頼先は個別の確認を要求する。

仕様に定める旧証明書の七日間の公開と、旧鍵での新規署名への復帰は区別する。
現在の API が保証しない旧鍵への巻き戻しを画面で約束せず、切り替え前の中止と、切り替え後の外部の信頼設定の修復を分けて手順にする。
秘密鍵は表示、ダウンロード、観測記録に含めない。
通常用途の「回転」に XML 鍵を混ぜる方式は、信頼更新前の切り替えを誘うため採らない。

## 計画

1. REQ-SIGNINGKEYS-006 と REQ-WSFEDERATION-001 に管理者の確認と中断後の再開を追加し、必要な DTO の変更を TypeSpec に定める。
2. 既存 API が返す用途とスコープ、Prepared の参照、共有する信頼先の列挙を確認し、足りない読み取りだけを補う。
3. 鍵の詳細と更新画面、全ドメインへの登録手順、切り替え後の確認と復旧 runbook を実装する。

## タスク

- [ ] T001 [Spec] 管理画面の更新操作、影響範囲、再確認と復旧の限界を定める。
- [ ] T002 [Acceptance] prepare 後も現在の鍵で発行し、確認後だけ新しい鍵で発行する試験を先に置く。
- [ ] T003 [UI] 二段階の更新、再取得、中止、確認と切り替えを実装する。
- [ ] T004 [App] 既存 API で不足する参照と確認の契約だけを追加する。
- [ ] T005 [Verify] 中断、二重送信、古い kid、未反映のドメインと共有する SAML 信頼先を検証する。

## 検証

- `mise run test-go-package -- ./backend/signingkeys/...` と `mise run test-go-package -- ./backend/wsfederation/...`：鍵を実際に変えた前後の Assertion とメタデータを検証し、準備だけで署名鍵が変わらないことを確かめる。
- `mise run test-ui-e2e-file -- <追加する XML 鍵更新の spec>`：画面から準備し、再訪問、対象全体の確認、activate、公開証明書の再取得まで確かめる。
- `mise run verify-changed`：実装の最終差分を検証する。契約を変更した場合は `check-contract-drift` と `check-published-api-compat` を加える。
- WI-79 の実テナントで、登録値を読み戻し、鍵切り替え後の新規サインインと既存の信頼先への影響を確認する。

## リスク

共有鍵の切り替えは Entra 以外の SAML 信頼先にも影響する。
対象の取りこぼしと、準備した鍵の取り違えを独立に検査する。
外部へ信頼させる証明書を変えるため不可逆とし、復旧は実際に提供する API と外部操作の範囲で記述する。
