---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-12
priority: p2
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  references: []
  reason: "開発補助の廃止と文書の参照整理であり、製品の機能、利用契約、既存データ、機能成熟度への影響はない。廃止後の操作は開発文書に反映する。"
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/formats/documentation-guide.md
    - docs/formats/specification-format.md
    - docs/formats/design-format.md
    - docs/development/coding-style.md
    - docs/development/writing-language.md
    - docs/development/testing.md
    - docs/development/task-context.md
    - docs/development/local-development.md
    - docs/design/application/backend.md
    - docs/design/application/design-guidelines.md
  typespec: []
  source:
    - mise.toml
    - tools/README.md
    - tools/check/README.md
    - tools/check/src/registry.ts
    - tools/check/src/command-map.ts
    - tools/package.json
    - .github/workflows/idmagic-ci.yaml
    - .agents/skills/implement-work-item/SKILL.md
    - .agents/skills/update-design/SKILL.md
    - .agents/skills/render-docs/SKILL.md
    - tools/brief/src/main.ts
    - tools/task-timing/src/main.ts
  tests:
    - tools/check/src/boundary-fitness.test.ts
    - tools/check/src/verification-tasks.test.ts
  stop_before_reading:
    - backend/
    - frontend/src/
    - spec/
    - work-items/done/
spec_impact:
  kind: none
  reason: "開発用ツールと文書の重複を整理する。製品の HTTP 契約、認証と認可の判定、永続状態、発行するイベント、外向きの呼び出しは変更せず、規範要件の内容と ID を保つ。"
---

# 費用対効果の低いツールと文書の重複を削り、保守する量を減らす

## 動機

開発用ツールが増え、選択、実行、保守にかかる費用に見合う価値があるかを見直す必要がある。
文書に同じ規則や説明が複数あれば、読む量に加えて同期する費用も増える。
ツールと文書を中心に整理し、コードとテストの細部を調べる費用は抑える。

## 対象範囲

- `tools/` と `mise.toml` のリポジトリ固有の検査、照会、報告、生成、実行補助を棚卸しし、価値の低いものを削除する。
  呼び出し元の `.github/`、`.agents/skills/`、`AGENTS.md` と関連文書も合わせて更新する。
- `docs/`、ルートの案内文書、リポジトリ内スキルにある同一内容の重複を、一次情報への参照へ置き換える。
  共通の開発規則とツールの説明から着手し、そこから見つかる重複へ進む。
- 削除したツールにだけ属する実装、テスト、fixture、設定、依存宣言を削る。
  製品コードと製品テストは構成と参照の確認にとどめ、明らかな不要物が見つかった場合だけ、その到達性と役割を確認して削除する。

## 対象外

- 製品コードと製品テストを全件読み、関数やケース単位で重複を探す調査。
- 製品の振る舞い、規範要件、公開契約、モジュール境界を変える変更。
- 汎用 CLI やランタイムの一括削減、依存関係の一括更新。
- 削減率の達成を目的にした短縮、独立した故障を検出するテストの削除。
- 削減判断を自動化する新しいツールや、恒久的な棚卸し台帳の追加。
- 完了済み作業項目の圧縮と、文書体系全体の再設計。

## 設計

### ツールを残す費用と価値を比較する

`tools/README.md` が示すとおり、個別検査と集約検査は同じ registry を参照する。
タスク名の数を独立したツールの数と見なさず、実装と入口を分けて評価する。
個別入口には局所検証を短くする価値があり、入口を減らす場合も実装を減らす場合も、その効果を区別して記録する。

候補ごとに、呼び出し元、固有の検出能力または作業短縮、実行費用、誤報や例外管理の負担、変更への追従費用、既存手段による代替を短く記す。
利用履歴が取れない場合、参照の少なさを利用頻度の証拠にしない。
照会と報告の補助、既存 CLI で容易に置き換えられる処理、重なる検査を優先して比較する。
削除候補が共有するコードは、残る利用者を確認してから整理する。

独自の機能があっても、効果が保守費用に見合わなければ廃止の候補にする。
代替なしで検査を廃止する場合は、失う検出能力、残るリスク、廃止を選ぶ理由をこの項目へ記録し、所有する開発規則を先に更新する。
必要な安全性や契約の証拠を代替手段で保てない候補は、今回の削除から外す。
単に検査を CI から外して実装を放置する整理は行わない。

