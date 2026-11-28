---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: federation-protocols
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-001 }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderConnectionRequest }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Contract.CreateIdentityProviderConnection }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Contract.UpdateIdentityProviderConnection }
---

# 上流の SAML IdP のメタデータから写した署名の証明書を、保存の時点で使える形に揃える

## 動機

SAML の上流の IdP の接続では、管理者が IdP のメタデータから SSO の URL と署名の証明書を写して保存する（OIDC の Discovery にあたる入力支援は SAML にない）。
メタデータの `<ds:X509Certificate>` は、XML Signature の規定どおり、PEM の見出しのない DER の base64 である。

2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、Microsoft Entra ID の公開の連携メタデータ（`https://login.microsoftonline.com/common/FederationMetadata/2007-06/FederationMetadata.xml`）を記録し、IDPSSODescriptor の五つの署名の証明書を写したままの文字列で確かめた。
記録は `backend/authentication/internal/federation/protocol_saml/testdata/entra-federation-metadata.json` である。

- `ValidateSigningCertificates` は五つとも「could not be parsed」を返す。同じ値を PEM の見出しで包むと、五つとも使える証明書になる。
- 接続の保存の検査（`IdentityProviderConnection.Validate`）は証明書が一つ以上あることだけを見るので、写したままの値でも保存と有効化が通る。
- 上流の応答の署名の検証（`validateSignature`）も PEM だけを読むので、その接続のログインはすべて「invalid SAML signing certificate」で失敗する。

管理者は保存の時点では誤りに気付けず、利用者のログインが失敗して初めて分かる。
規範の要件はこの形を述べていないが、偶然の振る舞いとして (c) 直すに分類した。
保存できた接続でログインがすべて失敗するので、グループの基準の p3 ではなく p1 とする。

## 対象範囲

- 次のどちらかを着手時に決めて仕様に書き、保存と検証を同じ規則で揃える。
  - DER の base64（空白と改行を含む）も証明書として受け付け、保存の時点で正規の PEM に揃える。
  - 保存の時点で証明書を解析し、使えない値を 400 で拒否する。
- `IdentityProviderConnectionRequest` の `saml_signing_certificates` の説明に、受け付ける形を書く。
- [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の `TestSigningCertificates_FromEntraMetadataAreUsableAsPEM` に、写したままの値で保存からログインの署名の検証までを確かめる検査を足す。

## 対象外

- SAML のメタデータの URL からの取り込みと定期の再取得。新しい入力支援を足す別の判断である。

## 設計

利用者が貼る値の形を最も多く受け付けるのは前者である。
後者は変換を持たないが、メタデータから写す管理者に毎回 PEM への書き換えを求める。
SP の登録の AuthnRequest の署名の証明書（`ParseCertificatePEM`）も PEM だけを読むので、同じ規則に揃えるかを着手時に決める。

## 計画

1. `spec-change` で受け付ける形を書く。
2. 記録の値で RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] 受け付ける証明書の形を仕様に書く。
- [ ] T002 [Acceptance] Entra ID のメタデータから写した値で RED を確認する。
- [ ] T003 [App] 保存と検証の規則を揃える。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/authentication/internal/federation/protocol_saml`、`mise run test-go-package -- ./backend/authentication/internal/federation/domain`、`mise run test-go-package -- ./backend/authentication/internal/federation/handlers_http`。

## リスク

変換を誤ると、別の証明書を信頼の根拠として保存する。
変換の前後で DER のバイト列が一致することを検査で固定する。
