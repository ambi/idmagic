# Go mutation testing tool の再評価

## 結論

2026 年 9 月 13 日、IdMagic の mutation testing tool を Gremlins 0.6.0 から gomutants 0.6.1 へ移行した。
Ooze は開発中の次版が公開されるまで採用しない。

gomutants は Go overlay を使うため、変異ごとに作業ツリーを複製しない。
パッケージ選択、変更差分による絞り込み、テスト単位の経路選択、単一 mutant の再実行、JSON と HTML と Stryker 形式の出力、永続キャッシュを備える。
IdMagic の対象パッケージで同種の演算子を実行した結果も、Gremlins より短く、既知の境界値不足を survivor として検出できた。

一方、gomutants は公開から日が浅い 0.x 系であり、キャッシュキーが対象パッケージから参照する別パッケージの変更を完全には追跡しない。
移行当初はキャッシュを無効にし、同一変更を反復調査するときだけ制約を理解した上で有効にする。

## 評価基準

IdMagic では、次の順で重視する。

1. survivor、killed、非コンパイル mutant を正しく区別できること。
2. 約 1.1 GB のリポジトリでも、パッケージ単位の手動実行を気軽に反復できること。
3. 変更箇所や関連テストへ実行範囲を狭められること。
4. 結果を追跡し、特定の mutant を再現できること。
5. 保守状況とリリースの成熟度が許容できること。

mutation score の閾値を CI gate にすることは目的に含めない。
主用途は、仕様例とテストの検出能力をパッケージ単位で調べ、survivor をレビューすることである。

## 比較

| Tool | 実行方式と絞り込み | 反復実行 | 成熟度 | 判定 |
| --- | --- | --- | --- | --- |
| gomutants 0.6.1 | overlay、coverage、package、changed-since、per-test | 永続キャッシュ、stable mutant ID、複数 report 形式 | 新しい 0.x 系 | 推奨 |
| Gremlins 0.6.0 | worker ごとに module 全体を byte copy、coverage、package、diff | 結果キャッシュなし | 更新頻度は低いが保守は継続 | 置き換える |
| Mutago 2.9.5 | coverage、git diff line、per-test、33 種類の operator | stable mutant ID、JSON など、永続結果キャッシュなし | 活発 | 次点 |
| Ooze 0.2.0 | mutant ごとに repository を materialize、Unix ではファイル単位の symlink | coverage prefilter、差分実行、結果キャッシュなし | 0.3.0 と 0.3.1 は retract、次版は未リリース | 見送る |
| gomu | source rewrite、coverage | Go test cache の影響を受ける | 実装上の懸念が未解消 | 見送る |
| go-mutesting | source rename、逐次実行 | coverage、差分、worker、結果キャッシュなし | 枯れている | 見送る |