### 文書の内容ごとに所有元を一つにする

[一次情報源の割り当て](../../docs/formats/documentation-guide.md#一次情報源の割り当て)と[開発文書の形式](../../docs/formats/documentation-guide.md#8-開発文書)に沿い、同じ規則、数値、一覧、手順の再掲を削る。
仕様と設計、規則と理由、索引と本文のように役割が異なる記述は、同じ語があるだけでは重複と判断しない。
参照先を探すための短い案内は残し、要件の意味と ID、運用時に必要な判断材料を保つ。
既存の生成元がある一覧は、手書きの複製を整理する。

### 調査の深さを限定する

最初はタスク定義、登録一覧、文書の見出し、参照検索で候補を絞る。
ソースの構造把握には `ast-grep outline` を使い、削除判断に必要な実装と呼び出し元だけを読む。
製品コードとテストの細部まで読まなければ判断できない候補は、今回の整理から外す。
削除に伴うツール専用テストの整理と、残るツールの局所検証は行う。

全ツールを維持して文書だけを短縮する案は、ツールの保守費用を減らせないため採らない。
全コードとテストを精査する案は、今回求める削減効果に対して調査費用が大きいため採らない。

## 計画

### 着手時の採否

タスク定義、registry の登録、ツールの構造と import、CI とスキルの参照を確認した。
利用履歴は取得しておらず、参照数を利用頻度の根拠にしない。
本番入口と製品テストの細部を読まずに判断できる次の四つを廃止する。
製品の境界と中核ロジックは変えず、残るツールの API と共有処理も変更しない。

| 候補 | 呼び出し元と固有の価値 | 費用と追従の負担 | 採否、代替と失う能力 |
| --- | --- | --- | --- |
| `report-change-coupling` | 設計ガイドラインとワークフローが任意の補助として紹介する。第一親の履歴からモジュールの組と分母を集計する | 現在の責務表と sqlc の配置を過去のパスへ当てる。走査数と一括変更の除外条件を解釈する必要があり、モジュール移動にも追従する | 廃止。対象のパスを指定した `git log --first-parent --name-status` で実際の変更シナリオを読む。組の自動集計と分母は失うが、合否の検査と境界判断の入力は維持する |
| `spec-route` | 仕様フォーマットが追跡の入口として紹介する。例のエラー型、経路、scope から HTTP 操作を順位付けし、兄弟 ID を引くテストを探す | TypeSpec と生成 Go を専用の正規表現で読むため、その表現と文書配置への追従が要る。候補は被覆や到達の証拠ではなく、全候補の精査は別途必要 | 廃止。`rg -n -F -- <ID> docs spec backend frontend`、`brief`、生成サイトの Traceability を使い、操作の契約とテストの観測を読む。エラー型を結合した操作の順位付けは失うが、規範 ID の被覆検査は維持する |
| `time-lint-go` | 個別タスクだけが入口。lint の debug ログからアナライザー別の合計、件数、最大値を集計する | 非公開の debug 表現を解析し、毎回キャッシュを消して Go 全体を lint する。通常の lint と別の timeout、実行引数を保守する | 廃止。`time-verify` でゲートを特定し、必要なときだけ `GL_DEBUG=goanalysis/analyze mise run lint-go` のログを読む。集計表と割合は失う。`clean-lint-cache` と標準 lint は維持する |
| `check-go-internal-visibility` | バックエンド設計が説明する個別タスク。独立した小さなモジュールで Go 自身の import 拒否を検査する | 製品の配置を読まず、fixture と Go module を別に維持する。製品の境界の逸脱は検出しない | 廃止。通常の製品ビルドとテストのコンパイル、`check-boundaries` が同じ禁止 import を拒否する。言語仕様そのものの canary は失うが、製品の公開目的の照合は維持する |

残す対象も役割ごとに比較した。
個別入口を減らすだけでは実装の保守費用が減らず、局所検証を長くするため、registry の入口は残す。

| 対象 | 維持する理由 |
| --- | --- |
| registry の検査、負債の増加検査、生成物との照合 | 文書の形式と参照、公開パッケージと SQL の所有、契約と実装、規範の引用、抑止の期限は異なる故障を検出する。共有入力だけを理由に同一視できない |
| `verification`、`report-contract-evidence`、`report-security-test-gaps`、`spec-review-candidates` | 実行、未解析、拒否後の作用、仕様を引くテストの被覆を別の事実として扱う。製品の細部を読まずに廃止できる同等の代替がない |
| `brief`、`changed-packages`、`spec-diff`、`time-verify` | 着手文脈、逆依存を含む検査対象、規範差分、集約タスクの計測を一次情報から導く。標準手順が使う範囲を維持する |
| `spec-where` | `rg` の薄い入口だが専用実装もテストもなく、複数の文書から利用される。廃止しても保守する実装は減らない |
| 生成、文書サイト、採番、作業項目の移動 | 派生物と参照を同期する既存の操作であり、手作業への移行は同期漏れを増やす |
| 依存管理、開発スタック、CI、デプロイと運用のタスク | リポジトリの標準入口であり、今回の汎用 CLI の削減対象ではない |

廃止対象は共有実装から参照されず、逆方向に使っている `boundary-fitness` と Gherkin 解析には残る利用者がある。
外部依存はすべて残るツールが使うため、`package.json` と lockfile は維持する。
未完了項目のツール追加計画（wi-458、wi-603、wi-606、wi-12283、wi-99632）と横断計画 wi-98584 を検索した。
廃止対象への依存や再導入計画はなく、その計画の採否は今回変えない。

### 文書の整理先

| 重複 | 所有元と整理 |
| --- | --- |
| ローカル開発の並列検証、依存の準備、失敗収集、`brief` の出力説明 | ワークフローの検証の段階と作業文脈の導出へ参照を寄せる。起動、ビルド、生成と計測の入口は残す |
| CONTRIBUTING の仕様先行、生成、作用の検査と検証順序 | ワークフローの該当節へ寄せ、PR の単位、コードレビューとコミットの規則は短い入口として残す |
| `update-design` の文書配置と割り当ての再掲 | 文書ガイドと設計形式への条件付き参照へ寄せる。更新の契機と検査は残す |
| `tools/check/README.md` の過去の件数、採否表、故障注入の履歴 | 既存の完了記録とテストが持つ事実を現在状態の説明から削る。検査とテストを採る判断基準は残す |
| `tools/README.md` の check の実行方式、検証結果と棚卸しの説明 | `check/README.md` とテスト方針へ参照を寄せる |

### 証拠と最終検査の選択

製品の Acceptance と Unit の振る舞いは変えないため、代替として削除対象の存在検査を実装前に失敗させる。
受け入れ代替は四つのタスク定義の不在、単体代替は専用ディレクトリ、解析実装と fixture の不在を検査する。
故障モデルは削除したタスクの呼び出し残り、残る共有処理の参照切れ、禁止 import の検出喪失である。
呼び出し残りは廃止後のタスクを CI へ一時注入し、`check-command-map` が拒否することを観測する。
禁止 import は残る `boundary-fitness.test.ts` の組み立て地点から `internal/` への拒否ケースで代替能力を確認してから fixture を削る。

最終検査には `check-command-map`、`check-agent-guidance`、`check-work-items`、`check-links` を選ぶ。
共有のタスク集合を使う利用者には `test-tools-file` の command-map、agent-guidance、registry、verification-tasks、mise-config、brief、timing を選ぶ。
削除後の型と lint は `typecheck-tools` と `lint-tools` で検査する。
タスク一覧と文書が集約入力にもなるため、最後は `verify-spec` で全ツールのテストと登録検査、生成サイトを確認する。
本番コード、TypeSpec、製品テストを変えないため、Go の全製品テストと UI 検証は追加しない。

### 実行順序

1. 着手時に `brief` から今回読む節と検証候補を取得し、変更対象と停止する範囲を確定する。
2. ツールと文書の候補を絞り、削除、参照への置換、維持の判断と理由をこの項目へ記録する。
   削除対象と失う能力の評価は、実装に入る前に確定する。
   ツール追加を計画する未完了作業項目も確認し、競合する計画があれば参照と判断を更新する。
3. 所有する開発規則を更新し、対象の実装、専用テスト、不要になった依存、タスク、CI とスキルの呼び出しをまとめて整理する。
4. 文書の重複を削り、参照先と入口を更新する。
5. 残る利用者と検査を確かめ、減った保守対象と失う能力を完了記録へ残す。

## タスク

- [x] T001 [Plan] ツールの費用と価値、文書の重複箇所と所有元を比較し、削除候補と維持理由を記録する。
  `check-work-items` は成功した。四つのタスクの不在検査は定義の 4 行を出して終了 1、専用実装の不在検査も終了 1 だった。
  `test-tools-file -- check/src/boundary-fitness.test.ts` は 27 件成功し、`rejects a composition point importing an undeclared package of an internal module` が同じ禁止 import を拒否した。
- [x] T002 [Docs] 廃止対象を前提とする開発規則と未完了の計画を同期する。
  共変更の履歴の読み方、ID の追跡、lint ログの取得、禁止 import の関門を所有文書へ反映した。
  未完了の計画に廃止対象との競合はなく、変更を加えない。`check-links` は成功した。
- [x] T003 [Tooling] 対象ツールと専用の実装、テスト、依存を削り、タスク、registry、CI、スキルの呼び出しを更新する。
  四つのタスクと 15 ファイルを削り、不在検査は GREEN になった。共有の registry、CI、外部依存の変更は不要だった。
  局所 7 ファイルの 66 テスト、`check-command-map`、`check-agent-guidance`、`typecheck-tools`、`lint-tools` が成功した。
  CI の lint の入口を一時的に `mise run spec-route` へ変えると、`check-command-map` が `workflow calls a task mise.toml does not define: spec-route` と終了 1 を返した。
  注入を戻した後は同じ検査が成功し、CI の差分がないことを確認した。
- [x] T004 [Docs] 重複する本文を削り、所有元への参照と索引を更新する。
  ローカル開発、CONTRIBUTING、ツール README、設計更新スキルを所有元への参照に整理した。
  検査 README の過去の棚卸し表と故障注入の履歴を削り、現行の判断基準と実行方式を残した。
  `check-links` と `check-work-items` は成功した。`check-agent-guidance` は直接参照 3 件の欠落を拒否したため、全体索引、バックエンド設計、モジュールの入口を戻して成功した。
- [x] T005 [Verify] 残る検査と呼び出し元を検証し、削減結果、維持した対象、残るリスクを記録する。
  `verify-spec` はサンドボックス内では Go 製ツール 3 件がビルドキャッシュの権限で失敗した。外で再実行すると全構成要素が成功し、994 テスト、生成文書 445 件、API 操作 342 件を検査した。
  廃止名の参照検索は、現在状態の文書、CI、スキル、ツール、タスクに一致なしだった。七つの視点で差分を読み、共有処理と独立した故障の検査を残していることを確認した。

## 検証

- 起票時は `mise run check-work-items` と `mise run check-links` で形式と参照を確認する。
- 実装時は、削除前後の参照検索で呼び出し残りと共有実装の利用者を確認する。
  `mise run check-command-map` と `mise run check-agent-guidance` で、残る CI とスキルの入口が実在することを確かめる。
- ツールの局所変更は `mise run test-tools-file -- <tools/ からの相対パス>` で残る利用者を検査し、`mise run typecheck-tools` と `mise run lint-tools` で不要な参照と型の不整合を検出する。
  検出能力を引き継ぐ場合は、同じ代表的な違反を引き継ぎ先が拒否することも確かめる。
- registry、共有処理、集約タスクへ波及した場合は `mise run verify-spec` を選ぶ。
  廃止する検査の代替証拠は、実装前に選んだ残存タスクまたは具体的な手動確認として記録する。
- 文書変更は `mise run check-links` と `mise run check-work-items`、文書の配置や形式に触れた場合は `mise run check-spec` を追加する。
  製品コードの不要物も削除する場合に限り、影響パッケージを対象とする `mise run test-go-changed` または該当 UI 検証を追加する。
  最終差分に合わせて実行対象と理由を確定し、全製品テストの調査を削減判断の前提にしない。

## リスク

- 低頻度でも重要な検査を削る恐れがある。
  頻度だけで決めず、検出する故障、代替、損失の重大さを比較する。
- 文書の重複削除で、固有の例外や現場で必要な説明を落とす恐れがある。
  所有元と差分を比較し、固有の内容を移してから参照へ置き換える。
- 削除した入口を CI、スキル、未完了の計画が使い続ける恐れがある。
  呼び出しと依存を同時に整理し、残る利用者の実行を確認する。
- 調査が製品コード全体へ広がると、削減による利益を調査費用が上回る。
  細部の精査が必要な候補は対象外にし、明確な削除と重複解消から完了させる。

## 完了

- **完了日**: 2026-10-12
- **要約**:
  `mise run spec-diff` は `no normative specification change against main` を返した。
  製品の振る舞い、規範要件と TypeSpec は変えていない。
  `report-change-coupling`、`spec-route`、`time-lint-go`、`check-go-internal-visibility` と専用の 15 ファイル（1,100 行）、タスク定義 44 行を削除した。
  検査 README の過去の棚卸し表と、ローカル開発、CONTRIBUTING、設計更新スキル、ツール README の規則の再掲を削り、所有元への参照へ整理した。
  registry の検査、CI のゲート、共有処理、外部依存、製品コードと製品テストは維持した。
- **受け入れ RED の証拠**:
  - **テスト**: 四つの廃止対象タスクの定義がないことを `rg` の検索と条件分岐で検査した。
  - **要件**: N/A: 製品要件を変えない開発補助の削除であり、正式なタスク入口を代替境界とする。
  - **観測した失敗**: 実装前は定義の 4 行を出力し、終了 1 だった。削除後は一致なしで終了 0 だった。
  - **検出できる理由**: 実装だけ消してタスクを残す整理と、タスクも廃止した結果を区別できる。
- **単体 RED の証拠**:
  - **テスト**: 専用ディレクトリ、lint ログ解析実装、Go fixture の不在をシェルの `test` で検査した。
  - **要件**: N/A: 内部の製品ロジックを変えず、専用実装の到達性と削除を代替境界とする。
  - **観測した失敗**: 実装前は存在するため終了 1、削除後は終了 0 だった。
  - **検出できる理由**: 入口だけを消して専用実装を放置する整理を区別できる。新しい恒久検査は追加していない。
- **変更耐性の結果**:
  CI へ廃止済みの `mise run spec-route` を一時注入すると、`check-command-map` が `workflow calls a task mise.toml does not define: spec-route` と終了 1 を返した。
  注入を戻すと成功し、CI に差分は残っていない。
  `boundary-fitness.test.ts` の禁止 import のケースを含む 27 件の成功で、残る関門の能力も確認した。
  文書整理では必須の直接参照を落とした時点で `check-agent-guidance` が 3 件を拒否し、参照を戻すと成功した。
  検査は照会の順位付け、共変更の集計、アナライザー別の集計表の喪失を補わない。
  これらは採否表に記した意図的な廃止であり、必要な個別調査は既存 CLI と一次情報から行う。
  Go の製品変更はないため、Go のミューテーションテストと特性化テストは対象外である。
- **検証結果**:
  - `check-work-items`、`check-links`、`check-command-map`、`check-agent-guidance` は、記録、文書と呼び出し残りの検査として成功した。
  - `test-tools-file` の boundary-fitness は 27 件、command-map、agent-guidance、registry、verification-tasks、mise-config、brief CLI、timing の 7 ファイルは 66 件成功した。
  - `typecheck-tools` と `lint-tools` は削除後の型と参照を検査し、成功した。
  - `verify-spec` は最初の実行で Go 製ツール 3 件がサンドボックスからのビルドキャッシュへの書き込みを拒否された。サンドボックス外で再実行し、全構成要素、994 テスト（71 ファイル）、生成文書 445 件と操作 342 件の検査が成功した。
  - 最終の文書整形後に `render-docs` が成功し、1,357 ページを再生成した。生成サイトと OpenAPI は追跡しない。
  - 完了記録を移動した後の `check-work-items`、`check-links`、`check-agent-guidance` は成功した。`check-published-api-compat` は未公開の契約として成功した。
  - 廃止名と専用実装への参照検索は、現在状態の文書、CI、スキル、ツール、タスクと未完了の計画（本項目を除く）で一致なしだった。
  - 七つの視点で差分をレビューし、実装と入口の削除を揃え、共有処理と独立した故障を検出するテストを保っていることを確認した。
- **対象外と残した事項**:
  全製品コードとテストの細部の重複調査、汎用 CLI の一括削減、依存更新、文書体系の再設計、完了済み項目の圧縮は行っていない。
  全製品テスト、UI E2E と Go の変更耐性検査は本番への差分がないため未実施であり、本項目の未完タスクはない。
