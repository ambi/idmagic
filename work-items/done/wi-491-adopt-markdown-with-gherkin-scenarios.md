---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-05
priority: p1
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "規範シナリオの正本形式、検査、生成表示、テストへの追跡粒度を変更するが、既存の REQ-* とそれが定める製品の振る舞い、TypeSpec、公開契約は変更しない。移行中に見つかった仕様の不足や規則の分割は、規範参照を持つ別の work item で扱う。" }
evidence_policy: risk-based-v3
documentation_impact: { level: none, reason: "製品の利用者向け文書と運用手順は変わらず、変更対象は仕様方法論、正本、検査、生成表示に限られる。", references: [] }
initial_context:
  source:
    - work-items/wi-491-adopt-markdown-with-gherkin-scenarios.md
    - SPECIFICATION_FORMAT.md
    - DOCUMENTATION_GUIDE.md
    - docs/development/specification-first-workflow.md
    - docs/README.md
    - docs/structure.md
    - tools/package.json
    - tools/check/src/specification-doc.ts
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/check-specifications.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/src/normative-coverage.test.ts
    - tools/check/src/security-controls.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/canonical-document-set.ts
    - tools/check/src/agent-guidance.ts
    - tools/check/src/work-item-references.ts
    - tools/render-spec-docs/src/render.ts
    - tools/workspace/src/workspace.ts
  stop_before_reading:
    - backend/
    - frontend/
    - spec/
---

# 規範シナリオを Markdown with Gherkin に置き換え、具体例ごとにテストへ追跡する

## Motivation

現在の `scenarios.md` は Gherkin の `GIVEN`、`WHEN`、`THEN` と Alistair Cockburn の Extensions に似た `ALT` を組み合わせた独自記法を使う。`ALT` は直前の `WHEN` または `THEN` の子項目となり、条件と結果を `→` で区切る。この設計は分岐を割り込む位置の近くに置けるが、分岐がどの通常ステップを置き換え、どこで通常経路へ戻り、どこで終了するかを表さない。

現状を測ると、ルートと 21 Context に規則が 308 件（生きた規則 306 件、退役済み 2 件）、`ALT` が 405 件ある。97 件の `ALT` は `→` を 2 個以上、19 件は 3 個以上含む。`REQ-OAUTH2-005` の認可コード再使用は、代替条件、2 回の操作、2 回の応答、トークンファミリーの失効、2 種類のイベント発行を 1 行に並べる。これは例外的な長文ではなく、分岐を 1 行に閉じる文法が複数操作と複数結果を表せないために生じている。

テストへの追跡もシナリオ単位で止まっている。現在の規範被覆は、テストファイルのどこかに `REQ-*` が 1 回現れれば、その見出しに属する正常経路とすべての `ALT` を被覆済みとみなす。正常経路だけを確認したテストが、同じ見出しに含まれる拒否、代替成功、副作用の不在まで確認したかは区別できない。`ALT` 自身には安定した住所がないため、テストが特定の分岐を名指すこともできない。

`DOCUMENTATION_GUIDE.md` は、同じ構造で入力だけが違う族を表で持ち、表の各行に対応するテストを要求すると定めている。しかし 22 個の旧 `scenarios.md` に表は 1 件もなく、`SPECIFICATION_FORMAT.md` に表の文法、行の識別子、被覆規則がない。規則と具体例と網羅すべき条件の組み合わせが、現在の文書では区別されていない。

