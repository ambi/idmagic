---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-09
depends_on: []
change_kind: maintenance
initial_context:
  source: [tools/ra/src/architecture-workspace.ts]
  tests: [tools/ra/src/architecture-workspace.test.ts]
  stop_before_reading: [spec, internal, frontend]
spec_impact:
  kind: none
  reason: TypeScript path alias の内部展開を正す保守修正であり、製品のモデル、外部インターフェース、状態、認可、シナリオを変更しないため。
---

# TypeScript alias の wildcard を完全かつリテラルに展開する

## Motivation
RA の architecture workspace 検査は TypeScript path alias の target に含まれる最初の
`*` だけを展開している。複数の wildcard が残る不正確な解決を防ぎ、CodeQL の
incomplete string escaping or encoding 指摘を原因箇所で解消する。

## Scope
- `tools/ra/src/architecture-workspace.ts` の TypeScript alias target 展開
- `tools/ra/src/architecture-workspace.test.ts` の回帰テスト

## Out of Scope
- TypeScript compiler と同等の module resolution 全体の実装
- alias pattern、target の選択順序、workspace module 判定の変更
- SCL、Architecture、公開 API の変更

## Tasks
- [x] T001 [Test] 複数 wildcard と置換記法を含む capture の回帰テストを追加。RED: `just test-tools` で未置換の `*` により追加した 2 test の失敗を確認。
- [x] T002 [Tooling] callback 形式の `replaceAll` で alias target の全 wildcard を capture のリテラル値として展開。
- [x] T003 [Verify] tools の test、typecheck、lint と work item 検査を通す。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just lint-tools`
- `just check-work-items`
- `just check-ids`

## Risk Notes
変更は alias target の wildcard 置換だけに限定する。capture は repository 内の import
specifier から得られる単純な文字列であり、再帰的な文法や認証・認可判断を伴わないため、
fuzz/property test は追加せず、複数 wildcard と JavaScript の `$&` 置換記法を直接テストする。

## Completion
- **Completed At**: 2026-08-09
- **Summary**:
  TypeScript alias target の全 wildcard を callback 形式の `replaceAll` で展開し、capture を
  JavaScript の置換記法ではなくリテラル値として扱うようにした。複数 wildcard と `$&`
  capture の回帰テストを追加した。
- **Verification Results**:
  - `just test-tools` - passed（297 tests）
  - `just typecheck-tools` - passed
  - `just lint-tools` - passed（82 files）
  - `just check` - passed
  - `just check-work-items` - passed
  - `just check-ids` - passed
  - `git diff --check` - passed
