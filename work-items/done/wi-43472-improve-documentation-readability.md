---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-26
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "文書サイトの表示、文書の語彙と構成、文書の検査だけを変え、規範 ID、TypeSpec のシンボル、設定キー、プロダクトの振る舞いは変えない。REQ-APPLICATION-014、REQ-IDGOVERNANCE-005、REQ-JOBS-011、REQ-PROVISIONING-016、REQ-SYSTEM-012 は、データベースの「行」「列」を「レコード」「カラム」と書き直すため spec-diff に現れるが、条件も要求する結果も同じである。"
documentation_impact:
  level: none
  reason: "変わるのは開発者と運用者が読む文書とその生成サイトだけで、利用者が観測する振る舞い、API、設定キーはどれも変わらない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/render-docs/src/render.ts
    - tools/render-docs/src/traces.ts
    - tools/check/src/terminology.ts
    - tools/check/src/docs-work-item-links.ts
    - tools/check/src/schema-tables.ts
    - docs/design/architecture/runtime.md
    - docs/design/data/database.md
  tests:
    - tools/render-docs/src/render.test.ts
    - tools/check/src/terminology.test.ts
    - tools/check/src/docs-work-item-links.test.ts
    - tools/check/src/schema-tables.test.ts
  stop_before_reading: [spec, frontend/src/features]
---

# 文書サイトを広い画面と大きな図で読みやすくし、文書の語彙と所在を整理する

## 動機

生成した文書サイトと、その元になる `docs/` の Markdown には、読み手が内容へたどり着くまでの障害が複数ある。

| 問題 | 現状 | 読み手が受ける影響 |
| --- | --- | --- |
| 本文の幅 | `tools/render-docs/src/render.ts` の `--measure:760px` が本文の最大幅を固定し、画面が広くても本文は 760px に収まる | 図が縮小され、表のセルが折り返しやすい |
| 図の大きさ | Mermaid の図は本文の幅へ縮小され、拡大する手段がない | 大きな ER 図や状態図の文字が読めない |
| 箇条書きの構造 | `docs/design/data/database.md` の「列型の選択」「`tenant_id` の保持区分」など、同じ属性の組を持つ項目を箇条書きで並べている | 項目間の比較に、各項目の文を読み比べる必要がある |
| データベースの語彙 | データベースのレコードとカラムを「行」「列」と書いている | 一般語の「行」「列」と区別できず、何を指すかを文脈から推定することになる |
| runbook の語彙 | 「runbook」「ランブック」と「運用手順書」が混在している | 同じ種類の文書を別の名前で呼ぶ |
| work item への参照 | 生成サイトのトレーサビリティページが、規則と例ごとに参照元の work item を「作業項目」カラムに並べ、`wi-` の識別子が 1205 件現れる。`docs/development/` は `check-work-item-references` の対象外で、`wi-409` への参照が残る | 静的な文書から、完了すると更新されない変更記録へ依存が向く |
| 実行単位の中身 | [ランタイムアーキテクチャ](../../docs/design/architecture/runtime.md)は API、Worker、Batch、Seed、フロントエンドゲートウェイの責務を一行ずつ述べるだけで、Worker が処理するジョブ、Batch の定期処理、Seed が投入するデータ、フロントエンドの画面の一覧と、その詳細の所在を示さない | 各実行単位が何をするかを知るために、文書全体とコードを読むことになる |

## 対象範囲

