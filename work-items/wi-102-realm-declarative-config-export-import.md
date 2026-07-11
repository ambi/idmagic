---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-04
---

# テナント（Realm）設定の宣言的 export/import を提供し GitOps・環境昇格・DR を可能にする

## Motivation
現状テナントとその配下設定（client / application / assignment / claim release /
federation trust / signing key policy / user attribute schema 等）は管理 API と
UI からしか構成できず、デモデータは Go の seed コードにハードコードされている。
本番運用では stg で作った構成を prod へ「同一に」昇格し、構成をコードとして
レビュー・バージョン管理し、DR 時に再現する手段が要る。手作業の再構成は
ドリフトと事故の温床になる。

Keycloak は realm を JSON で export/import でき、これが環境昇格・GitOps・
バックアップの基盤になっている。idmagic も tenant を単位とする宣言的な
設定ドキュメント（機密を除いた構成）を export し、冪等に import（差分適用）
できるべきである。これは WI-101 のデータバックアップとは別で、「構成 as code」を
対象とし、パスワードハッシュや秘密鍵素材などの機密は含めない。

## Scope
- **decision**:
  - 新規 ADR: export/import の対象範囲（構成のみか一部データも含むか）、機密除外方針 （秘密鍵・パスワード・client secret は含めず参照のみ）、import の冪等性と衝突解決 （create / update / fail-on-drift）を定義する。
- **scl**:
  - Tenancy context に ExportTenantConfig / ImportTenantConfig の objective を追加する。
  - export ドキュメントの版・スキーマ・含む集約の範囲を定義する。
- **go**:
  - tenant 単位で構成を集約し、安定した宣言的ドキュメント（JSON/YAML）へ直列化する export use case を追加する。列挙順を安定化し diff 可能にする。
  - 同ドキュメントを冪等に適用する import use case を追加する。既存との差分適用、 dry-run（適用せず差分表示）、参照整合（存在しない鍵/SP を指す割当を fail-closed で拒否）を持つ。
  - 機密フィールドは export に含めず、import 時は別経路（secret / KeyStore）参照で解決する。
  - 管理 API に export/import エンドポイントを追加し、既存 admin RBAC 下に置く。
- **documentation**:
  - README に環境昇格・GitOps・DR 再現での使い方と、機密が含まれない前提を書く。

## Out of Scope
- user 本体・認証イベント・トークンなど運用データの移送（WI-101 のバックアップが扱う）。
- 秘密鍵・パスワード・client secret 平文の export。
- 双方向リアルタイム同期。

## Plan
- [[ADR-032-tenant-as-first-class-aggregate]] と [[ADR-064-protocol-contexts-and-application-catalog]] の所有境界を保ち、Tenancyが全設定を直接更新せず、context別exporter/importer portをorchestrateする。manifestはtop-level realm identity/versionとcontext sectionから成る。
- export対象をtenant settings、attribute schema、groups/roles、Applications+protocol bindings/policies、upstream/provisioning metadata等のScope記載集合に固定する。users、consents、sessions、audit/events、secret/private key、generated IDsは除外またはlogical referenceにする。
- manifest JSON Schema/versionを公開し、stable logical keyでsection間参照を解決する。secretは`${secretRef}`等のopaque referenceだけを許し、export/import response/logに値を出さない。
- importはparse/schema validate→reference graph/cycle→current snapshotとのredacted semantic diff→plan→explicit applyの二段階にする。context順序はtenant→schema→groups/roles→applications→bindings/policiesで、各commandは既存use case/permission/eventを通す。
- applyはmanifest digest+operation keyで再実行可能にし、全realm巨大transactionではなくdependency stageごとにcheckpointする。失敗後のresume/compensateと削除policy（default preserve、explicit prune）を明示する。

## Tasks
- [ ] T001 [Inventory/ADR] context別export対象/除外/secretRef/logical key、merge/prune、transaction/checkpoint semanticsを決定する。
- [ ] T002 [SCL/Schema] RealmManifest/Plan/Operation/Result、Export/PlanImport/Apply interfaces/events/invariantsを追加しJSON Schemaと派生物を生成する。
- [ ] T003 [Context Ports] tenancy/identity/application/oauth2/saml/wsfederation等にcanonical export DTOとvalidate/apply command adapterを追加する。
- [ ] T004 [Planner] version migration、reference graph、redacted semantic diff、dependency order、manifest digest/idempotencyを実装する。
- [ ] T005 [Executor] stage checkpoint/resume、existing usecase経由apply、explicit prune、failure result/compensationを実装する。
- [ ] T006 [HTTP/CLI] streamed export、plan upload、review/apply/statusとGitOps向けnoninteractive commandをjust recipe経由で追加する。
- [ ] T007 [Verify] export→empty realm import→re-export、version upgrade、rename/reference、partial resume、prune、cross-tenant/secret injection、redactionを検証する。

## Verification
- `just test-go-race`
- `just lint-go`
- 手動: あるテナントを export → 空テナントへ import し、client/application/割当/claim policy が 再現されることを確認する。再 import しても差分ゼロ（冪等）であることを確認する。
- 手動: export に秘密鍵・パスワードハッシュ・client secret 平文が含まれないことを確認する。

## Risk Notes
export に機密が混ざると重大な漏えいになる。まず機密除外を型で強制し、
import の参照整合を fail-closed にしてから範囲を広げる。冪等性が崩れると
昇格のたびにドリフトするため、dry-run 差分を先に信頼できる形にする。
