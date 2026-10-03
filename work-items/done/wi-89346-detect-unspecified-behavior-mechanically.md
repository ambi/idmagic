---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発時の検査とテストと手順を加えるだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/development/specification-first-workflow.md
    - docs/domain/identity-management/user/README.md
    - docs/domain/identity-management/user/lifecycle.md
  typespec: []
  source:
    - tools/check/src/specification-doc.ts
    - tools/check/src/feature-specification.ts
    - tools/check/src/registry.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/src/check-event-contract.ts
    - spec/contexts/identity-management/models.tsp
    - backend/idmanagement/domain/events.go
  tests:
    - backend/idmanagement/handlers_http/refusal_effects_test.go
    - backend/idmanagement/handlers_http/lifecycle_refusal_effects_test.go
  stop_before_reading: [frontend, backend/idmanagement/user/db_postgres]
spec_impact:
  kind: none
  reason: "仕様の漏れを検出する検査とテストを加えるだけで、要件、TypeSpec の契約、永続状態、発行するイベント、外向きの呼び出しは変えない。検出した漏れの対応は別の work item で行う。"
---

# 仕様に書かれていない振る舞いを、仕様、実装、テストの各工程で機械的に検出する

## 動機

wi-17076 は、未記載の振る舞いを見つけたら (a) 要件にする、(b) 書かない、(c) 実装を直す、に分類して仕様、設計、実装へ反映する手順を定める。
しかし、手順だけでは、見つけるかどうかが作業者の注意に依存する。
IdManagement の 15 件の要判断は、実装と、既存コードから仕様を書き出す作業の後になって初めて見つかった。
漏れの多くは「状態と操作の組の結果」「エラーとイベントの発行条件」「同じ種類の操作の経路ごとの差」のように型が決まっているので、機械的に検出できる。

## 対象範囲

- 状態遷移の節の状態遷移表（マトリクス形式）に、結果のないセルがあれば拒否する検査。
- TypeSpec が宣言するエラーコードと、コードが発行するドメインイベントが、それぞれ一つ以上の要件に現れることを確かめる検査。
  `check-security-controls` と `check-event-contract` と同じ形で、導入時点の違反は許容リスト（減らす方向にだけ変更するベースライン）に載せる。
- 状態遷移表（マトリクス形式）から操作の列を生成して実装に適用し、表と実装の食い違いを検出するモデルベースのテスト。
  最初の対象は User のライフサイクルとする。
- `spec-review-candidates`、ミューテーションテストの生存した変異、`//spec:covers` のないテストの出力を、(a)(b)(c) の分類と work item の記録へつなぐ手順を、仕様先行の開発ワークフローに書く。

## 対象外

- 書式と分類の手順そのものの定義。
  wi-17076 が扱う。
- 検出した漏れの解消。
  IdManagement の許容リストは wi-83002 が書き直しの中で消す。
- User のライフサイクル以外の状態機械へのモデルベースのテストの適用。
  IdManagement は wi-86874、Tenancy は wi-71372、残りの Context は wi-35451 が扱う。
- IdManagement 以外の Context への語彙の検査の適用。
  wi-71372 と wi-35451 が、Context を書き直すたびに対象へ加える。

## 設計

検査は、既存の検査と同じく `mise run check` から呼び、導入時点の違反は許容リストに載せて順に解消する。
モデルベースのテストは、状態遷移表（マトリクス形式）を一次情報として読み、表を手でテストへ写さない。

### 状態遷移表（マトリクス形式）の検査

`specification-doc.ts` の状態機械の検査を広げる。
三つ目の表は、`| State |` で始まり `Kind` の列を持たない見出し行で見分ける。
表がない状態機械は拒否しない。
書式の「三つの表を置く」は *(checked)* でなく、15 件の状態機械のうち表を持つのは UserLifecycle だけなので、存在を求めると導入時点で 14 件が違反になるためである。

| 判定内容 | 失敗時の動作 |
| --- | --- |
| 結果のないセル（空のセル、列より少ないセル） | 状態と操作を名指して拒否する |
| セルの値が `→ <State>`、`何もしない`、`拒否：<返すもの>` のどれでもない。`<br>` で区切った各結果に、括弧の条件を添えてよい | 拒否する |
| 行の状態、または `→` の遷移先が状態の表にない | 拒否する |
| 状態の表の状態に行がない | 拒否する |
| `→ X` の結果に対応する `From`、`To` の行が遷移の表にない。遷移の表の行に対応する `→` の結果が表にない | 拒否する。二つの表が別の遷移を述べると、どちらかが漏れである |

### エラーコードとドメインイベントの要件への記載の検査

