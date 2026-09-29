---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-30
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 仕様文書の配置と、それを読む開発用の検査だけを変える。製品の API、設定、振る舞いは変わらず、リリースの読者へ知らせる内容がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/development/specification-first-workflow.md
    - docs/domain/authentication/README.md
    - docs/domain/authentication/scenarios.feature.md
    - docs/domain/authentication/federation/states.md
    - docs/domain/authentication/trusted-device/states.md
    - docs/domain/authentication/decisions.md
    - docs/domain/authentication/internals.md
    - docs/domain/authentication/glossary.md
  typespec: []
  source:
    - tools/check/src/specification-doc.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/gherkin-scenarios.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/canonical-document-set.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/src/registry.ts
    - tools/workspace/src/document-layout.ts
    - tools/workspace/src/workspace.ts
    - tools/check/schemas/work-item.schema.json
  tests:
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/spec-diff.test.ts
  stop_before_reading:
    - frontend
    - docs/domain/oauth2
    - backend/authentication
spec_impact: { kind: none, reason: "仕様文書の配置、規則一件の書式、それを読む検査を変え、authentication の既存の規範要素を ID を変えずに機能ごとのノードへ移すだけで、どの規範要素の意味も変えない。製品の API、永続状態、外向きの呼び出し、イベント、配備構成も変わらない。" }
---

# 仕様を、システム、コンテキスト、機能の木に組織し、規則一件の書式を定める

## 動機

仕様の細部を実装に合わせて書き起こすと、規則の数は大きく増える。
増えた規則を AI が迷わず書け、AI と人が目的の規則へたどり着ける形でなければ、書き起こした仕様は読まれず、変更の検出にも使われない。
今の仕様の組織には、これを妨げる問題が四つある。

**1. 機能の階層がない。**
コードは `backend/authentication/{session,totp,trusteddevice,password,federation,mfa,recovery,webauthn,securitynotification}/` のように機能ごとのスライスに分かれている。
一方、文書は `docs/domain/<context>/` の種類別ファイルで止まっている。
`scenarios.feature.md` は Rule を REQ 番号順、つまり追加した順に並べた平らな一覧である。
authentication は 37 の Rule（975 行）に、外部 IdP の連携、パスワード、TOTP、セッション、信頼済みデバイス、通知が混在し、oauth2 は 1,460 行ある。
`SPECIFICATION_FORMAT.md` は「機能別ディレクトリへ分割する」規則を持つが、使っているコンテキストはない。
さらに、`tools/check/src/specification-doc.ts` は `docs/domain/<context>/<file>` の一段だけを一次情報文書として認識する。
機能別ディレクトリに置いた文書は、`check-spec` からも `spec-diff` からも見えない。

**2. 一つの概念の仕様が種類別ファイルに散る。**
信頼済みデバイスの規則は、`glossary.md`、`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md`、`docs/design/data/database.md`、`docs/design/data/lifecycle.md` の 7 か所にある。
「信頼済みデバイスはいつ無効になるか」に答えるには、読み手がこれらを読み合わせて組み立てる必要がある。

**3. 規範が ID のない文章に紛れている。**
信頼済みデバイスの絶対期限の上限 90 日と、idle 期限 `min(30 日, max_age)` は、製品が守るべき値である。
それが `internals.md` の仕組みの説明の中にあり、ID も担保するコードの名前もない。
`spec-diff` は `internals.md` を比較しないため、値を変えても仕様の変更として検出されない。

**4. 上位の規則と下位の例外の関係が一方向にしか書かれていない。**
`docs/design/application/api-guidelines.md` の「ページサイズ」は、既定 50 件、上限 200 件、不正な `limit` はエラーという規則と、担保手段 `support_http.ParseLimit`、適用状況「全面適用」を書いている。
例外として、サインイン履歴（既定 10 件、上限 50 件）も上位の文書に列挙している。
ところがサインイン履歴は `ParseLimit` を使わず、独自の `parseLimitParam` で不正な `limit` を黙って既定値に置き換えている。
この逸脱はサインイン履歴の側のどこにも書かれておらず、「全面適用」という記述は実態と食い違ったまま気付かれていない。

## 対象範囲

- 仕様の木を三つの階層として定め、`SPECIFICATION_FORMAT.md` と `DOCUMENTATION_GUIDE.md` に書く。

  | 階層 | 置き場所 | 置く規則 |
  | --- | --- | --- |
  | システム | `docs/design/`、`docs/domain/scenarios.feature.md` | すべてのコンテキストが従う規則 |
  | コンテキスト | `docs/domain/<context>/` | そのコンテキストの複数の機能にまたがる規則 |
  | 機能 | `docs/domain/<context>/<feature>/` | 一つの機能の規則 |

