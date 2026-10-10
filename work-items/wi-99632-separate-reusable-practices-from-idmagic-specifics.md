---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-26
priority: p3
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: "開発方法、設計方針、文書体系の文書と skills を汎用部分と IdMagic 固有部分へ分けて配置し直すだけで、外部から観測できる振る舞い、TypeSpec の契約、規範要素は変えない。"
---

# 他のリポジトリでも使える開発方法と設計方針を IdMagic 固有の内容から分離し、リポジトリ内の独立した部分木へまとめる

## 動機

現在は未リリースであり、当面も未リリースを継続する。
2026-10-10 の簡素化調査を受け、実在する再利用先と共通化の効果が具体化するまで、この項目は延期する。
汎用化の階層、差し込み点、境界の検査を先に作ると、IdMagic の具体的な操作へ到達する参照と保守対象が増えるため、優先度を p3 とする。

このリポジトリの文書と skills には、IdMagic 以外でも通用する内容と、IdMagic の技術選択に結び付いた内容が同じファイルに混在している。

| 対象 | 汎用的な内容の例 | 固有の内容の例 |
| --- | --- | --- |
| `docs/development/specification-first-workflow.md` | 仕様先行の開発サイクル、証拠の要件、検証の段階 | `mise` タスク名、Go のミューテーションテスト、`//spec:covers` の Go コメント構文 |
| `docs/design/application/api-guidelines.md` | リソースの命名、ステータスコード、ページング、冪等性、長時間実行操作 | SCIM のバージョン、PostgreSQL の CHECK 制約とインデックスキー、TypeSpec でのボディ宣言 |
| `docs/design/application/design-guidelines.md` | モジュールの深さ、ポート、Aggregate 境界、作用の境界 | Go の型とパッケージを使った説明 |
| `docs/design/application/backend.md` | Context の内部構造、依存の向き、アーキテクチャ様式 | `backend/` と `frontend/` のディレクトリ、モノレポ構成、技術構成 |
| `docs/design/architecture/decisions.md` | Modular Monolith を選ぶ判断基準 | PostgreSQL を共有状態にする判断 |
| `.agents/skills/implement-work-item/SKILL.md` など | work item の起票と実装の手順 | `mise run` のタスク名、Go と Bun のテストコマンド |

`docs/formats/documentation-guide.md`、`docs/formats/specification-format.md`、`docs/formats/work-item-format.md` は、別のリポジトリでも使える汎用の方法論として書く方針をすでに採っている。
しかし、どの内容が汎用で、どの内容が固有かを判定する基準は文書化されておらず、境界を検査する仕組みもない。
そのため、固有の名前が汎用文書へ入り込んでも気付けず、逆に汎用的な方針が固有の設計文書に埋もれたままになる。
他のリポジトリで同じ方法を使うときは、ファイルを複製してから固有の記述を手で探して削る作業が毎回必要になる。

## 対象範囲

以下は再評価後に採否を決める候補であり、一律に実施する計画ではない。
着手前に再利用先の具体的な作業で必要な部分を選び、対象範囲、設計、タスクをその部分へ書き直す。

- 汎用と固有を分ける判定基準を定め、汎用部分の部分木の `README.md` に書く。
- 文書、skills、エージェント向けの規則を棚卸しし、各ファイルを「汎用」「固有」「分割する」のいずれかに分類する。
- 汎用部分を、リポジトリ直下の独立した部分木 `practices/` へ移す。
  - 対象は `docs/formats/documentation-guide.md`、`docs/formats/specification-format.md`、`docs/formats/work-item-format.md`、`docs/development/` と `docs/design/` と `docs/operations/` のうち汎用の内容、汎用化できる repo skills とする。
- 汎用と固有が混在するファイルは分割する。
  汎用の規則は `practices/` へ移し、固有の文書には IdMagic が採る選択と汎用規則との差分だけを残す。
