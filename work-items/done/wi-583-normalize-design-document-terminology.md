---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p1
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "文書の語彙と表記だけを変え、規範 ID、TypeSpec のシンボル、設定キー、プロダクトの振る舞いは変えない。EX-SIGNINGKEYS-012-01 の Given 1 行が「配備の」から「デプロイ先の」になるため spec-diff は REQ-SIGNINGKEYS-012 を挙げるが、条件も要求する結果も同じである。"
documentation_impact:
  level: none
  reason: "変わるのは開発者と運用者が読む設計文書の語彙だけで、利用者が観測する振る舞い、API、設定キーはどれも変わらない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - docs/domain/glossary.md
    - docs/design/architecture/deployment.md
    - docs/runbooks/backup-restore-dr.md
    - infra/backup/restore-drill.sh
    - tools/check/src/registry.ts
    - tools/check/src/runner.ts
    - tools/check/src/agent-guidance.ts
    - tools/check/src/check-agent-guidance.ts
    - tools/check/src/check-documents.ts
    - tools/workspace/src/workspace.ts
    - tools/check/README.md
  tests:
    - tools/check/src/registry.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading:
    - work-items/done/
    - backend/
    - frontend/
---

# 設計文書の用語を定着した外来語へ揃え、表記ゆれをなくす

## Motivation

設計文書の主要な概念が、一般的すぎる日本語へ訳されている。
「配備」「秘密」「実行時」「規則」「容量」「観測可能性」「基盤」は、どれも普通名詞としての読みが先に立ち、その節が deployment、secret、runtime、guideline、capacity、observability、platform のどれを指しているのかを読み手が文脈から推定することになる。
用語から概念へたどれないため、検索も外部資料との突き合わせも効かない。

同じ語の表記も揺れている。
`docs/design/architecture/deployment.md` は見出しで「参照トポロジー」、本文で「参照トポロジ」と書く。

意味を復元できない語もある。
`docs/design/architecture/deployment.md` の「Docker Compose はローカル開発と訓練」は、何の訓練を指すのか本文から決まらない。
一方で `recovery.md` や `system-acceptance.md` の「復元訓練」は restore drill を指しており、同じ「訓練」が二つの別のものに使われている。

## Scope

