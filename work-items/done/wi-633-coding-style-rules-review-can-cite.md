---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-20
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの内部規約だけを変更し、利用者が使える機能、互換性、移行手順は変えない。"
  references: []
initial_context:
  specification:
    - AGENTS.md
    - DOCUMENTATION_GUIDE.md
    - WORK_ITEM_FORMAT.md
    - docs/development/coding-style.md
    - docs/development/specification-first-workflow.md
    - docs/development/testing.md
    - docs/design/application/design-guidelines.md
    - .claude/rules/japanese-writing.md
  typespec: []
  source:
    - .golangci.yml
    - frontend/biome.json
    - backend/shared/security/actiontoken/action_token.go
    - frontend/src/lib/i18n/resolveLocale.ts
    - frontend/src/features/admin-groups/dynamicRuleCel.ts
    - frontend/src/lib/usePaginatedList.ts
    - .agents/skills/implement-work-item/SKILL.md
    - tools/check/src/agent-guidance.ts
  tests:
    - backend/shared/security/actiontoken/action_token_test.go
    - frontend/src/lib/i18n/resolveLocale.test.ts
    - frontend/src/features/admin-groups/dynamicRuleCel.test.ts
    - tools/check/src/agent-guidance.test.ts
  stop_before_reading:
    - spec/generated
    - docs/domain
    - backend/cmd
    - frontend/src/features/account
spec_impact:
  kind: none
  reason: "コードの形を選ぶための視点を一つの文書へ集めるだけで、外部から観測できる振る舞いも TypeSpec の契約も規範要素も変えない。"
---

# コードを書くときとレビューするときに使うコーディングスタイルを `docs/development/coding-style.md` に置く

## 動機

`DOCUMENTATION_GUIDE.md` §8.3 は `docs/development/coding-style.md` を置くと定める。
このファイルは無い。
`docs/development/` にあるのは README、仕様先行の開発ワークフロー、ローカル開発、継続的インテグレーション、テスト方針、リリース、開発プロセスの計測の 7 件で、コードの書き方を扱うものは一つも無い。

共通のコーディングスタイルが無いまま書かれたコードは、backend の実装だけで 1627 ファイル、12 万行ある。
frontend の TypeScript は 465 ファイルである。
この量の大半をエージェントが書く。
人は前後のファイルを読んで書き方を揃えるが、エージェントが読むのは与えられた文書と直近の文脈だけなので、揃える相手が依頼ごとに変わる。

コードの書き方に関わる指針は、現在は四つの層に分かれていて、どれも別の問いに答えている。

| 層 | 一次情報源 | 決めていること |
| --- | --- | --- |
| 設計 | `docs/design/application/design-guidelines.md` | モジュールの深さ、Seam とポート、Aggregate 境界、型の所有、作用の境界、エラーと拒否 |
| 工程 | `docs/development/specification-first-workflow.md` | リファクタリングを行う時点と、それが独自の証拠を持たない理由 |
| 言語 | `.claude/rules/japanese-writing.md` | コメントと説明文をどの言語で書くか |
| 機械 | `.golangci.yml`、`frontend/biome.json` | 整形と静的検査が拒否する形 |

空いているのは設計の層と機械の層の間である。
命名、コメントに何を書くか、関数と制御構造の形、重複を許す条件、整頓と振る舞いの変更の分離、テストコードの読みやすさは、どの層も持たない。
これらは lint が判定できず、モジュール境界の設計判断でもないが、コードを書くときとレビューするときの両方で判断が必要になる。

この空白はレビューの側にも現れる。
`code-review` スキルは、リポジトリが文書化した規則を探し、見つからない場合は Fowler の code smell の一覧を既定の基準として使う。
このリポジトリは今その既定の基準で読まれているので、レビューのたびに一般論から議論が始まる。
100 行を超える関数が 20 件あり、その上位は `backend/shared/http/server_http/routes.go` の経路登録と `backend/cmd/` 配下の起動処理である。
長さだけを基準にすればこれらは指摘になり、組み立てを行う関数は長くてよいと考えれば指摘にならない。
どちらを採るかをリポジトリが答えていないため、同じ判断が変更のたびに繰り返される。

