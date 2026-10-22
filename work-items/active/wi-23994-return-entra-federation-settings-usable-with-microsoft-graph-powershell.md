---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-23
priority: p3
depends_on: [wi-97369-issue-entra-federation-tokens-for-the-realm-and-issuer-entra-expects]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
---

# Entra の構成の結果を、Microsoft Graph PowerShell でそのまま使える形で返す

## 動機

Entra の構成の操作は、管理者が Entra 側に設定する値を `powershell` として返し、管理画面はそれを名前と値の組で表示する。
その値は、今の Entra の設定にそのまま使えない。

| 項目 | Microsoft Graph の設定（`internalDomainFederation`、`New-MgDomainFederationConfiguration`） | 現在の応答 |
| --- | --- | --- |
| 項目の名前 | `IssuerUri`、`PassiveSignInUri`、`ActiveSignInUri`、`MetadataExchangeUri`、`SignOutUri` | 廃止された MSOnline の名前（`PassiveLogOnUri`、`ActiveLogOnUri`） |
| 署名の証明書 | `SigningCertificate` に、証明書の公開部分の base64 | 「メタデータの証明書を使え」という英語の文 |
| 受動的なプロトコル | `PreferredAuthenticationProtocol` を `wsFed` と明示しなければ受動的なフローが動かない | 返さない |
| サインアウト | `SignOutUri` | 返さない。Entra からのサインアウトで idmagic のセッションが残る |
| MFA | `FederatedIdpMfaBehavior` | 返さない |

一次資料：[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)、[SAML 2.0 の IdP の構成例](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/how-to-connect-fed-saml-idp)。

## 対象範囲

- 構成の結果に、Graph の名前の設定値（`SigningCertificate` は現在の有効な証明書の base64、次の鍵があれば `NextSigningCertificate`）を返す。
- 値を埋めた `New-MgDomainFederationConfiguration` と `Update-MgDomainFederationConfiguration` のコマンドを返し、管理画面でコピーできるようにする。
- `SignOutUri` に、`wsignout1.0` を受ける受動的なエンドポイントを渡す。
- `known_limitations` などの英語の文を、`code` と辞書の文言に改める。

## 対象外

- idmagic から Microsoft Graph を呼んで Entra を自動で構成すること。

## 設計

コマンドの組み立ては、構成の値からの決定論的な計算にし、PowerShell の文字列のエスケープを単体テストで確かめる。
`FederatedIdpMfaBehavior` の既定の案内は、wi-84807 の実装の有無で変わるので、wi-84807 より先に実装するときは `acceptIfMfaDoneByFederatedIdp` を案内する。

## タスク

- [ ] T001 [Spec] REQ-WSFEDERATION-001 と `ConfigureEntraFederation` の応答を改める。
- [ ] T002 [Acceptance] 構成の応答の値で組み立てたコマンドが Graph の名前と証明書の base64 を持たないことを RED で確かめる。
- [ ] T003 [App] 応答とコマンドの組み立てを実装する。
- [ ] T004 [UI] コマンドのコピーと、辞書の文言での制限の表示を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run test-ui`、`mise run check`、`mise run check-spec-impact`。
- 実テナントでコマンドが通ることは wi-79 で確かめる。

## リスク

- 証明書の値を誤ると、Entra は署名を検証できずサインインを拒否する。メタデータの証明書と同じ値であることをテストで確かめる。