- 文書サイトの配置を、広い画面で本文の幅を広げ、表、図、コードブロックが本文より広い幅を使えるように変える。
- 文書サイトの Mermaid の図に、画面全体へ拡大し、拡大率の変更と移動ができる表示を加える。
- `docs/` と、文書体系が所有するルート直下の Markdown（`DOCUMENTATION_GUIDE.md`、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md`）で、同じ属性の組を持つ項目を並べる箇条書きを表にする。
- データベースのレコードとカラムを「レコード」「カラム」と書く。
- 「runbook」「ランブック」を「運用手順書」と書く。
- 文書サイトのトレーサビリティページから work item の参照を除き、`docs/development/` の work item 参照を除いて検査の対象に含める。
- ランタイムアーキテクチャに、実行単位ごとに実装する機能と詳細の所在を示す節を加え、フロントエンドの画面の一覧を[フロントエンド設計](../../docs/design/application/frontend.md)に加える。
- 上の語彙を `check-terminology` の規則として固定する。

## 対象外

- `docs/releases/` のリリースノートとアップグレードノート。変更単位で書く文書であり、ファイル名に work item 名を使うことは [WORK_ITEM_FORMAT.md](../../WORK_ITEM_FORMAT.md) が定める。生成サイトにも含まれない。
- `WORK_ITEM_FORMAT.md` の例に現れる `wi-48213`。work item の形式を説明する文書であり、例の番号は特定の記録を指さない。
- ディレクトリ名 `docs/runbooks/`、Prometheus のアノテーション `runbook_url` など、識別子とパスとしての `runbook`。
- CSV の行と列、表の行と列、ログの行など、データベース以外の「行」「列」。
- Worker のジョブ、Batch のサブコマンド、画面の一覧が実装と一致することの機械検査。一覧の粒度が定まってから、ずれが実際に生じるかを見て判断する。
- `docs/design/application/api-guidelines.md` の各規則にある「目的」「担保手段」「適用状況」の箇条書き。規則ごとに見出しで区切った一件分の記述であり、表にしても 2 カラムの項目と値の表になるだけで比較の助けにならない。

## 設計

### 本文の幅

文書サイトの本文カラムを、画面の幅に応じて広がる可変幅にし、要素の種類ごとに最大幅を変える。

| 要素 | 最大幅 | 理由 |
| --- | --- | --- |
| 段落、箇条書き、引用、見出し | `--measure` を 760px から 960px 程度へ広げる | 日本語の本文は一行が長すぎると行頭へ戻る視線の移動が長くなる。現状より広げつつ上限は保つ |
| 表、Mermaid の図、コードブロック | 本文カラムの全幅（目次を除き最大 1400px 程度） | 折り返しと縮小を減らす |
| ページ全体 | 目次を含めて上限を設け、超えた分は左右の余白にする | 4K などの広い画面で一行が際限なく伸びるのを防ぐ |

目次を隠す 1200px 以下、サイドバーを隠す 900px 以下の既存の段階はそのまま保つ。
正確な値は、1440px、1920px、390px の幅で表示を確かめてから決める。

採らない案は次のとおりである。

| 案 | 採らない理由 |
| --- | --- |
| `--measure` だけを一律に広げる | 表と図に必要な幅まで段落を広げると、本文の一行が長くなりすぎる |
| 本文カラムの上限をなくす | 広い画面で段落の一行が画面の端から端まで伸びる |

### 図の拡大表示

各 `.diagram-shell` に「拡大表示」ボタンを置き、押すと画面全体を覆う `<dialog>` に描画済みの SVG を複製して表示する。
実装は `render.ts` が出力する `site.js` と `site.css` に置き、新しい依存は加えない。

| 操作 | 振る舞い |
| --- | --- |
| 開いた直後 | 図全体が画面に収まる拡大率で表示する |
| マウスホイール、ピンチ | カーソルまたは二本指の中心を基準に拡大率を変える |
| ドラッグ | 図を移動する |
| 拡大、縮小、全体表示のボタン | キーボードでも同じ操作ができる |
| `Esc`、閉じるボタン | ダイアログを閉じ、フォーカスを元の「拡大表示」ボタンへ戻す |

ダイアログはページ内の要素なので、ライトテーマとダークテーマの CSS 変数と、`.diagram-shell .mermaid svg` に掛けている線の色の上書きを、ダイアログ内の図にも同じ選択子で適用する。
ボタンの文言は、生成サイトの他の文言と同じく日本語とする。

採らない案は次のとおりである。

| 案 | 採らない理由 |
| --- | --- |
| `svg-pan-zoom` などのライブラリを加える | 拡大と移動だけのために依存と配布物が増える |
| SVG を新しいタブで開き、ブラウザーの拡大機能に任せる | ページの CSS 変数と線の色の上書きが効かず、ダークテーマで図が読めなくなる |
| 図をクリックしたら拡大する | 図の上でのテキスト選択やスクロールと衝突し、キーボードからも操作できない |

### 表にする箇条書き

同じ属性の組を持つ項目が 3 件以上並ぶ箇条書きを表にする。
項目ごとに属性が異なる列挙や、手順の列挙は対象としない。
着手時点で確認した対象は次のとおりである。
実装では同じ基準で `docs/` 全体を見直し、見つけたものも同じ変更で表にする。

| 文書 | 箇所 | 表のカラム |
| --- | --- | --- |
| `docs/design/data/database.md` | カラム型の選択 | 値の種類、採用する型と制約、使わない型、理由 |
| `docs/design/data/database.md` | `tenant_id` の保持区分 | テーブルの種類、`tenant_id` の扱い、該当するテーブル、理由 |
| `docs/domain/structure.md` | 技術スタック | 領域、採用する技術 |
| `docs/domain/structure.md` | イベントのエンベロープと公開項目の語彙 | 構成要素、内容、利用者 |
| `docs/domain/oauth2/decisions.md` | 認証とアクセスの境界 | 経路、主体の決め方、到達できる範囲 |
| `docs/design/observability/logging.md` | ログで扱わない用途 | 用途、扱う場所 |
| `docs/requirements/quality.md` | 値の種別 | 種別、意味 |
| `docs/requirements/product-overview.md` | 利用者の種類 | 利用者、関心 |
| `docs/runbooks/backup-restore-dr.md` | 定期的な運用 | 運用、頻度または契機、内容 |

表の見出しは「判定内容」「失敗時の動作」のような形式的な名詞とし、セルを空にしない。

### データベースの語彙

データベースを主題とする文で、row を「レコード」、column を「カラム」と書く。
複合語も同じ扱いとし、「非キー列」は「非キーカラム」、「列型」は「カラム型」、「`tenant_id` 列」は「`tenant_id` カラム」とする。
PostgreSQL の機能名として定着した「行レベルセキュリティ」のような語が現れる場合は、製品の用語として残す。

`tools/check/src/schema-tables.ts` は `database.md` のテーブル一覧を見出し「`tenant_id` 列」と値「非キー列」で読むので、同じ変更で「`tenant_id` カラム」と「非キーカラム」へ改め、テストの入力も揃える。

`check-terminology` には、データベース以外の意味を取り得ない複合語だけを規則として加える。

| 採らない表記 | 採用する表記 | 通す literal |
| --- | --- | --- |
| `非キー列` | 非キーカラム | なし |
| `列型` | カラム型 | `配列型` |

単独の「行」「列」は CSV、表、ログにも使う一般語なので規則にしない。
これらは実装時に文書を読んで書き換え、機械検査は複合語に限る。

### runbook の語彙

地の文と表見出しの「runbook」「Runbook」「ランブック」を「運用手順書」に改める。
対象は `docs/`、ルート直下の文書体系の Markdown、`render-docs` が出力するナビゲーションと見出しの文言、`infra/` のスクリプトと設定にある日本語のコメントである。
`check-terminology` に `ランブック`、`runbook`、`Runbook` の規則を加え、パスの `runbooks/` とアノテーション名の `runbook_url` を通す literal とする。

### work item への参照

トレーサビリティページの「作業項目」カラムを除く。
`tools/render-docs/src/traces.ts` の work item の収集は、端末で読む `mise run brief` も同じ走査を使うので残し、生成サイトの表示からだけ外す。
規則や脅威を担う work item は、`mise run brief` と `mise run spec-where <id>` が引けるので、生成サイトに一覧を持たなくても調べる手段は残る。

`tools/check/src/docs-work-item-links.ts` の対象外から `docs/development/` を除く。
`docs/development/specification-first-workflow.md` の `wi-409` への参照は、その記録から現在も有効な内容を本文に書き、参照を除く。
`docs/development/local-development.md` のコマンド例 `mise run brief -- wi-123` は `mise run brief -- wi-<番号>` とする。
`docs/releases/` は対象外のまま残す。

### 実行単位が実装する機能

ランタイムアーキテクチャに「実行単位が実装する機能」の節を加え、実行単位ごとに機能の一覧または詳細の所在を示す。
各項目の詳細は所有する Context の文書に残し、この節は一覧と参照先だけを持つ。

| 実行単位 | この節に書く内容 | 詳細の所在 |
| --- | --- | --- |
| API | API リファレンスにあるすべてのエンドポイントを、API ガイドラインに従って実装することを一文で述べる | 生成サイトの API リファレンス、[API ガイドライン](../../docs/design/application/api-guidelines.md) |
| Worker | 登録する `JobKind` ごとに、投入する Context、処理の内容、詳細の所在を表にする | 各 Context の文書、[Jobs](../../docs/domain/jobs/README.md) |
| Batch | サブコマンドごとに、処理の内容、実行の契機、詳細の所在を表にする | `infra/k8s/base/batch-cronjobs.yaml`、各 Context の文書 |
| Seed | 投入する資源の種類（`first_party_clients`、`development_demo`）、マニフェストの与え方、起動の方法を述べる | [Seeding](../../docs/domain/seeding/README.md) |
| フロントエンドゲートウェイ | 画面群の一覧への参照を置く | [フロントエンド設計](../../docs/design/application/frontend.md)の画面の一覧 |

着手時点のコードでは、Worker は `backend/cmd/idmagic-worker/worker.go` で `noop_echo`、利用者、グループ、グループメンバーシップの CSV インポートのプレビューと確定、`dynamic_group_reconcile`、データエクスポート、`data_key_reencryption`、`lifecycle_workflow_run`、`provisioning_task`、`oauth2.RegisterJobHandlers` が登録するジョブを登録する。
Batch は `backend/cmd/idmagic-batch/main.go` で `retention-sweep`、`signing-key-lifecycle`、`data-key-reencryption-sweep`、`restore-consistency-check` を持つ。
実装時にはこれらを登録箇所から改めて列挙し、定期ディスパッチャーのように Worker のプロセス内で周期的に動く処理も Worker の行に含める。

Batch の実行間隔は `batch-cronjobs.yaml` が一次情報なので、節には「毎時」「毎日」のような契機だけを書き、cron 式を複製しない。

フロントエンドの画面の一覧は、`frontend/src/routes/` の約 100 経路をそのまま並べず、ログイン、アカウント、管理コンソールなどの画面群と、その主な画面を単位にする。
経路ごとの一覧は `frontend/src/routes/` が一次情報である。

## 計画

1. `render-docs` の配置と拡大表示を実装し、`render.test.ts` に配置の CSS と拡大表示のテストを加える。拡大表示のテストは happy-dom で `site.js` を実行し、ボタンを押すとダイアログが開いて SVG の複製を含むことと、`Esc` で閉じてフォーカスが戻ることを確かめる。
2. トレーサビリティページから work item を除き、`docs/development/` を `check-work-item-references` の対象に含め、該当箇所を書き換える。
3. `check-terminology` に語彙の規則を加えて RED を観測し、`schema-tables.ts` を含めて文書を書き換える。
4. 箇条書きを表にする。
5. ランタイムアーキテクチャとフロントエンド設計に実行単位の機能を書く。
6. `mise run render-docs` と `mise run serve-docs` で 1440px、1920px、390px の幅の表示と、ライトテーマとダークテーマの拡大表示を確かめる。

作るものを変え得る問いは残っていない。
本文と表の最大幅の正確な値は、手順 6 で表示を見て決める。

## タスク

- [x] T001 [Acceptance] 拡大表示と配置のテストを `render.test.ts` に加え、RED を確認する。
- [x] T002 [App] 本文カラムの可変幅と、要素ごとの最大幅を実装する。
- [x] T003 [App] 図の拡大表示を実装する。
- [x] T004 [Acceptance] `docs-work-item-links.ts` の対象外から `docs/development/` を除き、`mise run check-work-item-references` の RED を確認する。
- [x] T005 [Docs] `docs/development/` の work item 参照を書き換え、トレーサビリティページから work item を除く。
- [x] T006 [Acceptance] `check-terminology` に語彙の規則を加え、RED を確認する。
- [x] T007 [Docs] データベースの語彙と runbook の語彙を書き換え、`schema-tables.ts` とそのテストを揃える。
- [x] T008 [Docs] 同じ属性の組を持つ箇条書きを表にする。
- [x] T009 [Docs] ランタイムアーキテクチャに実行単位が実装する機能の節を加え、フロントエンド設計に画面の一覧を加える。
- [x] T010 [Verify] 生成サイトを複数の画面幅とテーマで確かめ、`mise run verify` を通す。

## 検証

- `mise run test-tools`（`render.test.ts`、`terminology.test.ts`、`schema-tables.test.ts`、`docs-work-item-links.test.ts` を含む）
- `mise run check-terminology`
- `mise run check-work-item-references`
- `mise run check-schema-tables`
- `mise run check-rendered-docs`
- `mise run serve-docs` で表示を目視確認する。
- `mise run verify`

## リスク

| リスク | 緩和策 |
| --- | --- |
| 本文の幅を広げると、狭い画面の配置が崩れる | 既存の 1200px と 900px の段階を保ち、390px の幅でも表示を確かめる |
| 拡大表示の複製した SVG で、Mermaid が SVG 内に埋め込む `id` が重複する | 複製の `id` と、それを参照する `url(#...)` を置き換えるか、表示中だけ元の図を移す。テストで同じ `id` が二つ存在しないことを確かめる |
| 「行」「列」の書き換えで、データベース以外の意味まで変える | 単独の「行」「列」は機械置換せず、文を読んで書き換える |
| Worker、Batch、画面の一覧が実装とずれる | 一覧の詳細は各 Context の文書に残し、この節は種類と参照先に限る。機械検査は対象外に記録した判断に従って後で検討する |

## 完了
- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` は REQ-APPLICATION-014、REQ-IDGOVERNANCE-005、REQ-JOBS-011、REQ-PROVISIONING-016、REQ-SYSTEM-012 の変更を挙げる。
  いずれもデータベースの「行」「列」を「レコード」「カラム」と書き直しただけで、条件も要求する結果も変わらない。
  規範以外では次を変えた。
  文書サイトは、段落を 960px までに保ったまま、表、図、コードブロックが目次を除く本文カラムの全幅（最大 1400px）を使う。
  各図に「拡大表示」ボタンを置き、描画済みの SVG を全画面のダイアログへ移して、拡大、縮小、全体表示、ホイールとピンチによる拡大、ドラッグによる移動を提供する。
  トレーサビリティページから「作業項目」カラムを除き、`docs/development/` を work item 参照の検査の対象に含めた。
  `check-terminology` に「非キー列」「列型」「ランブック」「runbook」「Runbook」の規則を加え、文書の語彙を「カラム」「レコード」「運用手順書」へ改めた。
  同じ属性の組を持つ箇条書きを表にし、ランタイムアーキテクチャに実行単位ごとの機能（Worker のジョブと周期処理、Batch のサブコマンド、Seed のプロファイル）と詳細の所在を、フロントエンド設計に画面の一覧を加えた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-item-references`、`mise run check-terminology`
  - **Requirement**: N/A: 文書と文書サイトの変更であり、規範となる製品要件がない
  - **Observed Failure**: `docs/development/` を対象に含めると `wi-123` と `wi-409` の 2 件を指摘した。語彙の規則を加えると `非キー列` 43 件、`列型` 2 件、`runbook` 29 件、`Runbook` 3 件、`ランブック` 8 件を指摘した。
  - **Detection Reason**: どちらも文書全体を走査する検査なので、書き換え漏れがあれば該当する行と語を示して失敗する。
- **Unit RED Evidence**:
  - **Test**: `tools/render-docs/src/render.test.ts` の `site layout and diagram viewer` の 4 件と `renders a linked multi-page documentation site`、`tools/check/src/terminology.test.ts`、`tools/check/src/docs-work-item-links.test.ts`、`tools/check/src/schema-tables.test.ts`
  - **Requirement**: N/A: 文書と文書サイトの変更であり、規範となる製品要件がない
  - **Observed Failure**: 実装前に、本文幅の CSS、拡大表示ボタンのマークアップ、ダイアログの表示、ツールバーでの拡大と閉じたときの復元の 4 件が失敗した。トレーサビリティページは「作業項目」を含んで失敗した。語彙の規則のテスト、`docs/development/` を対象とするテスト、`schema-tables` の見出しと値のテスト 7 件がそれぞれ失敗した。
  - **Detection Reason**: 拡大表示のテストは happy-dom で生成した `site.js` を実行し、ボタン操作後の DOM（ダイアログの開閉、SVG の位置、`id` の数、変換の変化、フォーカス）を観測する。
- **Change-Resistance Results**:
  SVG を移さず `cloneNode(true)` で複製する誤実装を注入すると、`id` の重複を表明するテストが失敗した。
  閉じたときに元のボタンへフォーカスを戻す行を削除すると、フォーカスを表明するテストが失敗した。
  データベースの「行」「列」の書き換えでは、データベースを主題とする 8 文書に限って直前の文字を条件とする正規表現で置換し、差分を読んで誤りを 3 件戻した（`文字列` が `文字カラム` になった箇所と、表の列を指す `capacity.md` の 2 か所）。それ以外の文書は文ごとにリテラルで置換した。
  生成サイトを `Bun.WebView` で 1920px、1440px、390px の幅に表示し、段落、表、図の幅（1920px で 960px、1278px、1278px）、横スクロールが出ないこと、拡大表示の開閉と、閉じた後に図が元の位置へ戻り、フォーカスがボタンへ戻ることを確かめた。390px ではボタンが横スクロール領域の外へ押し出されていたので、配置を直して再確認した。撮影した環境はダークテーマであり、ライトテーマの表示は目視していない。
- **Verification Results**:
  - `mise run test-tools` - 成功（592 件）
  - `mise run lint-tools`、`mise run typecheck-tools` - 成功
  - `mise run check-rendered-docs`、`mise run check-links`、`mise run check-terminology`、`mise run check-work-item-references`、`mise run check-schema-tables` - 成功
  - `mise run verify` - 成功
