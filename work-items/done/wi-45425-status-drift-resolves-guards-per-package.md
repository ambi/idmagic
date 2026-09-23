---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-23
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "リポジトリ内の検査器だけを変更し、利用者へ告知する機能、互換性、移行操作はない。"
  references: []
initial_context:
  specification:
    - docs/design/application/api-guidelines.md
  typespec: []
  source:
    - tools/check/src/status-drift.ts
    - tools/check/src/check-status-drift.ts
    - tools/check/src/contract-drift.ts
  tests:
    - tools/check/src/status-drift.test.ts
  stop_before_reading:
    - backend
    - frontend
spec_impact: { kind: none, reason: "契約と実装の食い違いを読む検査器の変更であり、シナリオ、TypeSpec、製品の振る舞いを変えない。検査が新しく見つけたずれの修正は別の work item が扱う。" }
---

# check-status-drift が同名の前提判定をパッケージごとに読み分け、読めない判定を失敗として報告する

## 動機

`check-status-drift`（`tools/check/src/status-drift.ts`）は、echo の context を持つ関数を名前だけで索引する。
同じ名前の定義が 2 つ以上あると、`collectResponders` はその名前への呼び出しを `unread` として扱い、定義をどちらも読まない。
前提判定（guard）が読まれないと、その判定が書く 401 と 403 の問題コードが operation に帰属せず、S1（未宣言のステータス）と B1（未宣言の 403 問題コード）の材料そのものが欠ける。
一方で `unread` と `unresolved` は要約の件数に数えられるだけで、検査を失敗させない。

[[wi-551-back-api-tokens-examples-with-tests]] では、ApiTokens と federation のハンドラーがどちらも `requireAdmin` という名前の前提判定を持っていた。
このため federation の 9 operation と ApiTokens の一覧・失効について、401 と 403 の宣言漏れと、どの経路も返さない 400 の宣言が、検査を通ったまま隠れていた。
ApiTokens 側を改名した時点で初めて表に出た。
無関係なパッケージで同じ名前を付けるだけで検査の網が黙って外れるので、名前の衝突は今後も起きる。

## 対象範囲

- 定義の索引を、名前だけでなく定義のあるパッケージ（Go ファイルのディレクトリ）で区別する。
- 呼び出しの解決は、呼び出し側と同じパッケージの定義を優先する。修飾なしの呼び出しと、受け手が同じパッケージの型である呼び出しが対象になる。
- それでも 1 つに決まらない前提判定の呼び出しは、黙って `unread` にせず、検査の失敗として operation 名と候補の定義位置を報告する。
- ハンドラー名そのものが曖昧な `handler-ambiguous` も同じ扱いにするかを、着手時に既存の件数を測ってから決める。
- 検査器の単体テスト（`tools/check/src/status-drift.test.ts`）に、別パッケージの同名の前提判定を持つ入力を加える。

## 対象外

- 契約そのものの修正。本項目の変更で新しいずれが見つかった場合は、件数と内容を記録し、別の work item で直す。
- 一般の Go の型解決（インターフェースの動的な呼び出し先、別パッケージを経由する埋め込み）。この検査は字面の読み手であり、完全な型検査器を作ることは目的にしない。
- `check-contract-drift` など、ほかの字面検査に同じ弱点があるかの監査。

## 設計

採用する方針は、索引の鍵を「名前」から「パッケージと名前」へ広げ、同一パッケージの定義を優先して解決することである。
製品のコードでは、前提判定は呼び出すハンドラーと同じパッケージに置かれることが多い。
wi-551 の `requireAdmin` はどちらもそうだったので、この解決で両方を正しく読める。

パッケージをまたぐ呼び出し（`d.Auth.RequireAdmin(c)` など）は、受け手の型を字面から決められない。
候補が 1 つだけならそれを読み、2 つ以上なら曖昧として失敗させる。
黙って読み飛ばす現在の挙動は、検査の結果を「通った」と「読めなかった」で区別できなくするので残さない。

