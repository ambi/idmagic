---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-06
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの検証手順と道具だけを変え、利用者向けの機能差分を生まない。"
  references: []
spec_impact: { kind: none, reason: "検証と着手支援の道具立てだけを変え、製品の観測可能な振る舞いも公開契約も変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - WORK_ITEM_FORMAT.md
    - docs/development/specification-first-workflow.md
    - docs/development/local-development.md
    - docs/development/continuous-integration.md
    - .github/workflows/idmagic-ci.yaml
    - tools/check/src/main.ts
    - tools/check/src/lib.ts
    - tools/check/src/check-specifications.ts
    - tools/check/src/command-map.ts
    - tools/render-spec-docs/src/main.ts
    - tools/workspace/src/check-workspace.ts
    - backend/shared/security/passwords_argon2id
    - frontend/package.json
    - frontend/bunfig.toml
    - frontend/src/test/register-dom.ts
    - frontend/tests/e2e
    - .agents/skills/implement-work-item/SKILL.md
  tests:
    - tools/check/src/lib.test.ts
    - tools/check/src/mise-config.test.ts
    - tools/check/src/work-item-markdown.test.ts
    - tools/workspace/src/check-workspace.test.ts
    - backend/shared/security/passwords_argon2id
    - frontend/tests/e2e
  stop_before_reading:
    - spec/contexts
    - docs/contexts
    - backend/oauth2
    - backend/saml
    - frontend/src
---

# work-item 1 件あたりの所要時間を、実測した律速から順に短縮する

## Motivation

work-item 1 件を実装しきるまでの時間が延びている。体感としての内訳は、着手時の関連コードと文書の把握、実装、検証、コミットの 4 つである。このうち検証とコミットと着手は、原因が道具と手順の側に測定できる形で残っており、推測なしで削れる。

### 実測

2026-09-06、8 コアの開発機で `verify` の構成タスクを 1 つずつ直列に実行した。作業ツリーには変更があり、Go のビルドキャッシュは温状態である。

| タスク | 実測 | 備考 |
| --- | --- | --- |
| `test-ui-e2e` | 下表のとおり | 5 本の spec を別々のプロセスで直列に走らせる |
| `lint-go` | 83 秒 | |
| `test-go-race` | 57 秒 | |
| `test-ui-unit` | 5 秒 | |
| `test-tools` | 4 秒 | |
| `compile-spec` | 1.1 秒 | |
| `check-api-compat` | 1.0 秒 | 依存の `compile-spec` を含む |
| ほかの `check-*`、`lint-*`、`typecheck-*`、`format-check-ui` | 各 1 秒未満 | |

`verify` は最大 8 並列で走るので単純な合計にはならない。1 秒台のタスクをどう並べ替えても総時間は変わらず、削る対象は `test-ui-e2e`、`lint-go`、`test-go-race` の 3 つに絞られる。以下の観察はすべてこの実測の上に置く。

`test-ui-e2e` は spec ごとに別プロセスなので、spec ごとに測った。

| spec | テスト数 | 実測 |
| --- | --- | --- |
| `ui-scenario-smoke.spec.ts` | 3 | 9.0 秒 |
| `localized-ui.spec.ts` | 1 | 8.6 秒 |
| `authorize-golden-path.spec.ts` | 2 | 6.4 秒 |
| `admin-session-recovery.spec.ts` | 1 | 5.6 秒 |
| `ui-scenario-actions.spec.ts` | 18 | 未測定 |

テスト 1 本の spec が 5.6 秒から 8.6 秒かかる。この下限はテストの中身ではなく、下記 O4 のスタック起動の費用である。spec が 5 本あるので、同じ起動を 5 回払っている。

`ui-scenario-actions.spec.ts` は起票時の観測では完走せず、`Expected a Response object, but received 'Response {...}'` を 3 回出して 31,129 行のダンプを吐いた。切り分けは T002 が行った。結果は Design の「T002 が切り分けたこと」に書いた。要点だけ言えば、完走しなかったのは 3 件分の未完了の変更のせいでも E2E の欠陥でもなく、観測に使ったエージェントのサンドボックスが `Bun.WebView` を動かせなかったためである。3 万行のダンプの方は本物の欠陥で、原因は別のところにあった。

### O1 検証ゲートは最初の失敗で全体を落とす

`mise run check` を実行すると、`check-links` が 745 ミリ秒で失敗した時点で走行中の `compile-spec` ごと打ち切られ、残り 15 個の検査は走らない。`verify` も同じである。したがって 1 回の実行で受け取れる失敗は 1 種類だけで、直しては数分かけて次の失敗を掘り出す往復になる。`lint-repo` は内部で 3 つの検査の失敗を集めて最後にまとめて出す形にしてあり、コメントにも「最初の失敗で止めず、落ちたものを全部並べて終わる」と書いてある。集約タスクだけがその規律から外れている。mise には `--continue-on-error` がある。

### O2 検証のはしごの狭い段と、最後のゲートがキャッシュを共有していない

[検証のはしご](../../docs/development/specification-first-workflow.md#5-verification-ladder) の第 3 段は `mise run test-go-package <package>`、すなわち `go test <package>` である。最後のゲートは `mise run test-go-race`、すなわち `go test -race ./...` である。`-race` は別のビルド構成なので、Go のテストキャッシュ項目は両者のあいだで共有されない。実装中に狭いテストを何度回しても、最後のゲートはその結果を 1 件も再利用しない。「部分テストと全体テストを両方走らせて損をしていないか」という問いに対する答えは、Go については「損をしている」である。損の出方はキャッシュの当たり外れではなく、狭い段が最後のゲートを一切前倒ししていないことにある。

### O3 `-race` の下では Argon2id のコスト設定が支配的である

`backend/shared/security/passwords_argon2id` の 5 テストは `-race` ありで 3.051 秒、なしで 0.569 秒だった。ハッシュ 1 回あたりおよそ 300 ミリ秒である。本番の OWASP 設定 (メモリ 19456 KiB、時間コスト 2) をそのまま組み立てるテストファイルは 27 ある。鍵導出関数そのものの正しさを検査するテスト以外に、この費用を払う理由はない。`test-go-race` の 57 秒のうちどれだけをここが占めるかは未測定であり、パッケージ別の実時間を取ってから判断する。

### O4 UI の E2E はスタックを 5 回起動する

`frontend/package.json` の `test:e2e` は 5 本の spec を `&&` で連結した 5 つの `bun test` プロセスである。各プロセスの `beforeAll` が `startE2EEnvironment()` を呼び、`go build ./cmd/idmagic`、API サーバーの起動、Vite 開発サーバーの起動、初期データの投入を毎回やり直す。テストは全部で 25 本、`Bun.WebView` の生成も 25 回、待ち合わせのポーリング間隔は `frontend/tests/e2e/` の 20 か所すべてが 150 ミリ秒である。ログインはフォーム入力で行うので、本番コストの Argon2id をテストごとに 1 回踏む。

上の spec 別の実測から、起動の費用はおよそ 5.5 秒である。5 回を 1 回にすれば、テストの中身を 1 行も変えずに 20 秒余りが消える。

あわせて、`startE2EEnvironment()` はポートがすでに使われていても止まらない。API と Vite は無条件に `spawn` され、束縛に失敗しても `waitForUp` は前の実行が残したサーバーに対して成功する。中断した実行の後始末が済んでいないときに、古いサーバーに対してテストが走る。単一起動へ移すときに、この取り違えも塞ぐ。

### O5 E2E は CI では走っていない

`.github/workflows/idmagic-ci.yaml` は `test-ui-e2e` を呼んでいない。ローカルの `verify` だけが律速を負担しており、しかも E2E の後退はサーバー側では捕まらない。どちらへ寄せるかを決めていない状態である。

### O6 成功した検査が大量の行を出す

`mise run check-work-items` は成功時に 500 行を出す。うち 1 件ごとの `ok` が 495 行を占め、これは記録の総数とともに増え続ける。`check-boundaries` は 2 行、`check-links` は 4 行で、成功時に黙る検査と全件を列挙する検査が混在している。エージェントはこの出力を読んで文脈に載せるので、行数はそのまま所要時間と文脈の消費になる。

### O7 着手時の読み込みに単一の入口がない

`docs/` は 13,104 行あり、`docs/contexts/oauth2/` だけで 2,187 行ある。Go は 297 パッケージ 1,550 ファイルである。[Context economy](../../docs/development/specification-first-workflow.md#8-context-economy) は work item の `initial_context` から読み始めよと書くが、`initial_context` は着手時に人またはエージェントが書くものなので、着手前には存在しない。つまり最初の探索だけは毎回素手で行われる。

素手でなくてよいはずの材料はすでにある。`tools/render-spec-docs/src/main.ts` の `collectTraces()` はリポジトリ全体を走査して、規範 ID ごとに「その ID を名指すコードとテスト」と「その ID を名指す work item」を集めている。ところがこの索引は Traceability の HTML を描くためだけに使われ、端末から引く手段がない。`mise run spec-where` はあるが、これは `docs`、コード、work items に対する 3 回の `rg` を見出し付きで並べるだけで、規範 ID から仕様本文、TypeSpec 記号、既存テスト、先行事例へはたどらない。

### O8 コミットが遅いのは 1 件ずつ切っていないからである

起票時の作業ツリーは 57 パスが変更済みで、wi-453、wi-454、wi-455 の 3 件分が同居していた。この状態からコミットを組み立てれば、どの変更がどの work item のものかを読み直す作業がまるごと発生する。`implement-work-item` の第 11 段は 1 件ごとのコミットを指示しているので、足りないのは手順ではなく実行である。「Completion に書いた内容の英語版がほぼそのままコミットメッセージになる」という見込みが成立するのも、1 work item が 1 コミットに対応しているときだけである。

## Scope

- タスクごとの実時間を記録する `mise` タスクを追加し、この Motivation の実測を再現可能にする。
- `verify` と `check` を、1 回の実行で落ちたゲートを全部報告する形にする。
- 検証のはしごの狭い段を `-race` にそろえ、変更したパッケージだけを対象にする段を追加する。
- 鍵導出関数そのものを検査しないテストから、本番コストの Argon2id を外す。
- UI の E2E のスタック起動を 1 回にし、ポーリング間隔を実測に基づいて詰める。
- E2E をローカルの `verify` に置き続けるか、独立したゲートにして CI へ移すかを決め、決めたとおりに配置する。
- 検査の成功時の出力を、要約と失敗だけにする。
- work item から読むべきものを 1 コマンドで出す `mise` タスクを追加し、既存の trace 索引を再利用する。
- 変わった手順に合わせて `implement-work-item` skill、`docs/development/local-development.md`、`docs/development/specification-first-workflow.md` の検証のはしごを更新する。

## Out of Scope

- 実装そのものの短縮を狙った施策。下記 Design のとおり、削れる根拠を実測で持っていない。
- `lint-go` の 83 秒の内訳調査と linter の取捨。検証ゲートの意味を変える判断であり、この work item の作業と混ぜない。
- E2E の spec 並列実行。spec ごとにポート集合と初期データを分ける設計が要り、単一起動の効果を測ってからでないと必要性を判断できない。
- CI の実行時間。対象はローカルの 1 件あたりの所要時間である。
- 1 件あたりの時間ではなく並列度を上げる方向。`parallel-work-items` skill が別に持つ手段であり、1 件あたりが長いという問題そのものは動かない。
- `check` の 16 個の検査を 1 プロセスへ統合すること。実測ではどれも 1 秒未満で、`check` 全体でも数秒である。統合しても総時間は変わらないので、やらないことをここに記録する。
- 変異テストの道具立て。[[wi-493-mutation-testing-tool-for-change-resistance-evidence]] が持つ。
- リードタイムそのものの継続計測。`docs/development/process-metrics.md` が採用条件を定めており、その条件を満たすのは別の作業である。
- E2E の速度制限そのものの検証。`fixtures.ts` は E2E 環境の IP 別上限を引き上げる。上限の正しさは Go のテストが持ち、ブラウザー E2E が見ているのは配線である。
- `tools/check/src/spec-diff.ts` の `noAdjacentSpacesInRegex` 警告。この変更の前から出ており、直すなら別に切る。

## Design

施策は「実測の律速から順に」並べる。O1 から O8 は独立に効くので、依存しない順に片付けられる。

### M0 実測の手段を先に作る

`mise run time-verify` を追加する。`verify` の各タスクを直列に実行し、タスク名、実時間、終了状態の表を出す。Motivation の実測はこれを手で書いた一時的なループで取ったもので、次に誰かが同じ数字を必要としたときに再現できない。これがないと、この work item の効果も、将来の後退も、体感でしか語れない。`docs/development/process-metrics.md` が言う Measurement の水準には届かないが、届かない理由はリードタイムの定義であって、タスクの実時間はここで測れる。

構成タスクの一覧は `mise.toml` から読む。`depends` の配列と、`run` の中の `mise run -c a ::: b` 形式の両方を読む必要があるので、その解決は `tools/check/src/verification-tasks.ts` に置き、`time-verify`、`check-work-items` の主要ユースケース到達性検査、`mise-config.test.ts` の 3 つが同じ関数を使う。`depends` だけを読む実装だと、M1 で書き換えた集約タスクは「構成要素なし」として読まれ、3 つの利用側すべてが黙って空集合で動く。

### M1 集約ゲートは 1 回で全部の失敗を報告する

`verify`、`verify-spec`、`check` を `mise run -c ... ::: ...` で走らせる。mise には `continue_on_error` にあたるタスク設定も設定項目もなく (`mise settings --all` に無い)、`-c` は実行時のフラグなので、`run` に書く形しか採れない。

`check` の構成要素は `verify` と `verify-spec` にも展開して並べる。入れ子の `mise run` にすると依存グラフが分かれ、`compile-spec` が `check` の内側と `check-api-compat` の外側で二重に走って、生成した OpenAPI を互いに書き換え合う。展開は重複なので、`check` の構成要素が両方の集約に含まれることと、並列版と逐次版が同じ一式を持つことを `mise-config.test.ts` が縛る。

`verify-serial` は逐次実行の意味を保つため、`lint-repo` と同じく失敗を集めて最後に並べる。呼び出しは `gate mise run <task>` の形をそのまま並べる。タスク名を変数に畳めば短くなるが、並列版と同じ一式かどうかを機械的に読めなくなり、drift が人の目に頼ることになる。

採らない案: 失敗したタスクだけを選んで再実行する仕組みを足す。失敗集合を持ち回す状態が増え、`mise tasks` から読めない挙動になる。1 回で全部出せば足りる。

### M2 狭い段を最後のゲートと同じ構成にそろえる

`test-go-package` を `go test -race` にする。`-race` は狭い段でも 5 倍のコストになるが (O3)、1 パッケージなら数秒であり、そのぶん最後のゲートがキャッシュから返る。あわせて `test-go-changed` を追加する。`git status --porcelain` から変更のあった Go ファイルを取り、`go list` でそのパッケージと逆依存を求め、その集合だけを `-race` で走らせる。「どこが関係するかを見極めるより回してしまった方が楽」という判断は正しく、だからこそ見極めを人ではなく `go list` にやらせる。

パッケージ選択は `tools/changed-packages` が持つ。`go list -f` の表形式を読み、`Deps`、`TestImports`、`XTestImports` を 1 つの依存集合として扱う。`go.mod` と `go.sum` は依存グラフに現れない全体入力なので、そこが変わったらモジュール全体を選ぶ。

`test-go-test` の `-count 1` は残す。RED を観測する段はキャッシュから返ってはいけない。

採らない案: 最後のゲートから `-race` を外す。競合検出はこのゲートの目的そのものである。

### M3 テストの Argon2id を軽くする

T002 のパッケージ別実時間は、`-race` の上位が Argon2id を踏むパッケージであることを示した (`shared/http/server_http` 28.87 秒を筆頭に、上位 5 件がすべてログインを経由する)。鍵導出関数そのものを検査しないテストは `backend/shared/security/testing_passwords` の `NewHasher()` からハッシャーを得る。コストは `argon2id_password_hasher_fuzz_test.go` が探索用に使っている値と同じ `MemoryCost: 64, TimeCost: 1, Parallelism: 1` である。`passwords_argon2id` パッケージ自身のテストは本番コストのままにする。

置き場所は本番 package の内側ではなく隣の `testing_*` package にした。`backend/authorization/testing_contract`、`backend/shared/storage/testing_postgres` と同じ形で、本番コードから安いコストの構成子が見えない。

### M4 E2E のスタック起動を 1 回にする

`test:e2e` を 1 つの `bun test tests/e2e/` にし、環境の起動と停止を preload の `setup.ts` へ移す。実装中に 3 つのことが分かった。

1 つ目。preload は `frontend/bunfig.toml` ではなく `frontend/tests/e2e/bunfig.toml` を `--config=` で与える。単体テストの preload は Happy DOM をグローバルへ登録するので、`Response` がブラウザー実装に差し替わり、コールバック受け口の `Bun.serve` が受け取れない値を返す。これが Motivation の 31,129 行のダンプの正体である。要求 1 件につきおよそ 1 万行を吐き、テスト自体は通るので誰も直さない。E2E が触るのは実ブラウザーであって登録された DOM ではないから、E2E では登録しない。なお `bun test -c <path>` は効かず、`--config=<path>` の形だけが効く。

2 つ目。1 プロセスになると 25 本のサインインが 1 分間に 1 つのアドレスから集中し、既定のログイン速度制限 (IP あたり 20 回/分) に当たって画面が "Too many requests." を出す。E2E 環境の起動時設定で IP 別の上限を引き上げる。上限そのものの正しさは Go のテストが持つ。

3 つ目。Vite を `bun run dev` 越しに起動していたため、停止時に殺せるのが中間プロセスだけで、Vite が孫として残っていた。次の実行はその残骸に当たる。`bun ./node_modules/vite/bin/vite.js` を直に起動する形にし、停止後はポートが空くまで待つ。

spec 間の状態独立については、共有フィクスチャ `demo` を書き換えていたのは 1 か所 (`demo.email = nextEmail`) だけで、その代入を読む側はもう無かったので落とし、`demo` を `Object.freeze` した。それ以外の spec は自分の操作対象を自分で作っている。単一プロセスで全 25 本が 2 回続けて通ることを確認した。

ポーリング間隔の 150 ミリ秒は `POLL_INTERVAL_MS = 25` に集約する。起動待ちだけは HTTP 要求そのものが刻みになるので `BOOT_POLL_INTERVAL_MS = 100` を別に持つ (元は 500 ミリ秒)。上限のタイムアウトは変えない。

### M5 E2E の置き場所を決める

`test-ui-e2e` は `verify` から外し、新しい `verify-full` に残したうえで CI の独立した job (`ui-e2e`) から実行する案を採った。ローカルの通常ゲートから律速を外しても、work item の完了前に画面へ届く変更なら E2E を明示的に実行する規約 (`implement-work-item` 第 10 段) と CI の job が後退検出を担うため、O5 の未検証状態を残さない。`check-work-items` の主要ユースケース到達性検査は `verify` の到達集合と CI が直接呼ぶタスクの和を見るので、`test-ui-e2e` を `e2e_test.task` に宣言している既存の記録はこの job で解決し続ける。

採らない案: `test-ui-e2e` を `verify` に残す。M4 で起動費用を減らしても、ブラウザー統合だけが必要とするスタック起動を通常の静的検査、単体テスト、ビルドと毎回組み合わせる理由にはならない。

### M6 検査は成功時に黙る

成功した対象の列挙を `--verbose` の下に移し、要約と失敗は既定でも出す。対象は `tools/check/src/main.ts` (work item 1 件ごとの `ok`) と `tools/check/src/check-specifications.ts` (正本文書 1 件ごとの `ok`) の 2 つで、`check` の出力のほぼ全部をこの 2 つが占めていた。あわせて成功時の要約を標準出力へそろえた。`check-ids` と `check/src/main.ts` だけが標準エラーへ出しており、ほかの検査と違っていた。出力が減るのは人間のためではなくエージェントの文脈のためであり、削るのは「成功した対象の列挙」だけである。

### M7 着手の入口を 1 コマンドにする

`mise run brief -- <work-item>` を追加する。work item の `affected_spec` を読み、規範 ID ごとに次を出す。

- その規範シナリオまたは標準の本文 (該当箇所だけ。文書全体ではない)
- TypeSpec 記号の宣言位置
- その ID を名指すコードとテストのパス
- その ID を名指す既存の work item のパス
- 対象パスの直近のコミットと、同じ文脈を触った直近のコミット
- 以上から組み立てた `initial_context` の下書き

trace 索引は `tools/render-spec-docs/src/traces.ts` へ切り出し、HTML の描画と端末からの照会が同じ索引を使う。索引が持つのは `REQ-*` と `EX-*` だけなので、標準要件 (`RFC6750-INVALID-TOKEN` のような) を名指す記録に対しては `git grep` で補う。下書きは source 側をパッケージ、tests 側を名指しているファイルにそろえる。読むのはパッケージで、回し直すのはファイルだからである。

採らない案: `spec-where` を拡張する。あれは任意の語をリポジトリ全体から探す道具で、規範 ID から仕様と実装をたどる道具とは入力も出力も別である。1 つにまとめると、どちらの用途でも余計なものが出る。

### M8 コミットは 1 work item 1 コミットに戻す

道具の変更ではなく手順の遵守である。`implement-work-item` の第 1 段に、着手時の作業ツリーに他の work item の変更が残っていないことを前提として書き、第 11 段に、コミットの本文は Completion の Summary を英語に直したものであって差分から書き起こすものではないことを明記する。

### T002 が切り分けたこと

`ui-scenario-actions.spec.ts` が完走しなかった原因は、エージェントのサンドボックスが `Bun.WebView` に必要なウィンドウサーバーへの接続を許さないことだった。`about:blank` への `navigate()` すら解決しない。サンドボックスの外では、変更前のツリーで 18 本すべてが 29.82 秒で通る。3 件分の未完了の変更に由来するものでも、E2E の欠陥でもないので、別の work item には切り出さない。

3 万行のダンプの方は本物で、原因は M4 の 1 つ目、単体テスト用 preload の Happy DOM が E2E にも効いていたことである。この work item が E2E の入口を組み替える過程で消えた。

### 実装そのものの時間について

削れる根拠を持っていないので、この work item では対象にしない。ただし実装時間に見えているもののうち 2 つは、上の施策が担う。1 つは RED/GREEN の 1 周あたりの待ち時間で、M2 が短くする。もう 1 つは「いまどのテストを回せばよいか」を毎回考え直す時間で、work item の Tasks に狭いテストの実行コマンドを書いておけば消える。後者は `WORK_ITEM_FORMAT.md` の変更ではなく、Tasks の書き方の実例として `implement-work-item` に置く。

## Plan

1. M0 の計測タスクを作り、変更前の値を記録する。以後の各施策は、この表の前後比較を証拠にする。
2. M1、M6 を先に片付ける。ほかの施策から独立で、以後の作業自体を速くする。
3. M2、M3 を行う。M3 の対象は M0 が出すパッケージ別の実時間が決める。
4. M4 を行い、実時間を記録する。
5. M5 の決定どおり、E2E を `verify-full` と CI の独立した job へ移す。
6. M7 を行う。
7. M8 と、手順文書の更新をまとめて行う。

実装内容を変える未決事項はない。M4 の実測値は単一起動とポーリング間隔の効果を評価する証拠として使い、M5 の配置判断は変更しない。

## Tasks

- [x] T001 [Tooling] `mise run time-verify` を追加し、`verify` の各タスクの実時間を表として出す。`mise run check-command-map` を通す。実行: `mise run test-tools`。
- [x] T002 [Verify] 変更のないツリーで変更前の実時間を記録した。`test-go-race` はキャッシュを消して 72 秒、パッケージ別の上位は `shared/http/server_http` 28.87 秒、`wsfederation/handlers_http` 12.17 秒、`idmanagement/user/usecases` 9.82 秒、`authentication/handlers_http` 9.00 秒、`authentication/password/usecases` 8.96 秒。`test-ui-e2e` は 56 秒、出力 41,515 行。`ui-scenario-actions.spec.ts` の不完走はエージェントのサンドボックスによるもので、別の work item には切り出さない (Design の「T002 が切り分けたこと」)。
- [x] T003 [Tooling] `verify`、`verify-spec`、`check` を、落ちたゲートを全部報告する形にした。`verify-serial` も同じ規律にそろえた。
- [x] T004 [Verify] 2 つのゲートを同時に壊し、1 回の実行で両方が報告されることを確認した。
- [x] T005 [Tooling] `check/src/main.ts` と `check-specifications.ts` の成功時出力を要約と失敗だけにし、`--verbose` を残した。成功時の要約は標準出力へそろえた。実行: `mise run test-tools`。
- [x] T006 [Tooling] `test-go-package` を `-race` にし、`test-go-changed` を追加した。実行: `mise run test-tools`。
- [x] T007 [Verify] `test-go-changed` の直後に `test-go-race` を走らせ、狭い段で走った 34 パッケージがすべてキャッシュから返ることを確認した。
- [x] T008 [App] 鍵導出関数そのものを検査しない 26 のテストファイルを、`testing_passwords.NewHasher()` へ移した。実行: `mise run test-go-package -- ./backend/shared/security/testing_passwords`。
- [x] T009 [Verify] `test-go-race` の実時間を T002 と比較した。72 秒 → 54 秒。
- [x] T010 [App] `test:e2e` を単一プロセス化し、環境の起動と停止を preload へ移した。spec 間の状態独立を確認し、共有フィクスチャの書き換えを 1 か所落とした。実行: `mise run test-ui-e2e`。
- [x] T011 [App] E2E のポーリング間隔を 150 ミリ秒から 25 ミリ秒へ、起動待ちを 500 ミリ秒から 100 ミリ秒へ下げた。
- [x] T012 [Verify] `test-ui-e2e` の実時間を T002 と比較し、25 本すべてが通ることを 2 回続けて確認した。56 秒 → 28 秒。
- [x] T013 [Tooling] M5 を決め、決定と根拠を Design へ書いた。`verify-full` と CI の `ui-e2e` job を追加した。
- [x] T014 [Tooling] trace 索引を `render-spec-docs/src/traces.ts` へ切り出し、`mise run brief -- <work-item>` を追加した。HTML の描画が同じ索引を使い続けることを `mise run check-spec` で確認した。
- [x] T015 [Verify] 完了済み work item 4 件 (wi-447、wi-453、wi-454、wi-455) に対して `brief` を走らせ、`initial_context` を再現できることを確かめた。
- [x] T016 [Docs] `implement-work-item` skill、`docs/development/local-development.md`、`docs/development/continuous-integration.md`、検証のはしご、`CONTRIBUTING.md`、`tools/README.md`、`frontend/tests/e2e/README.md` を更新した。
- [x] T017 [Verify] `mise run verify` を通し、`time-verify` の表を Completion に残した。

## Verification

- Acceptance RED: `mise run time-verify` と `mise run brief -- wi-497` が未知のタスクとして失敗し、変更前の `mise run verify-spec` が 3 重の故障のうち 1 つしか報告しないことを観測した。
- Unit RED: `tools/check/src/verification-tasks.ts`、`tools/task-timing/src/timing.ts`、`tools/changed-packages/src/changed-packages.ts`、`tools/brief/src/brief.ts`、`parseArgs` の `--verbose`、`testing_passwords.NewHasher()` について、対応する実装前にテストを失敗させた。
- `mise run verify`
- `mise run check-command-map`
- `mise run time-verify` の表 (下記 Completion)。
- 2 つのゲートを同時に壊した状態で `mise run verify-spec` を 1 回実行し、両方が報告される。
- `mise run test-go-changed` の直後の `mise run test-go-race` で、狭い段が走らせたパッケージが `(cached)` になる。

## Risk Notes

- `--continue-on-error` は、先行タスクが失敗しても後続を走らせる。`compile-spec` を壊して確認したところ、`check-spec` は失敗として報告される一方、生成 OpenAPI を読む `check-admin-scopes`、`check-contract-drift`、`check-status-drift` は作業ツリーに残っている前回の生成物に対して `ok` を出す。実行全体は `compile-spec` を名指して失敗するので緑と取り違えることはないが、これらの `ok` が指すのはディスク上の生成物であって、いま書いた仕様ではない。
- E2E の単一プロセス化は、spec 間の状態の独立という暗黙の保証を外す。`demo` を凍結して書き換えを実行時に失敗させ、全 25 本が 2 回続けて通ることを確認したが、これは順序依存が無いことの証明ではない。新しい spec を足すときは、自分の操作対象を自分で作る規約 (`frontend/tests/e2e/README.md`) に従う。
- E2E 環境で IP 別の速度制限を引き上げている。上限そのものの後退はここでは捕まらない。捕まえるのは `backend/shared/ratelimit` と `server_http` の Go テストである。
- ポーリング間隔を下げると、遅い環境で待ち合わせの取りこぼしが増えるのではなく、単に CPU を使う。上限のタイムアウトは変えていない。
- テストの Argon2id を軽くすると、本番コストでしか出ない不具合 (メモリ確保、パラメーターの符号化) を検出しなくなる。`passwords_argon2id` パッケージ自身のテストを本番コストのままにすることで、その検出はそこへ集約した。
- CI の `ui-e2e` job は手元から実行できないので、緑になることを観測していない。`Bun.WebView` は Linux では CDP 経由で Chrome を使う。最初の CI 実行が job の成否を決める。落ちるなら、修正するか job を外して `verify` へ戻すかを別に決める。
- 実測は 1 台の 8 コア機の 1 回の観測である。前後比較には同じ機械の同じ条件を使う。ほかの機械の絶対値をこの表と比べない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff main` は規範差分なしを報告する。TypeSpec の宣言も `docs/` の規範シナリオも変えておらず、変わったのは検証ゲートの構成、狭い段のビルド構成、テストが払う鍵導出コスト、E2E の起動回数、検査の出力量、着手時の入口、および手順文書である。観測できる差分は所要時間と出力行数に出る。`test-go-race` は 72 秒から 54 秒、`test-ui-e2e` は 56 秒から 28 秒、`mise run check` の出力は 685 行から 50 行、`test-ui-e2e` の出力は 41,515 行から 10 行になった。集約ゲートは 1 回の実行で落ちたゲートを全部報告するようになり、`test-ui-e2e` は `verify` から `verify-full` と CI の独立した job へ移った。
- **Acceptance RED Evidence**:
  - **Test**: `mise run time-verify`、`mise run brief -- wi-497`、および 3 つのゲートを同時に壊した状態での `mise run verify-spec`。
  - **Requirement**: N/A: 検証と着手支援の道具立てだけを変え、製品の規範要件を変えない。
  - **Observed Failure**: `time-verify` と `brief` は mise が「そのようなタスクは無い」として終了コード 1 で失敗した。`tools/README.md` に壊れたリンクを足し、`tools/task-timing/src/timing.ts` に型エラーを足した状態で `mise run verify-spec` を 1 回実行すると、`check-links`、`lint-tools`、`typecheck-tools` の 3 つが失敗したのに、実行の報告は `[lint-tools] ERROR task failed` の 1 行だけで、`check` の 16 検査のうち `check-spec`、`check-admin-scopes`、`check-contract-drift`、`check-status-drift`、`check-event-contract`、`check-security-controls`、`check-slo-references`、`check-vulnerability-suppressions` の 8 つは 1 行も出さずに打ち切られた。
  - **Detection Reason**: 打ち切られた 8 つが出力を持たないことは、mise の依存グラフが最初の失敗で残りを止めたことを直接示す。「報告が 1 件だった」だけなら失敗が本当に 1 件だった可能性を排除できないが、同時に壊した 3 件のうち 2 件が報告されていないことは排除する。
- **Unit RED Evidence**:
  - **Test**: `bun test check/src/verification-tasks.test.ts`、`task-timing/src/timing.test.ts`、`changed-packages/src/changed-packages.test.ts`、`brief/src/brief.test.ts`、`check/src/lib.test.ts` の `captures --verbose`、`mise run test-go-package -- ./backend/shared/security/testing_passwords`。
  - **Requirement**: N/A: 上と同じ。道具の単体境界には対応する規範シナリオが無い。
  - **Observed Failure**: 新規 4 モジュールは `Cannot find module` で失敗した。`parseArgs` は `--verbose` を `{ kind: 'error', code: 2, message: 'unknown flag: --verbose' }` として拒否した。`testing_passwords` は `no non-test Go files` でビルドに失敗した。
  - **Detection Reason**: それぞれの表明は、実装の有無ではなく実装の中身を見ている。`directTasks` は `mise run -c a ::: b` の継続行をまたいだ全構成要素を要求し、`depends` だけを読む実装を落とす。`changedGoPackages` は変更パッケージの逆依存を要求し、変更パッケージだけを返す実装を落とす。`extractDeclaration` は次の `## Rule:` で切ることと、他の規則の本文中の言及を宣言と取り違えないことを要求する。`testing_passwords` のテストは、安いコストで作ったハッシュを本番コストの検証器が受理することを要求し、PHC 形式の契約を切らないことを保証する。
- **Change-Resistance Results**:
  - 集約ゲートの構成要素解決: `verify` の一覧から `check-boundaries` を 1 行落とすと、`mise-config.test.ts` の `expands every member of check into the wider suites` と `runs the same set of gates in the parallel and serial suites` が両方失敗した。展開の重複が drift しても人の目に頼らない。
  - 狭い段の `-race` そろえ: `test-go-package` を `go test`（`-race` なし）に戻し、キャッシュを消して `./backend/apitoken/domain` を実行してから `mise run test-go-race` を走らせると、同じパッケージが `(cached)` ではなく `1.398s` で再実行された。`-race` を付けた形では `(cached)` になる。M2 の効果はビルド構成の一致そのものから来ている。
  - 成功時出力の抑制: `check-workspace --documents` は既定で `scenarios.feature.md` を出さず、`--verbose` を付けたときだけ出す。両方向を `check-workspace.test.ts` が表明するので、抑制しすぎ (要約まで消す) と抑制しなさすぎ (既定で全件出す) の両方が落ちる。
  - E2E の単一起動: 2 回続けて 25/25 が通った (29.13 秒と 27.89 秒)。実装中に、単体テスト用 preload を引き継いだ状態、ログイン速度制限に当たった状態、Vite が孫プロセスとして残った状態の 3 つがそれぞれ失敗として現れており、いずれも表明が検出した。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (25/25、2 回)
  - `mise run check-command-map` - passed
  - `mise run time-verify` (変更後、Go テストキャッシュを消した状態):

    | task | seconds | status |
    | --- | --- | --- |
    | test-go-race | 80 | ok |
    | test-ui-unit | 4.78 | ok |
    | check-work-items | 2.42 | ok |
    | check-spec | 2.13 | ok |
    | test-tools | 1.59 | ok |
    | lint-go | 1.53 | ok |
    | check-contract-drift | 1.06 | ok |
    | check-status-drift | 1.03 | ok |
    | check-api-compat | 0.99 | ok |
    | build-ui | 0.99 | ok |
    | check-admin-scopes | 0.98 | ok |
    | lint-repo | 0.73 | ok |
    | check-ui-dependencies | 0.55 | ok |
    | 残り 16 タスク | 各 0.24 以下 | ok |
    | total | 101 | ok |

    `lint-go` と `build-ui` が Motivation の 83 秒と数秒に対して 1.5 秒と 1.0 秒なのは、それぞれのキャッシュが温状態だからである。`time-verify` は状態を作らずに測るので、比較には同じ機械の同じキャッシュ状態を使う。

  - 変更前後の直接比較 (同じ機械、Go テストキャッシュを消した状態):

    | 対象 | 変更前 | 変更後 |
    | --- | --- | --- |
    | `test-go-race` の実時間 | 72 秒 | 54 秒 |
    | `test-go-race` の `shared/http/server_http` | 28.87 秒 | 3.69 秒 |
    | `test-ui-e2e` の実時間 | 56 秒 | 28 秒 |
    | `test-ui-e2e` の出力行数 | 41,515 行 | 10 行 |
    | `mise run check` の出力行数 | 685 行 | 50 行 |
    | `mise run check-work-items` の出力行数 | 500 行 | 3 行 |