- 汎用文書が固有の値を必要とする箇所（検証コマンド、インターフェース定義言語、規範 ID をテストから引用する構文、作業項目番号の採番手段など）を名前付きの差し込み点として定義し、IdMagic での値を一つの文書にまとめる。
- 汎用化できる repo skills を `practices/skills/` へ移し、`.agents/skills/` からは相対シンボリックリンクで参照する。
- `practices/` の境界を検査する。
  - `practices/` 内のリンクは `practices/` 内で解決しなければならない。
  - `practices/` の本文に、固有と判定した語（製品名、言語、ライブラリ、インフラ製品、固有のディレクトリ名、`mise` のタスク名）が現れてはならない。
- 移動に合わせて、`AGENTS.md`、`CLAUDE.md` が指す文書、各 `README.md` の索引、`tools/` の検査が参照するパスを更新する。

## 対象外

- `practices/` の別リポジトリへの抽出と、IdMagic からの参照方式（固定バージョンの複製、git submodule、外部 URL、パッケージ配布）の決定。
  本項目では、どの方式を選んでも成り立つよう `practices/` を自己完結させるところまでを行い、方式は抽出を扱う後続の work item で決める。
  比較の観点は「設計」に残す。
- `tools/` にある検査、文書の生成、作業項目番号の採番の実装の汎用化。
  これらは Bun と TypeScript で実装されており、判定基準では固有に当たる。
  汎用文書には何を検査するかだけを書き、実装の名前は書かない。
- `~/.agents/skills/` にあるマシン全体の skills の見直し。
  これらはすでにリポジトリの外にある。
- 汎用文書の英語版の作成と、文章の言語を利用するリポジトリが選べるようにする仕組み。
  `docs/development/writing-language.md` の言語方針は IdMagic 固有として残す。
- `docs/modules/`、`docs/requirements/`、`docs/runbooks/`、`spec/` の内容の汎用化。
  これらは IdMagic の製品仕様と運用手順そのものであり、固有に分類する。
  ただし、これらの文書形式の規則は `docs/formats/specification-format.md` と `docs/formats/documentation-guide.md` の一部として汎用側へ含まれる。
- 汎用化に伴う方針そのものの改訂。
  本項目は既存の規則を配置し直すだけで、規則の中身を変えない。
  移す過程で矛盾や古い記述を見つけた場合は別の work item として起票する。

## 設計

### 再評価と着手の条件

1. 実在する別のリポジトリと、共通の規則を使う具体的な作業を示す。
2. 複製している規則の更新費用と、抽出後の参照、差し込み点、同期の費用を比較する。
3. 両方で必要な最小範囲を選び、抽出が現在の開発を軽くする根拠を記録する。

条件が成立するまでは、新しい部分木、差し込み点、固有語の検査を導入しない。
規約自体の重複は[日常規約の整理](wi-12973-consolidate-daily-development-rules-and-separate-explanations.md)で先に減らせる。
既存の選択肢の比較は再評価の材料として残し、汎用化を完了するためだけに未使用の仕組みを作らない。

### 汎用と固有の判定基準

判定は内容の性質で行い、IdMagic が実際にその方針に従っているかどうかは問わない。

| 判定 | 内容の性質 | 例 |
| --- | --- | --- |
| 固有 | 特定のプログラミング言語、ライブラリ、フレームワーク、ツールに依存する | Go のパッケージ構成、React と TanStack の採用、`mise` のタスク、psqldef の規則、TypeSpec の記法 |
| 固有 | 特定のインフラ製品や配置先に依存する | PostgreSQL の可用性構成、Kubernetes マニフェスト、収集経路 |
| 固有 | 実際のディレクトリ構造の細部を示す | `backend/<context>/usecases/` のような実在のパス |
| 固有 | モノレポかどうか、リポジトリ内の配置を前提にする | フロントエンドとバックエンドを同じリポジトリで扱う手順 |
| 固有 | 製品のドメイン、要求、運用実績を書く | Bounded Context の仕様、SLO の数値、運用手順書 |
| 汎用 | 技術選択から独立した設計方針と判断基準 | Modular Monolith を選ぶ条件、モジュールの深さ、作用の境界、エラーと拒否 |
| 汎用 | 目指すべき構造（to-be）としての構成規則 | Clean Architecture や Vertical Slice Architecture に従うと各層と機能スライスがどう並ぶべきか、依存の向き |
| 汎用 | プロトコルと規格に基づく API の規則 | HTTP メソッドの割り当て、ステータスコード、ページング、冪等キー、条件付きリクエスト |
| 汎用 | 開発方法、文書体系、work item と仕様の形式 | 仕様先行のサイクル、証拠の要件、一次情報源の割り当て、運用手順書の形式 |