- `docs/` 配下の Markdown と、`AGENTS.md`、`CONTRIBUTING.md`、`DOCUMENTATION_GUIDE.md`、`README.md`、`SECURITY.md`、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md` の用語を、採用語の表へ揃える。
- 採用語と、それが指す英語の概念を [用語集](../../docs/domain/glossary.md) へ登録する。
- 見出しを変えた文書について、他文書からのリンクとアンカーを追従させる。
- 題名が「ガイドライン」になる 2 文書のファイル名を `api-guidelines.md`、`design-guidelines.md` へ改め、リポジトリ全体の参照を追従させる。
- 「訓練」のように意味が復元できない箇所は、置換ではなく元の意味を書き直す。
- 採用しないと決めた表記の再発を検出する検査を追加する。

## Out of Scope

- Go と TypeSpec のコメントの用語、`en` ロケールの UI 文言、ログとエラーの文言。用語の正本を文書側で固めてから別途扱う。改名した 2 文書を名指すコメント中のパスだけは、参照が切れるので追従させる。
- `platform.md` の改名。題名は「プラットフォーム設計」になるが、ファイル名は既に `platform.md` であり英語の概念名と一致している。
- `REQ-*`、`EX-*`、`SLO-*`、`CAP-*`、設定キー、mise タスク名、Go と TypeSpec の識別子。
- `work-items/` の記録。完了済みも起票済みも、書かれた時点の記録であり後から書き換えない。
- `CONFIGURATION.md` と `ROUTE_PRIORITY.md`。生成物であり、かつ英語で書かれている。
- `infra/`、`tools/`、`frontend/` の `README.md`。文書体系の外にあり、`DOCUMENTATION_GUIDE.md` が所有しない。
- 文書の内容の拡充。後続の各 work item が扱う。

## Design

採用語は次のとおりとする。
外来語を選ぶ基準は、その語が指す概念に定着した英語があり、日本語へ訳すと普通名詞へ吸収されることである。
訳しても概念が保たれる語（縮退、冗長性、監査、保持）はそのまま残す。

| 採らない表記 | 採用 | 理由 |
| --- | --- | --- |
| 配備 | デプロイ（行為）、デプロイメント（ビュー名） | 「配備アーキテクチャ」は deployment view を指す |
| 秘密 | シークレット | [用語集](../../docs/domain/glossary.md) の `ConfigurationReference` が既に「シークレット」を使う |
| 実行時アーキテクチャ | ランタイムアーキテクチャ | runtime view を指す。副詞的な「実行時」は対象ではない |
| API 規則 | API ガイドライン | 設計の観点を並べた指針であり、個々の強制点は TypeSpec と検査が持つ |
| 設計規則 | 設計ガイドライン | 同上。兄弟文書に別の外来語を当てない |
| 容量 | キャパシティ | 「容量設計」は capacity planning を指す |
| 観測可能性 | オブザーバビリティ | |
| 基盤設計 | プラットフォーム設計 | `infrastructure/platform.md` を指していることが題名から読めない |
| トポロジ | トポロジー | 同一文書内の表記ゆれ |
| 入場制御 | アドミッションコントロール | 過負荷時に入口で受け付けを止める機構。訳語が定着していない |
| 参照運用プロファイル | リファレンスワークロードプロファイル | キャパシティ算出の設計入力となる想定負荷の記述 |
| 構成算出規則 | サイジング計算式 | レプリカ数と接続数を求める式そのもの |
| 縮退順序 | ロードシェディング順序 | 優先度の低い経路から受け付けを落とす順序 |

「訓練」は語を置き換えるのではなく、箇所ごとに意味を確定させる。
restore drill を指す箇所は「復旧ドリル」とする。
`docs/design/architecture/deployment.md` の「ローカル開発と訓練」は「ローカル開発と復旧ドリル」とする。
`infra/backup/restore-drill.sh` が `infra/docker/docker-compose.dev.yaml` を使い捨てプロジェクトとして起動しており、Docker Compose が実際に担っている二つ目の用途はこれである。

**機械的な一括置換はしない。** 対象語は別の意味でも現れる。
残す共起は次のとおりで、これがそのまま検査の許可条件になる。

| 残す表記 | 理由 |
| --- | --- |
| 秘密鍵 | private key。シークレットではない |
| 秘密情報 | 復号できる形で保持する機微データ。起動時シークレットとは別の概念 |
| 実行時に、実行時の拒否条件、実行時刻、実行時間、初回実行時 | 副詞的または時刻としての「実行時」であり runtime view ではない |
| 保存容量、空き容量、容量超過 | ストレージの量であり capacity planning ではない |
| 認可規則、検証規則、動的グループ規則、日本語文章規則 | 普通名詞としての規則であり、guideline の訳ではない |
| 運用基盤、実行基盤、観測基盤、ジョブ基盤、配信基盤、インフラ基盤、認証基盤、デプロイ基盤 | platform ではなく「二つ目の永続化基盤」のような一般名詞 |
| `Rule:` で始まる Gherkin の行 | 規範シナリオの構文要素 |

採らない案を三つ記録する。
一つは、外来語を避けて日本語のまま用語を精緻化する案である。既に用語集が「シークレット」を使っており、外部資料との突き合わせも英語の概念名で行うため採らない。
一つは、`api-rules.md` と `design-rules.md` のファイル名を残したまま題名だけを「ガイドライン」にする案である。ファイル名は識別子なので変えない、という理由で当初はこちらを採った。採らないのは、`rules` と `guidelines` という二つの英語が同じ文書を指す状態が残り、`rg` で文書を探す人が題名からファイル名へたどれなくなるためである。改名の費用は、リポジトリ全体の参照 20 箇所あまりの追従で尽きる。
一つは、検査の例外を専用の台帳ファイルへ宣言させる案である。例外は「どの語のどの共起を残すか」であって「どのファイルを免除するか」ではないので、理由付きの共起として規則表の中に置く。ファイル単位の免除を作ると、その文書だけ用語が戻ってもだれも気付かない。

## Plan

1. 用語表と、残す共起の一覧を確定する。
2. 採用しない表記を検出する検査を `tools/check` へ追加し、現在の文書に対して落ちることを確かめる。この検査が観測可能な境界になる。
3. 採用語を [用語集](../../docs/domain/glossary.md) へ登録し、指す英語の概念を併記する。
4. 文書の所有単位ごとに置換する。`architecture/`、`design/`、`operations/` と `verification/`、`contexts/`、`development/` と `runbooks/` と `releases/`、ルート直下の順とし、単位ごとに差分を読む。
5. 「訓練」の各箇所を、意味を確定させて書き直す。
6. 見出しが変わった文書について、参照元のリンクとアンカーを更新する。
7. リンク、アンカー、仕様、作業記録、リポジトリ検査を通す。

## Tasks

- [x] T001 [Design] 用語表と、残す共起の一覧を確定する。
- [x] T002 [Tools] 採用しない表記を検出する検査を追加し、現在の文書に対して RED を観測する。`mise run test-tools-file -- check/src/terminology.test.ts`、`mise run test-tools-file -- check/src/repository-checks.acceptance.test.ts`、`mise run check-terminology`。
- [x] T003 [Docs] 採用語を用語集へ登録する。
- [x] T004 [Docs] `docs/design/architecture/` の用語と見出しを揃える。
- [x] T005 [Docs] `docs/design/` の用語と見出しを揃える。
- [x] T006 [Docs] `docs/operations/`、`docs/design/verification/`、`docs/requirements/` を揃える。
- [x] T007 [Docs] `docs/domain/` を揃える。
- [x] T008 [Docs] `docs/development/`、`docs/runbooks/`、`docs/releases/`、`docs/README.md` ほか `docs/` 直下を揃える。
- [x] T009 [Docs] ルート直下の人が読む Markdown を揃える。
- [x] T010 [Docs] 「訓練」の各箇所を、意味を確定させて書き直す。
- [x] T011 [Docs] `api-guidelines.md` と `design-guidelines.md` へ改名し、リポジトリ全体の参照を追従させる。
- [x] T012 [Verify] リンク、アンカー、仕様、作業記録、リポジトリ検査を通す。

## Verification

- `mise run check-terminology`
- `mise run test-tools`
- `mise run check-links`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run check-repository`
- `mise run verify`

