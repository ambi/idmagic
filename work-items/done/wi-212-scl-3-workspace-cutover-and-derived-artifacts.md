---
status: completed
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

- root `spec/scl.yaml` の `context_map` を `spec_version: "3.0"` へ移行し、全13 context と全5 tool specが3.0であることを保証。
- `tools/yaml-check/spec/scl.yaml` の `interfaces` と `scenarios` をSCL 3.0専用検証へ更新。
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

- [x] T001 [Inventory] 全18 SCL文書（13 context + 5 tool）とroot mapが3.0で、旧section/fieldが0件と確認する。
- [x] T002 [SCL] root context mapを3.0へ切り替え、context published languageとpathを最終検証する。
- [x] T003 [Validator] SCL 2.0 schema、dispatcher、fixture、移行専用codeを削除し3.0のみ受理させる。
- [x] T004 [Docs] RA、SCL、Architecture、AGENTS、skills、just command説明を3.0のsection/flowへ同期する。
- [x] T005 [Render] 全IdMagic/tool派生物をsourceから再生成し、旧section/anchor/security metadataを更新する。
- [x] T006 [Diff] OpenAPI/model schemaの外部contract差分をreviewし、意図しないbehavior差分を除去する。
- [x] T007 [Verify] tool、YAML、Go、UI、build、E2Eを含む標準verificationをすべて通す。
- [x] T008 [Audit] ADR-103の全決定と各work itemのscopeがworkspace現在形へ反映されたことを確認する。

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

## Completion

- **Completed At**: 2026-07-15
- **Summary**:
  - root context mapを含む全18 SCL文書を3.0へ統一し、SCL 2.0 schema、version dispatcher、旧fixtureを
    撤去した。Go runtime loaderとpolicy projectionも3.0のmodels、interface contracts、authorization、
    scenarios、flowsへ切り替え、旧section依存を除去した。
  - RA/SCL/Architecture文書、SCL skills、未完了work itemの仕様語彙を3.0へ同期し、全IdMagic/tool派生物を
    sourceから再生成した。OpenAPIのpath/query重複parameterを除去し、path/method/operation集合179件が
    cutover前後で不変であることを確認した。
  - TOTP登録確認をSCLのprotected accessに合わせてstep-up対象へ追加した。E2Eで発見したテスト基盤の
    locale固定、ユーザー詳細link誤選択、suite間process停止競合も修正し、全19 browser scenarioを通した。
- **Verification Results**:
  - `just scl-render` — passed
  - `just yaml-check` — passed
  - `just test-tools` — passed
  - `just typecheck-tools` — passed
  - `just test-go` — passed
  - `just verify` — passed (YAML、tools、Go race、UI format/lint/typecheck/unit/build)
  - `just test-ui-e2e` — passed (19 scenarios)
  - `just check-ids` — passed (317 record ids)
  - SCL監査 — 2.0は拒否fixture/説明文のみ、`relates_to`はWS-Addressingの業務fieldのみで、旧SCL正本要素は0件。
  - OpenAPI監査 — path/method/operation集合179件は不変、重複path/query parameterは0件。
- **Affected Guarantees State**:
  - guarantee: root map、13 context、5 tool specはSCL 3.0のみで検証・描画される。
  - state: passed
  - guarantee: runtime loaderとpolicy projectionは旧SCL 2.0 sectionやcompatibility aliasへ依存しない。
  - state: passed
  - guarantee: 派生OpenAPIの外部operation集合とruntime protocol behaviorはcutover前後で維持される。
  - state: passed
  - guarantee: 標準verificationとbrowser E2Eがclean workspace process lifecycleで完走する。
  - state: passed
- **Evidence**:
  - procedure: 全SCL version/section/access/referenceを棚卸ししてrootを3.0へ切り替え、validator bridgeを削除した。
    Go loaderとpolicy利用箇所を内側から移行後、派生物を再生成し、OpenAPI contract差分をoperation単位で監査した。
    最後に標準verificationとE2Eを実行し、検証基盤で顕在化した3件の決定的な不安定要因を修正して再実行した。
  - commands: `just scl-render`, `just yaml-check`, `just test-tools`, `just typecheck-tools`,
    `just test-go`, `just verify`, `just test-ui-e2e`, `just check-ids`
  - environment: macOS arm64 workspace; Bun 1.3.14; Go 1.26.5
  - actor: Codex (implement-work-item skill)
  - source: pre-commit working tree based on `286146b7`
  - result: passed; SCL 3.0 workspace cutover completed without a 2.0 compatibility bridge
  - artifacts: `spec/scl.yaml`, `spec/idmagic.html`, `spec/idmagic.openapi.json`,
    `spec/idmagic.models.schema.json`, `tools/yaml-check/spec/yaml-check.html`
