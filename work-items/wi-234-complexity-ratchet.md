---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-232-executable-architecture]
---

# UI container と Go source の複雑性を分割し再増加を ratchet で防ぐ

## Motivation
Frontend には 2,677 行・68 local state の Applications page、2,457 行・38 local state の Users page などがあり、文書化済みの thin container / presentation split を満たしていない。Backend にも 800 行を超える非生成 handler/usecase がある。方針を文章だけで運用すると再肥大化するため、意味単位の分割と機械的な上限が必要である。

## Scope
- Applications、Users、Groups、Agents、Settings の page module を route/resource operation 単位へ分割する。
- container、presentation、form validation、API orchestration、i18n の責務を分離する。
- SCIM usecase、OAuth2 authorize handler、IdentityManagement admin usecase など巨大 Go source を interface/aggregate operation 単位へ分割する。
- Architecture check に UI page/container 400行以下、local state hook 10個以下、非生成 Go source 新規800行超禁止の budget を追加する。
- 分割した presentation/helper/usecase の characterization/unit test を追加する。

## Out of Scope
- UI デザイン刷新。
- API wire contract や認可ルールの変更。
- 行数だけを減らすための機械的ファイル分割。
- generated route/sqlc code への budget 適用。

## Plan
- 最初に現在の挙動と route/API interaction を test で固定し、resource detail/create/edit/list の意味境界で分割する。
- shared mega-hook や巨大 props object へ複雑性を移さず、section-local container と純粋 presentation を使う。
- budget は後続の新規違反を即時 error とし、既存対象は本 WI の task と紐づく期限付き debt として順次解消する。
- Go は package/API を維持しながら operation 別 source へ分割し、循環依存や新しい shared bucket を作らない。

## Tasks
- [ ] T001 [Baseline] 対象 file の責務、route、state、test coverage を inventory 化する。
- [ ] T002 [UI] Applications と Users を route/resource operation 単位へ分割する。
- [ ] T003 [UI] Groups、Agents、Settings を同じ規約へ分割する。
- [ ] T004 [Go] 800行超の非生成 handler/usecase を意味単位へ分割する。
- [ ] T005 [Tests] 抽出した presentation/helper/usecase の test を追加する。
- [ ] T006 [Architecture] complexity budget と generated exclusion を有効化する。
- [ ] T007 [Verify] wire/route 挙動不変、coverage 非低下、全検証を確認する。

## Verification
- `just test-ui-unit`
- `just verify-ui`
- `just test-go`
- `just verify-go`
- `just yaml-check-architecture`
- `just verify`

## Risk Notes
大規模な挙動不変 refactor で merge conflict と回帰が起きやすい。resource 群ごとに小さく完了させ、既存 lifecycle workflow 変更と重なるファイルはその変更の統合後に扱う。