## 対象範囲

- `docs/development/coding-style.md` を新設し、人がコードの形を選ぶための視点と理由をまとめる。
- 扱う主題は、命名、コメントの密度と内容、関数と制御構造の形、重複と抽象化の判断、整頓と振る舞いの変更の分離、テストコードの書き方、依存を足すときの基準とする。
- 指針ごとに理由を書き、コードを書くときとレビューするときの両方で七つの視点を使うことを明示する。
- 公開された言語別スタイルガイドを表にまとめ、言語の慣用を調べる入口にする。
- 指針の根拠として参照する書籍を明示する。Kent Beck『Tidy First?』、Martin Fowler『リファクタリング』、Dustin Boswell と Trevor Foucher『リーダブルコード』、Steve McConnell『Code Complete』、Robert C. Martin『Clean Code』、John Ousterhout『A Philosophy of Software Design』、Michael Feathers『レガシーコード改善ガイド』を対象とする。
- 指針を理解しやすさ（Comprehensibility）、局所性（Locality）、凝集性（Cohesion）、カプセル化（Encapsulation）、テスト容易性（Testability）、変更容易性（Changeability）、単純さ（Simplicity）の七つの視点へ整理する。
- 汎用的な文書として書き、リポジトリ固有の名前、実例、検査設定は本文に持ち込まない。
- 入口を配線する。`docs/development/README.md` の表へ 1 行、`AGENTS.md` へ 1 行、`CONTRIBUTING.md` のレビュー観点へ参照を足す。
- `AGENTS.md` を作業別の参照先を示すルーターにし、コード編集とレビューの開始前に本文を読む時点を明示する。
- `implement-work-item` から本文を参照し、構造変更と振る舞いの変更を別コミットにできる手順へ直す。
- `check-agent-guidance` で `AGENTS.md` と `implement-work-item` から本文への導線を検査する。

## 対象外

- lint と整形器が判定する形式の文書化。機械検査の設定を一次情報源とする。
- 新しい lint 規則の追加と `.golangci.yml`、`frontend/biome.json` の変更。文書が固まってから、どの規則を機械へ移せるかを別に判断する。
- `docs/design/application/design-guidelines.md` が持つ設計レベルの規則の移設と改訂。
- テスト水準、実行境界、テストダブルの規則。`docs/development/testing.md` が持つ。本項目が扱うのはテストコードの読みやすさだけである。
- Seam の定義と、特性化テストの適用条件。前者は `design-guidelines.md`、後者は `testing.md` が既に持つ。『レガシーコード改善ガイド』から採るのは、テストを書ける形で新しいコードを書くための基準だけとする。
- 言語ごとのファイル分割。`DOCUMENTATION_GUIDE.md` §8.3 の通り、実際に衝突するまで一つのファイルで書く。
- 既存コードの一括是正。決めた規則は以降の変更へ適用し、触らない場所は書かれた時点の形のまま残す。
- コーディングスタイルに対応する機械検査の新設。人が判断する視点を先に文書化する。

## 設計

### 文書の範囲

新しい文書は、コードを書くときとレビューするときに使う汎用的な判断の視点を持つ。
命名、コメント、関数と制御構造、局所的な抽象化、テストコード、依存関係の追加を扱い、リポジトリ固有の名前、実例、検査設定は持たない。

モジュールやサービスの境界、外部 API の契約、テスト水準、テストダブル、整形、静的検査は対象外とする。
これらは利用するプロジェクトの設計文書、テスト方針、formatter、linter が決める。

### 判断に使う七つの視点

見出しは日本語を主とし、由来を追えるように英語を括弧で添える。

| 視点 | 判断する問い |
| --- | --- |
| 理解しやすさ（Comprehensibility） | 名前、コメント、本体から意図を読めるか |
| 局所性（Locality） | 一つの変更に必要な情報が近くにあるか |
| 凝集性（Cohesion） | 一つの単位が一つの理由で変わるか |
| カプセル化（Encapsulation） | 呼び出し側が知る必要のない事実が漏れていないか |
| テスト容易性（Testability） | テストのために本体を書き換えず、判断を再現できるか |
| 変更容易性（Changeability） | 同じ判断を一か所の変更で更新できるか |
| 単純さ（Simplicity） | 現在の要求に使わない仕組みが残っていないか |