呼び出し先の候補は、呼び出しの修飾の字面で次のように決める。

| 呼び出しの形 | 候補 |
| --- | --- |
| 修飾なし（`requireAdmin(d, c)`） | 呼び出し側と同じディレクトリの定義 |
| 呼び出し側の受け手の変数を通す（`d.requireAdmin(c)`） | 同じディレクトリの定義。なければ同名の全定義 |
| それ以外（`d.Auth.requireAdmin(c)`、`support.X(c)`、式の結果） | 同名の全定義 |

候補が 2 つ以上残り、その中に前提判定が含まれる呼び出しは `Ambiguity`（名前と候補の定義位置）として記録し、前提判定の連鎖を通じてハンドラーへ伝播する。
候補がすべてエラー値で分岐する対応付けの関数なら、1 つに決まっても読まないので、従来どおり `unread` とする。
`diffStatusCodes` は、曖昧な呼び出しを持つ operation と、ハンドラー名そのものが 2 つ以上の定義を持つ operation を、規則 `A1` の失敗として報告する。
曖昧な operation では S2（宣言過剰）を報告しない。

ハンドラー名の曖昧さ（`handler-ambiguous`）も失敗にした。
着手時の件数は 0 件であり、失敗にしても既存の検査結果は変わらない。
経路の登録はハンドラーの定義位置を持たないため、パッケージでの解決はハンドラー名には適用しない。

採用しない代替案は次のとおりである。

| 代替案 | 採用しない理由 |
| --- | --- |
| 曖昧な名前を失敗にするだけで、解決は変えない | wi-551 の衝突のように、別パッケージの同名の前提判定は正当な書き方であり、改名を強いる理由にならない |
| `go/types` で呼び出し先を解決する | 字面の読み手を型検査器へ置き換える規模の変更になる。現在の誤りは同一パッケージの優先だけで解消できる |
| 同名の定義をすべて読み、ステータスの和を帰属させる | 別のハンドラーのステータスを持ち込む。`handler-ambiguous` の注記が避けている誤りそのものである |

## 計画

1. 着手時に、現在の `unread` と `unresolved` の件数と理由の内訳を記録する。
2. 失敗する単体テストを先に書く。別パッケージにある同名の前提判定が、それぞれのハンドラーへ正しく帰属することを見る。
3. 索引と解決を変え、`mise run check-status-drift` を実行して、新しく見つかったずれを数える。
4. 新しいずれが出た場合は、この項目では検査を通すための宣言の変更をせず、別の work item へ切り出す。検査の失敗が残る間はこの項目を完了にしない。

## タスク

- [x] T001 [Acceptance] 別パッケージの同名の前提判定を持つ入力で、現在の検査がその判定のステータスを帰属しないことを観測する。
- [x] T002 [Tool] 定義をパッケージと名前で索引し、同一パッケージの定義を優先して解決する。
- [x] T003 [Tool] 解決できない前提判定の呼び出しを失敗として報告する。
- [x] T004 [Verify] リポジトリ全体に対する `check-status-drift` の結果を記録し、変更を検証する。

RED、GREEN、故障注入には `mise run test-tools-file -- check/src/status-drift.test.ts` を使い、リポジトリ全体の結果は `mise run check-status-drift -- --list-unresolved` で変更前と比べた。

## 検証

- `mise run test-tools`
- `mise run check-status-drift`
- `mise run verify`

## リスク

- **新しい失敗が一度に大量に出る。** 名前の衝突で隠れていたずれが、ほかにも残っている可能性がある。計画の 1 で現在の件数を記録し、出たずれは別の work item で直す。
- **同一パッケージの優先が誤った定義を選ぶ。** 同じパッケージに同名の関数は定義できないため、パッケージ内では誤らない。受け手が別パッケージの型である呼び出しを同一パッケージの定義へ誤って結び付けないよう、修飾付きの呼び出しは受け手の字面を見て区別する。

## 完了