- 機能ノードとコードの対応を定める。
  `docs/domain/<context>/<feature>/` は `backend/<context>/<feature>/` に対応させる。
  コードに機能スライスがない機能（たとえばサインイン履歴）の扱いと、名前の対応規則（`trusteddevice` と `trusted-device` など）も定める。
- 規則は一か所に書き、下位のノードは上位の規則からの例外だけを書く、という継承の規則を定める。
  例外は下位のノードで宣言し、上位の規則へリンクする。
  上位の文書にある例外の一覧は、下位の宣言から生成するか、下位の宣言と照合する。
- 機能ノードの中の規則の並びを、ドメインの概念のライフサイクル順に固定する。

  | 順 | 節 | 答える問い |
  | --- | --- | --- |
  | 1 | 生成 | いつ作るか、いつ作らないか |
  | 2 | 有効性 | いつ有効で、いつ無効か（境界を含む） |
  | 3 | 利用 | 使うと何が起き、何が変わるか |
  | 4 | 効果の範囲 | 何を肩代わりし、何を肩代わりしないか |
  | 5 | 失効と変更 | 何が失効させ、失効すると何が残るか |
  | 6 | 保持と削除 | どれだけ残し、いつ消すか |

  API の操作が中心の機能では、対象の操作 → 入力（既定値、上限、検証）→ 結果（順序、書式）→ 拒否 → 作用（永続状態、イベント、通知）の順とする。
- 規則一件の書式を定める。
  各規則は、ID、一文で書いた題名、箇条書きの規則文、理由（`decisions.md` へのリンク）、担保手段、上位の規則（例外の場合）、例（`EX-*`）、要判断の欄を持つ。
  規則文は一行に一つの義務だけを書く。
  条件の組み合わせは表にし、すべての入力がちょうど一行に一致するように書く。
  処理の順序が振る舞いを決める場合だけ、番号付きの手順で書く。
- `internals.md` に置いてよい内容を、仕組みの説明と壊れたときの直し方に限る。
  製品が守る値や条件は規則に置く。
- `tools/check/src/specification-doc.ts` と関連する検査を、機能ノードの文書も一次情報文書として認識するよう拡張する。
- `spec-diff` を、規則の本文（題名だけでなく規則文と表）の変更も、その規則 ID の変更として報告するよう拡張する。
- 次の検査を `check-spec` に加える。
  - 規則の担保手段に挙げたシンボルが、コードに存在する。
  - 例外として宣言した規則の「上位の規則」が存在する。
  - コードの機能スライスごとに、対応する機能ノードが存在する（導入時点で対応のないスライスは、基準リビジョンより増やさない）。
  - 規則文と要判断の欄に、定型句や曖昧な語（「適切に」「など」「必要に応じて」）がない。
- 試行として、authentication コンテキストを機能ノードへ再編する。
  既存の Rule を ID を変えずに機能ノードへ移し、種類別ファイルの内容も機能ごとに分ける。
  規範の意味は変えない。
- `SPECIFICATION_FORMAT.md` の該当節、`.agents/skills/spec-change/SKILL.md`、`.agents/skills/update-design/SKILL.md` を、新しい配置と書式に合わせて更新する。

## 対象外

- authentication 以外のコンテキストの再編。
  コンテキストごとの work item で扱い、`wi-95161` の書き起こしの work item と同じ単位で進める。
- `internals.md` の文章に紛れた規範を規則へ昇格させる作業。
  新しい規範要素（ID）を宣言する作業であり、書き起こしの work item で扱う。
  試行で見つけた規範は、書き起こしの work item の入力として記録する。
- 試行で見つけた文書と実装の食い違いの是正。
  たとえば `states.md` は「同じ理由での再失効は no-op」と書くが、`TrustedDevice.Revoke` は理由が異なっても最初の理由を保つ。
  こうした食い違いは記録し、書き起こしまたは是正の work item へ渡す。
- 上位の規則の担保手段が、範囲内のすべての操作で使われているかの静的な照合。
  サインイン履歴の `ParseLimit` 不使用はこの照合で機械的に見つかるが、規則ごとに範囲を定義する仕組みが別に必要なため、別の work item とする。
- ID に版を持たせ、古い版を引くテストを拒否する仕組み（OpenFastTrace 方式）。
  `wi-95161` の宣言の検査が同じ問題の一部を扱うため、その運用を見てから判断する。
- `docs/design/` の文書の再編。
  すでに範囲、対象外、子、隣接を宣言する固定の木であり、この work item は規則一件の書式だけを揃える。

## 設計

### 組織の原則

