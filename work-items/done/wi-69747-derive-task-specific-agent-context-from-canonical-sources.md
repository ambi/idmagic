---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  references: []
  reason: "初回公開前の開発用照会の改善であり、既存データと実利用契約への影響はない。"
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/writing-language.md
    - docs/formats/design-format.md
    - docs/formats/documentation-guide.md
    - docs/development/release.md
    - docs/design/application/design-guidelines.md
    - docs/design/architecture/logical.md
    - docs/modules/identity-management/README.md
    - docs/modules/identity-management/design/README.md
    - docs/design/architecture/risks.md
    - docs/requirements/quality.md
  typespec: []
  source:
    - AGENTS.md
    - mise.toml
    - tools/README.md
    - tools/brief/src
    - tools/check/src/agent-guidance.ts
    - tools/check/src/markdown-links.ts
    - tools/workspace/src/workspace.ts
    - .agents/skills/implement-work-item/SKILL.md
  tests:
    - tools/brief/src/brief.test.ts
    - tools/check/src/agent-guidance.test.ts
  stop_before_reading:
    - backend
    - frontend/src
    - spec/modules
spec_impact:
  kind: none
  reason: "AI が着手時に読む規約、設計、仕様、検証タスクの候補を一次情報から導出する。製品の HTTP 応答、認証と認可、永続状態、イベント、外向きの通知は維持する。"
---

# 作業種別と変更対象から、AI が読む最小の文脈と検証タスクを導出する

## 動機