- **Completed At**: 2026-09-23
- **Summary**:
  `check-status-drift` の定義の索引に定義位置のディレクトリと受け手の変数名を加え、呼び出しの修飾の字面から、同じパッケージの定義を優先して呼び出し先を解決するようにした。
  それでも候補が 2 つ以上残る前提判定の呼び出しと、2 つ以上の定義を持つハンドラー名は、読み飛ばさずに規則 `A1` の失敗として候補の定義位置を報告する。
  候補がすべてエラー値で分岐する関数なら、従来どおり部分解析として数える。
  着手時の件数は、部分解析 247 件、未到達 1 件（`route-not-found`）、`handler-ambiguous` 0 件で、変更後も一覧は同じであり、新しく見つかったずれはない。
  `api-guidelines.md` の担保手段へ、この解決規則と失敗の扱いを追記した。
- **Acceptance RED Evidence**:
  - **Test**: `same-named guards in different packages > reports the undeclared statuses each package guard writes`（`tools/check/src/status-drift.test.ts`）
  - **Requirement**: N/A: 製品の振る舞いではなく、契約と実装の食い違いを読む検査器の変更である。
  - **Observed Failure**: 2 つのパッケージが同名の `requireAdmin` を持つ入力で、変更前の検査は S1 を 1 件も報告しなかった。どちらの前提判定も読まれず、401 と 403 が operation に帰属しなかった。
  - **Detection Reason**: 経路、前提判定、ハンドラーを別パッケージのファイルとして渡し、`diffStatusCodes` の報告を比べるため、名前の衝突で帰属が黙って欠ける欠陥を検出する。実リポジトリでも、ApiTokens の `requireAdministrator` を `requireAdmin` へ戻した入力で、変更前は 11 operation が部分解析へ落ち、変更後は 0 件になることを確かめた。
- **Unit RED Evidence**:
  - **Test**: `same-named guards in different packages > attributes each guard to the handler in its own package`、`fails a guard call through a field whose definition it cannot settle`、`coverage > fails an operation whose handler name two definitions share`（`tools/check/src/status-drift.test.ts`）
  - **Requirement**: N/A: 検査器の解決規則を固定するテストであり、製品の要求 ID を持たない。
  - **Observed Failure**: 変更前は各ハンドラーの状態コードが `[200]` だけになり、曖昧な呼び出しとハンドラー名は失敗ではなく `unread` と `unresolved` に数えられた（6 件失敗）。
  - **Detection Reason**: パッケージごとの帰属、候補が残る呼び出しの失敗、ハンドラー名の曖昧さを別々の表明で見るため、解決規則のどれが崩れたかを区別できる。
- **Change-Resistance Results**:
  変異器は Go だけを対象とするため、次の故障を手で注入した。
  パッケージの優先の除去（3 件失敗）、修飾なし呼び出しを全定義へ広げる変更（2 件）、受け手の変数の分岐の除去（3 件）、曖昧な前提判定の `unread` への格下げ（1 件）、曖昧さの A1 報告の配線の除去（1 件）、ハンドラー名の曖昧さの報告の配線の除去（1 件）は、いずれも検出された。
  前提判定の連鎖を通じた曖昧さの伝播の除去と、曖昧な operation での S2 の抑止の除去は最初生き残ったため、`an ambiguous call behind a guard > fails the operation and reports no over-declaration` を追加し、どちらも 1 件失敗することを確かめた。
- **Verification Results**:
  - `mise run test-tools-file -- check/src/status-drift.test.ts` - passed（33 件）
  - `mise run check-status-drift -- --list-unresolved` - passed（0 件の指摘、342 operation 中 94 件を完全解析、247 件を部分解析、未到達 1 件。変更前と同じ一覧）
  - `mise run format-tools`、`mise run lint-tools`、`mise run typecheck-tools` - passed
  - `mise run spec-diff` - passed（規範の変更なし）
  - `mise run verify` - passed（tools 594 件、UI 単体 699 件、Go の競合検査を含む）
