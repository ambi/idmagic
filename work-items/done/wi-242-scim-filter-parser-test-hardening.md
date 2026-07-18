---
status: completed
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
  `UserFilterAttributes` / `GroupFilterAttributes` の非 dateTime 属性
  (string/boolean) に対しては `*domain.FilterError` (unsupported operator) に
  なることを明示するテストを追加する。（[[wi-244-scim-filter-grammar-extended-conformance]]
  がこの work item の起票後に `meta.created`/`meta.lastModified` への
  `gt`/`ge`/`lt`/`le` 対応を実装済みのため、本文起票時の「どの allowlist 属性でも」
  という記述は「dateTime 属性を除く」に読み替える。dateTime 属性側の
  `gt`/`ge`/`lt`/`le` は wi-244 の `TestParseFilterDateTimeComparison` が
  既に固定済み。）
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

- [x] T001 [Test] `ne` 演算子の直接テストを追加する（`TestParseFilterNotEqual`、
      scenario `SCIM clientはUsersとGroups collectionを検索できる` / interfaces.ListScimUsers）。
      string 属性 (`userName ne`) と boolean 属性 (`active ne`) の両方を固定した。
- [x] T002 [Test] `gt`/`ge`/`lt`/`le` が dateTime 以外の allowlist 属性で
      `invalidFilter` になることを明示するテーブル駆動テスト
      (`TestParseFilterOrderingOperatorsRejectedForNonDateTimeAttributes`) を追加した。
      wi-244 が本 work item 起票後に dateTime 属性 (`meta.created`/`meta.lastModified`)
      への対応を実装済みのため、`UserFilterAttributes`/`GroupFilterAttributes` を
      `Kind` で走査し dateTime 属性を除外する形にスコープを補正した。
- [x] T003 [Test] `FuzzParseFilter` (`backend/scim/domain/filter_fuzz_test.go`) を追加し、
      既存テストケース(複合式・文字列 escape・dateTime・URN プレフィックス・資源上限境界など
      34 件)を seed corpus として登録した。`-fuzz=FuzzParseFilter -fuzztime=30s` を実行し
      (1000 万回超の実行)、crasher は見つからなかった。
- [x] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を通した。

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

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  `backend/scim/domain/filter_test.go` に `TestParseFilterNotEqual`
  (string/boolean 属性への `ne` の直接検証) と
  `TestParseFilterOrderingOperatorsRejectedForNonDateTimeAttributes`
  (`UserFilterAttributes`/`GroupFilterAttributes` の非 dateTime 属性すべてに対し
  `gt`/`ge`/`lt`/`le` が `*domain.FilterError` になることをテーブル駆動で固定) を
  追加した。`backend/scim/domain/filter_fuzz_test.go` を新規追加し、`FuzzParseFilter`
  に既存ユニットテストケース(複合式・大文字小文字・文字列 escape・dateTime 比較・
  不正 dateTime literal・schema URN プレフィックス・資源上限境界・未終端文字列など
  34 件)を seed corpus として登録した。`filter.go` 自体への変更は不要だった
  (fuzz 実行で crasher は見つからなかった)。
  当初の Scope 記述(「gt/ge/lt/le はどの allowlist 属性でも常に invalidFilter」)は
  wi-238 時点の状態を前提にしており、本 work item の起票後に
  [[wi-244-scim-filter-grammar-extended-conformance]] が `meta.created`/
  `meta.lastModified` への `gt`/`ge`/`lt`/`le` 対応を実装済みだったため、
  T002 のテストは「dateTime 属性を除く」に補正して実装した。
- **Affected Guarantees State**: なし。テスト追加のみで `filter.go` の振る舞いは
  変更していない (spec_impact: none のとおり)。
- **Verification Results**:
  - `go test ./backend/scim/domain/... -v` — passed
  - `go test ./backend/scim/domain/... -run=^$ -fuzz=FuzzParseFilter -fuzztime=30s` —
    passed (1,025 万回超の実行、crasher 0 件)
  - `just yaml-check` — passed (251 work item、369 record id、ARCHITECTURE.md、
    traceability manifest/evidence すべて green)
  - `just test-go` — passed
  - `just verify-go` (golangci-lint 0 issues + `go test -race ./...`) — passed
- **対応していないこと (ADR-121 の開示義務)**:
  この work item は spec_impact: none のテスト追加のみで、`RFC7644-FILTERING` の
  `adoption: partial` の範囲([[wi-244-scim-filter-grammar-extended-conformance]]
  で開示済みの複数値属性複合フィルタ・custom schema 属性の未対応)を変更していない。
  Out of Scope に記載のとおり、複数値属性の複合フィルタ・`gt`/`ge`/`lt`/`le` の
  対象属性拡大・schema URN プレフィックス対応の拡張は本 work item では扱っていない。