汎用の方針は、利用するリポジトリがすべて採用するとは限らない。
Modular Monolith の採用は、そのリポジトリが単一の配置単位を選ぶ場合にだけ適用される。
汎用文書は、方針とともに適用する条件と採らない条件を書き、採否は利用するリポジトリが差し込み点で宣言する。

文書体系の配置（`docs/` の下に置く一次情報文書の名前と階層）は方法論の規約として汎用に含める。
コードのディレクトリ構造は、to-be としての層とスライスの規則を汎用に、実在のパスを固有に分ける。

### 配置

汎用部分は、リポジトリ直下の `practices/` へまとめる。

```text
practices/
├── README.md                  判定基準、差し込み点の一覧、読む順序
├── documentation/             docs/formats/documentation-guide.md、docs/formats/specification-format.md、docs/formats/work-item-format.md
├── development/               仕様先行の開発ワークフロー、コーディングスタイル、テスト方針
├── design/                    設計ガイドライン、API ガイドライン、UI の設計指針、構造の to-be
├── operations/                サービス管理と保守の汎用規則
└── skills/                    汎用化した repo skills
```

`docs/` は IdMagic の一次情報文書の置き場所として残す。
`docs/development/`、`docs/design/`、`docs/operations/` には、IdMagic が採る選択と、汎用規則への差分だけを書く。
差分を書く文書は、該当する汎用文書へリンクし、汎用の規則を書き写さない。

リポジトリ直下に部分木を置くのは、`practices/` を丸ごと別リポジトリへ切り出せる形にするためである。
`git subtree split --prefix=practices` で履歴ごと抽出でき、どの参照方式を選んでもパスの付け替えが `practices/` の外で完結する。

### 差し込み点

汎用文書は、固有の値を必要とする箇所で具体的な名前を書かず、差し込み点の名前で参照する。
差し込み点の一覧と意味は `practices/README.md` に置く。
IdMagic での値は、`docs/development/` に新設する一つの文書に表でまとめ、`AGENTS.md` からその文書へ導く。

差し込み点の候補は次のとおりで、棚卸しの結果に応じて増減させる。

| 差し込み点 | IdMagic での値 |
| --- | --- |
| 集約検証コマンド | `mise run verify` |
| タスクランナー | `mise` |
| インターフェース定義言語 | TypeSpec |
| 規範 ID をテストから引用する構文 | `//spec:covers <id>: <説明>` |
| 作業項目番号の採番手段 | `mise run work-item-number` |
| ミューテーションテストの実行手段 | `mise run test-go-mutation` |
| 採用するアーキテクチャ様式 | Modular Monolith、Context ごとの Clean Architecture |
| 文書の言語 | 日本語 |

### 混在ファイルの分割方針

