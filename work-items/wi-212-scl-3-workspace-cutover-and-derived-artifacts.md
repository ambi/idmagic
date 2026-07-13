---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: [wi-210-scl-3-application-oauth2-and-signing-migration, wi-211-scl-3-federation-and-provisioning-migration]
---

# IdMagic workspace を SCL 3.0 へ切り替え旧形式を撤去する

## Motivation

SCL 3.0 は互換 alias を持たない破壊的な言語改訂である。tool と全 context の移行後に、root context map、
派生物、RA/Architecture/skills、検証経路を一つの統合地点で同期し、限定的な SCL 2.0 bridge を残さず
workspace 全体を単一 versionへ切り替える必要がある。

## Scope

- root `spec/scl.yaml` を `spec_version: "3.0"` へ移行し、全13 context と全5 tool spec が3.0であることを保証。
- `tools/yaml-check` の SCL 2.0 schema/version dispatcher/fixtureを削除。
- `just scl-render` による `spec/idmagic.html`、`spec/idmagic.openapi.json`、
  `spec/idmagic.models.schema.json` と全 tool派生物の再生成。
- `REGENERATIVE_ARCHITECTURE.md`、`SPECIFICATION_CORE_LANGUAGE.md`、`ARCHITECTURE.md`、AGENTS instructions、
  SCL/implementation/render skills、just recipesの最終同期。
- workspace全体のSCL参照、section名、統計、anchor、ドキュメントlinkの旧形式撤去。

## Out of Scope

- SCL移行と無関係なruntime機能、UI、DB、protocol behaviorの変更。
- SCL 2.0 compatibility mode、converter、deprecated aliasの提供。
- 移行中に発見した既存仕様上の新機能・behavior修正。

## Plan

- [[wi-210]] と [[wi-211]] の両方を統合した地点でのみ実施する。
- 全SCL文書のversion/section/access/referenceを機械inventoryし、2.0残存が0件になってからbridgeを削除する。
- 派生物は部分branchの差分を採用せず、この地点でsourceから全再生成する。
- RA本文、Architecture、skills、AGENTSのsection網羅表をADR-103の語彙へ揃え、今後のagentが旧
  invariant/permission/UX規則を再導入しないようにする。
- full verification後もruntimeの外部contract差分が生成物に現れた場合、構造移行由来かbehavior変更かを
  分類し、後者は本itemで受け入れず別work itemへ分離する。

## Tasks

- [ ] T001 [Inventory] 全18 SCL文書（13 context + 5 tool）とroot mapが3.0で、旧section/fieldが0件と確認する。
- [ ] T002 [SCL] root context mapを3.0へ切り替え、context published languageとpathを最終検証する。
- [ ] T003 [Validator] SCL 2.0 schema、dispatcher、fixture、移行専用codeを削除し3.0のみ受理させる。
- [ ] T004 [Docs] RA、SCL、Architecture、AGENTS、skills、just command説明を3.0のsection/flowへ同期する。
- [ ] T005 [Render] 全IdMagic/tool派生物をsourceから再生成し、旧section/anchor/security metadataを更新する。
- [ ] T006 [Diff] OpenAPI/model schemaの外部contract差分をreviewし、意図しないbehavior差分を除去する。
- [ ] T007 [Verify] tool、YAML、Go、UI、build、E2Eを含む標準verificationをすべて通す。
- [ ] T008 [Audit] ADR-103の全決定と各work itemのscopeがworkspace現在形へ反映されたことを確認する。

## Verification

- `just scl-render`
- `just yaml-check`
- `just test-tools`
- `just typecheck-tools`
- `just verify`
- `just test-ui-e2e`
- `just check-ids`
- `rg -n "spec_version: ['\"]?2\\.0|^invariants:|^permissions:|^user_experience:|relates_to:|protects:" spec tools` の残存を文脈付きでreviewし、SCL正本では0件と確認する。

## Risk Notes

risk は high。全仕様・生成物・開発規則を同時に切り替える統合点である。依存item未完了では開始せず、
生成物は必ずsourceから再作成する。旧validatorを最後に削除し、full verificationをcutover gateとする。
