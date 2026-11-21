---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-21
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "変異テストの実行に使うツールだけを替え、製品の観測可能な振る舞い、永続状態、公開契約を変えない。"
---

# 変異テストのツールを gomutants から gamut へ置き換える

## 動機

`mise run test-go-mutation` は gomutants 0.7.3 を使い、変異体ごとにパッケージを構築し直す。
2026-11-21 に `backend/oauth2/internal/token/usecases` を既定の設定（ワーカー 2、5 種類の変異、キャッシュなし）で実行したところ、267 秒かかった。
統合モードで `backend/signinpolicy/domain` を実行したところ、802 秒かかった。

[gamut](https://github.com/ambi/gamut) 0.1.0 は全変異体を 1 回の構築に埋め込み（mutant schemata）、実行時に有効な変異体を選ぶ。
変異体ごとの構築結果を共有の Go ビルドキャッシュへ書かない。
[wi-49807](wi-49807-enable-gomutants-schemata-for-mutation-runs.md) が gomutants に求めていた実行方式を、gamut は既定で備える。

## 対象範囲

- `mise.toml` の固定ツールを `go:github.com/ambi/gamut/cmd/gamut` 0.1.0 へ替える。
- `test-go-mutation`、`test-go-mutation-mutant`、`check-go-mutation-tool` の各タスクを gamut のフラグへ移す。タスクの引数の順序と意味は変えない。
- `show-tool-versions` の版の表示を gamut へ替える。
- canary の検査（`tools/check/src/mutation-canary.ts`）を、gamut が書く mutation-testing-elements 形式の報告へ合わせる。
- タスク契約を固定する `tools/check/repository/mise-config.repository.ts` の期待を更新する。
- [テスト方針のミューテーションテスト](../../docs/development/testing.md#ミューテーションテスト)へ、実行方式、状態名、統合モードの準備時間を反映する。
- wi-49807 を、この項目で目的を達したものとして中止する。

## 対象外

- 既定の 5 種類の変異を広げること。gamut の既定集合（`RETURN_ZERO`、`STATEMENT_REMOVE` を含む 7 種類）は survivor の仕分け量を測ってから別に判断する。
- キャッシュを既定で有効にすること。gamut はモジュール内の依存パッケージの変更をキャッシュキーで追跡すると設計文書に書くが、この項目では確かめていない。
- gamut の設定ファイル `gamut.json`、許容の記録 `gamut.accepted.json`、臨界領域、閾値を導入すること。
- 変異テストを `check` または `verify` の必須ゲートに入れること。

## 設計

### フラグの対応

| タスクの引数 | gomutants 0.7.3 | gamut 0.1.0 |
| --- | --- | --- |
| ワーカー数 | `--workers` | `-workers` |
| キャッシュ（`off` または経路） | `--cache off`／`--cache <file>` | `-no-cache`／`-cache <file>` |
| 変異演算子 | `--only` | `-only` |
| 報告の経路 | `--output` | `-o` |
| 比較基準のリビジョン | `--changed-since` | `-since` |
| 統合モード | `--integration` | `-integration` |
| 1 件の再実行 | `--run-mutant-id` | `-mutant`（キャッシュを無視する） |

gamut の `-cache` は既定で作業ディレクトリの `.gamut/cache.json` を読み書きする。
リポジトリ内に状態を残さないよう、第 3 引数が `off` のときは `-no-cache` を渡す。

### 報告の形式

gamut の JSON 報告は mutation-testing-elements 第 2 版の形式で、`files` はモジュール相対の経路を鍵とするオブジェクト、各ファイルの変異は `mutants`、演算子名は `mutatorName` にある。
状態は `Killed`、`Survived`、`NoCoverage`、`CompileError`、`Timeout`、`Pending`、`Ignored` で表す。
gomutants の `LIVED` は `Survived`、`NOT VIABLE` は `CompileError` にあたる。
生存した変異は `statusReason` で、到達したが値が変わらない `not infected` と、値が変わったが検出されない `not detected` に分かれる。

変異の ID は `<ファイル>::<演算子>::<宣言>::<ハッシュ>` の形で、gomutants の `<ファイル>::<演算子>#<番号>` とは互換性がない。
作業項目に残した gomutants の ID は、gamut で再実行できない。

### 実測

2026-11-21 に、gamut 0.1.0 と gomutants 0.7.3 を同じ作業ツリーで比べた。

| 対象 | 測定項目 | gomutants 0.7.3 | gamut 0.1.0 |
| --- | --- | --- | --- |
| canary | 判定 | lived 1、not viable 1 | 同じ |
| `backend/oauth2/internal/token/usecases`（local） | 実行時間 | 267 秒 | 9 秒 |
| 同上 | 変異体 | 563 体 | 562 体。位置、演算子、置換で突き合わせて 551 体の判定が一致 |
| `backend/signinpolicy/domain`（integration） | 実行時間 | 802 秒 | 194 秒 |
| 同上 | 判定 | killed 29 | killed 27。突き合わせた 27 体はすべて一致 |

判定が食い違った 12 体の内訳は次のとおりで、どれも gamut の判定を誤りとは見なさない。

- 短絡評価の右辺 6 体（`a != "" && b != c` の `b != c` など）：gomutants は行単位の被覆で `LIVED`、gamut は式単位の被覆で `NoCoverage`。どのテストも右辺を評価していない。
- パッケージ水準の定数式 2 体（`30 * time.Second`）：gomutants は `NOT COVERED`、gamut は単独構築で判定して `Survived`。
- 文字列連結の `+` を `-` にする 3 体：gomutants は被覆がないので構築せず `NOT COVERED`、gamut は型検査で `CompileError`。
- gomutants は二項の `-` を `+` にする変異を `ARITHMETIC_BASE` と `INVERT_NEGATIVES` の両方で作る。gamut は一体にまとめる。統合モードの件数差 2 体も同じ理由である。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| gomutants を続け、wi-49807 の `--schemata` を待つ | 0.7.3 の `--help` は `--schemata` を公開しておらず、提供時期が分からない。gamut は同じ方式を既定で備え、判定も一致した |
| gomutants と gamut を併用し、タスクで選ばせる | 同じ証拠を二通りの ID と状態名で残すことになり、作業項目の記録を読み比べられない |
| gamut の既定集合の 7 種類へ同時に広げる | ツールの変更と演算子の変更を同時に持ち込むと、結果の差をどちらに帰属すべきか分からない |

## 計画

1. `mise-config.repository.ts` と `mutation-canary.test.ts` の期待を gamut へ替え、現在の設定に対する RED を確かめる。
2. `mise.toml` の固定ツール、三つのタスク、`show-tool-versions` を更新する。
3. canary の検査を mutation-testing-elements 形式へ合わせる。
4. テスト方針を更新する。
5. wi-49807 を中止する。

## タスク

- [ ] T001 [Unit] タスク契約と canary の検査の期待を更新し、RED を確かめる。
- [ ] T002 [Tooling] `mise.toml` の固定ツールとタスクを gamut へ移す。
- [ ] T003 [Tooling] canary の検査を mutation-testing-elements 形式へ合わせる。
- [ ] T004 [Docs] テスト方針を更新する。
- [ ] T005 [Verify] canary、代表パッケージ、1 件の再実行、差分モード、統合モードを確かめる。
- [ ] T006 [Docs] wi-49807 を中止する。

## 検証

- `mise run test-tools-file -- check/repository/mise-config.repository.ts` と `mise run test-tools-file -- check/src/mutation-canary.test.ts`：タスク契約と canary の検査を変えるため。
- `mise run check-go-mutation-tool`：ツールを替えた後に生存と構築不能の分類を検算する。
- `mise run test-go-mutation -- backend/oauth2/internal/token/usecases`：既定の引数の経路を確かめる。
- `mise run test-go-mutation-mutant -- backend/oauth2/internal/token/usecases '<生存した変異の ID>'`：1 件の再実行を確かめる。
- `mise run test-go-mutation -- backend/oauth2/internal/token/usecases 2 off /tmp/idmagic-mutation-since.json main`：差分モードを確かめる。
- `mise run check-work-items`、`mise run check-links`、`mise run verify`。

## リスク

- gamut 0.1.0 は 2026-11-21 公開の最初の版で、利用実績が少ない。canary と代表パッケージの突き合わせで判定を確かめ、gomutants 0.7.3 とタスクへは固定ツールとタスクの差し戻しで戻せる。
- 1.0.0 までは minor 版で報告の形と ID の計算が変わり得る。固定版を上げるときは gamut の CHANGELOG を読み、canary を再実行する。
- gamut は既定でログ出力などの arid なコードを変異させない。gomutants より変異体が減る箇所があるが、検出能力の証拠として扱う範囲には影響しない。