新しい検査 `unspecified-vocabulary` を `registry.ts` に登録し、`mise run check-unspecified-vocabulary` からも呼ぶ。
純粋な部分（収集と差分）と、ワークスペースを読む部分を、既存の検査と同じく二つのファイルに分ける。

| 語彙 | 収集元 | 要件に現れたとみなす表記 |
| --- | --- | --- |
| エラーコード | `spec/contexts/<context>/*.tsp` の `urn:idmagic:error:<code>` と、その `@doc` が付く `model <Name>` | 要件の本文に `<code>` または `<Name>` が単語として現れる |
| ドメインイベント | `backend/<package>/domain/*.go` の `EventType() string { return "<Name>" }` | 要件の本文または状態遷移の表に `<Name>` が単語として現れる |

- 対象は Context ごとの設定として持ち、最初は IdManagement（`identity-management`、`backend/idmanagement`）だけとする。
- 要件は、Context の木の下の機能仕様が宣言する `REQ-*` の見出しから次の見出しまでとする。状態遷移の表の `Event` 列はイベントの発行条件を述べるので、イベントでは表も記載とみなす。
- 導入時点の違反は `tools/check/unspecified-vocabulary-debt.json` に載せる。違反でなくなった項目が残っていれば、項目を消すよう求めて失敗する。許容リストは減る方向にだけ変わる。
- コードが発行するかではなく、宣言した `EventType` で数える。発行しないイベントの宣言は、それ自体が消すべきコードなので、検査が名指してよい。

### User のライフサイクルのモデルベースのテスト

`backend/idmanagement/handlers_http/user_lifecycle_model_test.go` に置き、管理 API の正式な入口を通す。
状態遷移表と遷移の表は、テストが実行時に `docs/domain/identity-management/user/README.md` から読む。
表の読み取りは、テスト専用のパッケージ `backend/shared/testing_statematrix` に置き、単体テストを持たせる。

| 型と操作 | 内容 |
| --- | --- |
| `statematrix.Machine` | `States []string`、`Transitions []Transition{From, Event, To}`、`Operations []string`、`Cells map[Cell][]Outcome` |
| `statematrix.Outcome` | `Kind`（`Transition`、`NoOp`、`Refusal`）、`To`、`Refusal`（`409 user_pending_deletion` のような返すもの）、`Condition`（括弧の条件。なければ空） |
| `statematrix.Read(markdown, machine string) (Machine, error)` | H3 の名前で状態機械を選び、三つの表を読む |
| `Machine.EventFor(from, to string) (string, bool)` | `→` の結果に対応する遷移の表の `Event` |

テストの側にだけ置く対応は、次の三つである。
対応のない名前が表に現れたら、テストは失敗する。

| 対応 | 理由 |
| --- | --- |
| 操作の列の名前から HTTP 要求 | 表は操作を日本語の名前で書き、経路を書かない |
| 条件の名前（`猶予期間内`、`猶予期間後`）から、`status_changed_at` の設定 | 条件は状態の外の値なので、テストが作る |
| 状態の名前から `UserStatus` の値 | `PendingDeletion` を `pending_deletion` へ変換する規則で導く |

各ステップでは、表から予測した結果と、実装の結果を比べる。

| 予測 | 表明 |
| --- | --- |
| `→ X` | 2xx を返し、状態が X になり、遷移の表がその遷移に付ける `Event` だけを、表に現れるイベントのうちから発行する |
| `何もしない` | 2xx を返し、状態と `status_changed_at` が変わらず、表に現れるイベントを発行しない |
| `拒否：<status> <code>` | その状態コードと Problem Details のコードを返し、状態と `status_changed_at` が変わらず、表に現れるイベントを発行しない |

操作の列は、すべての状態とすべての操作の組を一度ずつ試す網羅と、初期状態から乱数で操作を選び続ける列の二つで作る。
乱数の種は固定し、失敗時に種と列を表示する。

### 着手時に見つけた未記載の振る舞い

| # | 今の挙動 | 分類 | 扱い |
| --- | --- | --- | --- |
| 1 | `UserStatus` は状態の表にない `locked`、`staged`、`suspended` を宣言し、本番コードはどれにも遷移せず、TypeSpec にも現れない | (c) | wi-98121 |

## 計画

1. wi-17076 の完了後に、状態遷移表（マトリクス形式）の書式を確かめ、検査とモデルベースのテストの入力形式を決める。
2. 検査を加え、導入時点の違反を許容リストに載せる。
3. User のライフサイクルにモデルベースのテストを加える。
4. 検出結果を分類へつなぐ手順を書く。

## タスク

