---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-26
depends_on: []
change_kind: feature
initial_context:
  scl:
    SigningKeys: [models.KeyUsage, models.SigningKey, states.SigningKeyLifecycle]
    Saml: [interfaces.PublishSamlMetadata, interfaces.SamlSingleSignOn]
    WsFederation: [interfaces.PublishWsFederationMetadata]
  source:
    - backend/signingkeys
    - backend/saml
    - backend/wsfederation
    - backend/cmd/internal/bootstrap
  tests:
    - backend/signingkeys
    - backend/saml
    - backend/wsfederation
  stop_before_reading: [frontend]
affected_spec:
  - { context: SigningKeys, kind: model, element: KeyUsage }
  - { context: SigningKeys, kind: model, element: SigningKey }
  - { context: SigningKeys, kind: state, element: SigningKeyLifecycle }
  - { context: Saml, kind: interface, element: PublishSamlMetadata }
  - { context: Saml, kind: interface, element: SamlSingleSignOn }
  - { context: WsFederation, kind: interface, element: PublishWsFederationMetadata }
---

# SAML / WS-Fed の XML 署名資格情報をテナント別に永続化する

## Motivation

現在の FederationSigner はプロセス起動時に生成される単一の開発用自己署名証明書であり、
全テナントで共有され、再起動のたびに SP / RP の trust が壊れる。管理画面で正式な
metadata・証明書として案内する前に、XML federation の署名資格情報を tenant-aware な
永続 lifecycle に載せる必要がある。

## Scope

- `SigningKeys` に XML federation 用 key usage と X.509 certificate を追加する。
- memory / PostgreSQL / Vault の既存 KeyStore を usage-aware にし、tenant + usage ごとに
  active 鍵を1本持つ。
- SAML / WS-Fed / WS-Trust の署名と metadata を request tenant の XML federation 鍵へ切り替える。
- rotation overlap 中は metadata に active + verifying certificate を掲載する。
- signingkeys / saml / wsfederation の architecture ledger と設計正本を同期する。

## Out of Scope

- SP ごとの SAML IdP profile。
- BYOK、外部 CA 発行証明書、証明書 chain 管理。
- XML metadata 文書自体への署名。

## Plan

- KeyStore interface は維持し、XML federation の呼び出し元は context に
  `KeyUsage` を明示して、JWT と XML の lifecycle を分離する。
- XML用途の鍵生成時に同じ private key に対応する自己署名 X.509 certificate を作成し、
  credential row と同じ lifecycle で保存する。
- XML signer は `crypto.Signer` を受け取り、Vault Transit では署名方式を signer options
  に合わせて PS256 / PKCS#1 v1.5 から選ぶ。
- 新規署名は active のみ、metadata は期限内の active + verifying を使う。

## Tasks

- [x] T001 [SCL] key usage、certificate、tenant isolation、restart continuity、rotation overlap を仕様化して再生成する。
- [x] T002 [Domain] RED: usage context と XML credential validation の domain test を先に fail 確認（scenario `XML federation署名資格情報はテナントと用途で分離される`）→ GREEN。
- [x] T003 [Adapters] RED: memory/PostgreSQL/Vault の tenant+usage、certificate永続化、署名方式 test を先に fail 確認（同 scenario）→ GREEN。
- [x] T004 [SAML/WS-Fed] RED: tenant別署名とmetadata複数証明書 test を先に fail 確認（scenario `XML federation鍵の回転中も既存trustを検証できる`）→ GREEN。
- [x] T005 [Infrastructure] 起動時の process-global dev signer を除去し KeyStore resolver を配線する。
- [x] T006 [Architecture/Verify] 設計記録を同期し、全検証を通す。

## Verification

- `just check-scl`
- `just scl-render`
- `just test-go`
- `just verify-go`
- `just check`

## Risk Notes

XML署名方式と証明書の不一致は全SAML/WS-* trustを破壊する。既存 JWT usage を既定値として
維持し、XML用途を明示した context だけを新しい鍵集合へ向ける。外部入力parserは増やさないため
新規 fuzz test は不要と判断する。

## Completion

- **Completed At**: 2026-07-26
- **Summary**: Added tenant- and usage-scoped durable XML federation signing credentials, certificate publication during rotation overlap, and request-scoped signing across SAML, WS-Fed, and WS-Trust.
- **Verification Results**:
  - `just check-scl` - passed
  - `just scl-render` - passed
  - `just test-go` - passed
  - `just verify-go` - passed
- **Evidence**:
  - Domain and adapter tests cover separate JWT/XML lifecycles, tenant isolation, certificate persistence, and stable Vault-backed certificates across store recreation.
  - SAML and WS-Fed metadata tests cover publication of active and verifying certificates.
  - The process-global development signer and eager default-tenant key generation were removed; credentials are generated lazily for the request tenant and usage.
