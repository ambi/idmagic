---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: [wi-586-frontend-architecture-and-user-interface-design]
change_kind: refactor
spec_impact:
  kind: none
  reason: "どの画面からも使われていない補助の扱いを決めるだけで、画面の振る舞いは変えない。"
---

# 使われていないページング補助の扱いを決める

## Motivation

`frontend/src/lib/usePaginatedList.ts` と `frontend/src/components/ui/load-more.tsx` の `LoadMoreButton` は、「さらに読み込む」形式のページングの補助である。
どちらも自身のテストからしか参照されておらず、どの画面も使っていない。
一覧のページングは、URL の検索パラメーターと `PageNavigation` が担っている（[フロントエンド設計](../docs/design/application/frontend.md#一覧のページング)）。
Knip はテストからの参照を使用として数えるため、未使用として報告しない。

## Scope

- 二つの補助が残っている理由を確認する。一覧を `PageNavigation` へ移したときの取り残しか、「さらに読み込む」形式を使う予定の画面があるかを、導入と移行のコミットから判断する。
- 使う予定がなければ、補助とそのテストを削除し、共通辞書の不要になったキーも削除する。
- 使う予定があれば、どの画面がどの条件で使うかを[ユーザーインターフェース設計](../docs/design/application/user-interface.md#一覧)に書き、補助を残す。

## Out of Scope

- `PageNavigation` の変更。
- テストからの参照だけで使われているコードを Knip で検出する設定の変更。必要なら別の work item にする。

## Design

判断は導入時と移行時のコミットに基づく。
一覧の画面をカーソル方式のページ送りへ移したコミットで、「さらに読み込む」形式の置き換えが完了しているなら、取り残しとみなして削除する。

## Plan

1. 導入と移行のコミットを読み、判断を記録する。
2. 削除する場合は、補助、テスト、辞書のキーを同じ変更で削除する。

## Tasks

- [ ] T001 [Design] 残っている理由を確認し、削除するか残すかを決める。
- [ ] T002 [Refactor] 決定に従って削除するか、設計文書に使いどころを書く。
- [ ] T003 [Verify] `mise run verify-ui` を通す。

## Verification

- `mise run check-ui-dependencies`
- `mise run verify-ui`

## Risk Notes

共通辞書のキーは型で `ja` と `en` の両方に強制されているため、片方だけ残る状態は型検査で検出できる。