Gremlins 0.6.0 は worker ごとに module を一度複製し、その中で mutant を順に試す実装である。
README 自身も大規模 module で十分に動かないと説明している。
ただし、2025 年 12 月に 0.6.0 が公開され、2026 年 3 月にも修正が main に入っているため、放棄されたプロジェクトとは判断しない。
根拠は [Gremlins repository](https://github.com/go-gremlins/gremlins)、[0.6.0 の workdir 実装](https://github.com/go-gremlins/gremlins/blob/v0.6.0/internal/engine/workdir/workdir.go)、[0.6.0 release](https://github.com/go-gremlins/gremlins/releases/tag/v0.6.0) にある。

gomutants 0.6.1 は overlay により対象ファイルだけを差し替え、`go test -overlay` を実行する。
機能と制約は [gomutants README](https://github.com/szhekpisov/gomutants/blob/76c6e5989b7f5e252acc4641366a7091392a9733/README.md)、実装は [worker](https://github.com/szhekpisov/gomutants/blob/76c6e5989b7f5e252acc4641366a7091392a9733/internal/runner/worker.go) と [cache](https://github.com/szhekpisov/gomutants/blob/76c6e5989b7f5e252acc4641366a7091392a9733/internal/cache/cache.go) で確認した。
0.6.1 では sibling package のファイルをキャッシュキーへ含める修正が入ったが、対象が import する package の変更は既知の未解決事項である。

Mutago は operator の広さと変更行への絞り込みに優れる。
しかし永続的な結果キャッシュがなく、短いローカル反復という今回の優先事項では gomutants に劣る。
機能一覧は [Mutago repository](https://github.com/quality-gates/mutago) と [v2.9.5](https://pkg.go.dev/github.com/quality-gates/mutago/v2@v2.9.5) にある。

## IdMagic での試行

`backend/idmanagement/group/domain` を対象に、worker 数を 2 に固定して測定した。
実行前の通常テストは成功した。

| 試行 | 結果 | 実時間 |
| --- | --- | ---: |
| Gremlins 0.6.0、既存設定 | killed 56、lived 0、not covered 16 | 約 30.5 秒 |
| gomutants 0.6.1、Gremlins と同系統の 5 operator、cache off | killed 38、lived 14、not covered 16、not viable 1 | 約 22.0 秒 |
| gomutants 0.6.1、論理反転を含む 5 operator、cache cold | killed 43、lived 19、not covered 21、not viable 1 | 約 22.6 秒 |
| 同じ gomutants 実行、cache warm | 63 件を再利用 | 約 1.7 秒 |
| gomutants 0.6.1、デフォルトの 28 operator、dry-run | mutant 399 件、うち test 対象 299 件 | 約 1.3 秒 |

同じ AST に対する tool 間の mutant 分割と重複除去が異なるため、総数と score は直接比較しない。
時間はローカル環境での一回の観測であり、benchmark ではない。

既知の確認対象として、`dynamic_group_rule.go` の `>` を `>=` に変える mutant を単独実行した。
gomutants は約 4.1 秒で lived と判定し、以前の調査で見つかった境界値テスト不足を再現した。
一方、今回の Gremlins は同じ対象パッケージの全 covered mutant を killed と判定した。
Gremlins は終了コード 2 だけを not viable とし、終了コード 1 を killed とするため、compile error や setup failure が killed に混ざる可能性がある。
小さな独立パッケージでは Gremlins が lived を正しく報告したため、Gremlins 全般の故障ではなく、IdMagic の現在の構成との組み合わせで生じる判定上の異常として扱う。

## Ooze を採用しない理由

Ooze の main branch は 2026 年にも更新され、次版の作業も進んでいる。
しかし利用可能な最新 release は 2023 年の 0.2.0 であり、0.3.0 と 0.3.1 は retract されている。
現在の実装は mutant ごとに repository を materialize し、Unix でも全ファイルを走査して symlink tree を作る。
coverage prefilter、変更箇所限定、永続結果キャッシュもないため、IdMagic の性能上の懸念を解消する構造ではない。
根拠は [Ooze repository](https://github.com/gtramontina/ooze)、[retract を含む go.mod](https://github.com/gtramontina/ooze/blob/87c15dcb180492ba96f30278efc0146dd248f09b/go.mod)、[materialize 実装](https://github.com/gtramontina/ooze/blob/87c15dcb180492ba96f30278efc0146dd248f09b/internal/fsrepository/materialize_unix.go)、[次版の Pull Request](https://github.com/gtramontina/ooze/pull/82) にある。

## 導入条件

最初の移行では Gremlins と同系統の 5 operator に限定し、既存結果と比較できる状態を保つ。
対象は引き続き package directory とし、worker のデフォルト値は 2、PR の必須 CI gate にはしない。
初回と依存 package を変更した後は `--cache=off` を使う。
同一変更上で survivor を潰す反復だけは cache を許可する。

tool 更新時には、既知の survivor と既知の not viable mutant を固定した canary を実行する。
同じ入力を再実行して分類が一致することも確認する。
この条件により、過去に gomu で見つかった Go test cache の誤判定と、今回 Gremlins で観測した全 killed の異常を検出する。

gomutants の import 先を含むキャッシュ無効化が解決し、安定 release と利用実績が増えた時点で cache のデフォルト値を再評価する。
Ooze は retract されていない次版が公開された時点で、実行モデルと IdMagic 上の時間を再評価する。

## 実行方法

通常の調査は package directory を指定して実行する。
worker 数は 2、cache は無効、JSON report は一時領域の `idmagic-mutation-report.json` がデフォルト値である。

```console
mise run test-go-mutation -- backend/idmanagement/group/domain
```

同一変更上で反復するときだけ、第 3 引数へ cache file を指定する。
対象 package が import する package を変更した後は、以前の cache file を再利用しない。

```console
mise run test-go-mutation -- backend/idmanagement/group/domain 2 /tmp/idmagic-gomutants-cache.json
```

JSON report にある stable mutant ID を 1 件だけ再実行する場合は、専用 task を使う。
この task は cache を常に無効にする。

```console
mise run test-go-mutation-mutant -- backend/idmanagement/group/domain '<mutant-id>'
```

ツールの版を更新した後と change-resistance evidence を作る前には、固定した canary で verdict を検算する。

```console
mise run check-go-mutation-tool
```