- [x] T001 [Tooling] 状態遷移表（マトリクス形式）の網羅の検査を加える。RED：`mise run test-tools-file -- check/src/specification-doc.test.ts`。
- [x] T002 [Tooling] エラーコードとドメインイベントの要件への記載の検査を加える。RED：`mise run test-tools-file -- check/src/unspecified-vocabulary.test.ts`。
- [x] T003 [Test] User のライフサイクルにモデルベースのテストを加える。RED：`mise run test-go-test -- ./backend/shared/testing_statematrix <test>`、`mise run test-go-test -- ./backend/idmanagement/handlers_http TestUserLifecycleFollowsTheStateMatrix.*`。
- [x] T004 [Docs] 検出結果を分類と work item の記録へつなぐ手順を書く。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- 各検査が、意図して作った違反を検出すること。
- `mise run verify`

## リスク

- 検査が誤検出を重ねると、許容リストが膨らみ、検査が読まれなくなる。
  最初は IdManagement に限って導入し、誤検出の割合を確かめてから全体へ広げる。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範仕様の差分はない。
  `mise run check-spec` は、状態遷移表（マトリクス形式）の空のセル、語彙にない結果、状態の表にない状態、行のない状態、遷移の表と食い違う遷移を拒否するようになった。
  新しい検査 `mise run check-unspecified-vocabulary`（`mise run check` にも含む）は、IdManagement の TypeSpec のエラーコードと Go のドメインイベント 72 件のうち、どの要件にも現れない 17 件（エラーコード 8 件、イベント 9 件）を `tools/check/unspecified-vocabulary-debt.json` に許容リストとして載せた。17 件は、廃止した操作ごとの要約表にだけ書かれていたものと、どこにも書かれていないものである。
  `backend/shared/testing_statematrix` が機能仕様から状態機械を読み、`TestUserLifecycleFollowsTheStateMatrixInEveryCell` と `TestUserLifecycleFollowsTheStateMatrixAlongRandomWalks` が、UserLifecycle の全セルと乱数の操作列を管理 API で実行して表の予測と比べる。導入時点で表と実装は一致した。
  仕様先行の開発ワークフローに、各検出の出力を (a)(b)(c) の分類と記録先へつなぐ表を加えた。
  着手時に見つけた、どの遷移も到達しない `UserStatus` の三つの値を (c) とし、wi-98121 として起票した。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/specification-doc.test.ts` の `state matrix`、`tools/check/src/unspecified-vocabulary.test.ts`
  - **Requirement**: N/A: 開発時の検査とテストの追加であり、製品の要件を変えない。
  - **Observed Failure**: 実装前に、`state matrix` の拒否を期待する 6 件と、条件の付いた遷移先を読む 1 件が失敗した。`unspecified-vocabulary.test.ts` はモジュールがなく失敗した。
  - **Detection Reason**: 実際の文書へ違反を入れて確かめた。`user/README.md` の `Disabled × 無効化` を空にすると `mise run check-spec` が `state matrix gives no outcome for Disabled × 無効化` で失敗し、`lifecycle.md` から `restore_grace_expired` を消すと `mise run check-unspecified-vocabulary` が `error code restore_grace_expired appears in no requirement` で失敗した。
- **Unit RED Evidence**:
  - **Test**: `backend/shared/testing_statematrix/statematrix_test.go`
  - **Requirement**: N/A: テスト専用の読み取りであり、製品の要件を変えない。
  - **Observed Failure**: 実装前は `no non-test Go files` でビルドが失敗した。モデルベースのテストは、表と実装が一致していたので初回から通った。検出能力は下の故障注入で確かめた。
  - **Detection Reason**: セルの結果、条件、遷移の表のイベントを、文書の文字列から直接確かめる。
- **Change-Resistance Results**:
  表の `Disabled × 無効化` を `拒否：409 already_disabled` に変えると、全セルのテストと乱数の列のテストが `status=204 code="", want 409 already_disabled` で失敗した。
  `SetUserDisabled` から削除予約中の拒否を外すと、`PendingDeletion × 無効化` と `PendingDeletion × 再有効化` が `want 409 user_pending_deletion` で失敗した。
  再有効化の冪等の分岐を外すと、`Active × 再有効化` が `status_changed_at` の変化で失敗した。
  `mise run test-go-mutation -- backend/shared/testing_statematrix` の初回では 10 件が生存した。短い行と文書末尾の状態機械を読むテストを加えて 3 件を検出するようにした。残る 7 件のうち、`iota + 1` の符号、切り出しの開始位置（見出し行を含めても結果が同じ）、容量の指定と分割数の境界の 5 件は等価な変異である。セル数の境界 2 件（`len(cells) > 1`、`len(cells) < 4`）は、`mise run check-spec` が拒否する崩れた表でだけ差が出るので、テストを加えていない。
- **Verification Results**:
  - `mise run verify` - 成功
