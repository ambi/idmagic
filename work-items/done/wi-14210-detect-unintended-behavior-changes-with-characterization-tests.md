---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-06
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発時の検査と手順を加えるだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - docs/development/testing.md
  typespec: []
  source:
    - tools/check/src/check-spec-impact.ts
    - tools/check/src/spec-impact.ts
  tests:
    - tools/check/src/spec-impact.acceptance.test.ts
  stop_before_reading: [backend, frontend]
spec_impact:
  kind: none
  reason: "開発フローに特性化テストの工程を加え、仕様影響の宣言の検査に特性化テストの変更の条件を一つ足すだけで、要件、TypeSpec の契約、永続状態、発行するイベント、外向きの呼び出しは変えない。"
---

# 既存コードを変える前に特性化テストで現在の振る舞いを固定し、宣言のない振る舞いの変化を検出する

## 動機

仕様に書かれていない振る舞いを、宣言なしに変えてしまう変更を止める仕組みがない。

仕様影響の宣言（`check-spec-impact`）が確かめるのは、宣言と仕様差分の食い違いだけである。
仕様に書かれていない振る舞いを変えても仕様差分は出ないので、`Spec-Impact: none` のまま検査を通る。
`//spec:covers` を付けたテストも、要件として書いた振る舞いしか守らない。