## Risk Notes

一括置換は共起語を壊す。「秘密鍵」が「シークレット鍵」に、「実行時に評価する」が「ランタイムに評価する」になる誤りは、置換後の本文を読まないと見つからない。所有単位ごとに置換し、単位ごとに差分を読む。

複数行にまたがる置換では `sd -s` が終了コード 0 のまま何も置換しないことがある。着弾したことを `git diff --stat` で確かめる。

見出しの変更はアンカーを変える。`docs/` の内部リンクと `DOCUMENTATION_GUIDE.md` からの参照が切れるため、`mise run check-links` を各単位の完了時に実行する。

用語検査を素朴な禁止語一覧にすると、共起によって正当な用法まで落ちる。残す共起を理由付きで規則表へ置き、それ以外だけを拒否する。

## Completion

- **Completed At**: 2026-09-16
- **Summary**:
  設計文書の用語が、指す英語の概念へ一対一で戻せるようになった。
  deployment、runtime、platform、secret、capacity、observability、admission control、guidelines、load shedding order、sizing formula、reference workload profile の各概念が、それぞれ「デプロイ／デプロイメント」「ランタイム」「プラットフォーム」「シークレット」「キャパシティ」「オブザーバビリティ」「アドミッションコントロール」「ガイドライン」「ロードシェディング順序」「サイジング計算式」「リファレンスワークロードプロファイル」という一つの表記を持つ。
  「秘密鍵」「秘密情報」「保存容量」「容量超過」「実行時に」のように別概念を指す共起は残っており、規則表がその理由を持つ。
  意味の復元できなかった「訓練」は箇所ごとに確定した。`docs/design/architecture/deployment.md` の「ローカル開発と訓練」は「ローカル開発と復旧ドリル」になり、これは `infra/backup/restore-drill.sh` が `infra/docker/docker-compose.dev.yaml` を使い捨てプロジェクトとして起動している事実に基づく。
  題名が「ガイドライン」になった 2 文書はファイル名も `api-guidelines.md`、`design-guidelines.md` へ改めた。`rules` と `guidelines` の二つの英語が同じ文書を指す状態を残さないためである。
  `mise run spec-diff` は REQ-SIGNINGKEYS-012 を挙げる。EX-SIGNINGKEYS-012-01 の Given 1 行が「配備の `PERSISTENCE`」から「デプロイ先の `PERSISTENCE`」になったためで、id、条件、要求する結果はいずれも同じである。
  再発は `mise run check-terminology` が拒否する。免除は `tools/check/src/terminology.ts` の規則表が持つ理由付きの共起だけで、ファイル単位の免除は無い。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-terminology`（`tools/check/src/repository-checks.acceptance.test.ts` の「採らない表記を含む設計文書を、行と採用語つきで拒否する」「root 直下の文書も対象にする」が同じ境界を仮の作業ツリーで固定する）
  - **Requirement**: N/A: 文書の用語と表記の統一であり、規範のプロダクト要求を持たない
  - **Observed Failure**: 文書を 1 行も編集する前に、62 文書 264 箇所を `file:line:column` つきで拒否して exit 1 になった。内訳は配備 79、容量 35、秘密 17、訓練 9、その他 124 である
  - **Detection Reason**: 検査は語の有無ではなく occurrence の位置を見る。採用語へ置き換えた文書は通り、採らない表記が 1 箇所でも戻れば落ちる。行と列を出すので、指摘された箇所を読まずに直すことができない
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/terminology.test.ts` の「残すと決めた共起は通し、同じ行の別概念は通さない」「採用語が採らない表記を含む場合でも、採用語を落とさない」「許可 literal の外にある同じ語を、位置で区別する」
  - **Requirement**: N/A: 文書の用語と表記の統一であり、規範のプロダクト要求を持たない
  - **Observed Failure**: `verifyTerminology` が存在せず、import が解決しなかった
  - **Detection Reason**: 素朴な禁止語一覧との差がここに出る。「トポロジー」は「トポロジ」を、「秘密鍵」は「秘密」を含むので、許可を語の有無で判定する実装は採用語そのものを落とすか、逆に同じ行の別概念まで通す。occurrence の位置で覆いを判定していることを、この 3 件が固定する
- **Change-Resistance Results**:
  `risk: low` なので契約上は不要だが、検査の中心にある 2 つの判断へ故障を注入した。
  共起の許可を無視させる（`covered` を常に `false` にする）と `terminology.test.ts` が 9 件中 4 件落ちた。
  root 直下の文書集合を走査から外すと `repository-checks.acceptance.test.ts` が 17 件中 1 件落ちた。
  どちらも検出された。TypeScript 側に変異器は無いため、系統的な変異は行っていない。
- **Verification Results**:
  - `mise run check-terminology` - passed（224 文書）
  - `mise run test-tools` - passed（42 ファイル 538 件）
  - `mise run check-links` - passed（832 文書）
  - `mise run check-spec` - passed
  - `mise run check-work-items` - passed
  - `mise run check-repository` - passed
  - `mise run verify` - passed
