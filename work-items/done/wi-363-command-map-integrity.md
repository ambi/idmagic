---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-13
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、コマンドマップの記述と CI からの参照整合性を直すだけである。
initial_context:
  source:
    - justfile
    - .github/workflows/idmagic-ci.yaml
    - tools/check/src
  tests:
    - tools/check/src
  stop_before_reading:
    - backend
    - frontend
    - spec
---

# 壊れた CI 参照を直し、just --list をコマンドマップとして読める状態にする

## Motivation

`justfile` はこのリポジトリ唯一のコマンドマップであり、人と AI エージェントの入口である。その入口に
2 つの欠陥がある。

1. CI が `just traceability-strict` を呼び続けているが、この recipe は SCL 廃止（1b7b2cef）で
   `justfile` から消えている。`error: justfile does not contain recipe 'traceability-strict'` により
   **2026-08-10 以降、main の CI は 4 回連続で失敗している**。ローカルの `just verify` は緑なので、
   気付く手段が実質無かった。
2. `just --list` の説明が 11 recipe で意味不明になっている。`just` は直前の連続コメントのうち
   最終行だけを説明として表示するため、複数行コメントの recipe は説明が途中の断片になる。実例:
   `verify` は「bottleneck (and whether the grouping still makes sense) is always visible.」、
   `check-schema` は「no-ops). Runs in an isolated, disposable compose project (wi-308).」。
   `deploy-monitoring-operator` には説明が無い。発見手段そのものが壊れている。

recipe 数（71）は多いが、`verify-go` / `verify-ui` / `verify-spec` / `restore-drill` / `k6-smoke` などは
README・work item から実際に参照されており、削減対象ではない。数ではなく、参照の正しさと説明の質が問題。

## Scope

- CI workflow から存在しない `just traceability-strict` の呼び出しを除去し、SCL を指す古い step 名を直す。
- `.github/workflows/**` が参照する `just <recipe>` が `justfile` に実在することを検査する
  `check-command-map` を追加し、`just check` に組み込む。
- 全 recipe の説明を 1 行に整え、詳細は recipe 本体側のコメントへ移す。`deploy-monitoring-operator` に
  説明を付ける。
- recipe コメントに残った `(wi-101)` / `(wi-308)` などの work item 参照を除去する。

## Out of Scope

- recipe の統廃合。参照実績がある以上、数を減らすこと自体を目的にしない。
- CI workflow の構成変更（ジョブ分割、キャッシュ、実行環境）。
- 失われた traceability 検査の再実装。REQ とテストの対応は別途 wi-362 で扱う。

## Design

- 「CI が参照する recipe の実在」は、発生しやすく（recipe 削除は日常的なリファクタで起きる）、起きたときの
  影響が大きい（本番と同じ検査経路が丸ごと止まる）。ローカル検証では絶対に検知できない種類の欠陥なので、
  機械検査に値する。逆に「recipe が CI から使われているか」は検査しない。用途は CI だけではない。
- 説明は `just --list` に出る 1 行がすべてという前提に合わせる。読ませたい詳細は本体側コメントに置く。
  shebang 付き recipe では shebang 行より後にコメントを置く。

## Plan

1. CI workflow を直す。
2. `check-command-map` を実装し、単体テストを付けて `just check` に組み込む。
3. 説明を 1 行化する。
4. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [CI] 存在しない `just traceability-strict` の呼び出しを除去し、step 名を実態に合わせる。
- [x] T002 [Tooling] `check-command-map` を実装し、単体テストを追加して `just check` に組み込む。
- [x] T003 [Tooling] 全 recipe の説明を 1 行化し、work item 参照を除去する。
- [x] T004 [Verify] `just check` と `just verify` を通す。
- [x] T005 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just test-tools`
- `just --list`
- `just check`
- `just verify`

## Risk Notes

説明の 1 行化で、コメントに書かれていた運用上の注意（本番の image digest、非本番ガード、Prometheus
Operator の任意性）が失われると事故につながる。削除ではなく recipe 本体側へ移す。CI workflow の変更は
main への push まで実地検証できないため、参照検査をローカルで通すことで代替する。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  CI が呼んでいた存在しない `just traceability-strict` を除去し、SCL を指していた job 名を
  「Verify specification, backend, and UI」に直した。これで 2026-08-10 以降続いていた main の CI 失敗が
  解消する見込み。再発防止として `just check-command-map` を追加し、`.github/workflows/**` が呼ぶ
  `just <recipe>` が `justfile` に実在することを `just check` の一部として検査する。逆方向（recipe が
  CI から使われているか）は検査しない。
  `just --list` の説明は、`just` が直前コメントの最終行だけを表示する仕様に合わせて整えた。運用上の
  注意を失わないよう、複数行の説明は `[doc("...")]` 属性で 1 行の説明を与えたうえでコメント本文を
  残す形にした。`deploy-monitoring-operator` に説明を追加し、recipe コメントの `(wi-101)` /
  `(wi-308)` 参照を除去した。recipe の統廃合は行っていない。参照実績を確認した結果、
  `verify-go` / `verify-ui` / `verify-spec` / `restore-drill` / `k6-smoke` などはいずれも README や
  work item から使われており、削減対象ではなかった。
- **Verification Results**:
  - `just test-tools` - passed（98 tests）
  - `just check-command-map` - passed
  - `just --list` - 全 recipe が 1 行の説明を持つことを確認
  - `just verify` - passed