既存コードの仕様は、その機能に触れる変更のときに、触れる範囲だけを書き起こす方針である（[既存コードからの書き起こし](../../docs/development/specification-first-workflow.md#既存コードからの書き起こし)）。
この方針では、触れる前の振る舞いを何かで固定しておかないと、書き起こす前に振る舞いが変わっても気付けない。

`docs/development/testing.md` は特性化テストを手法として挙げているが、開発フローのどの工程で書くか、書いた後にどう扱うかを定めていない。

## 対象範囲

- Michael Feathers の特性化テストの手順を、開発フローの工程として定める。
  対象は、既存コードを変える作業のうち、触れる範囲の振る舞いを `//spec:covers` を付けたテストが固定していないもの。
- 特性化テストの書き方、置き場所、名前、書いた後の分類を、既存の仕組みの上に定める。
- `check-spec-impact` に条件を一つ加える。`Spec-Impact: none` のコミット、または `spec_impact: none` の work item の作業範囲が、既存の特性化テストを変えたり消したりしたら失敗させる。
- `docs/development/specification-first-workflow.md`、`docs/development/testing.md`、`implement-work-item` スキルを更新する。

## 対象外

- 新しいファイル形式、記録の生成と承認のタスク、専用のテスト基盤。
  通常の Go のテストと既存のタスクで足りる。
- Context 全体を前もって特性化すること。
  Feathers の手順どおり、変更する箇所の周りだけを特性化する。
- 特性化テストを仕様として扱うこと。
  特性化テストは現在の振る舞いの記録であり、維持すべき約束かどうかは分類で決める。
- フロントエンド。

## 設計

### 手順

Feathers の手順をそのまま使う。

1. 変更する箇所を、外部の振る舞いを観測できる最も狭い既存の境界（ユースケースまたはハンドラー）からテストで呼ぶ。
2. 結果について、わざと誤った期待値を表明する。
3. 失敗の出力から、実際の振る舞いを読む。
4. 実際の振る舞いを期待値にする。
5. 変更で動かし得る振る舞いが固定できるまで繰り返す。

固定できているかは、`mise run test-go-mutation -- <package-directory>` で、変更する箇所への変異を特性化テストが検出するかで確かめる。
検出しない変異があれば、その振る舞いの特性化テストを足す。

### 既存の仕組みとの関係

| 既存の仕組み | 特性化テストとの関係 |
| --- | --- |
| `//spec:covers` のないテストの表明を分類する工程（[仕様にない振る舞いの分類](../../docs/development/specification-first-workflow.md#仕様にない振る舞いの分類)の出力の表） | 特性化テストはこの「`//spec:covers` のないテスト」にあたる。新しい分類は作らない |
| `mise run spec-review-candidates` | 特性化テストを書く場所の候補を示す |
| ミューテーションテスト | 特性化テストが変更する箇所を固定できているかを確かめる |
| 仕様影響の宣言 | 特性化テストを変えるコミットを、`none` では通さない |
| 実行可能な具体例（`testdata/*.examples.json`） | 要件の具体例であり、特性化テストとは別に扱う。特性化テストに使わない |

### 名前と置き場所

特性化テストは、対象のパッケージの通常のテストファイルに `TestCharacterize<対象>` という名前で置く。
名前で区別するのは、仕様影響の検査が、テストファイルの中身を解析しなくても対象の関数を見分けられるようにするためである。
`//spec:covers` は付けない。

### 書いた後の分類

変更が済んだら、特性化テストが固定した振る舞いを[仕様にない振る舞いの分類](../../docs/development/specification-first-workflow.md#仕様にない振る舞いの分類)で分ける。

| 分類 | 特性化テストの扱い |
| --- | --- |
| (a) 要件にする | 外部仕様として維持すべき振る舞いだけを要件にし、テストの名前を変えて `//spec:covers` を付ける |
| (b) 書かない | 特性化テストを消す。消すコミットは `none` にできないので、work item で消す理由を記録する |
| (c) 実装を直す | 是正の work item を起票し、その work item で特性化テストを書き換える |
| 判断を後に回す | `TestCharacterize*` のまま残す。触れる人が次に分類する |

(a) で要件にするのは、[仕様として書く実装上の細部](../../SPECIFICATION_FORMAT.md#仕様として書く実装上の細部)の基準を満たす外部の振る舞いに限る。
特性化テストで固定したからといって、要件を増やさない。

### コミットの順序

特性化テストは、振る舞いを変え得る変更より前のコミットで追加する。
追加するコミットは本番コードを変えないので、仕様影響の宣言は要らない。
後のコミットでテストが落ちたら、宣言のない振る舞いの変化を検出したことになる。
変更を戻すか、work item で宣言して分類する。

### 仕様影響の検査に加える条件

| 条件 | 判定 |
| --- | --- |
| `Spec-Impact: none` のコミットが、基準に存在した `TestCharacterize*` 関数を変えるか消した | 失敗 |
| `spec_impact: none` の work item の作業範囲で、基準に存在した `TestCharacterize*` 関数が変わるか消えた | 失敗 |
| 宣言のない（トレーラーも宣言のある work item もない）コミットが、親コミットに存在した `TestCharacterize*` 関数を変えるか消した | 失敗 |
| `TestCharacterize*` 関数を新しく追加した | 成功 |
| `spec_impact: none` の work item が、作業範囲の中で足した `TestCharacterize*` 関数を変えるか消した | 成功 |

検査は、`backend/` 配下の `_test.go` について、基準（コミットなら親、work item なら作業範囲の起点）と変更後の `func TestCharacterize…(` から行頭の `}` までの本文を文字列として比べる。
テストの意味は解析しない。
ハンクの位置から関数を求める案より、差分の出力形式に依らず、純粋な関数として単体テストで固定できる。

### 着手時に見つけた未記載の振る舞い

| # | 今の挙動 | 分類 | 扱い |
| --- | --- | --- | --- |
| 1 | 本番コードを変えないコミットには宣言が要らないので、トレーラーのないテストだけのコミットで特性化テストを書き換えると、条件をすり抜ける | (a) | 宣言のないコミットによる書き換えも失敗にする。上の表の 3 行目 |
| 2 | (b) で、足した作業範囲の外から特性化テストを消すには、`affected_spec` を宣言する work item が要る | (a) | 対象範囲の「消すコミットは `none` にできない」の帰結として、ワークフローの (b) の行に書く |

### 採用しない案

| 案 | 採らない理由 |
| --- | --- |
| 外部境界の応答を承認済みファイルに記録して比べる（golden master） | 記録の形式、正規化、更新のタスクを新しく保守する必要がある。記録が大きいと差分を読まずに更新する運用になる。変更する箇所だけを通常のテストで固定すれば足りる |
| 特性化テストに専用のコメントやマーカーを付ける | 名前の規約で関数を見分けられる。注釈の文法を一つ増やすだけになる |
| 検査を足さず、レビューだけで特性化テストの書き換えを見つける | 落ちたテストを同じコミットで書き換えれば、検出が無効になる。この一点だけは機械で止める |
| Context 全体を前もって特性化する | 変更しない箇所の固定に費用をかけることになり、既存コードの書き起こしの方針とも合わない |

## 計画

1. `check-spec-impact` の条件の失敗例をテストで固定する。
2. 条件を実装する。
3. 開発フローの文書とスキルに、手順、名前、分類、コミットの順序を書く。

未解決の問いはない。

## タスク

- [x] T001 [Acceptance] `Spec-Impact: none` のコミットが既存の `TestCharacterize*` 関数を変えたときと消したときに、`check-spec-impact` が失敗する例を検査のテストで固定し、RED を確認する。新しく追加しただけなら成功する例も固定する。RED：`mise run test-tools-file -- check/src/spec-impact.acceptance.test.ts`。
- [x] T002 [Tooling] `check-spec-impact` に条件を加え、GREEN にする。単体：`mise run test-tools-file -- check/src/spec-impact.test.ts`。
- [x] T003 [Docs] `docs/development/specification-first-workflow.md` の既存コードからの書き起こしと出力の表、`docs/development/testing.md` の特性化テストの項に、手順、名前、分類、コミットの順序を書く。
- [x] T004 [Docs] `implement-work-item` スキルに、既存コードを変える前に特性化テストを書く工程を加える。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec-impact`
- `mise run check`
- `mise run verify`

## リスク

- 特性化テストが溜まり、分類されないまま残る。
  `TestCharacterize*` の名前で一覧できるので、触れる人が分類する。数の推移は、問題になったときに測る。
- 特性化テストを要件へ昇格させすぎると、仕様が偶然の振る舞いで膨らむ。
  昇格させるのは、仕様として書く実装上の細部の基準を満たす外部の振る舞いだけにする。
- リファクタリングで特性化テストの呼び出し方を変える必要が出ると、`none` のコミットで直せない。
  その場合は work item を起票して宣言する。手間は増えるが、振る舞いを変えていないことをレビューで確かめる機会になる。

## 完了
- **Completed At**: 2026-10-06
- **Summary**:
  `mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範仕様の差分はない。
  `mise run check-spec-impact`（`mise run check` にも含む）は、`Spec-Impact: none` のコミット、宣言のないコミット、`spec_impact: none` の work item の作業範囲が、基準に存在した `TestCharacterize*` 関数の本文を変えたか消したときに失敗するようになった。新しく足した特性化テストと、`none` の作業範囲の中で足した特性化テストの変更は通す。
  仕様先行の開発ワークフローに「特性化テスト」の節を加え、手順、名前、置き場所、コミットの順序、書いた後の分類を定め、既存コードからの書き起こしの手順、検査の出力の表、仕様影響の検査の表から参照した。
  `docs/development/testing.md` と `implement-work-item` スキルに、既存の Go コードを変える前に特性化テストを書く工程を加えた。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/spec-impact.acceptance.test.ts` の `spec-impact: 特性化テストの書き換え`
  - **Requirement**: N/A: 開発時の検査の追加であり、製品の要件を変えない。
  - **Observed Failure**: 実装前に、拒否を期待する 4 件（`none` のコミットによる変更と削除、宣言のないコミットによる変更、`none` の work item の作業範囲での変更）が `ok  spec impact` を返して失敗した。通すことを期待する 3 件は実装前から通った。
  - **Detection Reason**: 一時的な Git リポジトリでコミットを作り、`mise run check-spec-impact` と同じ入口を実行して出力と終了コードを確かめる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/spec-impact.test.ts` の `changedCharacterizations`
  - **Requirement**: N/A: 開発時の検査の追加であり、製品の要件を変えない。
  - **Observed Failure**: 受け入れテストの RED の後に関数と同時に加えたので、単体の RED は観測していない。検出能力は下の故障注入で確かめた。
  - **Detection Reason**: 本文の変更、削除、関数の外の変更、新しい関数、ファイルの追加と削除を、文字列の入力から直接確かめる。
- **Change-Resistance Results**:
  `test-go-mutation` は Go 専用なので、TypeScript の検査には手で故障を注入した。
  本文の比較を名前の有無の比較に変えると、単体 1 件と受け入れ 3 件が失敗した。
  work item の作業範囲への配線を外すと、`none` の work item の受け入れテストが失敗した。
  宣言のないコミットの条件を外すと、宣言のないテストだけのコミットの受け入れテストが失敗した。
- **Verification Results**:
  - `mise run check-spec-impact` - 成功
  - `mise run verify` - 初回は `AccountDataPage > downloads the export and clears the downloading state` が並列実行の負荷で失敗した。この変更はフロントエンドに触れず、同じファイルの単独実行は 3 回とも成功し、再実行した `mise run verify` は成功した
