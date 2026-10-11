---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-19
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "初回公開前の開発用検査だけを変更し、利用契約、保存データ、API と UI に影響しない。"
  references: []
initial_context:
  specification:
    - docs/development/task-context.md
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/writing-language.md
    - docs/development/README.md
    - docs/development/release.md
    - docs/formats/documentation-guide.md
    - docs/requirements/glossary.md
  typespec: []
  source:
    - tools/README.md
    - tools/check/README.md
    - tools/check/src/registry.ts
    - tools/check/src/agent-guidance.ts
    - tools/check/src/check-agent-guidance.ts
    - tools/check/src/verification-tasks.ts
    - tools/check/src/lib.ts
    - tools/check/src/terminology.ts
    - mise.toml
  tests:
    - tools/check/src/agent-guidance.test.ts
    - tools/check/src/verification-tasks.test.ts
    - tools/check/src/mise-config.test.ts
    - tools/check/src/lib.test.ts
    - tools/check/src/registry.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
  stop_before_reading:
    - backend
    - frontend
spec_impact:
  kind: none
  reason: "リポジトリ検査とその単体テストの取捨だけを扱い、プロダクトの振る舞いも規範要素も変えない。"
---

# リポジトリ検査とそのテストを基準で棚卸しし、理由を言えないものを落とす

## Motivation

2026-10-10 の簡素化調査では、エージェント手順の検査が `actually failed` などの文字列の存在と RED、GREEN、refactor の出現順を要求することを確認した。
意味を維持した文章の改善にも検査の更新が伴うため、文言そのものを固定する検査を具体的な削除候補として扱う。
現在は未リリースであり、当面も未リリースを継続する。
公開後の互換性整備より、現在の検査の保守と誤検出の費用の削減を優先する。
以下の件数と行数は起票時の観測であり、着手時に現在の registry と実装から再計測する。

`tools/check` には 18 の検査があり、実装 7,133 行、テスト 7,593 行からなる。
18 件すべてを走らせて 2.9 秒なので、費用は実行時間ではなく保守である。
検査もテストも、足した時点の理由は各ファイルの冒頭に書かれているが、足りているかを後から見直す機会が無かった。

量の偏りが、見直しをしていないことを示している。

| 対象 | 実装 | テスト | 比 |
| --- | --- | --- | --- |
| `lib.ts` | 121 行 | 674 行 | 5.6 |
| `normative-coverage.ts` | 283 行 | 478 行 | 1.7 |
| `security-controls.ts` | 394 行 | 505 行 | 1.3 |
| `contract-drift.ts` | 556 行 | 670 行 | 1.2 |

検査の対象数にも偏りがある。
`check-command-map` はワークフロー 1 ファイル、`check-agent-guidance` はスキル 3 ファイル、`check-vulnerability-suppressions` は抑止 1 件だけを読む。
これらは、規則の正本がリポジトリのどこにあり、検査以外の何が同じ違反を止めるのかを、いま一度確かめる対象である。

`mise.toml` は 1,032 行、130 タスクで、うち 17 は `bun run check/src/runner.ts <name>` を 1 行呼ぶだけの別名である。
`check` と `check-spec` はゲート集合を別々に列挙しており、`check-generated-contract`、`check-rendered-docs`、`documents` が両方に現れる。

ルート直下には生成物が 2 件追跡されている。
`CONFIGURATION.md`（160 行）と `ROUTE_PRIORITY.md`（310 行）である。
`docs/formats/documentation-guide.md` は生成物を追跡しない場所へ置くと定め、`tools/check` の用語検査はこの 2 件を明示的に読まない対象として名指ししている。
規約と実態のどちらかを動かす判断がされていない。

## Scope

- 18 件の検査それぞれを判定基準にかけ、判定と理由を表として `tools/check/README.md` に残す。
- 基準を満たさない検査を落とす。落とした検査の `mise` タスク、レジストリ登録、実装、テスト、文書からの参照を同時に落とす。
- エージェント手順の特定の文言と出現順を固定する検査を、必要な参照、実在するタスク、構造化した契約の検査へ縮小する。
- `tools/check/src` の単体テストをテストの基準にかけ、基準を満たさないものを落とす。
- `check` と `check-spec` のゲート集合の二重列挙を解消する。
- ルート直下の生成物 2 件について、規約と実態のどちらを動かすかを決め、決めた側を直す。

## Out of Scope

