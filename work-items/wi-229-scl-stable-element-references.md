---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: []
---

# SCL 3.0 の規範要素を検証対象として安定参照可能にする

## Motivation
RA の追跡可能性には、SCL のどの規範要素を実装・テスト・検証したかを機械的に指せる識別子が必要である。一方、独立した `assurance.obligations` に保証文を再記述すると、既存の model constraint、interface contract、state、authorization、scenario と意味が重複し、形式的な coverage と drift を増やす。

SCL 3.0 の「規則は実現・検証する所有要素へ局所化する」という原則を維持したまま、後続の追跡グラフが既存の規範要素を直接参照できる共通の参照モデルを確立する。

## Scope
- `SPECIFICATION_CORE_LANGUAGE.md` に context-qualified な SCL element reference の対象、正規形、解決規則を追加する。
- 参照可能な対象を、standard requirement、model、interface、state、authorization resource/principal/policy、objective、scenario、flow とする。
- `tools/yaml-check` に、workspace の context map を介して参照を解決し、未知 context・section・element・requirement を拒否する再利用可能な resolver と fixture を追加する。
- `tools/scl-to-html` の既存 anchor を同じ参照モデルへ揃え、規範要素から追跡情報へ安定してリンクできるようにする。
- ADR-103 の局所所有を維持し、独立した assurance section を導入しない判断を ADR に記録する。

## Out of Scope
- `assurance.obligations`、保証文、`evidence_kinds` の SCL への追加。
- SCL 3.1 の導入、既存 IdMagic context spec または tool self-spec の version migration。
- 実装ファイル、テスト、`just` recipe、CI artifact への binding。
- 匿名配列要素である個々の `requires` / `ensures` 式、model constraint、state transition、scenario step への位置ベース参照。
- IdMagic の既存テスト不足または仕様・実装不一致の解消。

## Plan
- authored reference は `context`、規範要素種別、所有要素名を分離した構造化値とする。standard requirement だけは standard 名と既存 requirement `id` の組で参照する。
- map key または既存 requirement `id` を安定識別子とし、配列 index や YAML 行番号を識別子にしない。細粒度の匿名 contract は所有する model/interface/state/scenario を参照対象とする。
- resolver は参照先の存在だけでなく、参照可能な規範要素種別かを検査する。glossary、annotations、context map、および生成物は verification target に含めない。
- SCL YAML の shape と `spec_version: "3.0"` は変更しない。参照は SCL の内側に逆向きリンクとして埋め込まず、後続 manifest が外側から保持する。

## Tasks
- [ ] T001 [Decision] assurance 台帳を追加せず既存の所有要素を直接参照する責務境界を ADR に記録する。
- [ ] T002 [Reference] SCL element reference の対象、構造化正規形、context-local 解決規則を言語リファレンスへ追加する。
- [ ] T003 [Resolver] context map と context spec を読み、参照を正規化・解決する共通 resolver を実装する。
- [ ] T004 [Validator] 未知 context・種別・要素・requirement、別 context の暗黙参照、位置ベース参照を拒否する positive/negative fixture を追加する。
- [ ] T005 [Renderer] HTML anchor と表示上の canonical reference を resolver の正規形へ揃える。
- [ ] T006 [Verify] tool、SCL、workspace の検証を通し、既存 SCL YAML が無変更で有効なことを確認する。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-scl`
- `just yaml-check-work-items`
- `just check-ids`
- `just verify`

## Risk Notes
参照粒度を匿名式や配列 index まで細分化すると、並べ替えだけで証跡が切れる。初期版は既存の安定 ID を持つ所有要素へ限定し、より細かな証明単位が実際に必要になった場合だけ、規範要素そのものへ名前を導入する別 WI で拡張する。