木は、読み手の問いの順序に合わせる。
読み手は「これはシステム全体の規則か」「どのコンテキストか」「どの機能か」「その機能のどの段階か」の順に絞り込む。
コードも同じ順に分かれているため、文書の木をコードの木と同じ形にすれば、AI はパッケージのパスから対応する仕様ノードを探索せずに決められる。
`wi-95161` の `spec-review-candidates` の結果も、そのまま一つの機能ノードに対応する。

規則を一か所に書くことは、既存の「同じ事実を二か所に書かない」原則をノードの間に広げたものである。
下位のノードが上位の規則を書き写すと、片方だけが更新される。
下位のノードが逸脱を宣言しなければ、サインイン履歴のように、逸脱はコードにだけ存在する暗黙の仕様になる。

### 規則一件の書式

`docs/design/application/api-guidelines.md` がすでに使っている「規則文、目的、担保手段、適用状況」の形を基にする。
新しい記法を持ち込むより、既存の読み手と既存の文書に揃える方が、AI の書き方も人の読み方も一つで済む。

信頼済みデバイスの機能ノードでは、次のようになる。

```markdown
## 有効性

### REQ-AUTHENTICATION-038 絶対期限と idle 期限の両方を満たす間だけ有効とする

- 絶対期限は、発行時刻にテナントの `trusted_device_max_age_seconds` を加えた時刻とする。上限は 90 日とする。
- idle 期限は、最終利用時刻に 30 日と絶対有効期間の短い方を加えた時刻とする。
- どちらの期限も、その時刻ちょうどで無効とする。
- 理由：[有効期間を二つの期限で切る](decisions.md#有効期間を二つの期限で切る)
- 担保手段：`TrustedDevice.Active`
- 例：EX-AUTHENTICATION-027-02
```

上位の規則からの例外は、次のようになる。

```markdown
### REQ-AUTHENTICATION-039 サインイン履歴のページサイズ

- 上位の規則：[ページサイズ](../../../design/application/api-guidelines.md#ページサイズ)
- 既定値は 10 件、最大値は 50 件とする。
- 0 以下または整数でない `limit` は、エラーにせず既定値として扱う。
- 担保手段：`authusecases.ListSignInActivity`、`parseLimitParam`
- 要判断：不正な `limit` の扱いが上位の規則と異なる。維持するか、400 を返すかを是正の work item で決める。
```

各欄の役割は次のとおりである。

| 欄 | 役割 |
| --- | --- |
| ID と題名 | テストの `//spec:covers`、`affected_spec`、`spec-diff` が指す単位 |
| 規則文 | 一行に一つの義務。条件の組み合わせは表 |
| 理由 | `decisions.md` の判断へのリンク。規則に理由を書き写さない |
| 担保手段 | 規則を実装するコードのシンボル。コードから規則へ、規則からコードへの両方向の追跡に使う |
| 上位の規則 | 例外のときだけ置く。上位の規則へのリンク |
| 例 | 代表例、境界、意外な挙動の `EX-*`。すべての場合の列挙はテストに置く |
| 要判断 | 現在の挙動を書いたうえで、維持するか是正するかが決まっていない点。是正の work item が決着させたら消す |

一行に一つの義務、曖昧な語の禁止、条件の表は、INCOSE の Guide to Writing Requirements の規則から、機械で検査できるものを採った。
要判断の欄は、GitHub Spec Kit の `[NEEDS CLARIFICATION]` に当たる。
番号付きの手順は、WHATWG の仕様が処理の順序を書く形に倣う。

### ID の体系

ID は `REQ-<CONTEXT>-NNN` のまま、コンテキスト単位の通し番号とする。
機能を ID に含めない。
規則を別の機能ノードへ移しても ID が変わらないようにするためである。
既存の「一度参照された ID は変更しない」規則とも両立する。
上の例の番号は説明のための仮のものであり、実際の番号は採番時に決まる。

### 規則と Gherkin の関係

今の規範の単位は、`scenarios.feature.md` の Gherkin の `Rule` である。
この書式を載せる方式は二つあり、試行の最初に決める。

| 方式 | 利点 | 負担 |
| --- | --- | --- |
| A. Gherkin の `Rule` の説明文に規則の欄を書く | 既存の構文解析、被覆の検査、`spec-diff` をそのまま拡張できる。ファイルの種類が増えない | Gherkin の構造では `Rule` の間に節の見出しを置けない可能性があり、ライフサイクル順の節を表せないおそれがある |
| B. 規則を `rules.md` に置き、`scenarios.feature.md` には Example だけを残す | 節の見出しを自由に置け、規則を文章として読みやすい | 新しい種類のファイルと構文解析が必要になる。Example を規則の ID へ結び付ける規則を新しく定める必要がある |