- `contract-drift` と `status-drift` の未解決率の縮小。342 操作のうち読み切れるのは 184 件と 83 件だが、これは Go を正規表現で読む手法の限界であり、縮小は別の設計問題である。本項目は両者を残すか落とすかだけを判定する。
- 検査ごとの `mise` タスク 17 件の廃止。判断済みで、残す。`mise tasks` の一覧は短くなるが、文書とスキルが個別の検査を名指しできる入口を失う方が損である。列挙が長いことは、検査が多いことの表示であって原因ではない。
- `backend/` と `frontend/` のテスト。本項目が扱うのはリポジトリ検査とその単体テストだけである。
- 規約文書の重複解消。[[wi-629-one-source-for-the-document-layout-and-format-rules]] が扱う。
- 証拠の適用条件の変更は[証拠の縮小](wi-95150-scale-verification-evidence-to-change-impact.md)、日常規約の整理は[規約の整理](../active/wi-12973-consolidate-daily-development-rules-and-separate-explanations.md)が扱う。
- API 互換性と告知の公開前の適用条件は[未リリース期間のゲート](wi-38041-suspend-release-documentation-and-compatibility-gates-before-publication.md)が扱う。

## Design

### 着手時の判断と検証境界

現行 registry は 25 件である。
18 件という起票時の観測から増えた検査も含め、現行の全件とテストファイルごとに判断する。
製品の仕様、モジュール境界、公開パッケージは変更しない。
検査の一次情報源は具体的な文書節、TypeSpec、スキーマまたは構成へ対応づけ、一般的なレビューを代替の検出器とは数えない。
検査全体の削除候補がなければ、T002 と T003 はその理由を記録して完了する。

`verifyAgentGuidance(documents: GuidanceDocument[], declaredTasks: ReadonlySet<string>): AgentGuidanceFinding[]` は文書参照、コマンド、構造化フィールドだけを検査する。
文章の意味を保つ言い換え、翻訳、説明順の変更では失敗させない。
`risk-based-v4` と `fault_model` は記録契約の識別子として残す。
文書読み取りは `checkAgentGuidance(snapshot)` が持ち、判定は入力値だけで行う。

仕様検査の TypeScript 集合は registry の `spec` group に一度だけ宣言する。
生成契約と生成サイトの共通ゲートを `check-spec-inputs` にまとめ、`check` と `check-spec` は同じ入口を参照する。
`check-repository` の全件実行と各個別タスクの入口は維持する。
生成物 2 件の追跡は維持し、文書ガイドの例外を明記する。

規範 id は N/A（製品要件を変更しない開発ツールの変更）とする。
受け入れ RED の代替検査は `verification-tasks.test.ts` の実際の `mise.toml` と registry の集合比較、単体 RED は `agent-guidance.test.ts` の言い換え受理と参照欠落拒否とする。
いずれも `mise run test-tools-file -- <file>` で確認する。
削除するテストは候補を一時的に外し、誤実装で残すテストが失敗することを確認してから落とす。
変更耐性は共通ゲートの配線切断と文書参照検査の無効化を注入し、同じ局所テストで確認する。
Go は変更しないので Go の特性化テストと変異器は適用しない。

検査の判定基準を三つ置き、三つすべてを満たすものだけを残す。

| 基準 | 内容 | 満たさないときに起きること |
| --- | --- | --- |
| 正本の外部性 | 規則の正本が検査の外（文書、TypeSpec、スキーマ、ワークフロー定義）にある | 検査自身が規則の正本になり、検査を直せば違反が消える |
| 違反経路の実在 | その規則に反する変更が、通常の作業の中で起こりうる | 起きない違反を守るために保守費用を払い続ける |
| 他の関門の不在 | 同じ違反を型検査、コンパイル、テスト、レビューが止めない | 同じ誤りを二重に止め、片方が壊れても誰も気付かない |

「過去に誤りを止めた実績」は基準に入れない。
通常の文章レビューができることだけで、同じ誤りを機械的に止められるとは判断しない。
代替する関門の対象と実行条件、検出する具体的な誤りを示す。
特定の文字列の有無が手順の意味を保証しない場合は、その文字列検査を残す理由と、参照や実行契約へ縮小したときに残る保証を比較する。
実績を基準にすると、足したばかりの検査と、守っている規則がまだ破られていない検査を落とすことになる。
`contract-drift` と `status-drift` の冒頭が記録する wi-381、wi-382、wi-386 の実績は、基準を満たすことの説明であって基準そのものではない。

テストの判定基準は一つである。
そのテストを消したときに素通りする誤実装を、一つ具体的に名指しできるかどうかで判定する。
名指しできないテストを落とす。
行の被覆率も、実装とテストの行数比も基準にしない。
比が大きいことは疑う手がかりであって、落とす理由にはならない。
`lib.ts` の 5.6 倍は、Ajv のスキーマ検証を通した frontmatter の受理と拒否を一件ずつ並べた結果かもしれず、その場合は残る。

