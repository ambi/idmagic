---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: bugfix
initial_context:
  scl:
    Saml:
      - standards.SAML2Core.SAML2Core-BearerAssertion
      - standards.SAML2Bindings.SAML2Bindings-RedirectPost
      - standards.SAML2WebBrowserSSO.SAML2Profile-WebBrowserSSO
      - models.SamlAuthnRequest
      - interfaces.SamlSingleSignOn
      - scenarios.SAML SP initiated SSO succeeds
      - scenarios.SAML rejects unregistered or mismatched request
  source:
    - backend/saml/domain/authnrequest.go
    - backend/saml/usecases/signin.go
    - backend/saml/adapters/http/sso_handler.go
    - backend/saml/ports
    - backend/saml/adapters/persistence/memory
    - backend/shared/adapters/persistence/valkey
  tests:
    - backend/saml/domain/authnrequest_test.go
    - backend/saml/adapters/http/saml_handler_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Saml, kind: standard_requirement, standard: SAML2Core, requirement: SAML2Core-BearerAssertion }
  - { context: Saml, kind: standard_requirement, standard: SAML2Bindings, requirement: SAML2Bindings-RedirectPost }
  - { context: Saml, kind: standard_requirement, standard: SAML2WebBrowserSSO, requirement: SAML2Profile-WebBrowserSSO }
  - { context: Saml, kind: model, element: SamlAuthnRequest }
  - { context: Saml, kind: interface, element: SamlSingleSignOn }
  - { context: Saml, kind: scenario, element: SAML SP initiated SSO succeeds }
  - { context: Saml, kind: scenario, element: SAML rejects unregistered or mismatched request }
---

# SAML IdP が AuthnRequest の意味を fail-closed に検証し重複発行を防止する

## Motivation

現在の SAML IdP は Issuer、ACS URL、Destination、署名方針、ForceAuthn を検証する一方、
`AuthnRequest` の `Version` と `IssueInstant` を解析・検証せず、`IsPassive`、
`ProtocolBinding`、`AssertionConsumerServiceIndex` を未解釈のまま既定の HTTP-POST / ACS へ
フォールバックする。未対応の NameIDPolicy format も許可され得る。さらに同じ SP-issued
request ID を記録する store がなく、同一の有効 AuthnRequest を繰り返すと assertion を繰り返し
発行できる。

これは「未対応要求を成功扱いしない」SAML Web Browser SSO の相互運用・安全性に反し、完了済み
wi-29 が residual risk として残した IdP 側 AuthnRequest replay 防御も未着手である。安全な既定値
への黙った変換をやめ、SP が意図した binding / interaction 制約だけを満たす応答を発行する必要がある。

## Scope

- `spec/contexts/saml.yaml` の `SamlAuthnRequest`、`SamlSingleSignOn`、SAML Core / Bindings /
  Web Browser SSO requirement と scenario に、受理する request version、IssueInstant の許容窓、
  passive request、response binding、ACS URL/index、NameID format、replay の契約を明文化する。
- AuthnRequest parser / validator が `Version="2.0"`、必須 `IssueInstant`、clock skew と最大年齢、
  `IsPassive`、`ProtocolBinding`、ACS URL/index の排他・対応可否、NameIDPolicy format を解析し、
  未対応または矛盾する要求を fail-closed に判定する。
- IdP が対応する response binding を HTTP-POST に固定して明示し、Redirect response や未解決 ACS
  index を要求されたときに黙って既定 ACS へ送らない。`IsPassive=true` で既存セッションが使えない
  ときはログイン画面へ遷移せず、検証済み ACS に SAML protocol の NoPassive 応答を返す。
- tenant / SP / AuthnRequest ID をキーに原子的に一度だけ確保する TTL replay store を port として
  追加し、memory と durable deployment 用 adapter に実装する。ログイン往復中は消費せず、発行直前に
  競合なく確保して、同じ request による assertion の二重発行を拒否する。
- domain、usecase、HTTP contract test で、古い / 未来の IssueInstant、version 不一致、passive 未認証、
  unsupported binding / ACS index / NameID、並行・逐次 replay、tenant 分離、正常な ForceAuthn / resume
  を固定する。

## Out of Scope

- SAML ECP、encrypted assertion、SAML SP として外部 IdP を消費する inbound federation
  （[[wi-30-inbound-federation-and-identity-broker]]）。
- XML parser の fuzzing と corpus 運用（[[wi-105-security-critical-parser-fuzzing]]）。
- SP metadata import の全 ACS binding/index model。初期実装では HTTP-POST response を明示的に
  サポートし、解決不能な index は拒否する。

## Plan

- SCL を先に更新し、現在の「省略時は先頭 ACS」という便利な既定を、属性が未指定の場合だけに限定する。
  明示されたが未対応の要求は protocol-level failure とする。
- request は parser で構造を取り出し、SP 解決後の semantic validator で登録設定・現在時刻・IdP
  capability と照合する。署名検証より前に request ID を replay store へ書き込まない。
- replay reservation は assertion 発行直前の usecase に置き、NeedLogin の resume を妨げない。
  atomic `RecordIfNew` と短い TTL を使い、失敗時も二重発行にならない outcome を定める。
- NoPassive / RequestUnsupported 等は、返信可能な ACS が確定している場合だけ署名済み SAML error
  response として返し、それ以外は IdP 側の generic rejection に閉じる。

## Tasks

- [ ] T001 [SCL] AuthnRequest の時間・passive・binding・ACS・NameID・replay 要件、失敗 scenario、SAML protocol error outcome を追加して派生 artifact を再生成する。
- [ ] T002 [Domain] RED: Version / IssueInstant / skew、IsPassive、ProtocolBinding、ACS URL/index、NameID format の table-driven test を先に失敗させ、parser と semantic validator を実装して GREEN にする。
- [ ] T003 [Port/Adapter] RED: tenant / SP / request ID の同時 `RecordIfNew`、TTL、tenant isolation を memory と durable adapter で先に失敗させ、replay store を実装して GREEN にする。
- [ ] T004 [Usecase/HTTP] RED: passive 未認証時の NoPassive、unsupported request の fail-closed、正常なログイン resume、逐次・並行 replay の HTTP contract test を先に失敗させ、発行判断と error response を実装して GREEN にする。
- [ ] T005 [Verify] SAML unit/HTTP、全 Go verification、SCL/work-item validation を実行し、署名必須 SP と署名不要 SP の両方を回帰確認する。

## Verification

- `just yaml-check`
- `just scl-render`
- `just test-go`
- `just verify-go`
- 手動: 同一の署名済み AuthnRequest を同一 tenant で二度送信し、最初だけ assertion が発行され、二度目は発行されないことを確認する。
- 手動: `IsPassive=true` の未認証 request が login redirect ではなく NoPassive response となり、未対応
  `ProtocolBinding` / ACS index / NameID format が既定 ACS への成功応答にならないことを確認する。

## Risk Notes

AuthnRequest validation を厳格化すると、従来暗黙に受理していた SP 設定が失敗へ変わり得る。しかし
未対応の要求を成功扱いして別の ACS/binding へ送る方が、同期失敗や意図しない認証応答のリスクが高い。
対応 capability を metadata / SCL で明示し、clock skew、resume、並行要求を contract test で固定して
互換性を管理する。replay store の可用性障害は fail-closed とし、TTL と容量を監視可能にする。