判断の基準は、ライフサイクル順の節を表せるかと、既存の検査の再利用の量である。
公式の Markdown with Gherkin の構文解析が `Rule` の間の見出しを許すなら A、許さないなら B とする。

**決定：方式 A を採る。**
`@cucumber/gherkin` の `GherkinInMarkdownTokenMatcher` で、`# Feature:`、`## 有効性`、`### Rule:`、`#### Example:` の順に見出しを置いた文書を試作して解析した。
キーワードを持たない `## 有効性` は構文エラーにならず読み飛ばされ、`Rule` と `Example` の親子関係も保たれた。
したがって、ライフサイクル順の節は方式 A で表せる。
一方、`Rule` の説明文のうち箇条書きの行は AST に残らない（試作では表の本文の行だけが `description` に残った）。
そのため、規則の欄と規則文は AST からではなく、`Rule` の見出しの次の行から次の見出しの直前までのソース行として読む。
方式 B が要する新しいファイルの種類、構文解析、Example と規則の結び付けの規則は、どれも不要になる。

### 機能ノードの配置

| 判断 | 内容 | 理由 |
| --- | --- | --- |
| 機能ノードに置けるファイル | `README.md`、`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md` | 既存の「コンテキスト内の分割」の規則が、境界の宣言と索引、共有語彙、採用した外部標準をコンテキストのルートに残すと定めている。`glossary.md` と `standards.md` を機能ノードに置くと、同じ語や標準の行が二か所に書ける |
| `glossary.md` の扱い | 機能ごとに分けず、コンテキストのルートに残す | 計画の 6 は用語集も分けるとしていたが、上の規則と衝突する。Aggregate root の一覧と語は機能をまたいで使われる |
| 機能ノードの下の階層 | 置かない | 木を三階層に固定する。四階層目を許すと、パッケージのパスから仕様ノードが一つに決まらない |
| 見出しの階層 | 機能ノードの `scenarios.feature.md` は `# Feature:`、`## <節>`、`### Rule:`、`#### Example:` とする | 節の見出しを `##` に置くため |
| 節の語彙 | ライフサイクル（生成、有効性、利用、効果の範囲、失効と変更、保持と削除）か、API（対象の操作、入力、結果、拒否、作用）のどちらか一方を、この順で使う。同じ節は一度だけ置き、すべての `Rule` をいずれかの節の下に置く *(checked)* | 並びを検査できる形にするため |
| コンテキストのルートの `scenarios.feature.md` | 節を要求しない | 機能をまたぐ規則は一つの概念のライフサイクルに並ばない |
| 名前の対応 | 機能ノードの名前からハイフンを除いた名前が、コードの機能スライスの名前と一致する（`trusted-device` と `trusteddevice`） | 文書はケバブケース、Go のパッケージは小文字の連結という、それぞれの既存の命名規則を変えずに対応させるため |
| コンテキストの名前の対応 | 同じ規則で対応させ、一致しないもの（`idmanagement` と `identity-management`）は検査の設定に別名として書く | 既存のディレクトリ名を変えないため |
| 機能スライスの判定 | `backend/<context>/<name>/` の直下に `domain/` または `usecases/` があるもの | `docs/domain/structure.md` の「機能ごとの垂直分割」の形 |
| スライスのない機能 | 機能ノードを置いてよい（サインイン履歴の `sign-in-activity`）。検査はスライスから機能ノードの向きだけを問う | コードの配置は仕様の境界の判断材料の一つにすぎない |
| 対応のないスライスの基準 | `tools/check/feature-node-debt.json` に導入時点の一覧を置き、一覧にないスライスが対応を欠く場合と、対応を得たスライスが一覧に残る場合の両方を拒否する | 既存の `boundary-debt.json` と同じ、減る方向にしか動かない基準 |

### 担保手段のシンボルの形

担保手段の欄のバッククォートで囲んだ各値を、`backend/` の Go のテスト以外のソースで宣言された名前と照合する。

| 形 | 一致とみなす宣言 |
| --- | --- |
| `<型>.<メソッド>` | その型をレシーバーとするメソッド |
| `<パッケージ名>.<名前>` | そのパッケージ名のパッケージで宣言された関数、型、定数、変数 |
| `<名前>` | どこかで宣言された関数、型、メソッド |

パッケージ名は `package` 句の名前とし、import の別名は使わない。
設計の例に書いた `authusecases` は別名であり、実際に書くときは `usecases.ListSignInActivity` とする。

### 検査の型と操作

検査はすべて純粋な関数とし、ファイルの読み取りは `check-*.ts` の側に置く。