生成物 2 件は、規約側を動かす。
`CONFIGURATION.md` と `ROUTE_PRIORITY.md` は、リポジトリを開いた運用者が設定項目と経路の優先順位をその場で読むための文書である。
追跡をやめると、読むために生成を実行させることになる。
`docs/reference/` へ移す案も採らない。59 ファイルからの参照を書き換える費用に対して、得るものがルート直下のファイル数だけである。
`docs/formats/documentation-guide.md` の「生成物は追跡しない場所へ置く」に、利用者が直接読む生成物は追跡する、という例外とその理由を書く。

## Plan

1. 18 件の検査を判定し、判定表を書く。落とす候補が出た検査は、その検査が止めていた違反を故意に作り、他の関門が止めないことを確かめてから判定を確定する。
2. 落とすと決めた検査を、レジストリ、実装、テスト、`mise` タスク、文書からの参照まで一度に落とす。
3. `tools/check/src` のテストファイルを 1 本ずつ基準にかける。落とす候補は、先に該当箇所へ誤実装を入れて他のテストが落ちることを確かめてから落とす。
4. `check` と `check-spec` の列挙を、片方がもう片方の部分集合であることが読める形にする。`verification-tasks.test.ts` が既にゲート集合を読んでいるので、そこへ関係を固定する。
5. `docs/formats/documentation-guide.md` の生成物の節へ例外を書く。

## Tasks

- [x] T001 [Tooling] 18 件の検査を三つの基準で判定し、判定表を `tools/check/README.md` へ書く。
  - 現行の 25 件を対象に、正本、違反経路、代替できない関門を一行ずつ記録した。
    着手時は実装 11,491 行、テスト 12,706 行（50 ファイル）であり、行数を採否の基準にはしなかった。
    文書の検査は `mise run check-links` と `mise run check-work-items` を選ぶ。
- [x] T002 [Acceptance] 落とす候補の検査について、止めていた違反を故意に作り、他の関門が止めないことを確かめる。
  - N/A: 検査全体の削除候補はない。
    別の関門が同じ範囲の故障を止めると主張して削除した検査はない。
    `agent-guidance` は参照と契約の欠落を守る独立した役割があるため、全体を残して文言検査だけを縮小した。
- [x] T003 [Tooling] 判定で落ちた検査を、登録、実装、テスト、タスク、参照まで落とす。
  - N/A: 判定で落ちた登録検査はない。
    25 件と個別タスクを維持し、不要な部分は T004 と T007 で整理した。
- [x] T004 [Tooling] `tools/check/src` のテストを基準で判定し、落ちたものを落とす。落とす前に誤実装で他のテストの検出を確かめる。
  - `lib.test.ts` の schema キー一覧の固定を一時的に除外し、`work-item` の登録先を変える故障で残る 25 テストが失敗した。
    `mise-config.test.ts` の集合の長さだけの表明を一時的に除外し、`verify` を `check` だけにする故障を集合一致と UI ゲートのテスト 2 件が検出した。
    両方とも故障を戻してから候補を削除した。
    実行はそれぞれ `mise run test-tools-file -- check/src/lib.test.ts` と `mise run test-tools-file -- check/src/mise-config.test.ts`、規範 id は N/A（開発ツールの重複表明）とする。
- [x] T005 [Tooling] `check` と `check-spec` の二重列挙を解消し、`verification-tasks.test.ts` で関係を固定する。
  - `mise run test-tools-file -- check/src/verification-tasks.test.ts` で共通入口 `check-spec-inputs` の欠落による RED を観測した。
    共通入口と registry の `spec` group を接続した後に同じ検査が成功した。
    規範 id は N/A（開発用ゲート集合の構成）とする。
- [x] T006 [Docs] `docs/formats/documentation-guide.md` へ、利用者が直接読む生成物を追跡する例外とその理由を書く。
  - 運用者が生成コマンドを実行せずに読む 2 文書の追跡を維持し、再生成とドリフト検査を伴う例外を生成物の節へ追記した。
    `mise run check-links` で参照を検査する。
- [x] T007 [Guidance] 意味を保つ言い換えで失敗する文言検査を整理し、必要な参照と実行契約の欠落の検出を維持する。
  - `mise run test-tools-file -- check/src/agent-guidance.test.ts` の言い換え受理テストで、編集前の検査が 9 件の文言欠落を誤検出する RED を観測した。
    固有文言と出現順を落とし、未定義タスクの検査を実在する `mise.toml` に接続して GREEN にした。
    文書参照検査を無効化する故障で、参照、オプション、構造化フィールドとファイルの欠落を検査する 4 テストが失敗した。
    規範 id は N/A（エージェント手順の参照と実行契約）とする。
