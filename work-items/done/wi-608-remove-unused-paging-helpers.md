---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: [wi-586-frontend-architecture-and-user-interface-design]
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  references: []
  reason: "未公開の製品で、画面から呼ばれない補助と辞書のキーだけを削除する。画面のページ送り、既存データ、HTTP 契約に影響せず、利用者の移行操作もない。"
initial_context:
  specification:
    - docs/development/task-context.md
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/development/writing-language.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/release.md
    - docs/design/application/design-guidelines.md
    - docs/design/application/frontend.md
  typespec: []
  source:
    - frontend/src/lib/i18n/common.i18n.ts
    - frontend/src/components/ui/page-navigation.tsx
  tests: []
  stop_before_reading:
    - backend
    - spec/modules
spec_impact:
  kind: none
  reason: "どの画面からも使われていない補助の扱いを決めるだけで、画面の振る舞いは変えない。"
---

# 使われていないページング補助の扱いを決める

## Motivation

`frontend/src/lib/usePaginatedList.ts` と `frontend/src/components/ui/load-more.tsx` の `LoadMoreButton` は、「さらに読み込む」形式のページングの補助である。
どちらも自身のテストからしか参照されておらず、どの画面も使っていない。
一覧のページングは、URL の検索パラメーターと `PageNavigation` が担っている（[フロントエンド設計](../../docs/design/application/frontend.md#一覧のページング)）。
Knip はテストからの参照を使用として数えるため、未使用として報告しない。

## Scope

- 二つの補助が残っている理由を確認する。一覧を `PageNavigation` へ移したときの取り残しか、「さらに読み込む」形式を使う予定の画面があるかを、導入と移行のコミットから判断する。
- 使う予定がなければ、補助とそのテストを削除し、共通辞書の不要になったキーも削除する。
- 使う予定があれば、どの画面がどの条件で使うかを[ユーザーインターフェース設計](../../docs/design/application/user-interface.md#一覧)に書き、補助を残す。

## Out of Scope

- `PageNavigation` の変更。
- テストからの参照だけで使われているコードを Knip で検出する設定の変更。必要なら別の work item にする。

## Design

判断は導入時と移行時のコミットに基づく。
一覧の画面をカーソル方式のページ送りへ移したコミットで、「さらに読み込む」形式の置き換えが完了しているなら、取り残しとみなして削除する。

`139a58641` は二つの補助を導入し、監査イベント、User、Group、Agent、Application の五つの一覧に結んだ。
`ecd74caff` は同じ五つの一覧から両方の import を削除し、URL のカーソルと `PageNavigation` によるページの置き換えへ移した。
`6f22aa964` はその方式に正確な件数と先頭、末尾への移動を追加している。
現在の両補助の利用元は専用テストだけであり、未完了の項目にも利用予定はない。
wi-371 の言及は利用できない理由であり、削除後は現行の `PageNavigation` を名指す形に更新する。
Jobs の「さらに読み込む」は機能固有の実装と辞書を使い、今回の補助を使わない。
したがって、両補助と専用テスト、共通辞書の `loadMore`、`loadingMore`、`loadMoreFailedError` を削除する。

テストの削除を含むため、テストを変えない refactor の証拠省略には当たらない。
製品の受け入れ境界と内部ロジックは追加しないので、両 RED の代替検査には既存の `mise run typecheck-ui` を使う。
補助を削除した段階で専用テストの import 解決が失敗することを観測し、残った利用元を検出できることを確かめる。
専用テストも削除した後に同じ検査が成功し、画面側の利用元が残っていないことを確認する。
共通辞書のキーへの参照も型検査で検査する。
削除した補助と専用テストはこの設計節に名前を残し、`initial_context` には削除後も解決できる参照を残す。

## Plan

1. 導入と移行のコミットを読み、判断を記録する。
2. 削除する場合は、補助、テスト、辞書のキーを同じ変更で削除する。

## Tasks

- [x] T001 [Design] 導入と移行のコミット、現在の利用元と未完了項目を確認し、取り残しとして削除すると決めた。変更前の `typecheck-ui` と専用テスト 11 件は成功した。
- [x] T002 [Refactor] 補助の削除後に `mise run typecheck-ui` が両専用テストの import に TS2307 を報告した。専用テストと不要な辞書キーの削除後は同じ検査と `check-ui-dependencies` が成功し、既存の `page-navigation.test.tsx` 2 件も成功した。製品の規範要件は N/A（利用経路のない補助の削除）。
- [x] T003 [Verify] `mise run verify` が成功し、`verify-ui` と同じ六つの検査も通った。`mise run test-ui-e2e` は 39 件成功し、仕様差分はなかった。完了移動後の記録とリンクの検査も成功した。

## Verification

- `mise run check-ui-dependencies`
- `mise run verify-ui`

## Risk Notes

共通辞書のキーは型で `ja` と `en` の両方に強制されているため、片方だけ残る状態は型検査で検出できる。

## Completion

- **Completed At**: 2026-10-11
- **Summary**:
  `mise run spec-diff -- main` は規範仕様の差分なしと報告した。
  URL と `PageNavigation` によるページ送りへ移行した後に残っていた二つの補助と専用テストを削除し、共通辞書から三つのキーを両言語で削除した。
  未着手の wi-371 が削除対象を名指していた箇所を、現行の `PageNavigation` の説明へ更新した。
  画面のページ送り、HTTP 契約、保存状態、通知とイベントは維持した。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 製品の利用経路を持たない補助の削除なので、代替検査に `mise run typecheck-ui` を使った。
  - **Requirement**: N/A: 製品の規範要件を変更しない。
  - **Observed Failure**: 補助の削除後、`load-more.test.tsx` の import に TS2307（`Cannot find module './load-more'`）が出て失敗した。
  - **Detection Reason**: 削除した部品を呼ぶ箇所が残れば、型検査が import の解決失敗を検出する。専用テストの削除後は成功し、画面側の利用元が残っていないことを確認した。
- **Unit RED Evidence**:
  - **Test**: N/A: 残す内部ロジックの変更がないので、代替検査に同じ `mise run typecheck-ui` を使った。
  - **Requirement**: N/A: 製品の規範要件を変更しない。
  - **Observed Failure**: 補助の削除後、`usePaginatedList.test.ts` の import に TS2307（`Cannot find module './usePaginatedList'`）が出て失敗した。
  - **Detection Reason**: フックの利用元の取り残しを検出する。専用テストと不要な辞書キーを削除した後は型検査が成功した。
- **Change-Resistance Results**:
  低リスクの未使用コードの削除で、残す計算や分岐の変更はない。
  補助だけを削除した段階で参照切れを検出し、残す `PageNavigation` の既存テスト 2 件も成功した。
  新しいテストと手書きの変異は追加していない。
- **Verification Results**:
  - `mise run setup`、`mise run lint-go` - 成功。
  - `mise run typecheck-ui`、`mise run check-ui-dependencies` - 成功。
  - `mise run test-ui-unit-file -- src/components/ui/page-navigation.test.tsx` - 2 件成功。
  - `mise run verify` - サンドボックスの外の固定したツリーで成功。UI 698 件、ツール 1,018 件、Go 6,133 件のテストが成功した。
  - `mise run test-ui-e2e` - サンドボックスの外で 39 件成功。
  - `mise run spec-diff -- main` - 規範仕様の差分なし。
  - `mise run check-work-items`、`mise run check-links` - 完了移動後も成功。
  - 初回の集約検証はサンドボックスの待受け制限などで失敗した。再実行中の指示追記もツリーの安定性検査が拒否したため、編集を止めた再実行で成功を確認した。環境による失敗は上の RED の証拠に数えていない。
  - `PageNavigation` と Knip の検出設定は対象外として変更していない。
