---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-06
priority: p2
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "変異テストの実行方式だけを変え、製品の観測可能な振る舞い、永続状態、公開契約を変えない。"
---

# gomutants の mutant schemata を有効にし、変異体ごとのビルドをなくす

## 動機

gomutants 0.6.1 は、変異体ごとに `-overlay` で書き換えたソースを渡し、パッケージのコンパイルとテストバイナリのリンクをやり直す。
この方式には二つの問題がある。

- **実行時間**：変異体ごとのコンパイルとリンクが実行時間の大半を占める。
- **共有ビルドキャッシュの肥大**：変異体ごとのビルド結果は二度と使われないのに、共有の Go ビルドキャッシュに書き込まれる。go コマンドの自動掃除は 5 日間使われていないエントリだけを消し、容量の上限を持たない。そのため、変異テストを繰り返すと数十 GiB に達し、ディスクが埋まりかける。

2026-10-06 に `backend/oauth2/token/usecases`（約 1,700 行、変異体 216 体）を `mise run test-go-mutation` の既定の設定で実行したところ、113 秒かかり、共有ビルドキャッシュに 2,769 件、454 MiB が加わった。

gomutants の作者は、全変異体を 1 回のコンパイルに埋め込み、どの変異体を有効にするかを実行時に環境変数で選ぶ mutant schemata を、`--schemata` オプションとして開発している（[gomutants の Pull Request #101](https://github.com/gomutants/gomutants/pull/101)、2026-10-06 時点で下書き）。

## 対象範囲

- `--schemata` を含む gomutants のリリース版へ、`mise.toml` の固定バージョンを更新する。
- `test-go-mutation`、`test-go-mutation-mutant`、`check-go-mutation-tool` の各タスクに `--schemata` を付ける。
- タスク契約を固定する `tools/check/src/mise-config.test.ts` の期待を更新する。
- [テスト方針の Go のミューテーションテスト](../docs/development/testing.md#go-のミューテーションテスト)へ、実行方式と実行時間の目安を反映する。

## 対象外

- 既定の 5 種類の変異を広げること。
- gomutants 以外のツールへ乗り換えること。採用しない理由は「設計」の「採用しない案」に記録する。
- 共有ビルドキャッシュを掃除するタスクを追加すること。採用しない理由は「設計」の「採用しない案」に記録する。
- 変異テストを `check` または `verify` の必須ゲートに入れること。

## 設計

### 採用する方式

gomutants の `--schemata` を有効にする。
変異の種類、変異体 id、JSON レポートの形式、変異体 1 体の再実行、結果のキャッシュは今のまま使える。
schemata に変換できない変異体は、gomutants が従来の変異体ごとのビルドで判定する。
schemata のビルドが失敗した変異体は「判定不能」に格下げされ、別の判定に置き換えられることはない。

### 開発中の版での実測

2026-10-06 に、Pull Request #101 の最新コミット（`v0.6.2-0.20260916024012-7de2bcb806d4`）を一時ディレクトリにインストールして測った。
対象は `backend/oauth2/token/usecases` で、worker 数 2、5 種類の変異、キャッシュ無効という既定の設定に `--schemata` だけを足した。

| 測定項目 | 0.6.1 | `--schemata` |
| --- | --- | --- |
| 実行時間 | 113 秒 | 5 秒 |
| 共有ビルドキャッシュの増加 | 2,769 件、454 MiB | 20 件、3.4 MiB |
| 1 回のビルドを共有した変異体 | なし | 211 体中 209 体 |
| 判定 | killed 188、lived 21、not covered 5、not viable 2 | 216 体すべてが変異体 id 単位で 0.6.1 と一致 |

canary（`tools/check/testdata/mutation-canary`）は、初回とキャッシュ再利用時の両方で lived 1、not viable 1 を返し、`mutation-canary.ts` の検査を通った。
not viable となる変異体は schemata から外れ、従来のビルドで判定された。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| 変異テストの実行中だけ一時的な `GOCACHE` を使い、終了時に消す | ゴミは残らないが、実行のたびに依存パッケージをキャッシュなしでビルドし直す。`backend/apitoken/usecases` では 13 秒が 26 秒になった |
| 共有キャッシュを APFS の `cp -c` で一時ディレクトリへ複製して使う | 3.4 万件の複製に 5.9 秒、削除に 2.7 秒かかり、キャッシュなしのビルドと大差がない。キャッシュが大きくなるほど遅くなる |
| 24 時間使われていないエントリを消す掃除タスクを追加する | 変異体のビルド結果は作られた直後なので、変異テストの前後に実行しても消えない。効果は `go clean -cache` と大差がない |
| 読み取りは共有キャッシュ、書き込みは一時ディレクトリに分ける `GOCACHEPROG` を自作する | 速度とキャッシュの肥大を両方解決できるが、cmd/go 内部のディスク形式に依存する。ツール側の schemata で解決するなら不要である |
| Assay 0.8.0 に乗り換える | 同じ方式を実装済みで、同じパッケージを 6 秒で判定し、重なる変異の判定も gomutants と一致した。ただし、作者 1 人で開発期間は 4 日間、2026-08-11 以降は更新がない。変異の種類を絞れず、変異体 1 体の再実行もできない。既定の変異で生き残りが 21 体から 181 体に増え、仕分けの負担が増える |
| Gremlins に戻る | 同じ方式の [Pull Request #318](https://github.com/go-gremlins/gremlins/pull/318) は外部の貢献者によるもので、未取り込みの前提 Pull Request が 5 つあり、本体の最終更新は 2026-06 である |
| go-mutants（P4suta）に乗り換える | 同じ方式を実装しているが、2026-08 に作成されたばかりで利用実績がない |
| ミューテーションテストツールを自作する | 既存のツールが同じ方式を実装済みまたは実装中であり、自作する理由がない |

## 計画

着手の条件は、`--schemata` を含む gomutants のリリース版が公開されることである。
それまでは、必要に応じて `go clean -cache` で共有ビルドキャッシュを掃除する。

1. リリース版の変更履歴で、`--schemata` の名前、既定値、Pull Request #101 からの変更点を確かめる。既定で有効になっていれば、オプションを付ける代わりにその既定を確認する。
2. `mise-config.test.ts` に `--schemata` と新しい固定バージョンを要求する表明を加え、現在の設定に対する RED を確認する。
3. `mise.toml` の固定バージョンと三つのタスクを更新する。
4. canary と `backend/oauth2/token/usecases` を実行し、実行時間、共有ビルドキャッシュの増加、0.6.1 の判定との一致を記録する。
5. テスト方針を更新する。

リリース版が `--schemata` を含まずに Pull Request #101 が閉じられた場合は、この work item を中止し、「採用しない案」から選び直す。

## タスク

- [ ] T001 [Tooling] リリース版の `--schemata` の仕様を確かめる。
- [ ] T002 [Unit] `mise-config.test.ts` の期待を更新し、RED を確認する。
- [ ] T003 [Tooling] `mise.toml` の固定バージョンと三つのタスクを更新する。
- [ ] T004 [Verify] canary と代表パッケージで、判定の一致、実行時間、キャッシュの増加を測る。
- [ ] T005 [Docs] テスト方針を更新する。

## 検証

- `mise run test-tools-file -- check/src/mise-config.test.ts`
- `mise run check-go-mutation-tool`
- `mise run test-go-mutation -- backend/oauth2/token/usecases` を 0.6.1 と新しい版で実行し、変異体 id ごとの判定を比べる。前後で `go env GOCACHE` 配下のファイル数と容量を測る。
- `mise run test-go-mutation-mutant` で、既知の生き残り 1 体を再実行する。
- `mise run verify`

## リスク

- schemata の書き換えに誤りがあると、判定が変わるおそれがある。変異体 id ごとに 0.6.1 と比べ、canary で既知の判定を確かめる。
- 式を書き換えると、インライン化やエスケープ解析の結果が変わる。メモリ割り当ての回数を表明するテストは、変異のない状態でも結果が変わるおそれがある。食い違いが出たら、その変異体を従来のビルドで再実行して切り分ける。
- 変更は固定バージョン、タスク、検査、文書に閉じており、0.6.1 と従来のタスクへ戻せる。
