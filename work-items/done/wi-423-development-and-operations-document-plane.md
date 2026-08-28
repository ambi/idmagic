---
depends_on: []
status: completed
authors: [tn]
risk: low
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/README.md, docs/structure.md, docs/deployment.md, docs/capacity.md]
  source:
    - README.md
    - CONTRIBUTING.md
    - docs/development/specification-first-workflow.md
    - SPECIFICATION_FORMAT.md
    - DOCUMENTATION_GUIDE.md
    - WORK_ITEM_FORMAT.md
    - infra/README.md
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
    - docs/runbooks/backup-restore-dr.md
    - work-items/wi-375-evaluate-yaml-fences-for-specification-tables.md
    - work-items/wi-425-method-document-reinforcement.md
    - work-items/done/wi-404-repository-entrance-documents-are-missing.md
    - work-items/done/wi-405-spec-and-docs-boundary-is-not-legible.md
    - work-items/done/wi-407-name-the-directory-after-the-kind.md
    - tools/check/src/specification-doc.ts
    - tools/render-spec-docs/src/main.ts
    - tools/render-spec-docs/src/render.ts
  tests: [tools/check/src/specification-doc.test.ts, tools/render-spec-docs/src/render.test.ts]
  stop_before_reading: [backend, frontend, spec/contexts, docs/contexts]