コードを書くときは、この七つの視点から実装の形を選び、実装後にも同じ視点で読み直す。
レビューするときは、好みではなく、どの視点で何が読みにくいか、どの変更が難しくなるかを理由とともに示す。

### 言語別スタイルガイド

Go と TypeScript に限定せず、公開された言語別スタイルガイドを表にまとめる。
将来使う言語が増えても、この文書の判断の視点は共通とし、命名、構文、整形の慣用は言語別スタイルガイドを参照する。

### 指針の書式

各項目は指針と理由の組で書く。
強制方法とリポジトリ固有の実例は、汎用的な本文には含めない。

### エージェント向けの導線

`AGENTS.md` は作業の種類から一次情報へ到達するための短いルーターとする。
コーディングスタイルの内容は複製せず、コード編集とレビューを始める前に本文を読み、七つの視点を判断に使うことだけを指示する。

`implement-work-item` は、最初のコード編集前と完了前の二か所で本文を参照する。
一つの work item は一つの意味上の変更を扱うが、構造変更と振る舞いの変更は別コミットにできるようにする。

`check-agent-guidance` は、`AGENTS.md` と `implement-work-item` の両方に `docs/development/coding-style.md` への導線があることを検査する。

### 参考文献の扱い

七冊は判断の由来として名指しし、本文を写さない。
関数の長さ、コメント、重複について主張が異なる箇所では、この文書が採る判断を明示する。

| 論点 | この文書が採る判断 |
| --- | --- |
| 関数の長さ | 行数の上限を置かず、抽象水準と、名前が本体を要約できるかで判断する |
| コメント | コードが答えられる問いは書かず、選択理由と呼び出し側が知る必要のある前提を書く |
| 重複 | 字面の類似ではなく、同じ判断または制約を表し、同じ理由で変わるかを確認してからまとめる |

### 採らない案

| 案 | 退ける理由 |
| --- | --- |
| ルート直下に `CODING_STANDARDS.md` を置く | `DOCUMENTATION_GUIDE.md` §8.3 が配置を `docs/development/` と定めている |
| ADR として記録する | 変更固有の判断は work item が持つ |
| `.agents/skills/` に skill として置く | skill は作業手順であり、コードの性質を示す文書の置き場所ではない |
| 七冊の要約を文書へ収める | 原典より粗い要約が、原典の更新から切り離されて残る |

## 計画

作業は次の順に進める。

1. 四つの既存文書との境界を確定し、重なる記述が無いことを確かめる。
2. 七つの視点ごとに文献を読み、コードを書くときとレビューするときに使う指針と理由へ落とす。
3. リポジトリ固有の名前や実例を含めず、汎用的な本文を書く。
4. 言語別スタイルガイドの表と、文書への入口を追加する。
5. エージェント向けの入口と実装手順を本文へ接続し、自動検査を追加する。
6. 文書体系とコミット手順の矛盾を解消する。
7. 検証する。

作るものを変えうる問いは着手前に解決した。
コメントの言語は `.claude/rules/japanese-writing.md` が持ち続け、新しい文書は密度と内容だけを持つ。
テストの水準は `docs/development/testing.md` が持ち続け、新しい文書はテストコードの読みやすさだけを持つ。
指針の粒度は、七つの視点から具体的な判断を行えるかどうかで決める。

## タスク

- [x] T001 [Docs] 文書の範囲を確定し、既存の四つの文書との重なりを解消する。
- [x] T002 [Docs] 七つの視点それぞれについて、該当する文献から判断原則を起こす。
- [x] T003 [Docs] Acceptance RED として `rg -q 'coding-style\.md' docs/development/README.md AGENTS.md CONTRIBUTING.md` が失敗することを確認し、三つの入口を追加する。
- [x] T004 [Docs] Unit RED は文書変更に単体境界が無いため適用せず、代替検査 `test -f docs/development/coding-style.md` の失敗を確認してから本文を書く。
- [x] T005 [Verify] 変更を検証する。
- [x] T006 [Docs] `AGENTS.md`、仕様先行ワークフロー、`implement-work-item`、文書ガイドをコーディングスタイルへ接続し、コミット方針を整合させる。
- [x] T007 [Tooling] `check-agent-guidance` が `AGENTS.md` と `implement-work-item` の導線欠落を検出する Unit RED を確認して実装する。
- [x] T008 [Verify] 拡張した変更を検証する。