```ts
type RuleBody = { id: string; line: number; lines: Array<{ line: number; text: string }> }
type RuleFields = {
  statements: Array<{ line: number; text: string }>
  guarantees: Array<{ line: number; symbol: string }>
  parents: Array<{ line: number; target: string }>
  openQuestions: Array<{ line: number; text: string }>
}
type GoDeclarations = { names: Set<string>; qualified: Set<string> }
type FeatureSlice = { context: string; name: string; path: string }

function ruleBodies(source: string): RuleBody[]
function ruleFields(body: RuleBody): RuleFields
function goDeclarations(files: Array<{ path: string; source: string }>): GoDeclarations
function verifyRuleFields(path: string, source: string, declarations: GoDeclarations, resolveLink: (from: string, target: string) => boolean): Finding[]
function verifySectionOrder(path: string, source: string): Finding[]
function verifyFeatureNodes(slices: FeatureSlice[], nodes: Set<string>, debt: FeatureNodeDebt): Finding[]
```

`spec-diff` は、規則の事実に `ruleBodies` の行を加える。
状態機械は、置いたファイルではなくコンテキスト（`docs/domain/<context>`）で同定する。
そうしないと、状態機械をコンテキストのルートから機能ノードへ移しただけで変更として報告される。

### authentication の割り当て

| ノード | 規則 | 節 |
| --- | --- | --- |
| ルート | 004、005、007、009 | なし |
| `federation` | 002、025（生成）、001（利用）、003（失効と変更）、037（保持と削除） | ライフサイクル |
| `password` | 024（有効性）、008、010、016（失効と変更） | ライフサイクル |
| `totp` | 011（生成）、017（利用）、012（失効と変更） | ライフサイクル |
| `mfa` | 018、019（生成）、015（利用）、020（効果の範囲）、022、023（失効と変更） | ライフサイクル |
| `webauthn` | 006（利用） | ライフサイクル |
| `recovery` | 036（利用） | ライフサイクル |
| `session` | 013、021、035（失効と変更） | ライフサイクル |
| `trusted-device` | 026（生成）、027（有効性）、029（効果の範囲）、028（失効と変更） | ライフサイクル |
| `security-notification` | 030、031、032、034（生成）、033（失効と変更） | ライフサイクル |
| `sign-in-activity` | 014（結果） | API |

ルートに残す 4 件は、ログイン、アカウントの API、無効なユーザーの拒否のように、複数の機能スライスをまたぐ。

### 試行の対象を authentication にする理由

authentication は、機能スライスが最も多く（9 個）、`scenarios.feature.md` に機能が最も混在し、`internals.md` に紛れた規範の実例（信頼済みデバイスの有効期間）と、上位の規則からの逸脱の実例（サインイン履歴のページサイズ）の両方を含む。
この書式と木がここで成り立てば、ほかのコンテキストでも成り立つ。

### 参考にした既存例