created_at: 2026-08-27
priority: p2
change_kind: docs
spec_impact: { kind: none, reason: "製品概要と手順の文書平面を新設するもので、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# 手順と製品概要の平面を、形式文書が宣言したとおりに存在させる

## Motivation

`SPECIFICATION_FORMAT.md` の配置図は `docs/product-overview.md`（問題、利用者、非目標）と `docs/development/`（環境、ビルド、生成、CI、テスト、リリースの手順）を載せている。どちらも存在しない。形式文書が宣言している配置と実体がずれており、配置図を信じて探した人は何も見つからない。

欠けているのは器だけではない。中身が本当に無い。

リリースの単位、頻度、後退の手順がどこにも書かれていない。`DOCUMENTATION_GUIDE.md` は 9.2 で「リリースと後退」を文書の一種として挙げ、9.3 で「バックアップと復旧」を挙げている。後者は `docs/runbooks/backup-restore-dr.md` として存在するが、前者は存在しない。`CONTRIBUTING.md` は「必須の検査の一覧をこの文書に複製しません。正本は CI のワークフローです」と、複製を避ける正しい判断をしているが、その結果として「どうやってリリースするか」を書いた場所がどこにも無い。

Extreme Programming の技術プラクティスのうち、小さな変更、継続的統合、リファクタリング、テスト先行は入っているのに、小さなリリースだけが完全に欠落している。開発プロセスそのものを測る指標（変更のリードタイム、デプロイの頻度、変更失敗率、復旧時間）も無い。`docs/capacity.md` が製品の性能について確立した Evidence classes の規律を、開発プロセスへ適用する余地がそのまま残っている。

製品概要も同様である。`docs/README.md` は Context Map と索引を持つが、この製品が誰のどんな問題を解くのか、何を目標としないのかを述べた場所が無い。Bounded Context の索引表から製品像を再構成することはできない。

手順の文書は仕様とは種類が違う。仕様は「製品が何をするか」を述べ、手順は「人が何をするか」を述べる。同じ文書平面に混ぜないという判断は `docs/README.md` が既にしている（「手順はこの平面には置かない」）。その受け皿を作る。

## Scope

- **`docs/product-overview.md`**：解く問題、想定する利用者、非目標を書く。`SPECIFICATION_FORMAT.md` が既に宣言している内容に従う。
- **`docs/development/`**：環境、ビルド、生成、CI、テスト、リリースの手順を置く。`README.md` と `CONTRIBUTING.md` にある手順は、正本をこの平面へ移して参照へ置き換える。
- **リリースと後退**：リリースの単位と頻度、版の付け方、後退の判断基準と手順を書く。
- **開発プロセスの指標**：変更のリードタイム、デプロイの頻度、変更失敗率、復旧時間を測るかどうかを判断し、測るなら `docs/capacity.md` と同じ Evidence classes の区分で書く。
- **手順文書の分類**：手順、参照、解説をどう分けるかの方針を決め、`docs/development/` の中の命名に反映する。`docs/runbooks/` の 8 本が既に「事象の最中に読むもの」として分かれていることと整合させる。
- **索引の更新**：`docs/README.md` の文書索引へ新しい平面を加える。

## Out of Scope

- CI ワークフローの変更。必須の検査の正本は `.github/workflows/idmagic-ci.yaml` であり、本 work item はそれを文書へ複製しない。
- 実際のリリース作業と版付けの開始。手順を書くところまでを担う。
- 指標の収集基盤の構築。測るかどうかの判断と、測るなら何をどう測るかの定義までを担う。
- `DOCUMENTATION_GUIDE.md` の改訂。同文書はリポジトリに依存しない一般論を述べる位置づけであり、本件の対象はこのリポジトリの実体である。

## Design

`docs/development/` と `docs/runbooks/` は `SPECIFICATION_FORMAT.md` が述べるとおり「閉じたファイル集合の下に位置し、手順は固定した種類の集合ではないので自由に名前を付ける」。したがって新しいファイル名の検査は不要で、既存の検査器に手を入れる必要はない。`docs/product-overview.md` は閉じた集合の側に属し、`specification-doc.ts` の許可リストに既に含まれているかを確認する。含まれていなければ加える。

手順文書の分類には Diátaxis を用いる。学習のための手引き、目的を達成するための手順、事実の参照、背景の解説という 4 分類は、この製品の手順文書の実態によく合う。`docs/runbooks/` は「事象の最中に読むもの」として既に目的達成型の手順に相当し、`CONFIGURATION.md` は参照に相当する。採用の理由は、`docs/` の正本文書が既に「ファイル名がその中身の種類を語る」という同じ原理で組まれており、手順の平面へ同じ原理を延長するだけで済むことである。4 分類を機械検査はしない。手順は種類が固定されないという `SPECIFICATION_FORMAT.md` の判断を覆さないためである。

開発プロセスの指標については、測ることの費用と得られるものを見て判断する。デプロイの頻度と変更のリードタイムは Git と CI の履歴から導けるため費用が低い。変更失敗率と復旧時間は、何を失敗と数えるかの定義が要り、その定義が曖昧なままだと数値が意味を持たない。前 2 者だけを採る判断も正当であり、その場合は後 2 者を採らない理由を書く。

リリースと後退の書き場所には、`docs/development/` に置く案と `docs/runbooks/` に置く案がある。採るのは分割で、計画されたリリースの手順は `docs/development/` に、後退の判断と実行は `docs/runbooks/` に置く。後退は事象の最中に読むものであり、runbook の既存の書式（着手条件、最初に確認すること、実施、確認、失敗したとき）がそのまま当てはまる。

`DOCUMENTATION_GUIDE.md` を配置の基準にする。ルートの `DEVELOPMENT.md` は仕様先行ループ、証拠契約、検証のはしごを持つ開発文書であり、ルートに置く例外を支える別の役割はない。内容を `docs/development/specification-first-workflow.md` へ移し、エージェント向けの案内、work item 書式、スキル、生成仕様サイトを含む参照を張り替える。

既存の記述は種類に従って移す。`README.md` の製品説明、対象範囲、対象外は `docs/product-overview.md` へ移し、環境とコマンドの手順は `docs/development/` へ移す。`CONTRIBUTING.md` は Pull Request の規則だけを残し、手順を参照する。同じ内容を 2 か所に置かない。

リリースは API、ワーカー、バッチ、UI を同じコミットと版に束ねる。版は Semantic Versioning の `vMAJOR.MINOR.PATCH` タグで識別し、利用者に見える互換な変更は `MINOR`、互換性を壊す変更は `MAJOR`、修正だけなら `PATCH` とする。固定した暦上の頻度は置かず、依存関係のない完了済み work item を不必要に束ねない小さなリリースを原則とする。現在はリリース用 CI が存在しないため、文書は検証、版付け、成果物の同一性、段階的展開、確認までを定め、実行の自動化は対象外に残す。

開発プロセスの 4 指標は、現時点では採らない。Git と CI だけでは「本番へ配備された時刻」「その配備が障害を起こしたか」「復旧した時刻」を確定できず、変更のリードタイムとデプロイ頻度を含む 4 指標を再現可能な Measurement にできないためである。`docs/development/process-metrics.md` に、目的、必要な事象、Evidence classes による採用条件、再評価条件を記録する。配備と障害の事象を保存する基盤は Out of Scope のままとする。

生成仕様サイトは、正準仕様と方法論に加えて `docs/development/*.md` を独立した Development グループとしてレンダリングする。これにより、`docs/README.md` から手順へ移動しても生成サイト上のリンクがリポジトリ外へ抜けない。`docs/runbooks/` は障害時に別の入口から読む平面であり、本変更では生成サイトへ取り込まない。

## Plan

1. `specification-doc.ts` の許可リストに `product-overview.md` があるかを確認する。無ければ加える。
2. `DEVELOPMENT.md` を `docs/development/specification-first-workflow.md` へ移し、`SPECIFICATION_FORMAT.md`、`docs/README.md`、`docs/structure.md` と全参照を更新する。
3. `docs/product-overview.md` を書き、`README.md` の製品説明と対象外を移す。
4. `docs/development/` を作り、環境、生成、CI、テストの手順を移す。`README.md` と `CONTRIBUTING.md` は入口と規則だけに絞る。
5. リリースの手順を `docs/development/` に、後退を `docs/runbooks/` に書く。
6. 開発プロセスの指標を採らない理由と再評価条件を書く。
7. 生成仕様サイトへ Development グループを加える。

## Tasks

- [x] T001 [Baseline] 許可リスト、既存の手順、リリース用 CI とタグ、配備資材、開発プロセス指標の情報源を確認した。
- [x] T002 [Spec] `docs/product-overview.md` を書き、既存の許可リストで受け入れられることを確認した。
- [x] T003 [Spec] `DEVELOPMENT.md` を `docs/development/specification-first-workflow.md` へ移し、`SPECIFICATION_FORMAT.md`、`docs/README.md`、`docs/structure.md` と全参照を更新した。
- [x] T004 [Docs] `docs/development/` に環境、生成、CI、テストの手順と索引を書いた。
- [x] T005 [Docs] `README.md` と `CONTRIBUTING.md` を入口と規則へ絞り、移した手順を参照させた。
- [x] T006 [Docs] リリースの手順を書いた。
- [x] T007 [Docs] 後退の runbook を既存の書式で書いた。
- [x] T008 [Docs] 開発プロセスの 4 指標を採らない理由、必要な事象、再評価条件を書いた。
- [x] T009 [Acceptance] `mise run spec-render` 後に `spec/generated/docs/development/index.html` が無い RED を確認し、Development 平面を生成サイトへ加えた。
- [x] T010 [Unit] `renderSpecificationSite` が Development 文書を独立したグループへ出すテストを RED から GREEN にした。
- [x] T011 [Verify] `mise run check-spec` と `mise run verify` を通し、生成された仕様サイトから新しい平面へ到達できることを確認した。

## Verification

- `mise run check-spec` が新しいファイルを受け入れ、閉じた集合の側に許可されていないファイル名は引き続き拒否する。
- `SPECIFICATION_FORMAT.md` の配置図に載る全ての項目が実在する。
- 生成された仕様サイトから `docs/development/` と `docs/product-overview.md` へ到達できる。
- 同じ内容が `README.md`、`CONTRIBUTING.md`、`docs/development/` の各文書で重複していない。
- `mise run verify`

Acceptance RED は `mise run spec-render` に続けて `test -f spec/generated/docs/development/index.html` を実行する。製品の規範的な振る舞いは変えないため Requirement は `N/A` とし、生成された入口から手順平面へ到達できない現在の失敗を代替の観測可能境界とする。

Unit RED は `mise run test-tools` で、`renderSpecificationSite` に Development 文書を渡しても独立したページとナビゲーション群が生成されない失敗を観測する。製品のドメインまたはユースケースロジックを変えないため Requirement は `N/A` とする。

## Risk Notes

手順の平面を作ると、既存文書からの重複が生まれやすい。特に `CONTRIBUTING.md` と仕様先行ワークフローは内容が近く、移設と参照の区別が曖昧になると複数箇所に似た記述が並ぶ。参照だけを置き、内容を書かないという原則を各ファイルの冒頭に明記する。

`docs/product-overview.md` は書き手の主観が入りやすく、製品の現状ではなく願望を書いてしまう危険がある。非目標を先に書くと、書ける範囲が自然に定まる。

開発プロセスの指標は、測ること自体が目的化しやすい。何のために測るか（たとえば「変更のリードタイムが延びていることを検知して、work item の粒度を見直す」）を先に書き、書けない指標は採らない。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を報告した。製品の規範的な振る舞いは変えず、ルートの `DEVELOPMENT.md` を `docs/development/specification-first-workflow.md` へ移し、製品概要、ローカル開発、CI、テスト、リリース、開発プロセス計測、リリース後退の文書を種類別の平面へ配置した。ルート `README.md` と `CONTRIBUTING.md` は入口と規則へ絞った。生成仕様サイトには `docs/development/*.md` を表示する Development グループを追加した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run spec-render` の後に `test -f spec/generated/docs/development/index.html` を実行した。
  - **Requirement**: N/A: 製品の規範的な振る舞いではなく、開発文書の生成サイト上の到達性を変更するため
  - **Observed Failure**: 実装前は Development の入口が生成されず、ファイル存在検査が終了コード 1 になった。
  - **Detection Reason**: 文書を作っただけで生成器へ登録しない実装では同じ失敗になり、生成サイトから開発文書へ到達できるという受け入れ境界を区別できる。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools` の `renderSpecificationSite > renders development documents as their own navigable plane`。
  - **Requirement**: N/A: 製品のドメインまたはユースケースロジックではなく、仕様サイト生成器の分類と経路を変更するため
  - **Observed Failure**: `development/index.html` が `undefined` となり、新しいテスト 1 件だけが失敗した（220 passed、1 failed）。
  - **Detection Reason**: `docs/development/` を従来どおり方法論文書または正本文書として扱う実装では、独立した出力経路とナビゲーション群を生成できず失敗する。
- **Independent Verification**:
  新しい文脈のエージェント `/root/wi423_independent_review` に仕様適合と規約適合の独立レビューを依頼したが、エージェント実行環境の利用上限により所見を返す前に終了した。本 work item は `risk: low` であり、`risk-based-v2` が独立検証を必須にする `medium` 以上には当たらない。実装者による差分監査では、旧パスの残存、Markdown リンク、文書配置、コマンドの実在、生成サイトの経路を確認した。
- **Change-Resistance Results**:
  `risk: low` の文書・ツール変更なので追加の mutation または fault injection は行っていない。Development 文書を収集しない従来実装が Acceptance RED と Unit RED の双方で検出されることを確認した。
- **Verification Results**:
  - `mise run test-tools` - passed（221 tests）。
  - `mise run check-links` - passed（612 documents）。
  - `mise run spec-render` - passed（987 pages、147 documents）。
  - `test -f spec/generated/docs/development/index.html` - passed。
  - `mise run spec-diff` - passed（no normative specification change against main）。
  - `mise run verify` - passed。

## Left Undone

- 成果物を公開するリリース用 CI と外部のリリース経路は実装していない。実際の版タグ作成、成果物公開、本番配備も行っていない。
- 配備と障害の事象を保存する計測基盤は実装していないため、変更のリードタイム、デプロイ頻度、変更失敗率、復旧時間の計測は開始していない。
- `DOCUMENTATION_GUIDE.md` 自体は変更していない。リポジトリ固有の配置と規則は `SPECIFICATION_FORMAT.md` と `docs/development/` に反映した。
