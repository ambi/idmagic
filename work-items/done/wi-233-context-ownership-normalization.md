---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-232-executable-architecture]
---

# ClaimMapping・SigningKeys・domain event の物理所有を SCL context map に一致させる

## Motivation
ClaimMapping と SigningKeys は独立した SCL context だが対応する Go context がなく、型・usecase・adapter が OAuth2、protocol context、`backend/shared` に分散している。また context 固有 event が巨大な `backend/shared/spec/events.go` に集約され、変更時に無関係な context を読む必要がある。公開語彙と技術共有を分離し、SCL から物理構造を再生成できる配置へ揃える。

## Scope
- `spec/scl.yaml` の `context_map.Application.depends_on.ClaimMapping`
- `spec/scl.yaml` の `context_map.ClaimMapping.publishes`
- `spec/scl.yaml` の `context_map.SigningKeys.publishes`
- `spec/scl.yaml` の `context_map.{Saml,WsFederation}.depends_on`
- `backend/signingkeys/{domain,ports,usecases,adapters}` を新設し、鍵 lifecycle、repository port、管理 API/usecase を移す。
- `backend/claimmapping/{domain,ports,usecases}` を新設し、protocol-neutral policy、projection、validation を移す。
- SAML、WS-Fed、OAuth2、Application は context map で宣言した公開面だけに依存する。
- context 固有 event struct を各 owning context へ移し、shared には event envelope と技術的 primitive だけを残す。
- memory/PostgreSQL/Vault/crypto adapter の責務と DI module を新しい ownership に合わせる。
- `ARCHITECTURE.md` と SCL context map の publishes/depends_on を実態に同期する。

## Out of Scope
- claim release の新機能追加。
- 鍵アルゴリズムや rotation policy の変更。
- audit event wire name/payload の破壊的変更。
- PostgreSQL schema の無関係な正規化。

## Plan
- 挙動不変 refactor として、先に公開 interface と contract test を固定してから consumer を段階移行する。
- SigningKeys は lifecycle と key material metadata を所有し、JWT/XML の wire signing は protocol/crypto adapter に残す。
- ClaimMapping は protocol-neutral な issued claim までを所有し、OIDC JSON、SAML Attribute、WS-Fed URI への wire 変換は各 protocol context に残す。
- event consumer は具象 event struct の中央 registry ではなく安定した envelope/type discriminator を境界にする。

## Tasks
- [x] T001 [Contracts] 現行公開型・event wire・repository の characterization test を追加する。
- [x] T002 [SigningKeys] context package と module を新設し consumer を移行する。
- [x] T003 [ClaimMapping] context package と projection contract を新設し consumer を移行する。
- [x] T004 [Events] context 固有 event を owning package へ移し shared registry を縮小する。
- [x] T005 [Adapters] memory/PostgreSQL/Vault/crypto と bootstrap DI を再配線する。
- [x] T006 [Architecture] context map と Architecture realization/dependency を同期する。
- [x] T007 [Verify] wire compatibility、adapter parity、全検証を通す。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-go`
- `just build-go`
- `just verify`

## Risk Notes
認証・署名・監査の中心境界を移す高リスク refactor。ファイル移動と意味変更を混在させず、event type/payload、DB row、JWKS、JWT/XML 出力の characterization test を先に固定する。

## Completion
- **Completed At**: 2026-07-17
- **Summary**:
  ClaimMapping と SigningKeys の domain/usecase/adapter を SCL 上の ownership に対応する context package へ移し、consumer と bootstrap DI を公開面経由に再配線した。context 固有 event は owning domain へ移し、shared には envelope、wire marshal、複数 context が発行する技術的通知 primitive のみを残した。SCL 派生物と `ARCHITECTURE.md` も実装へ同期した。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` - passed
  - `just verify-go` - passed（lint 0 issues、race test を含む）
  - `just build-go` - passed
  - `just verify` - passed
- **Affected Guarantees State**:
  既存の claim release、署名鍵 lifecycle、JWKS、JWT/XML 署名、domain event の type/payload、永続化契約を維持した。新しい利用者向け挙動は追加していない。
- **Evidence**:
  - procedure: `just verify` による SCL、work-item、architecture、traceability、Go、UI の統合検証
    environment: local macOS workspace
    actor: Codex
    source_revision: `23753291a36b741170a673166b584db00b695f12` + WI-233 working tree
    result: passed
    storage: repository tests and generated artifacts
    summary: context ownership の再配置後も全検証と wire compatibility tests が成功
