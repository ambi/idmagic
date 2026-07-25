---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-19
depends_on: [wi-258-inbound-integration-taxonomy]
---

# 既存 `Scim` context を inbound-honest な名前 / 構造へ rename・再配置する

## Motivation

`Scim` context は SCIM **server** (外部 IdP が我々の API を叩く受動 inbound) 専用だが、名前が方向を
叫ばない。ADR-128 §コンテキスト (2) が指摘した通り、outbound provisioning の追加で「scim だけでは
inbound / outbound のどちらか分からない」曖昧さが顕在化した。ADR-128 はこの rename を高リスクな
[[wi-45-outbound-scim-provisioning]] には含めず専用 WI に切り出す方針とし、
[[wi-258-inbound-integration-taxonomy]] が inbound の target 構造を確定する。本 WI はその確定構造へ
`Scim` を physical rename / 再配置する。

## Scope
- `spec/scl.yaml` context_map の `Scim` エントリを [[wi-258-inbound-integration-taxonomy]] 確定の
  名前 / 構造へ rename (単独 context の rename か、統一 inbound context 配下の source feature slice
  への移設かは wi-258 の決定に従う)。
- `spec/contexts/scim.yaml` を新 path へ移し、canonical ref namespace (`Scim/...`) を新名へ更新する。
- published ref `ScimUserRef` / `ScimGroupRef` の rename 要否を判断する (ソース SCL 内で他 context
  から未参照であることは確認済み。参照は自 context の publishes と派生生成物のみ)。
- `backend/scim/` を新配置へ `git mv` し、Go import path を一括置換する。context 横断ハブ
  (`backend/shared/http/server_http/routes.go` の Scim 配線等) の named import を修正する。
- `ARCHITECTURE.md` の context / module 台帳を同期し、派生生成物を再生成する。

## Out of Scope
- SCIM server の振る舞い・endpoint・wire 契約の変更 (純 rename / 配置変更)。
- CSV import の移設 ([[wi-260-relocate-csv-user-import-to-inbound]])。
- active-pull connector ([[wi-95-ldap-ad-user-federation]]、
  [[wi-30-inbound-federation-and-identity-broker]])。
- outbound provisioning ([[wi-45-outbound-scim-provisioning]])。

## Plan
- [[wi-258-inbound-integration-taxonomy]] の確定構造を待ってから着手する (規約が固まる前に動かさない)。
- context rename は canonical ref (`Scim/model/...`, `Scim/interface/...` 等) の namespace 変更を
  伴うため、SCL 内参照・生成物・backend 実装を一括で追随させる。`ScimUserRef` / `ScimGroupRef` は
  他 context 未参照ゆえ cross-context 波及は小さいが、rename する場合は publishes と生成物を同期する。
- backend の物理移動は [[wi-254-backend-feature-vertical-slice-convention]] 系 (wi-255 / wi-256) と
  同じく `git mv` で履歴を保持し、import prefix 一意性と `just build-go` / `just test-go` で網羅検証する。

## Tasks
- [x] T001 [SCL] context_map と `contexts/scim.yaml` を確定名 / 構造へ rename し、canonical ref・
      publishes を更新する → `git mv contexts/scim.yaml contexts/sourcing.yaml` (`context: Sourcing`)、
      wi-258 の scaffold glossary を同ファイルへ統合、context_map の `Scim` エントリを削除して
      `Sourcing` に統合。`just check-scl` 緑。
- [x] T002 [Go] `backend/scim/` を `git mv` で再配置し、import path 一括置換・named import 修正 →
      6 layer dir を `backend/sourcing/scim/` へ、`module.go` を context ルート
      `backend/sourcing/module.go` (`package sourcing`) へ。24 ファイルの import を置換、
      `Module.Repo` → `Module.ScimRepo`、hub 側の `Scim` フィールド → `Sourcing`、`sqlc.yaml` の
      queries / out パスも追随。
- [x] T003 [Architecture] `ARCHITECTURE.md` を同期し、派生生成物を再生成する → contexts 台帳から
      `Scim` を削除、module id を `sourcing-scim-*` / `sourcing-public` / `sourcing-composition` へ
      rename、Context Map の行を統合。`just scl-render` 実行。
- [x] T004 [Verify] 下記 Verification を緑にする。

## Verification
- `just verify` (check / traceability-strict / test-tools / typecheck-tools / lint-go / test-go /
  format-check-ui / lint-ui / test-ui-unit / build-ui) — 全緑。WI 起草時の `just verify-go` /
  `just yaml-check` は現行 justfile に存在しないレシピ名だったため読み替えた。
