---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: [wi-361-derived-normative-diff]
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、正規シナリオを起点に定義・コード・work item を引くための索引と検索を追加するだけである。
initial_context:
  source:
    - tools/render-spec-docs/src
    - justfile
  tests:
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading:
    - backend
    - frontend
---

# 正規シナリオを起点にコードと work item を引ける索引と検索を追加する

## Motivation

正規シナリオは 249 個あるが、ある REQ を変更したいエージェントが対応する実装・テスト・過去の判断に
辿り着く手段が全文検索しかない。1,231 行の `oauth2/SPECIFICATION.md` を読ませる導線しか無く、タスク
ごとのコンテキストコストが大きい。`decisions/` を廃止した現方式では「なぜ今この挙動なのか」の追跡先も
work item だけになっており、REQ を起点に辿れる導線が特に効く。

一方、テスト側への REQ 注記を機械検査で強制することは見送る。テスト自体は存在しており、欠けているのは
注記だけなので、検査が守るのは「新しい REQ に注記を付け忘れないこと」でしかない。発生しても影響が
軽く、摩擦だけが残る。索引と検索の提供に絞る。

## Scope

- 生成される仕様サイトに Traceability ページを追加する。全正規シナリオについて、定義位置へのリンク、
  その ID を書いているコード・テストのパス、その ID に言及している work item を一覧にする。
- `just spec-where <term>` を追加する。仕様・コード/テスト・work item を横断し、該当位置だけを返す。
- `DEVELOPMENT.md` の Context economy を、索引と `spec-where` を初手にする記述へ更新する。

## Out of Scope

- REQ 注記の有無を失敗させる検査（ラチェット含む）。
- 既存 249 シナリオへの注記の一括付与。
- 索引ファイル（`index.json` など）の生成。同期対象を増やさない。

## Design

- 対応関係の正本はコード側の注記 1 箇所だけとし、仕様側には書かない。双方向に書くと必ずドリフトする。
  索引は毎回リポジトリを走査して導出するので、生成物が実態から乖離しない。
- 参照の収集は `render-spec-docs` の実行時に行い、`spec/` 配下と生成物を除外したテキストファイルから
  `REQ-<CONTEXT>-NNN` を拾う。`work-items/` 配下は別カラムとして扱う。
- `spec-where` は生成物を持たず `rg` でその場で探索する。索引ファイルを作ると同期対象が増え、現方式が
  意図的に捨てた登録簿の再来になる。

## Plan

1. Traceability ページを `render-spec-docs` に追加する。
2. `just spec-where` を追加する。
3. `DEVELOPMENT.md` の導線を更新する。
4. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [Tooling] 参照収集と Traceability ページを実装し、サイトのナビゲーションに追加する。
- [x] T002 [Tooling] `just spec-where` を追加する。
- [x] T003 [Docs] `DEVELOPMENT.md` の Context economy を更新する。
- [x] T004 [Verify] `just check` と `just verify` を通す。
- [x] T005 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just test-tools`
- `just render-spec-docs`
- `just spec-where REQ-SOURCING-006`
- `just verify`

## Risk Notes

参照収集はリポジトリ全走査のため、レンダリング時間が伸びる。テキスト拡張子に限定し、生成物・依存物を
除外して抑える。ID の単純一致で拾うため、無関係な文脈での言及も参照として出る可能性があるが、索引の
用途では誤検出より取りこぼしのほうが害が大きいので許容する。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  仕様サイトに Traceability ページ（`spec/generated/docs/traceability/index.html`、サイドバーの
  References から到達）を追加した。全 249 正規シナリオについて、定義位置へのアンカーリンク、ID を
  書いているコード・テストのパス、言及している work item を表にする。参照はレンダリング時に作業ツリー
  を走査して収集するため、authored な索引を持たず乖離しない。現状は 249 中 7 シナリオが参照を持つ。
  併せて `just spec-where <term>` を追加した。仕様・コード/テスト・work item を横断して該当位置だけを
  返し、仕様本文を丸ごと読ませない初手を提供する。`DEVELOPMENT.md` の Context economy をこの導線に
  更新した。
  当初案にあった「新規 REQ にテスト注記が無ければ失敗」というラチェット検査は入れていない。テストは
  実在しており、検査が守るのは注記の付け忘れだけで、摩擦に見合わないと判断した。
- **Verification Results**:
  - `just test-tools` - passed（98 tests）
  - `just render-spec-docs` - 775 ページ生成、Traceability ページを含む
  - `just spec-where REQ-SOURCING-006` - 仕様 1 件・コード/テスト 5 件・work item 1 件を出力
  - `just verify` - passed