| 例 | 取り込んだもの |
| --- | --- |
| [EARS](https://en.wikipedia.org/wiki/Easy_Approach_to_Requirements_Syntax) | 条件の種類を明示して一文に一つの義務を書く考え方 |
| [INCOSE Guide to Writing Requirements](https://www.incose.org/docs/default-source/working-groups/requirements-wg/guidetowritingrequirements/incose_rwg_gtwr_v4_summary_sheet.pdf) | 一つのことだけを述べる、曖昧な語を使わないなど、機械で検査できる規則 |
| [OpenFastTrace](https://devdocs.jabref.org/code-howtos/requirements.html) | 要求に ID を付け、実装とテストから引く追跡。版の仕組みは対象外とした |
| [StrictDoc](https://strictdoc.readthedocs.io/en/latest/strictdoc_01_user_guide.html) | 要求一件を決まった欄で書き、欠けた欄を機械で検出できるようにする考え方 |
| [WHATWG Infra Standard](https://infra.spec.whatwg.org/) | 順序が意味を持つ処理を番号付きの手順で書く形 |
| [GitHub Spec Kit](https://learn.microsoft.com/en-us/training/modules/spec-driven-development-github-spec-kit-enterprise-developers/5-write-effective-spec-file) | 未決定の点を本文の中で明示する印 |

### 採用しない代替案

| 代替案 | 採用しない理由 |
| --- | --- |
| 種類別ファイルの平らな配置のまま、`scenarios.feature.md` の中で並べ替える | 1,000 行を超えるファイルが残り、状態、判断、仕組みは別のファイルに散ったままになる |
| 機能を ID に含める（`REQ-AUTHENTICATION-TD-001`） | 機能の境界を引き直すたびに ID が変わる |
| 上位の文書に例外を列挙し続ける | 機能を読む人と編集する AI から例外が遠く、逸脱を宣言しないまま実装できてしまう |
| EARS の英語のキーワードをそのまま使う | 日本語の文書に英語の構文を混ぜることになる。条件の種類は規則文の中の表現と表で十分に表せる |
| すべての規則を Gherkin の Example で書く | 規則全体を読むために多数の例を読む必要があり、読み手の負担が大きい |

### 書き起こしと是正への入力

authentication の再編で見つけた項目を、書き起こしの work item（`wi-95161` と同じ単位）と是正の work item の入力として記録する。
この work item では、どれも規範へ昇格させず、是正もしない。

`internals.md` の文章に紛れた、製品が守る値と条件は次のとおりである。

| 置き場所 | 値または条件 |
| --- | --- |
| `trusted-device/internals.md` | 絶対期限の上限 90 日、idle 期限 `last_used_at + min(30 日, max_age)` |
| `session/internals.md` | セッションの有効期限は作成時から固定 1 時間で延長しない、`last_seen_at` の更新は最短 5 分間隔、期限切れのレコードの保持 90 日 |
| `password/internals.md` | `min_length=12`、`max_length=128`、4 文字未満の識別子は照合しない、`history_depth=5`、`max_age_days` は 30〜3,650 日、リセットのトークンの `ttl=1800s` |
| `internals.md`（ルート） | ログインの流量制限（アカウント単位 900 秒に 10 回、IP 単位 900 秒に 30 回、拒否 900 秒）、集計の閾値（5 分、アカウント 10 回、IP 50 回、テナント 1000 回）、保持期間（成功 365 日、失敗 30 日、集計 90 日）、ステップアップの直近性 5 分、TOTP のパラメーター、WebAuthn の `challenge_bytes=32` と `timeout_seconds=120`、復旧コード 10 文字 10 個 |
| `security-notification/internals.md` | SMTP の待ち時間の上限 10 秒、既知の端末のレコードを `last_seen_at` から 365 日で消す |

文書と実装の食い違いは次のとおりである。

| 文書 | 実装 | 食い違い |
| --- | --- | --- |
| `trusted-device/states.md` の「同じ理由での再失効は安全な no-op」 | `TrustedDevice.Revoke`（`backend/authentication/trusteddevice/domain/trusted_device.go`） | 実装は失効済みなら理由を問わず何もせず、最初の理由を保つ |
| `docs/design/application/api-guidelines.md` のページサイズの規則（不正な `limit` はエラー、適用状況は全面適用） | サインイン履歴の `parseLimitParam`（`backend/authentication/handlers_http/account_activity_handler.go`） | `ParseLimit` を使わず、0 以下または整数でない `limit` を黙って既定値にする |

文書の中の重複も見つけた。
ルートの `decisions.md` は、セルフサービス API の境界と機微な自己操作のステップアップを二つの項目で重ねて書き、`mfa/decisions.md` は登録専用フローへの到達条件を二つの項目で重ねて書いている。
再編では規範の意味を変えないため、項目を移しただけで統合していない。

## 計画

1. 方式 A と B を、信頼済みデバイスの規則で試作して比べ、方式を決める。
   決めた方式と理由をこの work item の「設計」に追記する。
2. `SPECIFICATION_FORMAT.md` と `DOCUMENTATION_GUIDE.md` に、木の階層、継承の規則、機能ノードの並び、規則一件の書式を書く。
3. `specification-doc.ts` と関連する検査を、機能ノードの文書も認識するよう拡張する。
4. `spec-diff` を、規則の本文の変更も報告するよう拡張する。
5. 担保手段、上位の規則、機能ノードの対応、曖昧な語の検査を `check-spec` に加える。
6. authentication を機能ノードへ再編する。
   既存の Rule は ID を変えずに移し、`states.md`、`decisions.md`、`internals.md`、`glossary.md` の内容を機能ごとに分ける。
   再編の前後で `spec-diff` が規範の変更を報告しないことを確かめる。
7. 再編の過程で見つけた、`internals.md` に紛れた規範と、文書と実装の食い違いを一覧にし、書き起こしと是正の work item の入力として記録する。
8. skills を新しい配置と書式に合わせて更新する。

作るものを変え得る問いは、計画の 1 の方式の選択だけである。
これは実装の最初の段階で試作によって決め、決まるまで後続の段階に着手しない。

## タスク

- [x] T001 [Spec] 方式 A と B を信頼済みデバイスで試作し、方式を決めて「設計」に記録する。
- [x] T002 [Spec] `SPECIFICATION_FORMAT.md` と `DOCUMENTATION_GUIDE.md` に、木、継承、並び、規則の書式を書く。
- [x] T003 [Acceptance] 機能ノードに置いた規則の本文を変えた作業ツリーで、`mise run spec-diff` がその規則の変更を報告しないことを観測する。
  規範要素は N/A（`spec_impact: none` の検査の変更）。
  予定する RED：`REQ-AUTHENTICATION-026` を `trusted-device/scenarios.feature.md` へ移して本文を一行変え、`mise run spec-diff` が変更ではなく削除として報告する（機能ノードの文書を読まない）。
- [x] T004 [App] `specification-doc.ts` と関連する検査を機能ノードへ拡張する。単体 RED を確認し、GREEN にしてからリファクタリングする。
  予定する RED：`specification-doc.test.ts` に、`docs/domain/<context>/<feature>/scenarios.feature.md` の `documentKind` が `scenarios` になり、`glossary.md` は機能ノードでは一次情報文書にならない例を加える。
  反復の検査：`mise run test-tools-file -- check/src/<file>.test.ts`、振る舞いが GREEN になったら `mise run test-tools`。
- [x] T005 [App] `spec-diff` を規則の本文の変更へ拡張する。
  予定する RED：`spec-diff.test.ts` に、規則の本文の表の一行だけを変えた版で `changedScenarios` に ID が入る例と、状態機械をルートから機能ノードへ移しても `changedTransitions` が空の例を加える。
- [x] T006 [App] 担保手段、上位の規則、機能ノードの対応、曖昧な語、節の並びの検査を `check-spec` に加える。
  予定する RED：`specification-rules.test.ts` に、存在しないシンボル、存在しない上位の規則、対応のないスライスの追加、曖昧な語、逆順の節の各例を加える。
- [x] T007 [Docs] authentication を機能ノードへ再編し、再編の前後で規範の差分がないことを確かめる。
- [x] T008 [Plan] 見つけた規範と食い違いを一覧にし、書き起こしと是正の work item の入力として記録する。
- [x] T009 [Docs] `spec-change` と `update-design` の skills を更新する。
- [x] T010 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run check-spec`
- `mise run spec-diff`（authentication の再編の前後で、規範の変更が報告されないこと）
- `mise run check`
- `mise run verify`

変更への耐性として、少なくとも次の誤実装を注入し、検査が失敗することを確認する。

- 機能ノードに置いた `scenarios.feature.md` を一次情報文書として認識しない。
- 規則の本文の表の一行を変えても、`spec-diff` が変更を報告しない。
- 存在しないシンボルを担保手段に書いても通る。
- 存在しない上位の規則を例外に書いても通る。
- 対応する機能ノードのない機能スライスを新しく追加しても通る。

## リスク

- 再編で規範の意味が変わる。
  再編の前後で `spec-diff` を実行し、規範の変更が報告されないことを確かめる。
  移動と内容の変更を同じコミットに混ぜない。
- 機能の境界がコードのスライスと一致しない。
  サインイン履歴のようにスライスを持たない機能もある。
  対応の規則に例外の書き方を用意し、境界の判断は `SPECIFICATION_FORMAT.md` の「コンテキスト内の分割」の基準に従う。
- 規則の書式が重くなり、書き起こしの量が増える。
  欄のうち必須は ID、題名、規則文、担保手段に限り、理由、上位の規則、例、要判断は該当する場合だけ置く。
- `wi-99632` が `SPECIFICATION_FORMAT.md` と `DOCUMENTATION_GUIDE.md` を汎用の部分木へ移す。
  同時に進めると衝突するため、どちらかを先に完了させる。
  この work item の書式の規則は汎用の方法論として書き、IdMagic 固有の値（`backend/` のパス、`mise` のタスク名）は差し込み点として扱える形にする。
- 検査を機能ノードへ広げると、これまで認識されていなかった文書が検査にかかり、既存の違反が表面化する。
  authentication の再編と同時に解消する。

## 完了

- **Completed At**: 2026-09-30
- **Summary**:
  `mise run spec-diff` は再編の後も「no normative specification change against main」を報告し、規範要素の追加、削除、変更はない。
  仕様をシステム、コンテキスト、機能の三階層の木に置く規則、機能ノードとコードの機能スライスの対応、継承の規則、機能ノードの中の節の並び、規則一件の書式を `SPECIFICATION_FORMAT.md` に定め、`DOCUMENTATION_GUIDE.md` と開発ワークフローから参照した。
  `documentNames` を配置の唯一の判定として導入し、`documentKind`、文書検査、ファイル集合の検査、作業ツリーの文書探索、文書サイト、`spec-route`、`brief`、セキュリティ統制の検査、work item のスキーマが機能ノードの文書を一次情報として扱うようにした。
  `spec-diff` は規則の本文（規則文、欄、表）を規則の事実に含め、状態機械をコンテキストで同定する。
  `check-spec` に `specification-rules` の検査を加え、担保手段のシンボル、上位の規則のリンク、曖昧な語、機能ノードの節の並び、機能スライスと機能ノードの対応（`tools/check/feature-node-debt.json` を基準とする）を確かめる。
  authentication の 37 件の規則のうち 33 件を 10 の機能ノードへ ID を変えずに移し、`states.md`、`decisions.md`、`internals.md` を機能ごとに分けた。
  移した規則を指す work item と文書の参照を移動先へ書き換えた。
  `glossary.md` は既存の分割の規則に従いコンテキストのルートに残し、計画の 6 から外した。
- **Acceptance RED Evidence**:
  - **Test**: `REQ-AUTHENTICATION-026` を `docs/domain/authentication/trusted-device/scenarios.feature.md` へ移して本文を一行変えた作業ツリーでの `mise run spec-diff`。
  - **Requirement**: N/A: 仕様の配置と検査の変更であり、製品の規範要素を変えない。
  - **Observed Failure**: `removed scenarios: REQ-AUTHENTICATION-026` を報告した。機能ノードの文書を読まず、移した規則が消えたと扱った。実装後は同じ作業ツリーで `changed scenarios: REQ-AUTHENTICATION-026` を報告し、本文を戻すと差分なしを報告した。
  - **Detection Reason**: 移動だけなら差分なし、本文の変更なら変更という二つの観測で、機能ノードを読まない実装と本文を比べない実装の両方を区別できる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/specification-doc.test.ts` の `reads a feature node one level below its context with the grammar its name gives`、`spec-diff.test.ts` の `reports a rule whose body changed even when its title and steps did not` と `reports nothing when a rule and its state machine move into a feature node`、`specification-rules.test.ts` の 9 件（空の実装に対して）、`brief.test.ts` の `returns a rule under a lifecycle section and stops at the next section`、`render.test.ts` の `nests a feature node and its documents under the context that owns it`、`repository-checks.acceptance.test.ts` の `reads the scenarios of a feature node as normative`。
  - **Requirement**: N/A: 仕様の配置と検査の変更であり、製品の規範要素を変えない。
  - **Observed Failure**: `documentKind` が `undefined` を返した。`spec-diff` が本文だけの変更を報告せず、状態機械の移動を `docs/domain/demo#Lifecycle` と `docs/domain/demo/task#Lifecycle` の変更として報告した。空の検査はどの違反も報告しなかった。`brief` は `### Rule:` の規則を抽出できなかった。文書サイトは機能ノードの題名から所有の接頭辞を外さなかった。文書検査は機能ノードを「not a canonical specification document」とし、さらにコンテキストの索引表に載っていないと報告した。
  - **Detection Reason**: 各テストは機能ノードの経路と規則の本文を入力にし、読まない実装、比べない実装、拒否しない実装のそれぞれで失敗する。
- **Change-Resistance Results**:
  次の誤実装を手で注入し、すべて検出された。変異ツールは Go だけを対象とするため、TypeScript の検査には手書きの障害を使った。
  - 機能ノードの段に空の名前集合を返す：`specification-doc.test.ts` と受け入れテストの 2 件が失敗し、実リポジトリの文書検査は 36 件の「not a canonical」を報告した。
  - `spec-diff` の事実から本文を落とす：`spec-diff.test.ts` の 1 件が失敗した。
  - 担保手段をすべて受理する：単体 3 件と受け入れ 1 件が失敗した。
  - 上位の規則のリンク解決を無視する：単体と受け入れの 2 件が失敗した。
  - 対応のない機能スライスを報告しない：単体と受け入れの 2 件が失敗した。
  - 節の順序の判定を外す、曖昧な語の判定を外す、節の下にない規則の判定を外す：それぞれ単体 1 件が失敗した。
  - セキュリティ統制の検査が機能ノードのシナリオを読まない（配線の除去）：実リポジトリで `ChangePassword` と `UnlinkExternalIdentity` の拒否が宣言されていないという R4 の失敗が出た。
  - work item のスキーマから機能ノードのパスを外す：実リポジトリの `mise run check-work-items` がスキーマ違反で失敗した。
  注入の途中で、受け入れテストのファイル全体が一度だけ短時間で一斉に失敗したが、同じ注入で再実行すると再現しなかった。環境による一過性の失敗として扱い、判定には使っていない。
- **Verification Results**:
  - `mise run test-tools` - 成功（663 件）
  - `mise run check-spec` - 成功
  - `mise run check-work-items` - 成功
  - `mise run spec-diff` - 規範の変更なし
  - `mise run verify` - 成功（初回は `SPECIFICATION_FORMAT.md` の「既定」の表記で用語検査が失敗し、「デフォルト」に直して再実行した）
