---
status: pending
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
- [ ] T001 [SCL] context_map と `contexts/scim.yaml` を確定名 / 構造へ rename し、canonical ref・
      publishes を更新する。`just yaml-check` を通す。
- [ ] T002 [Go] `backend/scim/` を `git mv` で再配置し、import path 一括置換・named import 修正。
- [ ] T003 [Architecture] `ARCHITECTURE.md` を同期し、派生生成物を再生成する。
- [ ] T004 [Verify] 下記 Verification を緑にする。

## Verification
- `just verify-go` / `just build-go` / `just test-go` — 新 import path 解決とテスト緑。
- `just yaml-check` / `just check-ids` — context_map・canonical ref・Architecture 整合。
- `git log --follow` で `git mv` の履歴保持、旧配置 / 旧 context 名への参照残存ゼロを grep で確認。

## Risk Notes
稼働中 context の rename は canonical ref namespace の変更を伴い、SCL 参照・backend import・生成物へ
広く波及する。振る舞いは不変だが、`ScimUserRef` / `ScimGroupRef` の rename 判断と派生生成物の
同期を検証ゲートで担保する。パス移動は並行ブランチと衝突しやすいので、並行 work-item が少ない
タイミングで実施する。