brief は affected_spec と識別子への言及から読む候補を返すが、仕様影響のない作業では起点がなく、共有設計、品質要件、作用の確定境界も自動でそろわない。
AGENTS.md は作業種別ごとに文書を読むよう求め、[コンテキストの節約](../../docs/development/specification-first-workflow.md#8-コンテキストの節約)は無関係な方法論文書を事前に読み込まないよう求める。
どの節が今回必要かの判断が AI に残る。

現在は未リリースであり、当面も未リリースを継続する。
リリースや互換性の準備より、日々の設計と開発の探索コストを下げる方を優先し、p1 とする。
初回リリースの準備を、この改善の前提にしない。

## 対象範囲

- brief を、仕様 ID に加えて作業種別と明示した変更対象からも候補を導出できるようにする。
- 適用する規約の該当節、横断要件、公開契約、共有設計、品質要件、既知リスク、検証タスクへの案内。
- 各候補に、参照元と今回読む必要がある理由を表示する。
- 最初の候補を小さくし、関連履歴や隣接モジュールを追加取得できる形にする。
- AGENTS.md、リポジトリ内スキル、ワークフローの参照範囲を同じ導出にそろえる。
- 読む量と探索の往復を、代表的な作業で変更前後に比較する。

## 対象外

- リポジトリ全体の自動読込、ベクトル検索、外部の検索基盤の導入。
- 独立した手書きの仕様とコードの対応台帳。
- 参照や言及の存在から、実装済みまたは検証済みと判定すること。
- 起票時の initial_context の固定。着手時に実際に読んだ最小範囲を記録する。

## 設計

既存の文書の配置、モジュールの責務、明示した参照、コードの構造、mise のタスクを使って候補を導く。
仕様影響のない作業では、作業種別と対象パスを起点とし、製品の規範 ID を架空に作らない。
規範上の参照と、探索用の候補を区別し、意味を推定した候補はその理由と限界を表示する。
全関連情報を一度に展開せず、必要な契約と規約の該当節を先に返す。

着手時に、起点の指定方法、節と対象の選択規則、追加取得の単位を決める。
候補は initial_context の下書きに留め、実際に読む対象と止める範囲は作業者が確定する。
規約の内容をツールへ複製せず、一次情報の節を参照する。

### 独立に決めた代表例

| 作業 | 起点 | 必要な情報 | 初期取得から除く情報 |
| --- | --- | --- | --- |
| 機能変更 | feature、backend/idmanagement/user | 仕様影響、証拠契約、コード規約、テスト方針、IdManagement の公開契約と品質、関連 TypeSpec の候補 | 他モジュール、履歴、全設計領域の本文 |
| 構造変更 | refactor、backend/idmanagement/user、boundary | 境界を選ぶ判断手順、責務表、バックエンドの依存規則、既知リスク | 無関係な機能の要件本文 |
| 文書変更 | docs、docs/development/writing-language.md | 文章の言語、該当文書形式、リンク検査 | Go と UI の規約、製品契約、TypeSpec |
| tooling 変更 | tooling、tools/brief | コード規約、ツールの入口、tooling のテストと型検査 | 製品仕様、ブラウザー検査 |
| 横断作用 | refactor、backend/idmanagement/user、effects | 作用の境界、一貫性の境界、モジュールの確定と監査の所有文書 | 隣接モジュールの自動展開 |

現行 brief は仕様参照のない四例に空の下書きを返す。
AGENTS.md の条件を手で照合し、文書を開いて見出しを探し、mise tasks から局所検査を探す往復が必要である。
変更後は同じ独立した必要情報を検査し、初期候補の本文行数と、その候補のファイル全体の行数を比較する。
探索の往復は実時間ではなく、起点から必要な参照とタスクをそろえる照会回数で比較する。

### 採用する境界と操作

起点は既存の change_kind と affected_spec、および繰り返せる CLI オプション --target、--kind、--concern とする。
製品の結果を変える未解決の問いはない。
concern は boundary、effects、contract、security、performance とし、パスだけで変更意図が決まらない場合に明示する。
履歴と隣接モジュールは --include history と --include neighbors で追加する。
CLI の対象はリポジトリ内の既存パスに限り、initial_context を起点として再利用しない。

主要型は ContextSeed（kind、targets、concerns）、ContextCandidate（location、origin、reason、basis）、ContextPlan（文書と検証タスクの候補）とする。
deriveContext(seed, snapshot) は候補を返し、snapshot がファイルの取得境界、CLI が Git と表示の境界になる。
AGENTS.md の既存の参照表の行へ条件コメントを付け、リンク先と理由をその行から読む。
タスクの適用条件は mise.toml の当該タスクへコメントで添え、実行入口と説明は TOML から導く。
規約本文、仕様とコードの対応表、別の作業別台帳は作らない。
論理責務表のリンクとパッケージ接頭辞から所有モジュールを決め、公開契約と品質を案内する。
作用の所有文書は設計索引のリンクと説明から候補を選び、意味による推定であることを表示する。
候補の参照元、理由、明示参照か配置または意味の推定かを常に表示し、実装と検証の完了を主張しない。
参照表がなくても仕様参照の既存表示を保つ。
リポジトリの製品モジュールと公開パッケージの境界は変えない。

### 発見と分類

既存の coverageLine は、名前を含むファイルがない場合に「実装がない」と表示していた。
これは言及を証拠としない対象範囲と矛盾するため、開発用表示の誤りとして修正した。
製品の結果は変えず、要件の追加も行わない。
TypeSpec の候補は Go のディレクトリ名ではなく、責務表の文書リンクが所有するモジュール名から導く。
設計索引の意味から作用の所有を推定する限界は task-context.md へ移し、完全な対応台帳は作らない。

### 読込範囲と探索の比較

比較対象は同じ候補の Markdown 文書集合であり、変更前の文書を全文で読む上限と、変更後の選択節の行数を比較した。
変更前の本文は作業開始コミット bd8a16542、変更後はこの作業ツリーを使った。
節は Markdown の構造で区切り、同じファイルの重複する節は一度だけ数えた。
TypeSpec と実装本文、エージェントが実際に消費したトークン数、実時間はこの計測に含めない。

| 代表例 | 文書数 | 変更前の全文上限（行） | 変更後の選択節（行） | 文脈候補 | 検証タスク |
| --- | --- | --- | --- | --- | --- |
| 機能変更 | 9 | 1,636 | 464 | 15 | 3 |
| 構造変更 | 13 | 1,953 | 623 | 19 | 4 |
| 文書変更 | 4 | 1,748 | 424 | 6 | 2 |
| tooling 変更 | 6 | 1,255 | 411 | 8 | 4 |

現行ツリー同士の全文と節の比較は mise run test-tools-file -- brief/src/context-measurement.test.ts で再実行できる。
前後の差は読込範囲の上限を示し、実際のエージェントがこの行数を読んだとは主張しない。
旧 brief は四例のような仕様参照のない記録へ文脈もタスクも返さず、brief、AGENTS.md の条件照合、mise tasks の検索という三つの照会段階を要した。
新 brief は一回で必要な場所と局所タスクを返す。
参照先の本文を読む操作と、故障に合う検証を選ぶ判断は前後とも必要であり、照会回数の比較から除いた。
履歴と隣接候補はどちらも初期出力へ混ぜない。

| 案 | 判断 |
| --- | --- |
| 既存の brief と生成サイトの探索を拡張する | 採用候補。既存の一次情報と構造を再利用できる |
| すべての関連文書を連結して返す | コンテキストコストを増やすため採らない |
| 手書きの作業別チェックリストを別に維持する | 規約の変更で写しが古くなるため採らない |

## 計画

1. 通常の機能変更、構造変更、文書変更、tooling 変更の代表例で必要な文脈と探索量を記録する。
2. 起点、候補の理由、追加取得の境界を決める。
3. 必要な横断契約の欠落と、無関係な文書の過剰取得を検出する RED を確認する。
4. brief と案内を更新し、同じ代表例で探索量を比較する。

## タスク

- [x] T001 [Inventory] 作業種別ごとの代表例と、必要な最小文脈を独立に決める。既存 brief の空の下書きと、上の代表例を確認した。
- [x] T002 [Design] 起点、参照の導出、候補の理由、追加取得の単位を決める。上の型、一次情報、追加取得の単位を確定した。
- [x] T003 [Tooling] brief と生成する案内を拡張する。mise run test-tools-file -- brief/src/context.test.ts brief/src/main.test.ts で、仕様参照のない tooling の規約、タスク、横断作用の候補が空になる RED を観測した。CLI の passes explicit targets through the mise entry and works without affected_spec と、導出の follows the effect index to the owner of commit and notification semantics を GREEN にした。規範 ID は N/A: 開発用照会。既存の coverageLine が言及の不在を実装不在と呼ぶ問題も does not infer missing implementation from absent naming files の RED を確認して修正した。typecheck-tools と lint-tools は成功した。
- [x] T004 [Docs] エージェント指示、implement-work-item、spec-change、update-design、ワークフローを task-context.md の同じ起点と追加取得へ同期した。check-agent-guidance と check-links は成功した。mise run test-tools-file -- check/src/agent-guidance.test.ts の rejects each entry that loses the canonical context selection pointer は、案内へのポインターの欠落を検査器が見逃す RED を観測してから GREEN にした。規範 ID は N/A: エージェント向け案内の同期。
- [x] T005 [Verify] 代表例の欠落と過剰取得、TypeSpec の起点、隣接本文の未取得、直接の規範参照の維持、不正な CLI 入力を検証した。対象の 41 テストと、導出の二種類の故障注入が期待どおりに失敗することを確認した。読込範囲と照会段階の比較を設計に記録した。集約 verify は、文書の例と環境制約による初回失敗を解消した後に成功した。

T003 は mise run test-tools-file -- brief/src/context.test.ts と brief/src/main.test.ts で RED と GREEN を確認する。
受け入れ境界の代わりに CLI テストで、仕様参照がない場合の規約と検証タスクの欠落を観測する。
単体境界の代わりに導出テストで、横断作用の所有文書の欠落と、無関係なモジュールの過剰取得を観測する。
製品要件は N/A: 開発用照会であり、製品の規範要件を作らない。
T004 は mise run check-agent-guidance、T005 は対象テスト、mise run check-links、mise run check-work-items、mise run verify で検証する。
中リスクの変更耐性は、条件による絞り込みと作用索引への配線を外して、対応するテストが失敗することを確認する。

## 検証

- mise run test-tools-file で brief と参照導出の対象テストを実行する。
- spec_impact: none の作業でも、対象の規約と検証タスクに到達できる。
- 横断的な作用を変える作業では、その確定境界の所有文書が候補に現れる。
- 仕様への言及だけを実装や検証の証拠として表示しない。
- mise run brief、mise run check-agent-guidance、mise run check-links、mise run check-work-items を実行する。
- mise run verify を実行し、代表例の読む量と探索の往復、必要な参照の維持を記録する。

## リスク

探索量だけを減らすと、必要な横断契約まで落とし得る。
代表例の必要情報を独立に決めて欠落を検査し、取得量の削減と必要な情報への到達を別々に比較する。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は main に対する規範仕様の変更がないと報告した。
  製品の HTTP 応答、認証と認可、永続状態、イベント、外向きの通知は維持した。
  brief は仕様参照のない作業でも、作業種別と明示した対象から、規約の節と検証タスクを参照元と理由付きで返す。
  境界と作用の変更、TypeSpec の起点、公開契約、品質、設計索引と既知リスクを案内し、履歴と隣接モジュールは明示した場合に追加する。
  AGENTS.md、三つのリポジトリ内スキル、ワークフローと検査器を同じ導出に同期した。
  名前の言及の不在から実装の不在を断定する表示を修正し、参照と推定を実装または検証の証拠として扱わない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run test-tools-file -- brief/src/main.test.ts` の `passes explicit targets through the mise entry and works without affected_spec`。
  - **要件**: N/A: 製品の受け入れ境界を持たない開発用照会なので、mise を経由する CLI 検査を代替境界にした。
  - **観測した失敗**: 旧 CLI は `--target` を mise の入口から受け取らず、coding-style の節も test-tools-file も表示しなかった。
  - **検出できる理由**: 実際の mise タスクを起動して場所と実行タスクを表明するため、導出ロジックだけ通り、正式な入口の引数転送が外れた実装を拒否できる。
- **単体 RED の証拠**:
  - **テスト**: `mise run test-tools-file -- brief/src/context.test.ts` の `routes tooling with no normative reference to code guidance and narrow tools checks` と `follows the effect index to the owner of commit and notification semantics`。
  - **要件**: N/A: 製品のドメイン計算を変えない tooling 作業なので、一次情報からの候補導出を代替境界にした。
  - **観測した失敗**: 初期実装は空の候補を返し、コード規約、局所タスク、確定と通知を所有する audit-events の候補が欠けて失敗した。
  - **検出できる理由**: 独立に選んだ必要な場所と、無関係な場所の不在を同時に表明し、全候補を返すだけの実装でも通らない。
- **変更耐性の結果**:
  AGENTS.md の条件による絞り込みを外すと、文書変更に coding-style が混ざり、`keeps document changes away from production code rules and contracts` が失敗した。
  同じ接頭辞で始まる別のディレクトリを排除する検査も失敗した。
  作用の設計索引への配線を外すと、`follows the effect index to the owner of commit and notification semantics` と、所有リンクを差し替える検査が失敗した。
  両方の注入を戻して対象の 41 テストが成功した。
  新しい所有文書のリンクは索引だけを差し替えて候補にでき、隣接モジュールの本文を取得しないことも確認した。
  生き残った注入はないが、TypeScript の体系的な変異ではなく、二つの現実的な配線障害を選んだ検査である。
  Go のソースを変えていないため test-go-mutation は対象外とした。
  作用の説明の語による推定は所有文書の網羅を保証せず、その限界を一次情報の task-context.md に記した。
- **検証結果**:
  - `mise run setup`、初回 `mise run lint-go`、対象の 41 テスト、`typecheck-tools`、`lint-tools`、`brief`、`check-agent-guidance`、`check-links`、`check-work-items` は成功した。
  - `mise run verify` は成功した。tooling の 979 テスト、UI の 709 テスト、Go の race 検査、仕様と生成文書、依存、lint、型、UI ビルドを通過した。
  - 初回 verify は、案内文書の例に実在する作業番号を入れたことと、サンドボックス内の Go 解析およびポート起動の制約で失敗した。例を汎用記法へ直し、許可されたサンドボックス外での再実行が成功した。
  - 七つの視点から、条件の所有を一次情報に残すこと、既存 Markdown パーサーと snapshot を使うこと、作用の入力境界、推定の限界、局所テストと正式な入口を確認した。
  - ブラウザーへ到達するコードを変更しないため test-ui-e2e は対象外とした。
  - 完了記録を done へ移動した後の `mise run check-work-items` と `mise run check-links` は成功した。