## 検証

- `mise run check-links`
- `mise run check-terminology`
- `mise run check-rendered-docs`
- `mise run test-tools-file -- check/src/agent-guidance.test.ts`
- `mise run check-agent-guidance`
- `mise run verify`

## リスク

指針が抽象的すぎると、コードを書く人とレビューする人が異なる判断を行う。
各項目を指針と理由の組にし、七つの視点から指摘理由を説明できる形にする。

言語をまたぐ共通の視点が、各言語の慣用を上書きする危険がある。
命名、構文、整形は言語別スタイルガイドとプロジェクトの機械検査を優先する。

エージェント向けの導線が文書名だけを検査すると、別の文脈にリンクが残っていてもコード編集前には読まれない可能性がある。
自動検査は、文書のパスに加えてコード編集とレビューの開始条件も確認する。

## 完了

- **Completed At**: 2026-09-20
- **Summary**:
  規範仕様の差分はない。
  コードを書くときとレビューするときに使う汎用的なコーディングスタイルを七つの視点で定め、言語別スタイルガイドをまとめた。
  `AGENTS.md` を作業別の参照先を示すルーターにし、実装 skill に編集前の読込と完了前のレビューを追加した。
  構造変更と振る舞いの変更を別コミットにできるよう手順を整合させ、導線の欠落を `check-agent-guidance` で検出するようにした。
- **Acceptance RED Evidence**:
  - **Test**: `rg -q 'coding-style\.md' docs/development/README.md AGENTS.md CONTRIBUTING.md`、`rg -q 'docs/development/coding-style\.md' .agents/skills/implement-work-item/SKILL.md`
  - **Requirement**: N/A: 利用者から観測できるプロダクトの振る舞いを変えない開発文書とリポジトリツールの変更である。
  - **Observed Failure**: 最初の検査は文書作成前に終了コード 1 となり、三つの入口から参照できなかった。二つ目の検査はエージェント導線の拡張前に終了コード 1 となり、実装 skill がコーディングスタイルを読まなかった。
  - **Detection Reason**: 文書の発見可能性と、実装手順からの読込を別々に検出するため、入口だけまたは本文だけを追加した不完全な変更では両方に成功しない。
- **Unit RED Evidence**:
  - **Test**: `test -f docs/development/coding-style.md`、`mise run test-tools-file -- check/src/agent-guidance.test.ts`
  - **Requirement**: N/A: 文書自体に実行可能な単体境界はなく、エージェント導線の検査ロジックだけが単体境界を持つ。
  - **Observed Failure**: 文書作成前の存在検査は終了コード 1 となった。検査ロジックの実装前は、追加した 7 件中、`AGENTS.md` と `implement-work-item` の欠落を検出する 2 件が失敗し、残る 5 件が成功した。
  - **Detection Reason**: 二つの失敗は、文書パスが無い場合だけでなく、コード編集前に読むという開始条件が無い場合も検出する。
- **Change-Resistance Results**:
  N/A: 文字列マーカーの欠落を直接与える二つの否定側テストが変更の目的を検出しており、実行時のプロダクトロジックを変更していない。
- **Verification Results**:
  - `rg -q 'coding-style\.md' docs/development/README.md AGENTS.md CONTRIBUTING.md` - passed
  - `rg -q 'docs/development/coding-style\.md' .agents/skills/implement-work-item/SKILL.md` - passed
  - `test -f docs/development/coding-style.md` - passed
  - `mise run test-tools-file -- check/src/agent-guidance.test.ts` - 7 passed, 0 failed
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-agent-guidance` - passed, 4 guidance files checked
  - `mise run check-links` - passed
  - `mise run check-terminology` - passed
  - `mise run check-rendered-docs` - passed
  - `mise run spec-diff` - no normative specification change against main
  - `mise run verify` - passed