| ファイル | 汎用側へ移す内容 | IdMagic 側に残す内容 |
| --- | --- | --- |
| `docs/development/specification-first-workflow.md` | 開発サイクル、証拠の要件、検証の段階、現在状態の文書、コンテキストの節約 | タスク名、言語別の変異テスト手段、注釈の構文 |
| `docs/development/testing.md` | テスト水準、主要ユースケース、性質とファジング、テストダブルの選択基準 | Go のミューテーションテスト、メモリーと PostgreSQL のアダプターの扱い |
| `docs/design/application/api-guidelines.md` | 命名、値の表現、メソッド、ステータスコード、エラー形式、コレクション操作、冪等性、条件付きリクエスト、長時間実行操作、安定性と非推奨化 | SCIM、データベースの制約、ヘッダーの設定主体、TypeSpec での宣言方法 |
| `docs/design/application/design-guidelines.md` | 全体 | 言語に依存する例示だけを差分文書へ移す |
| `docs/design/application/frontend.md` | 機能スライスの境界、依存の向き、コンテナと表示用コンポーネントの分離 | 採用ライブラリ、ルーティングのファイル規約、ビルドと配信、タスク |
| `docs/design/application/user-interface.md` | 情報の順序、表示状態、破壊的な操作、フォームと検証、エラーの文言、アクセシビリティ、国際化の原則 | 対応言語、辞書の実装、テストのロケール |
| `docs/design/application/backend.md` | Context の内部構造の to-be、Context 間イベントの公開言語と互換性、アーキテクチャ様式 | 実在のディレクトリ、技術構成、HTTP ルーティング |
| `docs/design/architecture/decisions.md` | Modular Monolith を選ぶ判断基準と見直し条件 | PostgreSQL とブラウザーの同一オリジン境界に関する判断 |
| `docs/design/observability/logging.md` | 共通フィールド、`event_name` の命名、ログレベル、秘密情報と個人情報の扱い | 収集経路、保持期間、容量 |
| `docs/design/data/schema-management.md` | 宣言的スキーマ、収束の検査、拡張と縮小 | psqldef の性質から来る規則、適用する地点 |
| `docs/operations/service-management.md` | エラー予算、変更の分類、ポストモーテム | 当番とエスカレーション、目標値 |

表にないファイルは棚卸しで分類する。
`docs/development/coding-style.md` はすでに汎用として書かれているので、そのまま移す。

### skills の分類

| skill | 分類 | 扱い |
| --- | --- | --- |
| `new-work-item` | 汎用 | 採番と検査のコマンドを差し込み点へ置き換えて移す |
| `implement-work-item` | 汎用 | テストと検証のコマンドを差し込み点へ置き換えて移す |
| `parallel-work-items` | 汎用 | ワークツリーの配置とタスク名を差し込み点へ置き換えて移す |
| `spec-change` | 汎用 | インターフェース定義言語を差し込み点へ置き換えて移す |
| `update-design` | 汎用 | 一次情報文書の名前は汎用の文書体系に従うので、そのまま移す |
| `render-docs` | 固有 | TypeSpec の生成手順なので残す |
| `update-all-dependencies` | 固有 | 依存の層が IdMagic の技術構成に依存するので残す |

skill は他の文書を取り込めないので、汎用化した skill は差し込み点を「`AGENTS.md` が示す値」として参照する。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| ファイルを移さず、節ごとに汎用か固有かの印を付ける | 抽出時に結局ファイルを分割する必要があり、印は検査しない限り実態からずれる |
| 汎用版の複製を作り、既存ファイルは残す | 同じ規則が二か所に存在し、一次情報源を一つにする文書体系の原則に反する |
| `docs/practices/` に置く | `docs/` は IdMagic の一次情報文書の置き場所であり、文書配置の検査の対象でもあるため、抽出する部分木と混ざる |
| 汎用文書にテンプレート変数を埋め、生成時に固有の値を代入する | 生成を経ないと読めない文書になり、生成の仕組みそのものが固有の実装になる |
| 本項目で別リポジトリへ抽出する | 分類と境界の検査が固まる前に抽出すると、固有の記述を含んだまま二つのリポジトリで管理することになる |

### 後続の抽出で比較する参照方式

本項目では決めないが、`practices/` の構成がどの方式でも成り立つように、観点だけを記録する。

