---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-11
depends_on: []
---

# Work-item の完了前提を機械可読な依存グラフとして管理する

## Motivation
本文リンクだけでは、実装順序を止める前提と単なる関連を区別・検証できない。

## Scope
- `tools/yaml-check/spec/scl.yaml` の work-item 検査仕様
- work-item format、Schema、横断依存グラフ検査、HTML 表示
- 未完了 work-item の `depends_on` 移行

## Out of Scope
- 完了済み work-item の履歴移行。
- 依存先が未完了であることによる status 遷移の禁止。

## Plan
- `depends_on` を完了前提だけの配列として定義し、存在・自己参照・循環を検査する。
- 本文の WI 参照を自動変換せず、未完了 WI を個別に判定して移行する。

## Tasks
- [x] T001 [SCL] work-item 依存グラフの仕様を追加する。
- [x] T002 [Tooling] Schema・横断検査・HTML 表示を実装する。
- [x] T003 [Data] 未完了 WI に依存情報を追加する。
- [x] T004 [Verify] 検証と派生物同期を行う。

## Verification
- `just yaml-check`
- `just test-tools`
- `just verify`

## Risk Notes
本文の全 WI リンクを依存と誤認すると実装順序を不必要に拘束する。明示的な完了前提だけを採用する。

## Completion
- **Completed At**: 2026-07-11
- **Summary**:
  未完了 work-item の完了前提を `depends_on` として正本化し、Schema・依存グラフ検査・HTML 表示を追加した。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-tools` - passed
  - `just typecheck-tools` - passed
  - `just verify` - blocked by the pre-existing Go lint error: `context loading failed: no go files to analyze`
