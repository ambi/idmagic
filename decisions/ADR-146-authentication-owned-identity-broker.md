---
status: accepted
authors: [tn]
created_at: 2026-07-27
---

# ADR-146: login-time identity broker を Authentication が所有する

## コンテキスト

外部 OIDC / SAML IdP からの対話的ログインは、protocol wire、local User、OAuth
authorization transaction の三つにまたがる。`Saml` / `OAuth2` のいずれかへ broker 全体を
置く案、`Sourcing` に外部 identity correlation を共通化する案、Authentication に
login-time capability として置く案があった。

同時に、自動 account linking は利便性と account takeover risk のトレードオフを持つ。
email 一致を既定にする案、常に明示 link だけにする案、tenant が verified-email policy を
明示した場合だけ自動化する案を比較した。生 client secret の保存には envelope encryption
が必要だが、その基盤はまだ存在しない。

## 決定

Identity provider connection、external subject link、JIT / linking policy、login attempt と
orchestration は `Authentication` が所有する。OIDC RP / SAML SP はその upstream protocol
port を実装し、IdManagement と OAuth2 には published interface で接続する。現在の設計は
[`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md) に置く。

自動 link は tenant が `VerifiedEmail` policy を明示し、upstream が検証済み email を返し、
tenant 内で一意に一致するときだけ許可する。既定は `None` とする。JIT は明示 policy と
claim mapping を要求し、IdManagement が password credential の無い User を作る。

生 secret は保存せず external `secret_reference` だけを永続化する。SAML は既存
`etree` / `goxmldsig` の検証 primitive を再利用し、encrypted assertion と
IdP-initiated unsolicited response は初期範囲に含めない。

## 却下した代替案

- `Saml` または `OAuth2` に broker 全体を置く: もう一方の protocol と link/JIT policy が
  protocol context に従属し、login session ownership と逆向きになる。
- `Sourcing` の correlation link を共有する: inventory synchronization と interactive login
  は lifecycle と consistency requirement が異なり、実装前の抽象化になる。
- email 一致を既定で自動 link する: issuer 境界を越えた account takeover を招く。
- client secret を平文保存する: 未実装の暗号基盤を迂回し、運用上の秘密境界を偽装する。
- SAML framework を新規導入する: 現行の XML trust primitive と二重の署名モデルを持ち込む。

## 影響

- SCL は `Authentication/models.IdentityProviderConnection`、`FederatedIdentity`、
  `StartFederatedLogin` / `CompleteFederatedLogin` / link・admin interfaces と、
  `IdentityManagement/interfaces.ProvisionFederatedUser` を正本とする。
- Authentication に federation feature slice と設計正本を追加し、IdManagement / OAuth2
  との依存を architecture ledger に記録する。
- encrypted assertion、IdP-initiated SAML、生 secret の内蔵保存は対応していないことを
  work item 完了時に開示する。