- [x] T008 [Verify] 変更を検証する。
  - 七つの視点で差分をレビューし、ファイルを固定した状態で `mise run verify` が成功した。
    Go 6,133 件、ツール 1,018 件を含む集約ゲートを通過した。
    初回に検出された用語を修正し、再実行中の記録追記による入力変化の拒否も、入力を固定した再実行で解消した。
    完了記録の移動後に `mise run check-work-items` と `mise run check-links` が成功した。

## Verification

- `mise run test-tools`
- `mise run check`
- `mise run verify`
- エージェント手順の意味を保った言い換えを受理し、必要な文書参照や実行タスクの欠落を引き続き検出する。

## Risk Notes

検査を落とすと、その検査が止めていた違反が通るようになる。
落とす判断の根拠は「他の関門が止める」なので、根拠が正しいことを T002 で観測してから落とす。
観測せずに落とした検査は無い、という状態を完了時に記録する。

テストを落とす作業は、検出能力を落としながら緑のままで進む。
落とす前に該当箇所へ誤実装を入れ、残すテストが落ちることを確かめる手順を T004 に置く。
確かめずに落としたテストが一本も無いことを完了時に記録する。

判定表そのものが陳腐化する恐れがある。
表は判定の時点の根拠であり、検査を足すときに同じ三基準で書くことを `tools/check/README.md` に規則として書く。

## Completion

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は main に対する規範仕様の変更なしと判定した。
  現行の 25 検査と着手時の 50 テストファイルを棚卸しし、検査全体は維持した。
  手順の特定文言と出現順の検査を参照、実在するタスクと契約フィールドの検査へ縮小し、意味を保つ言い換えを受理するようにした。
  削除候補を外した状態でも残るテストが同じ故障を検出することを観測し、重複する表明を 2 件削除した。
  仕様検査の集合と生成入力ゲートを共通化し、運用者が直接読む生成文書 2 件を追跡する例外を文書ガイドへ明記した。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/verification-tasks.test.ts` の仕様検査と全件検査の集合比較。
  - **要件**: N/A: 製品要件を変更しない開発用ゲートの構成であり、実際の `mise.toml` と registry を読む代替検査を用いた。
  - **観測した失敗**: 共通入口 `check-spec-inputs` が存在せず、両方の検査から到達するという表明が失敗した。
  - **検出できる理由**: 実際のタスク呼び出しと依存を展開し、共通ゲートの到達と registry の部分集合関係を比較するため、片方の配線欠落を区別できる。
- **単体 RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/agent-guidance.test.ts` の意味を保つ言い換えの受理。
  - **要件**: N/A: エージェント手順の参照と実行契約の検査であり、製品の規範要素を追加しない。
  - **観測した失敗**: 必須参照と契約を残した言い換えに対し、変更前の検査が 9 件の文言欠落を報告した。
  - **検出できる理由**: 文言と説明順を変えても同じ参照と契約を持つ入力を受理し、それらが欠ける入力を拒否する表明で、過剰な固定と検査不足の両方を区別する。
- **変更耐性の結果**:
  文書参照の検査を無効化する故障は 4 テスト、共通ゲートを片方から外す故障は集合比較の 1 テストが検出した。
  キー一覧の表明を除外して schema の登録先を変える故障は 25 テスト、集合の長さの表明を除外して `verify` を `check` だけにする故障は 2 テストが検出した。
  すべての故障を戻し、局所テストの成功を確認してから表明を削除した。
  テストファイルを全体として削除したものはなく、観測していない代替関門を根拠に削除した登録検査もない。
  TypeScript の配線と判定の代表的な故障を手で注入した結果であり、体系的な全変異の検査とは扱わない。
- **検証結果**:
  - `mise run verify` — 入力を固定した再実行で成功。Go 6,133 件、ツール 1,018 件が成功した。
  - 完了記録を `work-items/done/` へ移動後、`mise run check-work-items` と `mise run check-links` — 成功。
  - `mise run spec-diff` — 規範仕様の変更なし。
  - `mise run test-tools-file -- check/src/agent-guidance.test.ts` — 成功。
  - `mise run test-tools-file -- check/src/check-agent-guidance.test.ts` — 同じ workspace のタスク宣言への接続を確認して成功。
  - `mise run test-tools-file -- check/src/verification-tasks.test.ts`、`check/src/lib.test.ts`、`check/src/mise-config.test.ts` — 復元後に成功。
  - `mise run test-tools-file -- render-docs/src/documentation-quality.test.ts` — 全体検証で検出した用語を修正し、成功。
  - `mise run lint-tools`、`mise run typecheck-tools`、`mise run check-agent-guidance` — 成功。
  - コーディングスタイルの七つの視点で差分をレビューし、読み取りの作用と入力値だけの判定、共通パーサーの再利用、別 workspace での配線検査を確認した。
  - ブラウザーへ到達する変更はないため、UI E2E は N/A。
