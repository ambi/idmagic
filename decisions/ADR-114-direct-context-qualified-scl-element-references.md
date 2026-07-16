---
status: accepted
authors: [tn]
created_at: 2026-07-16
---

# ADR-114: SCL の保証対象は独立台帳へ複製せず context-qualified な規範要素参照で指す

## コンテキスト

SCL の規範要素と Architecture、実装、検証、実行証跡を結ぶには、どの要素を対象にしたかを
機械的かつ rename まで安定して識別する必要がある。一方、既存の model constraint、interface
contract、state、authorization、objective、scenario、flow と同じ保証文を独立した
`assurance.obligations` 台帳へ複製すると、二つの正本が drift し、形式的な coverage だけが増える。

ADR-103 は規則を実現・検証する所有要素へ局所化し、匿名の contract 式には補助 ID を持たせないと
決めている。この局所所有を維持したまま、複数 bounded context で同名になり得る要素と、配列内に
既存 ID を持つ standard requirement を一意に指す共通参照が必要である。

## 決定

SCL の外側から規範要素を指す値は、`SPECIFICATION_CORE_LANGUAGE.md §6.1` の
context-qualified SCL element reference とする。通常要素は `context`、`kind`、`element`、standard
requirement は `context`、`kind: standard_requirement`、`standard`、`requirement` を持つ。

参照対象は standard requirement、model、interface、state、authorization resource/principal/policy、
objective、scenario、flow に限定する。map key、または standard requirement の既存 `id` を identity
とし、配列 index、YAML 行番号、HTML slug は用いない。匿名の constraint、`requires` / `ensures`、
transition、scenario step は、その規則を所有する model、interface、state、scenario を参照する。

SCL に assurance section、保証文、evidence kind、実装・検証への逆向き link は追加しない。外側の
manifest が SCL element reference を保持し、SCL の規範文を直接 target にする。HTML anchor も同じ
構造化参照から生成する。

## 却下した代替案

- `assurance.obligations` を SCL に追加する案: 所有要素の規範文を再記述し、drift と重複 coverage を
  生むため採用しない。
- `section.name` だけで参照する案: bounded context 間と authorization の3種間で衝突し、一意に
  解決できないため採用しない。
- 配列 index、YAML 行番号、renderer の slug を使う案: 並べ替えや表示変更だけで証跡が切れ、元の
  identity を復元できないため採用しない。
- 全ての匿名規則へ ID を追加する案: authoring cost と形式的な粒度を先行させるため、実際に細粒度の
  証明単位が必要になるまで導入しない。

## 影響

- `SPECIFICATION_CORE_LANGUAGE.md §6.1` が参照対象、構造化正規形、canonical text、解決規則を定義する。
- `tools/yaml-check` は workspace context map を介して参照を解決し、未知 context、kind、element、
  standard、requirement、context 不一致を拒否する共通 resolver を提供する。
- `tools/scl-to-html` は canonical reference から衝突しない anchor と表示値を生成する。
- SCL YAML の shape と `spec_version: "3.0"`、既存 IdMagic context spec は変更しない。
- element rename は外側の参照を更新する必要がある破壊的変更になる。
