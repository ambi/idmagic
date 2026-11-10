---
status: pending
authors: [tn]
risk: critical
created_at: 2026-10-24
priority: p1
depends_on: [wi-69206-run-a-directory-connector-that-reads-ad-and-ldap]
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
---

# AD から取り込んだ User のパスワードの確認を Directory Connector の LDAP の bind に委ね、ログインと WS-Trust の usernamemixed で使えるようにする

## 動機

Directory Connector で取り込んだ User はパスワードを持たない。
AD を使い続ける組織では、利用者は AD のパスワードで IdMagic にサインインしたい。
WS-Trust の能動的な STS（`/trust/usernamemixed`、REQ-WSFEDERATION-004）も UsernameToken のパスワードをログインと同じ照合で確かめるので、取り込んだ User は、MSAL のようなリッチクライアントから Entra のフェデレーションを経て Microsoft 365 などのトークンを得られない。
Entra のドメインフェデレーションで想定する、AD を Entra Connect と IdMagic の両方へ同期する構成では、この経路が使えないことが連携の欠落になる。
パスワードを IdMagic へ写さずにこれを実現するのが、認証の委譲である。

Hybrid Join 後の Windows のパスワードログオンでは、CloudAP が入力された AD 資格情報でフェデレーション IdP から SAML を取得し、Entra に新規 PRT を要求する。
端末が sync join を終えていても、このユーザー認証が通らなければ Windows からの標準的な SSO は完結しない。
ローカルパスワードを追加した User や直接作った認証済みセッションではこの経路を検証できないため、本項目を実テナント検証 [[wi-79-entra-real-tenant-end-to-end-verification]] の前提とする（[Microsoft の PRT 初回発行](https://learn.microsoft.com/en-us/entra/identity/devices/concept-primary-refresh-token#prt-issuance-during-first-sign-in-windows)）。

| 製品 | 機能 |
| --- | --- |
| Okta | AD Agent が delegated authentication を担い、Okta のサインインで受けたパスワードを AD で確かめる |
| Microsoft Entra | Pass-through Authentication の Agent が、外向きの 443 だけでパスワードの確認の要求を待ち受けて答える。受け口のポートを開けない |

Connector は外向きの接続だけを行うので、IdMagic から Connector へ要求を届ける経路が要る。
この作業項目でその経路も作り、後続の書き戻し（[[wi-90960-write-password-changes-back-to-active-directory]]）と AD へのプロビジョニング（[[wi-19734-provision-users-from-idmagic-into-active-directory]]）が同じ経路を使う。

## 対象範囲

- Connector が IdMagic へ長いポーリングで接続し、IdMagic がその接続へ確認の要求を渡し、Connector が結果を返す経路。要求は短い期限と一度だけの ID を持つ。
- 委譲を有効にした取り込み元の User のパスワードのサインインで、Authentication が Connector に確認を求め、結果で成功か失敗を決める。
- パスワードは要求の中で Connector の公開鍵で暗号化し、IdMagic のログ、保存先、イベントに残さない。
- Connector が応答しないときの扱い（失敗にする、ほかの認証方式へ誘導する）を決める。
- AD のアカウントのロック、期限切れ、パスワードの変更の要求を、サインインの結果へ写す。
- WS-Trust の usernamemixed の UsernameToken の照合も同じ委譲を使い、AD の結果を REQ-WSFEDERATION-004 の 401 と 403 の拒否へ写す。Connector が応答しないときの扱いは、ほかの認証方式へ誘導できない WS-Trust でも決める。
- AD 同期ユーザーにローカルパスワードを作らずに、AD の資格情報で認証し、AD 側のパスワード変更後も AD の照合結果に従う。AD が失効させた旧値を IdMagic のローカル照合で受け付けない。
- 管理者が対象の取り込み元の委譲認証を試し、資格情報の拒否、Connector の停止、DC への接続失敗を調べられるようにする。利用者向けの応答の違いでアカウントの存在を漏らさない。

## 対象外

- 複数の Connector の Agent への振り分け。[[wi-86455-run-several-directory-connector-agents-for-one-identity-source]] の後に足す。
- パスワードの書き戻しと AD へのプロビジョニング。
- 実 Windows と Entra での新規 PRT、WAM、ブラウザー SSO の成立確認。WI-79 が担う。

## 設計

### 未解決の問い

- 経路をポーリング、WebSocket、Server-Sent Events のどれにするか。エッジの構成と、閉域のプロキシを通るかで選ぶ。
- 確認の待ち時間の上限と、サインインの性能目標への影響。
- パスワードの暗号化の方式（Connector の鍵での HPKE など）と、Connector の鍵の更新との関係。
- 委譲を有効にした User に IdMagic のパスワードを持たせないことの保証と、移行の手順。

## 計画

1. 経路の方式と脅威モデルを設計に書き、利用者に確認する。
2. Authentication と Sourcing の仕様、TypeSpec、DirectoryConnector の仕様を更新してから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いと脅威モデルを決める。
- [ ] T002 [Spec] 要件と Connector API の経路を書く。
- [ ] T003 [App] 経路、委譲、Connector 側の bind を実装する。
- [ ] T004 [Verify] OpenLDAP の E2E と故障注入で確かめる。
- [ ] T005 [Acceptance] ローカルパスワードを持たない同期ユーザーを正式な入口から認証し、AD 側でのパスワード変更と再試行を確かめる。
- [ ] T006 [UI] 管理者による委譲認証の試験と、秘密情報を含まない障害の診断を提供する。

## 検証

- OpenLDAP のコンテナと正式なルーターの E2E で、正しいパスワード、誤ったパスワード、Connector の停止のそれぞれのサインインの結果を確かめる。
- 同じ三つの場合について、`/trust/usernamemixed` の RST が RSTR、401、Connector の停止時に決めた応答を返すことを確かめる。
- `mise run test-go-openldap-e2e`：取り込みと委譲の設定から開始し、正式なルーターの usernamemixed で認証して署名済み RSTR を検証する。試験用ディレクトリでパスワードを変更し、旧値を無効にした後に旧値を拒否して新値で発行することまで確かめる。
- 診断の管理 API は権限不足と別テナントを拒否し、試験用のパスワードをログ、永続状態、イベントに残さないことを確かめる。実装時に最小の試験境界と mise タスクを決める。

## リスク

- パスワードが IdMagic と Connector の間を流れる。暗号化、期限、一度だけの ID のどれかを誤ると、平文のパスワードを読まれるか再送される。
- Connector が止まると、委譲した User はサインインできない。冗長化（wi-86455）を先に入れるかを着手時に判断する。