[Markdown with Gherkin](https://github.com/cucumber/gherkin/blob/main/MARKDOWN_WITH_GHERKIN.md) は、公式 Gherkin パーサーが扱う GitHub Flavored Markdown の方言である。見出しを `Feature`、`Rule`、`Example`、`Scenario Outline` に使い、箇条書きを `Given`、`When`、`Then`、`And`、`But` に使うため、現在の正本が持つ Markdown と生成文書を維持できる。Gherkin の `Rule` は一つの規則に属する複数の具体例をまとめ、`Scenario Outline` の `Examples` 行はそれぞれ独立した実行例へ展開される。この構造へ置き換えれば、独自の分岐文法を保守せず、規則、例、テストの対応を分離できる。

## Scope

- ルートと全 Context の規範シナリオ正本を `scenarios.md` から `scenarios.feature.md` へ置き換え、公式 Markdown with Gherkin パーサーが受理する文法にする。
- 各ファイルを一つの `Feature` とし、既存の生きた `REQ-*` 見出しを同じ識別子を持つ `Rule` へ移す。移行だけを理由に `REQ-*` を改名、削除、分割しない。
- 正常経路、代替成功、拒否を独立した `Example` にし、`ALT` と `→` を廃止する。各 `Example` に `EX-<CONTEXT>-<REQ-NNN>-<sequence>` 形式の一意な識別子を付ける。
- 同じステップ構造で条件値だけが異なる族を `Scenario Outline` と `Examples` 表で表す。各行に `example_id` を置き、1 行を一つの具体例として追跡する。
- 複数の独立した条件から結果が決まる規則に限り、`Examples: Decision table (Unique)` を使えるようにする。条件、結果、`any` が表す無関係値、実行可能な組み合わせの扱いを `SPECIFICATION_FORMAT.md` に定める。
- `ACTOR` を Gherkin ステップから外す。行為者は `When` の主語へ書き、複数例に共通する主役を残す必要がある場合は `Rule` の説明に置く。
- 公式 Gherkin パーサーを `tools/package.json` に固定し、構文解析と AST または Pickle への展開を `mise run check-spec` に接続する。Cucumber のステップ定義やテストランナーは導入しない。
- `REQ-*` の被覆を、配下にあるすべての生きた `EX-*` がテストから名指しされていることとして導出する。テストは引き続き Go と TypeScript のネイティブなテストとし、一つのパラメーター化テストが複数の `EX-*` を名指してよい。
- 既存の `REQ-*` 引用を子の `EX-*` へ自動的に配賦しない。移行時にテストを読んで対応を確定できた例だけを名指しし、それ以外は理由付きの例被覆負債へ入れる。
- 生成仕様サイトのシナリオ表示とトレーサビリティを `Feature`、`Rule`、`Example`、Decision Table の階層に対応させ、各例からテストソースと負債状態を確認できるようにする。
- `SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`docs/development/specification-first-workflow.md`、`docs/README.md`、`docs/structure.md`、リポジトリ内の agent guidance と skills を新しい正本名と追跡規則へ同期する。
- `affected_spec`、`initial_context`、Markdown リンク、生成器、検査器、負債報告に残る `scenarios.md` の参照を `scenarios.feature.md` へ移す。

## Out of Scope

- シナリオが定める製品の振る舞い、TypeSpec のモデルと API、認証または認可機構の変更。
- 移行中に見つかった不足した拒否、境界、状態遷移、並行性の規範追加。該当する Context の `REQ-*` または TypeSpec symbol を参照する別の work item で扱う。
- 既存の広い `REQ-*` を複数の規則へ意味分割すること。移行後の新規変更には「一つの `Rule` に一つの規則」を適用するが、既存要素の分割は規範変更として別に扱う。
- Cucumber、Godog、または別の BDD テストランナーによるステップ定義とテスト実行。公式パーサーは正本の構文検査と例の展開にだけ使う。
- 完全な Decision Model and Notation（DMN）、FEEL、複数ヒット方針、規則優先順位、決定サービスの導入。最初の形式は重ならない `Unique` の決定表に限る。
- Decision Table による状態機械、再試行列、時間順序の表現。これらは通常の `Example` または `states.md` が所有する。
- Example Mapping の質問を current-state の正本に保存すること。未解決の質問は実装前に work item で解決し、永続する結論だけを正本へ移す。
- 既存テストを Gherkin ステップから生成すること、または Gherkin からテストコードを生成すること。

## Design

### 正本の構造

`scenarios.feature.md` は Markdown with Gherkin として公式パーサーで解析できる文書にする。キーワードは既存のリポジトリ規則に合わせて英語、説明とステップ本文は日本語とする。

```markdown
# Feature: OAuth2

## Rule: REQ-OAUTH2-005 認可コードは正しい PKCE 検証後に交換できる

### Example: EX-OAUTH2-005-01 正しい verifier で交換する

- Given クライアントに未使用の認可コードが発行されている
- When クライアントが正しい PKCE verifier で認可コードを交換する
- Then access token、ID token、refresh token が返る
- And 認可コードは使用済みになる

### Example: EX-OAUTH2-005-02 誤った verifier では交換できない

- Given クライアントに未使用の認可コードが発行されている
- When クライアントが誤った PKCE verifier で認可コードを交換する
- Then `InvalidGrantError` が返る
- And トークンは発行されない
```

一つの `Example` は分岐を持たない一本の経路とする。エラー応答、発行されないトークン、変更されない状態、発行される監査イベントのように結果が複数ある場合は、同じ例の複数の `Then` または `And` として書く。これにより、結果の列を `→` で直列化せず、テストが確認すべき観測を一段ずつ読める。

`ACTOR` は現在の検査で個数だけが検証され、テスト対応や認可境界へ接続されていない。Gherkin にないステップ種を残す理由がないため、行為者を `When` の主語へ移す。主役の宣言がシナリオ群の理解に必要な場合も、自由記述の `Primary actor` として `Rule` の説明へ置き、実行ステップとして扱わない。

### 規則と例の識別子

`REQ-*` は現在と同じく規範となる振る舞いの識別子であり、`Rule` の名前の先頭に置く。移行時には一つの旧シナリオを一つの `Rule` へ写し、正常経路と各 `ALT` をその子の例へ展開する。旧見出しに異なる規則が混在していると分かっても、本項目では意味を再配分しない。

`EX-*` はテストがどの具体例を実行したかを指す住所である。通常の `Example` は名前の先頭に持ち、`Scenario Outline` は `Examples` 表の `example_id` 列に持つ。識別子はリポジトリ全体で一意とし、同じ親 `REQ-*` の下で連番を割り当てる。表の行順や文章の並べ替えでは変更しない。

被覆は二段階に分ける。`Rule` は少なくとも一つの生きた例を持たなければならず、配下のすべての `EX-*` がテストから名指しされるか、理由付きの負債に載っている場合だけ `REQ-*` を被覆済みとする。テスト中の旧 `REQ-*` 引用は高位の追跡情報として残せるが、特定の `EX-*` の被覆には数えない。

移行によって数百件の `EX-*` が一度に生まれるため、根拠のない自動対応は行わない。各例について既存テストの入力と観測が一致する場合だけ、そのテストへ `EX-*` を追加する。一致を確認できない例は例被覆負債に理由付きで置く。負債は新しい例を受け入れず、テストが識別子を得たら削除を要求する既存のラチェットに従う。

### Decision Table

同じ操作に対し、独立した条件の組み合わせが結果を決める場合は `Scenario Outline` の `Examples` 表を Decision Table として使う。

```markdown
### Scenario Outline: `prompt=none` の結果

- Given 登録済みの redirect URI が確定している
- And 既存セッションは <session> である
- And 必要な同意は <consent> である
- When クライアントが `prompt=none` で認可を要求する
- Then 結果は <outcome> になる
- And 認可コードの発行は <authorization_code> になる
- And 対話 UI は表示されない

#### Examples: Decision table (Unique)

  | example_id | session | consent | outcome | authorization_code |
  | --- | --- | --- | --- | --- |
  | EX-OAUTH2-005-03 | absent | any | `login_required` | no |
  | EX-OAUTH2-005-04 | present | absent | `consent_required` | no |
  | EX-OAUTH2-005-05 | present | present | success | yes |
```

最初に扱うヒット方針は `Unique` だけとし、実行可能な入力はちょうど一行に一致させる。条件セルの `any` はその条件が行の結果に影響しない値を表す。公式 Markdown with Gherkin パーサーはハイフンだけのセルを含む行を GFM の区切り行として AST から除外するため、一般的な決定表で使われる `-` は採用しない。自然言語で宣言された値域から表の完全性を機械的に証明することはしないが、各行の `example_id`、重複または `any` による重なり、空の結果、テスト被覆は検査する。閉じた値集合を使う表では、実行可能な組み合わせが欠けていないことをレビューと表駆動テストで確認する。

Decision Table と単なるデータ表は区別する。境界値や代表値を同じステップへ代入するだけなら通常の `Examples` とし、条件の組み合わせから規則の結果を決める場合だけ `Decision table (Unique)` と名付ける。順序、履歴、再試行、時間経過で結果が変わる場合は表へ畳まず、独立した例または状態遷移表にする。

### 構文検査とテスト実行の境界

公式 `@cucumber/gherkin` の JavaScript 実装を固定し、`tools/check` が Markdown with Gherkin を解析する。パーサーが返す AST と Pickle を使って `Feature`、`Rule`、例、`Examples` 行の所属を読み、正規表現で見出しと字下げを再実装しない。

Gherkin の構文を採用しても、Cucumber の実行器は導入しない。製品のテストは現在の Go と TypeScript の境界に残し、`EX-*` の引用で具体例との対応を示す。これにより、複数言語へ同じステップ定義層を設けず、既存の単体、受け入れ、E2E テストをそのまま証拠として使える。

拒否の契約検査は、行頭の `ALT` または `THEN` を読む現在の正規表現を廃止し、例の結果ステップからエラー型を取得する。拒否かどうかの自然言語分類は増やさない。拒否例には応答と防いだ効果の双方を書くという既存規則を維持するが、その文章の質を構文検査だけで保証したとは扱わない。

### 移行

検査器と生成器は移行中だけ `scenarios.md` と `scenarios.feature.md` のどちらか一方を受理し、同じディレクトリに両方ある場合は拒否する。代表的な三つの規則を先に移し、構文、表示、被覆、Decision Table の形を確定してから Context 単位で移す。全 Context の移行、参照の更新、負債の作成が終わった時点で旧形式の読み取りを削除し、`scenarios.md` を再び作れないようにする。

試行対象は、複数の `→` を持つ `REQ-OAUTH2-005`、`THEN` の途中へ `ALT` が入る `REQ-SAML-006`、複数条件の積を持つ `REQ-AUTHORIZATION-004` とする。三つはそれぞれ長い代替経路、通常結果を中断する拒否、Decision Table の適用判断を検証する。

試行移行では、`REQ-OAUTH2-005` の通常経路と 7 個の `ALT` を `EX-OAUTH2-005-01` から `08` へ対応させ、認可コード再使用時の二回の応答、トークンファミリー失効、`RefreshTokenReuseDetected`、`TokenRevoked` を同じ例の独立した結果として残した。`REQ-SAML-006` は通常経路と 5 個の `ALT` を `EX-SAML-006-01` から `06` へ対応させ、検証途中の拒否と Assertion 発行後の再利用拒否を別経路にした。`REQ-AUTHORIZATION-004` は通常経路と 3 個の `ALT` を 4 行の `Decision table (Unique)` へ対応させ、主体関係、代行者関係、チェーン状態、スコープ、判定を明示した。三件とも旧経路の条件と結果の各断片が新しい例に一度以上現れることを移行検査で確認した。

移行検査は旧文書から断片の一覧を作り、新文書と突き合わせる。`Scenario Outline` のステップは表の値を代入して初めて旧ステップの文になるため、生の本文だけでなく Pickle へ展開した各行のステップ本文も照合対象にし、空白の有無だけの差は落として比較する。`REQ-AUTHORIZATION-004` の Decision Table は、この展開後に旧文の語順がそのまま残るよう条件ステップの区切り位置を選んだ。検査自身が安全網であるため、断片の分割、エラー型の収集、条件の欠落、結果だけの欠落をテストで固定したうえで移行に使った。移行の受け入れ後、変換器、負債の初期化、監査は一度きりの道具として削除した。監査は作業ツリーを移行直前のリビジョンと比べるため、以後の正常な仕様変更をすべて欠落として報告するようになり、残せば誤報の発生源にしかならない。

### 却下した代替案

**現在の `ALT` を複数行へ拡張する案。** 条件、代替ステップ、通常経路への復帰、終了を表す子要素と識別子が必要になり、Gherkin の `Example` と Cockburn の extension fragment を独自に再実装することになる。公式パーサーと編集支援を使えず、今回の問題を別の独自文法へ移すため却下する。

**EARS をシナリオ正本にする案。** EARS は前提、引き金、システム応答を一つの要求文に分け、不要な振る舞いを `IF` と `THEN` で明示するため、原子的な規則の文章には使える。しかし複数操作の流れ、複数の具体例、表の各行、テストとの一対一の住所を持たない。`Rule` の文章をレビューする補助手段にはできるが、シナリオ記法の置換にはならないため採用しない。

**Cockburn の fully dressed use case を正本にする案。** Main Success Scenario と Extensions は分岐の発見に適するが、各 extension は主経路のステップ番号、処理断片、復帰または終了を必要とし、テストには経路ごとの具体例を別途導出する必要がある。要求発見の方法として使うことは妨げないが、テストへ直接対応する正本にはしない。

**通常の `.feature` を別に追加する案。** 既存の `scenarios.md` と同じ振る舞いを二つの正本に持つため却下する。Markdown with Gherkin は `scenarios.feature.md` を置き換える唯一の正本とし、この問題を生じさせない。

**すべての `ALT` を Decision Table へ変換する案。** トークン再使用、配送再試行、状態遷移のような経路は条件の積ではなく順序に意味がある。表へ変換すると履歴と因果が失われるため、Decision Table は同一時点の条件から結果が決まる規則に限定する。

## Plan

1. Markdown with Gherkin の依存を固定し、代表的な文書を AST と Pickle へ解析する検査を Acceptance RED と Unit RED で置く。
2. `REQ-*` を `Rule`、通常の `Example` と `Examples` 行を `EX-*` として収集する純粋なモデルを定義し、重複、親の不一致、例のない規則、行 ID の欠落、同一ディレクトリの二重正本を拒否する。
3. `REQ-OAUTH2-005`、`REQ-SAML-006`、`REQ-AUTHORIZATION-004` を試行移行し、旧文書との意味対応表を work item の Design へ追記する。旧経路ごとに Given、When、Then、拒否時に防いだ効果が新しい例に残ることを独立に確認する。
4. `Example` と Decision Table 行の被覆を検査し、旧 `REQ-*` 引用が子の `EX-*` を自動的に被覆しないことを RED で固定する。例被覆負債のラチェットと報告を追加する。
5. 生成仕様サイトを Gherkin AST から描画し、Rule ごとに例、Decision Table、テストソース、負債を表示する。
6. Context 単位で残りの正本を移行する。機械変換した後に旧シナリオと新しい例を一件ずつ照合し、意味を判断できない `ALT` は負債へ隠さず移行を停止して、仕様欠陥の follow-up work item を起票する。
7. 全 work item、正本文書、検査、生成器、agent guidance、skills のパス参照を更新する。`docs/README.md` と `docs/structure.md` に正本名と構造を反映する。
8. 全 Context の移行後に旧形式の互換読み取り、`ALT` の検査、`ACTOR` の装飾、`→` の分割処理を削除する。
9. 仕様差分が既存 `REQ-*` の追加、削除、退役、本文の意味変更を報告しないことを確認し、意味変更が見つかった場合は本項目から外して規範参照を持つ work item を起票する。

## Tasks

- [x] T001 [Acceptance] Markdown with Gherkin の有効な正本を現行検査が受理できず、独立した `EX-*` の一つだけがテスト未対応でも現行被覆検査が通ることを RED として記録する。
- [x] T002 [Tooling] 公式 Gherkin パーサーを固定し、`scenarios.feature.md` を AST と Pickle へ解析する境界を追加する。`@cucumber/gherkin` を `tools/package.json` に固定し、`tools/check/src/gherkin-scenarios.ts` が `Parser`、`AstBuilder`、`GherkinInMarkdownTokenMatcher`、`compile` だけを使って解析する。
- [x] T003 [Core] `Feature`、`Rule`、`Example`、`Scenario Outline`、`Examples` 行、親子関係、`REQ-*`、`EX-*` を表す純粋なモデルと検査を実装する。`parseScenarioDocument` が `ScenarioRule` と `ScenarioExample` を返し、`gherkin-scenarios.test.ts` が各拒否を固定する。
- [x] T004 [Pilot] `REQ-OAUTH2-005`、`REQ-SAML-006`、`REQ-AUTHORIZATION-004` を移行し、旧経路と新しい例の意味対応を Design に記録する。
- [x] T005 [Coverage] `EX-*` 単位のテスト引用、理由付き負債、ラチェット、`REQ-*` 被覆の導出を実装する。`checkNormativeCoverage` の宣言集合を standards と example に限り、`example-coverage-debt-baseline.json` で新規 id の受け入れを拒否する。
- [x] T006 [Decision Table] `Scenario Outline` の通常の例表と `Decision table (Unique)` を検査し、各行の `example_id` とテスト被覆を要求する。`example_id` 列の欠落、EX id でない行、空の結果、`any` を含む重なりを拒否する。
- [x] T007 [Security] 拒否のエラー型収集と契約検査を Gherkin の結果ステップへ移し、変更前後で宣言済みエラー型の集合が一致することを固定する。`mise run audit-scenario-migration` が旧 43 件と新 43 件の一致を確認する。
- [x] T008 [Render] 生成仕様サイトとトレーサビリティを Rule、Example、Decision Table、テストソース、負債の階層へ対応させる。トレーサビリティ表が Rule と Example を別行にし、例ごとにテストソースまたは負債理由を表示する。
- [x] T009 [Migrate] ルートと全 Context の規範シナリオを移行し、既存テストへ確認済みの `EX-*` を追加し、未確認分を理由付き負債へ入れる。22 ファイル、308 規則、405 `ALT` を 711 例へ移した。既存テストの観測が例の結果ステップ全体と一致する例はなかったため、確認済みの `EX-*` は 0 件、711 件すべてを理由付き負債に置いた。
- [x] T010 [References] work item、正本文書、検査、生成器、報告、agent guidance、skills の `scenarios.md` 参照を `scenarios.feature.md` へ更新する。`docs/releases/**` の見出しアンカーも `#rule-req-*` へ移し、`mise run check-links` が 681 文書で通る。
- [x] T011 [Docs] `SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`specification-first-workflow.md`、`docs/README.md`、`docs/structure.md` を新しい正本形式と追跡規則へ同期する。
- [x] T012 [Remove Legacy] 旧形式の互換読み取り、`ALT`、`ACTOR`、`→` に依存する検査と表示を削除し、旧 `scenarios.md` の再導入を拒否する。`docs/contexts/audit/scenarios.md` を置くと正本集合の検査が拒否することを確認した。`spec-diff` だけは履歴を読むため旧文法の読み取りを残す。
- [x] T013 [Verify] 標準検証と意味保存の監査を通し、移行で見つかった製品仕様の変更候補を別の work item へ分離する。`mise run verify` と `mise run audit-scenario-migration` が通り、`mise run spec-diff` は規範的な意味変更を報告しない。

## Verification

- `mise run test-tools`
- `mise run typecheck-tools`
- `mise run lint-tools`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run render-spec-docs`
- `mise run spec-diff`
- `mise run verify`
- すべての `scenarios.feature.md` を公式 Gherkin パーサーが解析し、各ファイルが一つの `Feature` を持つ。
- 既存の生きた `REQ-*` の集合が移行前後で一致し、追加、削除、退役、改名がない。
- すべての生きた `Rule` が一つ以上の生きた具体例を持つ。
- 通常の `Example` と `Examples` の各実行可能行が一意な `EX-*` を持つ。
- テストが名指ししない `EX-*` は理由付き負債になければ失敗し、テストが付いた負債項目は削除を要求される。
- 親の `REQ-*` だけを引用するテストでは、子の `EX-*` が被覆済みにならない。
- 一つのパラメーター化テストが複数の `EX-*` を名指した場合は、対応する例をそれぞれ被覆済みと数える。
- `Decision table (Unique)` の全行が `example_id` と結果を持ち、重複した ID、空の結果、例のない表を拒否する。
- `scenarios.md` と `scenarios.feature.md` の併存を拒否し、完了時には `scenarios.md` が一つも残らない。
- `ALT`、シナリオ文法としての `ACTOR`、条件と結果を分ける `→` が正本に残らない。
- 移行前にシナリオから収集したエラー型の集合と、移行後に結果ステップから収集した集合が一致する。
- 生成仕様サイトからすべての `REQ-*` と `EX-*` へ到達でき、各例にテストソースまたは負債が表示される。
- `affected_spec` と `initial_context` にある旧パスが残らず、全参照が解決する。
- `mise run spec-diff` が製品の規範的な意味変更を報告しない。書式変更を意味変更として報告する場合は、比較器を Rule と Example のモデルに合わせてから判定する。

## Risk Notes

リスクは high。実行時の製品コードは変えないが、308 件の規範シナリオと 405 件の代替経路を移すため、条件、結果、拒否時に防いだ効果のどれかを落としても文書と検査が同時に新形式へ移れば気付かない可能性がある。旧構造から得た経路の一覧と新しい `EX-*` の一覧を別々に抽出して比較し、代表三件の手作業監査後に Context 単位で進める。

第二のリスクは、`EX-*` を大量に追加した直後に被覆を見かけ上満たすことである。旧 `REQ-*` の引用をすべての子へ自動配賦すれば、今回直そうとしている粗い追跡を別名で残すことになる。テストの入力と観測を確認できた例だけを対応済みとし、残りを負債として明示する。

第三のリスクは、Decision Table が「表にしたから網羅した」という誤解を生むことである。構文検査が保証できるのは行 ID、値の有無、重複、テスト引用までであり、自然言語の値域に対する完全性は保証できない。この限界を仕様形式と生成された被覆表示に明記し、閉じた値集合では表駆動テストとレビューで実行可能な組み合わせを確認する。

第四のリスクは、公式パーサーを使いながら周辺に独自規則を増やすことである。`REQ-*` と `EX-*` はリポジトリの追跡契約として必要だが、分岐、背景、例の展開、表の解析は Gherkin AST に委ねる。パーサーが表現できない構造が必要になった場合は、独自構文を足す前に規則または例の分割を選ぶ。

`reversibility` は irreversible とする。ファイル名と検査実装は戻せても、新しい `EX-*` をテストと履歴が引用し始めた後に、その識別子体系を存在しなかった状態へ戻すことはできない。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を報告する。308 件の規則 id、その本文が名指しするエラー型 43 件、旧経路の条件と結果の断片はすべて残り、追加、削除、退役、改名はない。変わったのは正本の形式と追跡の粒度だけである。規範シナリオの正本は 22 ファイルの `scenarios.md` から `scenarios.feature.md` へ移り、306 件の生きた規則が `Rule` に、正常経路と 405 件の `ALT` が 711 件の `Example` と `Examples` 行に分かれ、各例が `EX-<CONTEXT>-<REQ-NNN>-<sequence>` の住所を得た。被覆の宣言集合は規則から例へ移り、テストが名指さない例は理由付き負債に載る。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 製品の振る舞いを変えない仕様方法論と検査の変更であり、観測可能な製品境界を持たない。
  - **Observed Failure**: 移行前の検査は `scenarios.feature.md` を正本と認めず、`docs/contexts/*/scenarios.feature.md: not a canonical document` で拒否した。逆向きの確認として、移行後に `docs/contexts/audit/scenarios.md` を置くと `not a canonical document; docs/contexts/audit/ holds only README.md, glossary.md, standards.md, states.md, decisions.md, internals.md, scenarios.feature.md` で拒否する。
  - **Detection Reason**: 受け入れ境界の代わりに正本集合の検査が実際に失敗した。この検査は正本の名前を集合として持つため、「新形式を受理する」と「旧形式を再導入できない」の両方向を区別する。片方だけを満たす実装は、どちらかの実行で必ず落ちる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/gherkin-scenarios.test.ts` と `tools/check/src/normative-coverage.test.ts`
  - **Requirement**: N/A: 検査器の内部境界であり、規範シナリオを持たない。
  - **Observed Failure**: `parseScenarioDocument` の実装前、`rejects examples outside a rule and a live rule without examples` などは `parseScenarioDocument is not a function` で落ちた。被覆側では `does not let a rule citation cover the examples under it` が、親の `REQ-*` 引用を子の `EX-*` へ配賦する実装で落ちる。
  - **Detection Reason**: この項目が直そうとしている粗い追跡は「親を引用すれば子も被覆済み」という実装として現れる。被覆側のテストは引用集合と宣言集合を別々に与え、`REQ-OAUTH2-005` の引用が `EX-OAUTH2-005-01` を含まないことを直接主張するため、配賦する実装と配賦しない実装を分ける。
- **Change-Resistance Results**:
  変更した判定ロジックへ系統的に故障を注入した。`gherkin-scenarios.ts` と `normative-coverage.ts` の各判定を無効化する 11 個の変異を作り、`mise run test-tools` が検出するかを測った。初回は 8 件が検出、3 件が生存した。生存したのは (a) `example_id` 列の欠落、(b) `Examples` 行の値が EX id でない場合、(c) 引用 id の左端の境界である。(a) と (b) は行に住所がないまま表を受理する変異、(c) は `PRE-EX-DEMO-001-02` のような別 id の一部を引用と数える変異で、いずれも移行後の追跡粒度をそのまま無効にする。三つを直接固定するテストを追加し、再測定で 11 件すべてを検出した。
  移行の安全網である `audit-migration.ts` 自身にも同じ測定を行い、断片比較を無効化する変異が生存することを確認した。純粋な比較部分を `fragment-audit.ts` へ分け、断片の分割、エラー型の収集、条件の欠落、結果だけの欠落を固定するテストを追加した。
  方法の限界: 変異は判定条件の無効化に限り、`@cucumber/gherkin` のパーサー本体、`render-spec-docs` の描画、711 件の例の日本語本文が旧経路の意味を保っているかは対象外である。最後の点は構文検査で保証できないため、断片の照合と代表三件の手作業監査で扱った。同義の言い換えは断片照合では検出できず、これは移行検査が持つ本質的な限界である。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run audit-scenario-migration` - passed (22 files, 308 rules, 405 alternatives, 711 examples, 43 error types)
  - `mise run spec-diff` - no normative specification change against main
  - `mise run check-links` - ok (681 documents)
