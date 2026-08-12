---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: [wi-359-spec-first-method-docs-realignment]
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、着手中の work item が指す読み込み対象の実在を検査するだけである。
initial_context:
  source:
    - tools/workspace/src/check-workspace.ts
  tests:
    - tools/workspace/src/workspace.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec/contexts
---

# 着手中の work item の initial_context が実在する対象を指すことを検査する

## Motivation

`check-workspace.ts` は `affected_spec` の参照を既に検査している（パスの実在、`requirement` 文字列の
出現、TypeSpec symbol の末尾セグメントの宣言）。一方 `initial_context` は一切検査されていない。

`initial_context` は AI エージェントが着手時に最初に読む唯一の導線であり、そこが壊れていると読む対象を
自力で探し直すことになり、コンテキストコストが跳ね上がる。実際、pending 90 件のうち 13 件が削除済みの
`decisions/ADR-*.md` を参照している。wi-359 で `initial_context` を着手時必須へ移したので、着手の瞬間に
「そこに書いたものが実在するか」だけを検査すれば、腐った導線で作業を始めることはなくなる。

併せて `requirement` の照合が単なる部分一致である点を、正規シナリオ見出しとしての宣言の有無に変える。
別の REQ ID の写し間違いが本文中の言及で通ってしまうのを防ぐ 1 行の変更で済む。

## Scope

- `tools/workspace/src/check-workspace.ts` の work item 検査に、`status: in_progress` の記録に限って
  `initial_context` の各パスが実在することの検査を追加する。
- `initial_context.specification` は `spec/**/SPECIFICATION.md#REQ-...` 形式を許容し、ファイルの実在と
  REQ 見出しの宣言を検査する。
- `affected_spec[].requirement` の照合を、部分一致から `### <REQ-ID>:` 見出しの宣言へ変更する。
- 検査対象を絞る根拠を `WORK_ITEM_FORMAT.md` に 1 行で残す。

## Out of Scope

- 新しい `just` recipe や新規ツールの追加。既存の `just check-work-items` の内側で完結させる。
- TypeSpec symbol の名前空間まで含めた解決。誤りは実在する（wi-298 の `IdMagic.Contract.UserInfo` は
  実際には `IdMagic.OAuth2.Operations.UserInfo`）が、末尾セグメントが合っていれば読み手は到達でき、
  影響が軽い。コンパイル済みプログラムへの依存を持ち込む価値がない。
- pending / done の `initial_context` 検査。腐ることを前提に、着手時に書き直す運用で受ける。
- 廃止キー（`scl:` / `decisions:`）を名指しで拒否する規則。歴史的経緯に固有の規則を検査に持ち込まない。

## Design

- 新しい検査層やコマンドを足さず、既に work item を走査している 1 箇所に条件を加えるだけにする。方式を
  重くしないことを優先し、`in_progress` という 1 状態にだけ効く規則にする。
- `specification` 参照は `path#REQ-ID` の 1 形式に限る。複数形式を許すと検査もエージェントの書き方も
  ぶれる。

## Plan

1. 検査を `check-workspace.ts` に追加する。
2. 壊れた入力が赤になる単体テストを追加する。
3. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [Tooling] `in_progress` の `initial_context` パス実在検査と `specification` の REQ 宣言検査を
      追加する。
- [x] T002 [Tooling] `affected_spec[].requirement` の照合を見出し宣言へ変更する。
- [x] T003 [Tooling] 単体テストを追加する。
- [x] T004 [Docs] `WORK_ITEM_FORMAT.md` に検査範囲を 1 行で明記する。
- [x] T005 [Verify] `just check` と `just verify` を通す。
- [x] T006 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just test-tools`
- `just check`
- `just verify`

## Risk Notes

`requirement` 照合の厳格化により、既存記録が赤くなる可能性がある。全 361 件で確認し、赤が出た場合は
参照先の修正で解消する。`in_progress` の記録は現時点で本 work item のみのため、新検査の影響範囲は狭い。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  `check-workspace.ts` に散在していた work item の参照検査を `tools/check/src/work-item-references.ts`
  の純関数へ切り出し、単体テストを付けた。検査内容は 2 点変えた。`affected_spec[].requirement` は
  `REQ-` 始まりのとき `### <ID>:` 見出しの宣言を要求する（standard ID は従来どおり出現照合）。
  `status: in_progress` の記録に限り `initial_context` を解決し、`source` / `tests` /
  `stop_before_reading` のパス実在と、`specification` の `path#REQ-ID` 形式の解決を検査する。
  新しいコマンドや設定は追加しておらず、`just check-work-items` の内側で完結する。
  本 work item 自身を `in_progress` にして新検査を通し、運用として成立することを確認した。
- **Verification Results**:
  - `just test-tools` - passed（87 tests）
  - `just check` - passed（361 records）
  - `just verify` - passed
