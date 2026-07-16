---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: []
---

# SCL 3.1 に仕様要素の保証義務を表す assurance を追加する

## Motivation
RA は仕様・実装・テスト・検証結果の追跡可能性を要求しているが、SCL 3.0 には仕様要素を何で証明するかを表す正規の構造がない。そのため、保証の対応が Go の手書き `AssuranceManifest` や完了 WI の自由記述へ分散し、仕様追加時のテスト漏れを機械的に検出できない。

## Scope
- `SPECIFICATION_CORE_LANGUAGE.md` に SCL 3.1 と `assurance.obligations` の意味・型・参照規則を追加する。
- `tools/yaml-check` の SCL 3.1 schema、version dispatch、semantic validation を追加する。
- `tools/scl-to-html` が assurance obligation と coverage を表示できるようにする。
- 全 tool self-spec と IdMagic の context spec を SCL 3.1 へ移行する。
- obligation は既存の `standards`、`models`、`interfaces`、`states`、`authorization`、`objectives`、`scenarios`、`flows` を `covers` で参照し、業務要件そのものを重複記述しない。

## Out of Scope
- 実装ファイル・テストファイル・CI artifact の SCL 内への記載。
- IdMagic の既存テスト不足の解消。
- Architecture と実 import の検証。
- SCL 3.0 文書の即時サポート終了。

## Plan
- SCL 3.0 validator は読み取り互換のため維持し、3.1 は加算的な新 version として実装する。
- `assurance.obligations.<id>` は `description`、context-local な `covers`、非空の `evidence_kinds` を持つ。実装固有 binding は後続 WI の verification manifest が所有する。
- 不明な参照、空の coverage、未対応 evidence kind、別 context の暗黙参照を semantic error にする。
- 3.1 の positive/negative fixture と tool self-spec を先に更新し、IdMagic spec の移行後に派生物を一括再生成する。

## Tasks
- [ ] T001 [Decision] assurance と外部 evidence binding の責務境界を ADR に記録する。
- [ ] T002 [SCL] SCL 3.1 の言語リファレンスと schema を追加する。
- [ ] T003 [Validator] version dispatch、参照解決、negative fixture を追加する。
- [ ] T004 [Renderer] HTML に obligation と coverage を描画する。
- [ ] T005 [SelfSpec] 全 tool self-spec を 3.1 へ移行する。
- [ ] T006 [IdMagic] root と全 context spec を 3.1 へ移行する。
- [ ] T007 [Derived] SCL 派生物を再生成する。
- [ ] T008 [Verify] tool、SCL、workspace の検証を通す。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-scl`
- `just scl-render`
- `just verify`

## Risk Notes
言語 version と全仕様文書を横断するため高リスク。3.0 と3.1の意味を暗黙に混在させず、version fixture、未知 field 拒否、全 context 一括移行で silent downgrade を防ぐ。