- `just build-go` / `just test-go-race` / `just lint-go` — 新 import path 解決、race テスト緑、lint 0。
- `just sqlc-generate` — パス変更後に再生成しても生成物に差分が出ないことを確認 (冪等)。
- 契約不変の確認: 再生成した `spec/idmagic.openapi.json` の operationId 247 件と
  `spec/idmagic.models.schema.json` の model 565 件が rename 前後で集合として同一 (差分は context
  順序による並び替えのみ)。
- `git log --follow` で `git mv` の履歴保持、旧配置 / 旧 context 名への参照残存ゼロを grep で確認。

## Risk Notes
稼働中 context の rename は canonical ref namespace の変更を伴い、SCL 参照・backend import・生成物へ
広く波及する。振る舞いは不変だが、`ScimUserRef` / `ScimGroupRef` の rename 判断と派生生成物の
同期を検証ゲートで担保する。パス移動は並行ブランチと衝突しやすいので、並行 work-item が少ない
タイミングで実施する。

## Completion

- **Completed At**: 2026-07-25
- **Summary**: [[ADR-141-inbound-identity-sourcing-taxonomy]] 決定 2・4 の目標構造へ SCIM inbound
  server を移設した。SCL は `spec/contexts/scim.yaml` → `spec/contexts/sourcing.yaml`
  (`context: Sourcing`) へ移し、wi-258 が置いた taxonomy の glossary と統合、context_map の `Scim`
  エントリを削除して `Sourcing` に一本化した。Go は `backend/scim/{domain,ports,usecases,handlers_http,
  db_memory,db_postgres}` を `backend/sourcing/scim/` 配下へ `git mv` し、`module.go` は context ルート
  `backend/sourcing/module.go` (`package sourcing`) に据えた。振る舞い・wire 契約・DB スキーマは不変
  (SCIM endpoint パス、`scim:*` scope、`scim_user_refs` / `scim_group_refs` テーブルは変更なし)。
- **Human Decisions**: なし (構造と命名は wi-258 / ADR-141 で確定済み)。
- **Verification Results**:
  - `just verify` - passed (10 check 並列すべて ok)
  - `just build-go` / `just test-go-race` / `just lint-go` - passed (lint 0 issues)
  - `just sqlc-generate` - passed (パス変更後の再生成で生成物差分なし)
  - `just scl-render` - passed (`spec/idmagic.html` / `.models.schema.json` / `.openapi.json` を再生成。
    差分は context 順序の並び替えのみで、operationId 247 件・model 565 件の集合は不変)
- **Affected Guarantees State**: 外部契約は不変。SCIM の RFC7643 / RFC7644 requirement の
  `adoption` (`RFC7643-CORE-RESOURCES` などの `partial`) は移設前の値をそのまま引き継いでおり、
  本 WI で範囲を狭めていない。
- **Semantic Diff**:
  - `ScimUserRef` / `ScimGroupRef` は **rename しない**と判断した。両者は SCIM id ↔ 内部 principal の
    相関 (= ADR-141 の `SourceCorrelation` の scim 実体) であり、source 非依存の名前にすると
    どの source の相関かが名前から消える。他 context から未参照であることは grep で確認済み。
    ただし context の published language からは外した (`Sourcing.publishes: []`) — 消費者が無く、
    Sourcing の published 語彙は source 非依存に保つ (ADR-141 決定 7、`Provisioning` と同じ扱い)。
  - `Sourcing` の `depends_on` から `Jobs` を外した。wi-258 の scaffold 時点では目標構造の宣言だったが、
    実要素が入った現在 `JobRef` を使う SCL 要素は無い (scim slice は request 単位で適用する)。
    `IngestionRun` を durable job で回す directory source (wi-95) が入る時に再度宣言する。
  - `sourcing.Module` のフィールドを `Repo` → `ScimRepo` に改名した。context ルートの module が
    複数 source slice を束ねる構造 (ADR-141 決定 4) では、どの source の repository か名前で判別できる
    必要がある。hub 側 (`bootstrap.Dependencies` / `server_http.Deps`) のフィールドも `Scim` →
    `Sourcing` に改名した。
  - `Scim` context 名を参照していた work item の `affected_spec` / `initial_context.scl`
    (pending: wi-246〜251、done: wi-238 / wi-239 / wi-244) の context key を `Sourcing` へ更新した
    (traceability の解決に必要)。pending の 6 件は `backend/scim/...` のパス参照も新配置へ更新した。
    done 3 件は本文・パス記述を歴史記録として残し、checker が要求する context key だけを直した。
    ADR 本文の旧パス言及 (ADR-122 / ADR-123 等) は `ADR_FORMAT.md` の「ADR は改変しない」に従い
    そのまま残した。
- **Residual Risk**: 稼働中 context の物理移動なので、並行 work-item ブランチがあるとパス衝突する。
  現時点で並行ブランチは無い。`backend/sourcing` の root は現在 `module.go` のみの thin root で、
  source 非依存コアの抽出は wi-95 に委ねている (ADR-141 決定 3)。
