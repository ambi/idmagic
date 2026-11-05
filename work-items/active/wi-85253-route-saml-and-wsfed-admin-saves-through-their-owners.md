---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: spec/modules/saml/models.tsp, symbol: IdMagic.Contract.SamlServiceProvider }
---

# SAML の SP と WS-Fed の RP の管理 API も、所有者の登録操作を通す

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、[wi-10848](../done/wi-10848-keep-saml-and-wsfed-configuration-updates-with-their-owners.md) が直した不変条件の一部が、管理 API の経路に入っていないことが分かった。

| 経路 | 保存する箇所 | 署名の証明書の確認 |
| --- | --- | --- |
| Catalog の作成と更新 | `ServiceProviderAdministration` | あり（`ErrSigningCertificateRequired`） |
| 管理 API の SP の登録（`admin_service_provider_handler.go`） | ハンドラーが `SamlSPRepo.Save` を直接呼ぶ | なし |

`IdMagic.Contract.SamlServiceProvider` は、`want_authn_requests_signed` が true のとき証明書を必須と宣言する。
管理 API は、証明書がないか読めない PEM の SP を保存し、その SP の SSO は AuthnRequest の検証で失敗する。
WS-Fed の RP も、管理 API と Entra の構成の経路がハンドラーから `WsFedRPRepo.Save` を直接呼び、検証を経路ごとに書いている。
同じ保存型を作る経路が、検証を別々に持つ形が残っている。

## 対象範囲

- 管理 API の SP の登録で、`want_authn_requests_signed` が true なら読める証明書を求める。
- SAML の SP と WS-Fed の RP を保存する全経路を、各モジュールの登録操作へ通す。

## 対象外

- 不変条件を型の構築へ移すこと。[wi-15826](wi-15826-enforce-domain-state-invariants-at-construction-and-update.md) が扱う。

## 設計

### 所有の形

候補 A を採る。
SAML と WS-Fed の管理 API のハンドラーは、`ServiceProviderAdministration` と `RelyingPartyAdministration` の操作を呼ぶ。
リポジトリの `Save` をハンドラーの依存から外し、ユースケースの外から保存できないようにする。
管理 API は既存の SP の上書き（PUT の意味）を持つので、Catalog の作成と更新とは異なる入力を、同じ検証を通る操作として足す。
IdP プロファイルの規則の違反を 400 に写す現在の応答は保つ。

## タスク

- [ ] T001 [Test] 証明書なしの `want_authn_requests_signed` が管理 API で保存されることを RED にする。
- [ ] T002 [App] SAML の管理 API を登録操作へ通す。
- [ ] T003 [App] WS-Fed の管理 API と Entra の経路を登録操作へ通す。

## 検証

- `mise run test-go-package -- ./backend/saml/...`
- `mise run test-go-package -- ./backend/wsfederation/...`

## リスク

- 既に証明書なしで保存した SP は、次の更新で拒否される。移行は対象外とし、拒否の応答で証明書の登録を案内する。
