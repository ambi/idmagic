---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-14
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/domain/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-004 }
---

# realm 配下の branding アセット転送例を現行 gateway と一致させる

## Motivation

`EX-TENANCY-004-03` は、realm 配下の `logo_url` を gateway が backend へ転送せず、画像取得が失敗すると記述する。

現在の `frontend/vite.config.ts` は realm 配下の転送対象へ `tenant-branding-assets` を含める。
サーバー側にも同じ経路が登録されているため、具体例が記録する失敗と現行の経路構成が一致しない。

## Scope

- `EX-TENANCY-004-03` が要求する最終状態を、現行の成功経路と過去の回帰事例のどちらとして扱うか決める。
- 現行の成功経路を要求する場合は、具体例を成功条件へ書き換え、gateway と backend の双方を通るテストを対応付ける。
- 失敗事例を履歴として残す場合は、規範の具体例から外し、適切な変更記録へ移す。

## Out of Scope

- branding アセットの URL 形式または保存形式の変更。
- 他の gateway 転送対象の棚卸し。
- 越境したアセット取得のエラー契約。`EX-TENANCY-004-02` は [[wi-578-align-cross-tenant-branding-asset-refusal]] が扱う。

## Design

規範シナリオは必要な製品の振る舞いを記述するため、解消済みの失敗を `Then` に残すと、正常な実装を不適合として扱う。

採用候補は、具体例を「realm 配下の `logo_url` が gateway から backend へ転送され、画像取得に成功する」へ直す案である。
過去の失敗を残す必要がある場合は、work item の履歴または回帰テストの説明に置く。

## Plan

1. gateway と backend の現行経路を正式な入口から観測する。
2. `EX-TENANCY-004-03` の規範上の役割を決める。
3. 具体例を決定済みの成功条件へそろえ、gateway の回帰テストを追加する。
4. UI が返された `logo_url` を表示へ使う経路も確認する。

## Tasks

- [ ] T001 [Decision] `EX-TENANCY-004-03` を規範の成功例として直すか、履歴へ移すか決める。
- [ ] T002 [Spec] Tenancy の具体例を決定へそろえる。
- [ ] T003 [Acceptance] gateway と backend を通る realm 配下の画像取得を検証する。
- [ ] T004 [UI] 返された `logo_url` が画像表示へ渡ることを検証する。
- [ ] T005 [Verify] 仕様、UI、ブラウザー経路の検証を通す。

## Verification

- `mise run check-spec`
- `mise run test-ui-unit-file -- ./src/components/Brand.test.tsx`
- `mise run test-ui-e2e`
- `mise run verify`

## Risk Notes

設定 API の成功だけを見ても、gateway が画像 URL を転送できることは証明できない。
受け入れテストは画像 URL への HTTP 要求まで進め、画像本文を観測する。
