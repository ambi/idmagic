# check

`check/` は、リポジトリ検査を一つの深いモジュールとして提供する。
検査名と実装は `registry.ts` だけで対応づけ、`runner.ts` が共有 snapshot に対して選択した規則を並行実行する。
各規則は `process.argv`、`process.exit`、リポジトリ root の推測を持たず、所見を値として返す。

利用者向けの入口は `mise` タスクである。

```bash
mise run check
mise run check-spec
mise run check-work-items
mise run check-api-compat
```

`mise run check` は登録済みの TypeScript 検査を一回の Bun 起動で実行し、一つの規則が例外を投げても残りの結果を集める。
個別タスクは同じ registry から一つの規則を選ぶため、検査ごとの CLI shell は持たない。

`check-spec` の TypeScript 集合は registry の `spec` group に置き、`all` の部分集合にする。
生成契約と生成サイトのゲートは `check-spec-inputs` に一度だけ列挙し、`check` と `check-spec` が共有する。
`check-repository` は引き続き全登録規則を実行する。

既存バージョン API の必須互換性ゲートと告知の最低水準は、[公開状態](../../docs/development/release.md#公開状態と日常開発)を共通の入力とする。
`mise run check-api-compat` は集約に含めず、未公開でも保存済みベースラインとの比較を明示的に行う。

work item の識別子はファイル名の stem 全体である。
同じ stem を `work-items/active/` と `work-items/done/` に置くと `mise run check-work-items` が拒否する。
status と配置の不一致、および旧配置の `work-items/` 直下に残った記録も拒否する。
先頭の番号は割り当ての目安であり、同じ番号だけでは衝突としない。

## 規範 id の被覆

`mise run check-spec` は、宣言したすべての規範 id（`standards.md` の行と、実行可能な scenario example）を、いずれかのテストが名指すことを求める。
どのテストも名指さない id は検査に落ちる。
未検査の id を免除する台帳は無い。
標準の側の台帳は wi-495 が、具体例の側の台帳は wi-496 が空にして削除した。
同じ名前のファイルを置き直しても、どの検査も読まない。

## 用語

`mise run check-terminology` は、設計文書が採らないと決めた表記を拒否する。
対象は `docs/` の Markdown と、`terminology.ts` の `TERMINOLOGY_ROOT_DOCUMENTS` が挙げる root 直下の文書である。
生成物の `CONFIGURATION.md` と `ROUTE_PRIORITY.md`、および書かれた時点の記録である `work-items/` は読まない。

規則は `terminology.ts` の `TERMINOLOGY_RULES` が持つ。
1 件は、採らない表記、代わりに使う表記、そして残す共起からなる。

```ts
{
  term: '秘密',
  adopt: '「シークレット」',
  allow: [{ literal: '秘密鍵', reason: 'private key であってシークレットではない' }],
}
```

`allow` の literal は対象語を含み、その literal が覆う位置に現れた occurrence だけを通す。
「トポロジ」と「トポロジー」のように、採用した表記が採らない表記を含む場合もこれで扱える。

免除はこの規則表にしかない。
ファイル単位の免除は持たない。文書ごとに外せるようにすると、その文書だけ用語が戻ったことに誰も気付かないためである。
正当な用法が落ちたときは、理由を書いた `allow` を足す。

## 検査の判定基準

検査を追加または変更するときは、次の三つをすべて満たすかを判断し、下の表へ根拠を書く。

1. 規則の一次情報源が検査の外にある。
2. 通常の変更でその規則への違反が起こりうる。
3. 同じ違反を止める別の関門がない。

代替する関門があるという判断には、その入力、実行条件と検出する故障を示す。
一般的なレビューや、検査自身の単体テストが通ることだけを代替とは数えない。
削除候補がある場合は、違反を故意に作り、代替の関門の結果を観測してから削除する。
実行時間、行数、過去に誤りを止めた回数は、採否の基準にしない。

### 現行 registry の判定

2026-10-11 の着手時点には 25 件の登録規則、実装 11,491 行、テスト 12,706 行（50 ファイル）があった。
行数は `tools/check/src/*.ts` を `.test.ts` とそれ以外に分けた値であり、起票時の 18 件とは対象時点が異なる。
全検査を削除する根拠はなく、`agent-guidance` の保証範囲を縮小した。
各行の一次情報源が外部性、違反例が実在する変更経路、最後の列が他の関門の不在の根拠を示す。

| 検査 | 判定 | 検査の外にある一次情報源 | 通常の変更で起こる違反 | 他の関門では止まらない理由 |
| --- | --- | --- | --- | --- |
| `executable-examples` | 残す | [実行可能な具体例](../../docs/formats/specification-format.md#実行可能な具体例と派生表示)と例の入力データ | データだけを変え、Markdown の派生表示を更新しない | データのテストは表示を読まず、文書の構文検査は値の一致を見ない |
| `documents` | 残す | [文書ガイド](../../docs/formats/documentation-guide.md)、[仕様フォーマット](../../docs/formats/specification-format.md)、[設計フォーマット](../../docs/formats/design-format.md)、[規範IDの参照](../../docs/development/specification-first-workflow.md#規範idをテストから参照する) | 閉じた配置の外に仕様を作る、未引用の例を追加する | TypeSpec のコンパイルは Markdown の配置、状態表と引用集合を読まない。リンク検査は実在する誤配置を通す |
| `specification-rules` | 残す | [要件と例](../../docs/formats/specification-format.md#5-要件と例) | 要件の担保手段を改名したままにする、仕様なしの機能スライスを加える | 文書の構文検査だけでは Go の宣言と機能スライスの対応を照合できない |
| `boundary-debt-ratchet` | 残す | [検査と負債](../../docs/design/application/backend.md#検査と負債) | 新しい境界違反を台帳に加えて許してしまう | `boundaries` は現状と台帳の一致を見る。基準 revision から増えたかはこの検査だけが見る |
| `document-layout` | 残す | [一次情報源の割り当て](../../docs/formats/documentation-guide.md#一次情報源の割り当て)と [文書配置の形式](../../docs/formats/specification-format.md#2-仕様文書) | 許可する文書種別を追加したのに、人向けの配置図に載せない | `documents` は実際の配置を検査し、配置図の説明漏れを見ない |
| `work-items` | 残す | [作業項目フォーマット](../../docs/formats/work-item-format.md)と記録スキーマ | 着手時の文脈、完了証拠、依存先、告知を欠いた記録を作る | TOML、TypeScript、TypeSpec の検査は Markdown frontmatter と参照の整合を検査しない |
| `spec-impact` | 残す | [仕様影響の宣言](../../docs/development/specification-first-workflow.md#仕様影響の宣言) | `none` と宣言しながら規範仕様を変える、未申告の本番変更をコミットする | `work-items` は記録の形と参照を検査するが、Git の作業範囲と規範差分を照合しない |
| `links` | 残す | [参照の向き](../../docs/formats/documentation-guide.md#設計文書からコードへの参照)と文書の索引 | リンク先や見出しを移し、参照を更新しない | 文書の構文検査と生成サイトの検査は、ルート文書、スキル、変更中の記録まで同じ範囲を走査しない |
| `terminology` | 残す | [文書の見せ方](../../docs/formats/documentation-guide.md#文書の見せ方)、[用語集](../../docs/requirements/glossary.md)と各設計文書の採用語 | 設計の単位を旧名称へ戻す、技術用語を別の一般語へ置換する | 文書の構文検査と英文検査は日本語の同義語を受理する。規則表は採用語の検出用の対応であり、概念の定義は文書が持つ |
| `work-item-references` | 残す | [現在状態の文書](../../docs/development/specification-first-workflow.md#6-現在状態の文書) | 現在の設計を、完了後に更新されない作業記録へ委ねる | `links` はリンク先が実在すれば通し、参照の方向は判定しない |
| `boundaries` | 残す | [モジュール間の依存規則](../../docs/design/application/backend.md#モジュール間の依存規則)、[責務表](../../docs/design/architecture/logical.md#モジュールの責務)、[テーブル所有](../../docs/design/data/database.md#所有と書き込みの境界) | 共有ライブラリからモジュールへ依存する、他の所有者のテーブルを書く | Go は `internal/` を拒否するが、公開パッケージのモジュール循環、Domain の作用、SQL の所有は拒否しない。sqlc は SQL の妥当性だけを検査する |
| `raw-sql` | 残す | [ポートとアダプター](../../docs/design/data/database.md#ポートとアダプター) | 静的 SQL を sqlc の外で追加し、例外理由を書かない | Go は SQL を文字列として通し、sqlc は自分の入力にない呼び出しを検査しない |
| `command-map` | 残す | `mise.toml` と `.github/workflows/` | 個別タスクを削除し、CI の呼び出しを残す | 手元の `verify` は CI 固有の呼び出しを実行しない。actionlint はカスタムタスクの存在を知らない |
| `agent-guidance` | 縮小して残す | [文脈の導出](../../docs/development/task-context.md)、[開発サイクル](../../docs/development/specification-first-workflow.md#3-開発サイクル)、[作業記録の契約](../../docs/formats/work-item-format.md)、`mise.toml` | 手順から必須参照や `fault_model` を落とす、削除済みタスクを呼ぶ | `links` は参照自体が消えたことを検出できず、`command-map` は CI だけを読む。特定の言い回しと説明順は手順の意味を保証しないので検査しない |
| `admin-scopes` | 残す | [スコープ宣言](../../docs/design/application/api-guidelines.md#api-アクセストークンのスコープ宣言)と TypeSpec | 管理 API の注釈を落とす、別の名前空間の scope を使う | TypeSpec の拡張注釈はコンパイルでき、Go の認可テストは全契約の注釈の網羅性を検査しない |
| `contract-drift` | 残す | [HTTP ボディの宣言](../../docs/design/application/api-guidelines.md#http-ボディの宣言)、TypeSpec と Go の DTO | 契約だけにエンベロープを足す、DTO の項目を契約へ反映しない | TypeSpec と Go は独立にコンパイルでき、各操作の HTTP テストに宣言との全件照合はない。未解決を成功と数えない |
| `status-drift` | 残す | [ステータスコードの宣言](../../docs/design/application/api-guidelines.md#ステータスコードの宣言)と TypeSpec | ガードが返す 401 を契約に追加し忘れる | ボディの集合比較では状態コードを比較しない。HTTP テストは返却値を固定しても宣言の過不足を読まない。部分解析の限界は維持する |
| `event-contract` | 残す | [イベントの公開契約](../../docs/design/application/backend.md#イベントの公開契約)と TypeSpec のイベント語彙 | payload のキーを片側だけ改名する、宣言したイベントを構築しなくなる | 不透明な JSON のキーは Go の型検査で照合できず、import 検査にも現れない |
| `security-controls` | 残す | [拒否と作用の証明](../../docs/development/specification-first-workflow.md#レスポンスだけでは副作用を証明できない場合)、TypeSpec と拒否の例 | ガードで writer の nil を返す、契約が約束する拒否の例を落とす | status のテストだけでは拒否後の続行を通す。`status-drift` はコードと契約を見ても例の存在を見ない |
| `unspecified-vocabulary` | 残す | [仕様に書く範囲](../../docs/formats/specification-format.md#仕様に書く範囲)と各モジュールの要件 | エラーコードやイベントだけを追加し、振る舞いの要件を作らない | TypeSpec と Go は語彙の存在を受理する。引用検査は宣言済みの例の引用だけを調べ、存在しない要件を発見できない |
| `slo-references` | 残す | [閾値の所有](../../docs/design/observability/monitoring.md#閾値の所有)と品質要件 | SLO ID を誤記する、アラートの運用手順書を移す | promtool は PromQL とルールを検査するが Markdown の宣言や参照先は読まない。`check-monitoring` は標準ゲート外でもある |
| `schema-tables` | 残す | [スキーマの読み方](../../docs/design/data/database.md#スキーマの読み方)と PostgreSQL スキーマ | 新しいテーブルや LOGGED 区分を文書へ反映しない | `check-schema` はデータベースと SQL の収束を見るが、Markdown の一覧を読まない |
| `vulnerability-suppressions` | 残す | [依存関係の脆弱性](../../SECURITY.md#依存関係の脆弱性)と `osv-scanner.toml` | 理由や期限を欠いた抑止を追加する | OSV-Scanner は理由と期限を任意とし、未設定なら無期限に抑止する。既知脆弱性の走査を成功させても理由の欠落は止まらない |
| `api-compat` | 個別入口だけ残す | [破壊的変更の検出](../../docs/design/application/api-guidelines.md#破壊的変更の検出)と保存したベースライン | 任意の比較時に既存の項目の削除を見落とす | 現在契約との一致は過去契約との互換性を保証しない。初回公開前には集約から外し、明示した比較だけで実行する |
| `published-api-compat` | 残す | [公開状態](../../docs/development/release.md#公開状態と日常開発)と `spec/release-state.json` | 未公開に比較を強制する、公開後に比較を省略する | 個別の `api-compat` は公開条件を持たない。この検査が実行条件を所有し、状態ファイルの異常を免除にしない |

登録の 25 件は、`all` に属する 24 件と明示比較の 1 件である。
個別の `mise` タスクは、文書とスキルから検査を名指す入口として維持する。
`documents` と `links` の一部、`boundaries` とその増加検査の一部は同じ入力を読むが、それぞれ対象範囲や比較軸が異なるため、一方を代替とはしない。

## テストの判定基準

そのテストを消すことで素通りする誤実装を具体的に名指せる場合に残す。
表はテストファイル単位の判定であり、故障例は各ファイルが固定する契約の代表例である。
正常系も、無条件拒否や過剰な検査という誤実装を区別する場合は残す。
CLI の受け入れテストは純粋関数のテストと値が同じでも、読み取りや登録の配線を外す別の故障を検出する。

### テストファイルの判定

ファイル名は `src/` からの相対名である。
着手時の 50 ファイルを対象とし、タスク読み取りの新しい配線には `check-agent-guidance.test.ts` を加えた。

| テストファイル | 判定 | 消すと見落とす誤実装の例 |
| --- | --- | --- |
| `admin-scopes.test.ts` | 残す | 管理 API から scope の注釈が消えても通す、SCIM の scope を管理用と数える |
| `agent-guidance.test.ts` | 書き換える | 必須参照と契約フィールドの欠落を通す、言い換えを拒否する、集約コマンドの後続タスクを照合しない |
| `api-compat.test.ts` | 残す | `$ref` と隣接する制約を片方だけ読む、項目の削除や必須化を通す、互換な追加を拒否する |
| `boundary-debt-ratchet.test.ts` | 残す | 台帳へ加えた新規違反を許す、追加した書き込みカラムを既存違反と数える |
| `boundary-fitness.test.ts` | 残す | パッケージ循環をモジュール循環と取り違える、公開方式や所有者のないコードを黙って無視する |
| `canonical-document-set.test.ts` | 残す | 機能スライスに共有語彙を許す、拡張子の大小を違法配置として報告せず候補名も誤る |
| `check-api-compat.test.ts` | 残す | 公開条件の分岐を逆にする、状態ファイルの欠落を未公開として免除する |
| `check-boundaries.acceptance.test.ts` | 残す | sqlc の別ディレクトリの入力やテーブル所有の照合を実際の読み取りから外す |
| `check-links.test.ts` | 残す | 変更していない完了記録を再検査する、変更した完了記録まで除外する |
| `check-work-items.test.ts` | 残す | 読み取りを繰り返す、公開後も告知を免除する、旧配置の記録を列挙から落とす |
| `command-map.test.ts` | 残す | tools の宣言を task と数える、削除した CI 用タスクを通す |
| `contract-drift.test.ts` | 残す | ネストした response を平坦化する、異なる成功ボディを解決済みと数える、同形の曖昧な route を一意とする |
| `docs-work-item-links.test.ts` | 残す | リリース文書まで拒否する、実在する work item への参照を現在状態の文書に許す |
| `document-layout-format.test.ts` | 残す | 配置図の段を読み損ねる、旧文書形式だけを求めて機能仕様の欠落を見逃す |
| `documentation-impact.test.ts` | 残す | 別の作業記録へ仕様差分を帰属する、未公開期間の告知を遡及要求する、昇格証拠を省略する |
| `event-contract.test.ts` | 残す | エンベロープを payload に混ぜる、別の型の構築をイベントの発行と数える |
| `executable-examples.test.ts` | 残す | 入力と期待結果の対応変更を見逃す、コード例の marker を実行する、親パスを許す |
| `feature-layout.acceptance.test.ts` | 残す | 文書の木の列挙から例や内部設計を外す、委譲先と品質要件へのリンクを解決しない |
| `gherkin-scenarios.test.ts` | 残す | Background を別の Rule に適用する、Outline の行値や Unique の wildcard を誤る |
| `lib.test.ts` | キー一覧の固定だけ削除 | 登録 schema を呼ばず無条件受理する、enum、必須フィールド、証拠の境界を許す、診断を誤った行へ返す |
| `markdown-links.test.ts` | 残す | コード例をリンクと数える、未使用の参照定義や改行したリンクを無視する |
| `mise-config.test.ts` | 集合の長さだけの固定を削除 | 並列と逐次の集合をずらす、lockfile の走査や UI ゲートの配線を落とす |
| `mutation-canary.test.ts` | 残す | lived と not viable を取り違えて誤った変更耐性を報告する |
| `normative-coverage.test.ts` | 残す | 文章中の id を被覆と数える、親規則の引用だけで子の例を被覆済みにする |
| `operation-verification.test.ts` | 残す | 二つの未解決理由を持つ同じ操作を二度数える |
| `primary-use-case-evidence.test.ts` | 残す | 故障モデル、E2E の理由、実在テストや標準タスクからの到達を検査しない |
| `production-code.test.ts` | 残す | 生成物を手書き本番コードと数える、例外マーカーをファイル全体の免除にする |
| `raw-sql.test.ts` | 残す | 引数のない `Query()` を DB 呼び出しと数える、離れた例外印を許す |
| `registry.test.ts` | 残す | 同じ検査を二重登録する、明示比較を集約に入れる、個別 selector を取り違える |
| `repository-checks.acceptance.test.ts` | 残す | runner から規則を外す、root の用語と閉じた文書配置を読み取らない |
| `runner.test.ts` | 残す | 一つの規則の例外で全結果の収集を打ち切る |
| `schema-tables.test.ts` | 残す | SQL のコメントをカラムと数える、LOGGED と tenant_id の区分を文書と照合しない |
| `security-controls.test.ts` | 残す | writer のラッパーを追わない、別パッケージの同名関数をガードと取り違える |
| `slo-references.test.ts` | 残す | 文章中の SLO を宣言と数える、flow 形式の annotation を読まない、page の手順書を求めない |
| `spec-diff.test.ts` | 残す | DocString、値だけの変更、TypeSpec の本文変更を見逃す、移動や整形だけを仕様変更とする |
| `spec-impact.acceptance.test.ts` | 残す | checkpoint と移動した記録の Git 履歴を誤り、着手後の仕様変更を比較範囲から落とす |
| `spec-impact.test.ts` | 残す | `conforms` を仕様差分の申告とする、別作業の変更を `none` と矛盾させる |
| `specification-doc.test.ts` | 残す | 宣言の H1、状態表、遷移表、マトリクスと標準の閉じた語彙を検査しない |
| `specification-rules.test.ts` | 残す | 担保手段の型や上位の要件を取り違える、EARS の主体、句の順序と応答を受理しすぎる |
| `status-drift.test.ts` | 残す | mapper の全分岐を各操作の返却と数える、未読があっても過剰宣言を指摘する、同名 guard の所属を誤る |
| `table-writes.test.ts` | 残す | CTE、upsert と TRUNCATE の書き込み先を漏らす、未解決 SQL を成功扱いする |
| `terminology.test.ts` | 残す | 許可共起が一つあるだけで行全体を免除する、採用語に含まれた旧表記まで拒否する |
| `typespec-declarations.test.ts` | 残す | 文字列内の波括弧で宣言を分割する、文字列の空白や decorator の変更を落とす |
| `unspecified-vocabulary.test.ts` | 残す | 文書中の雑談を要件と数える、解消した語彙を台帳に残しても通す |
| `verification-tasks.test.ts` | 関係の検査を追加して残す | `depends` だけを読み集約を空にする、共通生成ゲートを片方から外す、`spec` group の要素を失う |
| `vulnerability-suppressions.test.ts` | 残す | 引用符付き日付や table を期限とする、期限当日を有効にする、パッケージ抑止を見逃す |
| `work-item-dependencies.test.ts` | 残す | stem の重複、間接循環、未知の依存を通す、番号割り当ての境界を一つずらす |
| `work-item-markdown.test.ts` | 残す | 日本語見出しを記録フィールドに戻せない、主要ユースケースの YAML を読み落とす |
| `work-item-references.test.ts` | 残す | 言及だけの要件を宣言と数える、着手前から読書一覧の存在を要求する |
| `worktree-exclusion.test.ts` | 残す | ルート走査に `.worktrees/` を混ぜ、別ブランチのリンクやコードを検査する |
| `check-agent-guidance.test.ts` | 追加 | 手順だけを読んで `mise.toml` のタスク集合を渡さない、別 workspace のタスクで判定する |

### 削除と縮小の観測

`lib.test.ts` の schema のキー集合を直接固定する表明を一時的に外し、登録先を `work-item-disabled` に変えた。
残る schema 検証の 25 テストが失敗したので、登録欠落はそちらで検出できる。
別の正当な schema の追加まで禁止する理由はなく、キー集合だけの表明を削除した。

`mise-config.test.ts` の `verify` のメンバー数が 1 より大きいという表明を一時的に外し、集合を `check` だけにした。
並列と逐次の集合一致、および UI ゲートを含むことの 2 テストが失敗した。
集合の長さだけでは具体的な欠落を特定できず、残す関係の検査が同じ故障を止めるため削除した。

手順の文言と出現順を固定していたテストは、参照を保つ言い換えの受理、参照を消す変更の拒否へ書き換えた。
言い換え受理のテストは変更前に 9 件の文言欠落で失敗した。
縮小後に参照の検査を無効にすると 4 テストが失敗し、共通生成ゲートの配線を外すと集合の関係のテストが失敗した。
故障はすべて戻し、局所テストを再実行した。
全検査を削除する候補は採らなかったため、未観測の代替関門を根拠にした検査の削除はない。