| 方式 | 利点 | 負担 |
| --- | --- | --- |
| 固定バージョンを複製して同期する | 相対リンクと文書の生成がそのまま動き、オフラインで読める | 同期の手段と、複製を手で編集していないことの検査が必要になる |
| git submodule | 参照するバージョンが明示される | ワークツリーの作成、CI、エージェントの作業手順が増える |
| 外部 URL で参照する | リポジトリに複製を置かない | リンクの解決を検査できず、オフラインで読めない |
| パッケージとして配布する | バージョン管理と更新の通知を既存の仕組みに任せられる | 文書と skills を配布する手段がパッケージ管理の種類に依存する |

## 計画

この計画は上の着手条件が成立した時点で、選択した最小範囲へ改訂する。

1. 棚卸し表を作り、各ファイルと各節を「汎用」「固有」「分割する」へ分類する。
   分類表は本項目の完了節に残す。
2. `practices/README.md` に判定基準と差し込み点の一覧を書く。
3. 境界の検査を追加し、現状で違反が検出されることを確認する。
4. 汎用のみのファイルを移し、参照を更新する。
5. 混在するファイルを分割する。
   一つのファイルごとに、汎用側へ移した規則と IdMagic 側に残した差分を突き合わせ、規則が落ちていないことを確かめる。
6. skills を汎用化して `practices/skills/` へ移し、`.agents/skills/` からのシンボリックリンクを張る。
7. IdMagic の差し込み点の値をまとめた文書を書き、`AGENTS.md` と `CLAUDE.md` の参照先を更新する。

未解決の問い:

- `practices/` という名前は仮である。
  抽出後のリポジトリ名と揃えたい場合は、着手時に決める。
  名前は後から変えられるが、変えるとリンクをすべて付け替える必要がある。
- 差し込み点の値をまとめる IdMagic 側の文書の名前と置き場所は、棚卸しの結果に合わせて着手時に決める。
- 境界の検査で使う固有語の一覧を、検査の実装に置くか、`practices/` の外の設定として置くかは着手時に決める。
  一覧自体は IdMagic 固有なので、`practices/` の中には置かない。

## タスク

- [ ] T001 [Docs] 文書、skills、エージェント向けの規則を棚卸しし、分類表を作る。
- [ ] T002 [Docs] `practices/README.md` に判定基準、差し込み点、読む順序を書く。
- [ ] T003 [Tooling] `practices/` の境界の検査を追加し、現状の違反を検出することを確認する。
- [ ] T004 [Docs] 汎用のみのファイルを `practices/` へ移し、参照を更新する。
- [ ] T005 [Docs] 混在するファイルを分割し、規則の欠落がないことを突き合わせる。
- [ ] T006 [Docs] 汎用化できる skills を `practices/skills/` へ移し、`.agents/skills/` からリンクする。
- [ ] T007 [Docs] IdMagic の差し込み点の値をまとめた文書を書き、`AGENTS.md` と `CLAUDE.md` を更新する。
- [ ] T008 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- 境界の検査が `practices/` の外へのリンクと固有語を検出し、移動後は違反がないこと。
- `mise run verify`

## リスク

- 文書の移動で、リンク、`tools/` の検査が参照するパス、skills の参照が壊れる。
  移動のたびに `mise run verify` を通し、文書配置の検査と `AGENTS.md` の導線の検査で検出する。
- 混在ファイルの分割で規則が落ちる、または両側に重複する。
  ファイルごとに分割前後の規則を突き合わせ、分割と内容の変更を同じコミットに混ぜない。
- 汎用文書を差し込み点で書くと、IdMagic の開発者とエージェントが具体的なコマンドへたどり着くまでの参照が一段増える。
  `AGENTS.md` から差し込み点の値の表へ直接導き、よく使うコマンドは `AGENTS.md` にも残す。
- 固有語の検査は誤検出（一般語としての Go など）と見逃しの両方が起こり得る。
  リンクの閉包の検査を主とし、固有語の検査は補助として扱う。
