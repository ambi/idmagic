---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-18
depends_on: [wi-238-scim-inbound-list-query-conformance]
change_kind: maintenance
spec_impact:
  kind: none
  reason: >-
    テストのみを追加する。wi-238 で実装した RFC 7644 filter grammar parser
    (backend/scim/domain/filter.go) の既存の振る舞い・SCL 契約は変更しない。
---

# SCIM inbound filter parser のテスト網羅性と fuzz 耐性を強化する

## Motivation

wi-238 で `backend/scim/domain/filter.go` に RFC 7644 §3.4.2.2 filter grammar の
parser/evaluator を実装したが、完了報告時点でテストに以下の欠落があった。

- `ne` 演算子を直接検証するテストがない（`not` 経由の間接テストのみ）。
- `gt` / `ge` / `lt` / `le` が構文として認識されつつ、どの allowlist 属性でも
  許可されず常に `invalidFilter` になることを明示するテストがない。
- untrusted な外部入力（SCIM client が送る任意の filter 文字列）を parse する
  security-sensitive なコードであるにもかかわらず、手書きテストケースのみで
  fuzz test が存在しない。wi-238 の Risk Notes は injection・過剰計算量を
  明示的にリスクとして挙げていたが、fuzzing によるパニック・無限ループ耐性の
  検証はカバレッジに含まれていなかった。

これらのギャップは実装完了報告で自己申告されず、ユーザーからの直接的な質問で
初めて判明した。テスト網羅性を仕様どおりに固定し、将来の回帰・未知の
クラッシュ入力を検出できるようにする。

## Scope

- `backend/scim/domain/filter_test.go` に `ne` 演算子の直接テストを追加する。
- `backend/scim/domain/filter_test.go` に、`gt`/`ge`/`lt`/`le` が
  `UserFilterAttributes` / `GroupFilterAttributes` のいずれの属性に対しても
  `*domain.FilterError` (unsupported operator) になることを明示するテストを追加する。
- `backend/scim/domain/filter_fuzz_test.go` に `FuzzParseFilter` を追加し、
  `ParseFilter` が任意のバイト列に対してパニック・ハングしないことを検証する。
  既存のユニットテストケース（複合式、文字列 escape、資源上限境界など）を
  seed corpus として登録する。

## Out of Scope

- 複数値属性の複合フィルタ (`emails[type eq "work"]` のような bracket 構文) の実装。
- `gt`/`ge`/`lt`/`le` を実際に使える日付・数値属性の追加。
- schema URN プレフィックス付き属性名、custom/extension schema 属性への対応。
- これらは振る舞い変更を伴うため、必要になった時点で SCL 変更を伴う別 work item で扱う
  （wi-238 の Plan で意図的に allowlist に閉じた決定を踏襲する）。

## Plan

- 振る舞い変更なし、`filter.go` 自体は変更しない前提でテストのみを追加する。
- fuzz test で crasher が見つかった場合はテスト追加の範囲を超えるため、
  この work item 内で `filter.go` の bug fix として扱ってよい（振る舞いが
  「パニックしない」という既存の暗黙契約を満たす方向の修正であり、新機能では
  ないため spec_impact: none の範囲内とみなす）。ただし修正が allowlist や
  文法の対応範囲を広げる場合は、この work item をいったん止めて SCL 変更が
  必要かどうか判断する。
- fuzz corpus はリポジトリに残し、`just test-go` の通常実行では `-fuzz` なしで
  seed corpus を通常のテストケースとして再生し続ける（Go の fuzz test は
  `-fuzz` を指定しない限り seed のみ実行される標準動作）。

## Tasks

- [ ] T001 [Test] `ne` 演算子の直接テストを追加する（`TestParseFilterNotEqual` 等、
      scenario `SCIM clientはUsersとGroups collectionを検索できる` / interfaces.ListScimUsers）。
- [ ] T002 [Test] `gt`/`ge`/`lt`/`le` が全 allowlist 属性で `invalidFilter` になることを
      明示するテーブル駆動テストを追加する。
- [ ] T003 [Test] `FuzzParseFilter` を追加し、既存テストケースを seed corpus として登録する。
      ローカルで最低 30 秒 `-fuzz=FuzzParseFilter -fuzztime=30s` を実行し、crasher が
      出ないことを確認する。crasher が見つかった場合は `testdata/fuzz/FuzzParseFilter/`
      に再現ケースとして残り、`filter.go` を修正して GREEN にする。
- [ ] T004 [Verify] `just test-go`、`just verify-go` を通す。

## Verification

- `go test ./backend/scim/domain/... -v`
- `go test ./backend/scim/domain/... -run=^$ -fuzz=FuzzParseFilter -fuzztime=30s`
- `just test-go`
- `just verify-go`

## Risk Notes

振る舞い変更を伴わないテスト追加が中心のため低リスク。唯一の不確実性は
fuzz test が `filter.go` の未知のパニック／無限ループ・過大なメモリ確保を
見つけた場合で、その場合は `filter.go` 自体の修正が必要になる
（MaxFilterLength/MaxFilterDepth の適用漏れや、lexer の境界条件など）。
見つかった場合も「既存の allowlist・grammar 範囲内でパニックしない」という
修正であり、対応範囲の拡大にはあたらない。
