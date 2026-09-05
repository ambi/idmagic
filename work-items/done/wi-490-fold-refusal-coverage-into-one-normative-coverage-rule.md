---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: tooling
priority: p1
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 規範 id の網羅を数える仕組みと開発者向けの規範文書だけが変わり、製品の振る舞い、公開契約、運用手順のどれも変わらないため、リリース読者に伝える差分がない。
  references: []
spec_impact: { kind: none, reason: "規範 id の網羅を数える仕組みと、テストの規範を述べる文章の変更であり、製品の振る舞い、公開契約、TypeSpec のどれも変えない。R1、R2、R4 の判定結果は変更の前後で同一であることを検証で固定する。" }
initial_context:
  source:
    - tools/check/src/security-controls.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-specifications.ts
    - tools/check/src/report-coverage-debt.ts
    - tools/check/example-coverage-debt.json
    - tools/check/standards-coverage-debt.json
    - tools/check/README.md
    - docs/development/specification-first-workflow.md
    - mise.toml
  tests:
    - tools/check/src/security-controls.test.ts
    - tools/check/src/normative-coverage.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec/generated
---

# 拒否の特別扱いをやめ、規範 id の網羅を 1 つの規則、1 つの台帳、1 つの検査に畳む

## Motivation

[[wi-390-security-control-test-standard-and-gate]] は 4 つの規則を置いた。

R1 と R2 は、応答を書いたうえで `nil` を返す防護という欠陥の形を表現不可能にする構造的な検査であり、試作の時点で現存する同型の欠陥を 2 件見つけている。

これらは今も正しく、本 work item は触れない。

R3 と R4 は追跡可能性の仕組みで、インシデントの枠組みを引き継いで「セキュリティ上の拒否」の側に置かれた。

その特別扱いが何を支えているのかを測ったところ、**振り分け以外の何も支えていない**ことが分かった。

### 測定 (2026-09-05)

拒否かどうかの判定は `security-controls.ts` の `refusalSteps` が行い、`ALT` と `THEN` の段に対して 2 つの腕で照合する。

エラー型名 `\b[A-Z][A-Za-z0-9]*Error\b` と、15 語の日本語・英語の語彙である。

| 測定 | 値 |
|---|---|
| 生きたシナリオ | 305 |
| うち拒否を宣言していると判定されるもの | 184 (60%) |
| 判定の内訳 | エラー型名の腕 135 / 語彙のみの腕 49 |
| テストが id を引用していないシナリオ | 191 = 拒否 102 + 非拒否 89 |
| 未検証の割合 | 拒否 102/184 = 55% / 非拒否 89/121 = 74% |

判定器の消費点は 3 つしかない。

**R4 に対する寄与は測定上ゼロである。**

`declaredRefusalTypes` は拒否段からエラー型名を取り出すが、エラー型名を含む段は定義上つねにエラー型名の腕で拒否段になるため、語彙の腕はここに 1 つも型を足さない。

21 Context すべてで、語彙の腕を外しても R4 の入力は変わらず、拒否フィルタ自体を外して「どの段に現れるエラー型も宣言とみなす」形にしても、やはり変わらない。

**R3 の規則は、非拒否側の規則と同一である。**

`normative-coverage.ts` は冒頭で「The shape is `checkRefusalCoverage`'s ... a second style of the same comparison is a second place to go wrong」と述べており、同じ 3 方向の比較を 2 つの様式で実装している。

`accounted` と `accountedPath` は、2 つの台帳を単に別々ではなく互いに素に保つためだけに存在する。

**R1 と R2 は判定器を使わない。**

したがって、15 語の正規表現、`refusalScenarioIds`、重複した検査、専用の台帳、互いに素という不変条件からなる一式が支えているのは、**id がどちらの JSON ファイルに載るか**だけである。

### 振り分けだけのために払っている代償

判定器は語彙の照合なので、両方向に漏れる。

語彙のみの腕 49 件には、エラー型名を書いていない本物の拒否が含まれる。

`REQ-AUTHORIZATION-004` の「許可しない」、`REQ-AUTHORIZATION-005` の「拒否理由を添えて許可しない」、`REQ-APPLICATION-011` の「割り当てがないためフェデレーションを拒否する」、`REQ-SAML-002` の「フェイルクローズで拒否する」。

同じ 49 件に、拒否でないものも含まれる。

`REQ-AUTHORIZATION-009` は監査イベントの**フィールド一覧**に「拒否理由」という語があるためにマッチしている。

`REQ-SYSTEM-015` は「再認可から復旧できない → 再ログイン導線を提示する」の**条件節**にマッチしており、結果は何も拒否していない。

`REQ-AUDIT-003`、`REQ-CLAIMMAPPING-001`、`REQ-SIGNINGKEYS-002`、`REQ-AUTHENTICATION-031` は「含まれない」でマッチしているが、これらは拒否ではなく**出力の最小化**である。

### 特別扱いが優先度の装置としても働いていない

台帳を分ける根拠は「セキュリティ防護の拒否が未検証であることは、振る舞いが未検証であることより大きな事実だ」という主張である。

しかし判定されたのは 305 件中 184 件、60% であり、多数派を特別扱いとは呼べない。

さらに未検証の割合は拒否側 55%、非拒否側 74% で、**声を大きくした側のほうが状態が良い**。

分離が注意を向けた先は、より手当ての進んでいる半分だった。

加えて、拒否の台帳だけが各エントリーに理由を持たない。

`normative-coverage.ts` は「A list this size stops being read the moment its entries are indistinguishable from one another, and then it stops shrinking」と書いて理由欄を足したが、102 件を抱える拒否の台帳にはそれが無い。

## Scope

- 規範 id の網羅を 1 つの規則に統一する。「宣言された規範 id は、少なくとも 1 つのテストがその id を引用する」。
- `security-refusal-debt.json` の 102 件を `example-coverage-debt.json` へ統合し、各エントリーに理由を付ける。台帳はラチェットのまま、縮むことしか許さない。
- `checkRefusalCoverage` を廃し、`checkNormativeCoverage` に一本化する。`accounted` と `accountedPath` による互いに素の維持を削除する。
- `refusalScenarioIds` と `REFUSAL_WORDS` を削除する。
- `declaredRefusalTypes` を「シナリオの段に現れるエラー型名すべて」に変え、拒否の述語を外す。R4 の判定結果は変わらない。
- 「声の大きさ」を保管された事実ではなく**導出される属性**にする。報告タスクが TypeSpec から、その id を持つ Context が状態変更操作に 403 を宣言しているかを引き、並べ替えと絞り込みに使う。
- `docs/development/specification-first-workflow.md` の「Testing a refusal」を、拒否に限らない形へ書き換える。
- 本 work item より前に作成された Context ごとの 18 件について、参照する台帳のパスと検査名を機械的に更新する。

## Out of Scope

- R1 と R2 の変更。
  欠陥の形を表現不可能にする構造的な検査であり、本件の議論はこれに一切触れない。
- R4 の判定単位を操作ごとへ細かくすること。
  [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 台帳の粒度を `ALT` の分岐単位へ下げること。
  シナリオ書式に分岐の住所を導入する変更であり、独立に扱う。
- シナリオと TypeSpec 操作を結ぶ参照の導入。
  本件の統合はこれを必要としない。導入すれば R4 が操作単位になり、状態変更かどうかも導出できるが、書式変更の規模が別物である。
- 「副作用の不在」を機械検査の門にすること。
  wi-390 が意図的に見送った判断であり、ここで覆さない。
- 台帳に載っている 325 件を減らす作業そのもの。
  Context ごとの work item が持つ。

## Design

### 統合後の形

規則は 1 つになる。

宣言された規範 id は、少なくとも 1 つのテストがその id を引用する。

引用が無い id は台帳に理由付きで載っていなければならず、載っている id が引用を得たら台帳から外さなければならない。

対象はシナリオと `standards.md` の行の両方で、拒否かどうかは問わない。

保管するのは id と理由だけにする。

その id が拒否を宣言しているか、状態を変える操作に属するか、どの Context とパッケージが持つかは、すべて報告時に導出する。

属性を保管すると、属性の判定が変わったときに台帳の内容が変わり、ラチェットが「縮むだけ」でなくなる。

導出にすれば、判定を直しても台帳は動かない。

### 「声の大きさ」は導出する

分離をやめても、拒否が未検証であることの重さは失われない。

報告タスクが TypeSpec から、その Context が状態変更操作に対して宣言している 403 のエラー型を引き、シナリオがその型を名指しているかで重みを付ける。

これは語彙ではなく契約から導かれるので、日本語の書きぶりに依存しない。

`REQ-SYSTEM-015` の条件節や `REQ-AUTHORIZATION-009` のフィールド一覧が重みを得ることはない。

この重みは意図的に狭く、過小に報告する。

非 GET 操作の 403 以外で拒否する防護は届かないので、WorkloadIdentity のアテステーション拒否、DataKeys のフェイルクローズ、HTTP の境界を持たない Seeding は、いずれも重み 0 になる。

wi-390 が書かれた動機そのものの拒否が 0 と出るのは弱点だが、広げるなら契約が述べる内容を広げるべきで、スクリプトの推測を広げるべきではない。

### R4 は述語を外しても同じ結果になる

`declaredRefusalTypes` から拒否の述語を外し、シナリオの段に現れるエラー型名をすべて宣言とみなす。

21 Context で測ったところ、結果の集合は完全に一致した。

述語は R4 に対して何もしていないので、外すのは意味の変更ではなく重複の削除である。

この一致は変更の前後で検査し、Verification に固定する。

### 規範の文章を拒否から一般化する

`docs/development/specification-first-workflow.md` の「Testing a refusal」は、規則の中で唯一、機械検査されていないぶん実際に効いている部分である。

これを拒否に限定しているのが、そもそもの誤りだと考える。

効いている性質は「**観測される応答が、効果を含意しない段**」であり、拒否はその最大かつ最も危険な部分集合にすぎない。

成功経路では応答が効果から導出されるので、応答を assert すれば効果も推移的に assert される。作られた行が本体として返り、発行されたトークンが実際に通る。

拒否経路ではそうならない。403 はガードが別の分岐で書くもので、効果の経路とは独立に生成される。

同じ形は拒否の外にもある。

[[wi-470-record-control-plane-state-changes-in-the-audit-log]] のクォータ更新は、監査イベントを 1 つも発行しないまま 200 を返していた。

200 は発行の有無から独立に生成されるので、応答は何も語らない。

これは拒否ではないため、どの台帳にも現れなかった。

したがって節を「Testing a step whose response does not entail its effect」として書き直し、拒否を先頭の例、監査イベントの欠落を 2 番目の例として置く。

### 統合の前に測って解決した 2 つの問い

2 つの検査は同じ規則を実装しているが、入力の取り方が違う。

統合すると債務の id が別の入力に晒されるため、着手前に差分を測った。

**引用の探索範囲。**

`check-security-controls` は `backend/**/_test.go` だけを読み、`REQ-[A-Z0-9]+-\d+` で引用を数える。

`check-specifications` は `backend` と `frontend` の両方から `_test.go`、`*.test.tsx?`、`*.spec.tsx?` を読み、宣言済み id そのものを模様として照合する。

統合後は後者が債務の 102 件にも適用される。

測ったところ、102 件を名指すテストは backend の Go テスト、backend の TypeScript テスト、frontend のテストのいずれにも **0 件**だった。

したがって統合によって「引用が増えたので台帳から外れる」id は発生しない。

これは重要な確認である。もし発生していれば、効果を確かめるテストを書かないまま id が台帳から消え、Context ごとの 18 件が禁じている注記だけの解消と同じ結果になっていた。

**廃止されたシナリオの扱い。**

`check-specifications` は `(superseded by REQ-...)` の見出しを宣言から外すが、`refusalScenarioIds` は外さない。

廃止済みのシナリオは `REQ-SHAREDSIGNALS-002` と `REQ-IDMANAGEMENT-012` の 2 件で、どちらも債務の 102 件に含まれない。

統合によって「宣言が無くなったので台帳から外せ」と言われる id は発生しない。

なお `REQ-IDMANAGEMENT-012` は拒否を宣言しており、今日は R3 の対象に入っているが、統合後は廃止済みとして対象から外れる。

引退した振る舞いにテストを求めないのは `check-specifications` 側の判断が正しいので、この差分は受け入れる。

### 却下した案

**分離を保ったまま判定器を直す案。**

条件節を除く、フィールド一覧を除く、といった手当てを重ねても、語彙の照合であることは変わらない。

そして測定によれば、判定器が支えているのは振り分けだけである。

直す価値のある出力を持たない判定器を直すのは、複雑さを保存したまま作業を増やす。

**シナリオと TypeSpec 操作を結ぶ参照を先に入れる案。**

それが最終的な姿だとは考える。状態変更かどうかも、契約が約束する拒否も、分岐の住所も、すべてそこから導出できる。

しかし本件の統合はそれを必要としない。

大きい設計を小さい整理の前提にすると、整理が止まる。

参照の導入は独立に判断する。

**台帳を `ALT` 単位へ下げることを同時に行う案。**

粒度の問題は実在する。`REQ-OAUTH2-040` は 6 分岐、`REQ-OAUTH2-036` は 5 分岐を持ち、id を引用したテスト 1 本ですべてが充足済みになる。最大は 10 分岐である。

ただしこれはシナリオ書式に分岐の住所を導入する変更であり、統合とは独立に成立する。

同時に行うと、どちらの是非も測れなくなる。

**3 つの台帳のうち `standards-coverage-debt.json` だけ残す案。**

`standards.md` の行とシナリオは宣言の場所が違うだけで、規則は同じである。

分ける理由は無い。

### 着手前に定めた RED

**Acceptance RED。**

台帳を統合したうえで検査を変えないまま `mise run check-spec` を走らせると、`accounted` の照合によって 102 件の「already listed in security-refusal-debt.json. Keep the two lists disjoint.」が出る。

これは互いに素という不変条件が実在し、統合を妨げていることを観測する形であり、`accounted` を外した後に消えることで統合の完了を示す。

**Unit RED。**

`security-controls.test.ts` から `errorTypesNamedByScenarios` を import するテストは、関数が存在しない時点で失敗する。

拒否の述語を外した抽出が、拒否の語彙を 1 つも含まない段のエラー型も返すことを、この単体テストで固定する。

R4 の判定そのものには RED が存在しない。

述語を外しても結果が変わらないことが本件の主張なので、そこは RED ではなく**基準との一致**で確かめる。

## Plan

1. 変更前の基準を取る。R1、R2、R4 の findings と、`report-refusal-debt` の分類を記録する。
2. `security-refusal-debt.json` の 102 件へ理由を付けて `example-coverage-debt.json` へ統合し、統合後の件数が 191 であることを確認する。
3. `checkRefusalCoverage` の呼び出しを `checkNormativeCoverage` へ寄せ、`accounted` の受け渡しを削除する。
4. `refusalScenarioIds` と `REFUSAL_WORDS` を削除し、`declaredRefusalTypes` から拒否の述語を外す。
5. R4 の findings が基準と一致することを確認する。一致しなければ統合を止めて原因を記録する。
6. 報告タスクを、契約由来の重み付けで並べ替える形へ書き換える。
7. `specification-first-workflow.md` の節を一般化し、拒否と監査イベントの 2 例を置く。
8. Context ごとの 18 件について、台帳のパスと検査名の参照を更新する。
9. `tools/check/README.md` の 3 つのラチェットの表を書き直す。

## Tasks

- [x] T001 [Baseline] 変更前の R1、R2、R4 の findings と分類を記録する。
- [x] T002 [Tooling] 102 件へ理由を付けて台帳を統合し、ラチェットが縮むだけであることを確認する。
- [x] T003 [Tooling] `checkRefusalCoverage` と `accounted` を削除し、網羅の検査を 1 つに寄せる。
- [x] T004 [Tooling] `refusalScenarioIds` と `REFUSAL_WORDS` を削除し、`declaredRefusalTypes` の述語を外す。
- [x] T005 [Verify] R4 の findings が基準と完全に一致することを確認する。
- [x] T006 [Tooling] 報告タスクを契約由来の重み付けへ書き換える。
- [x] T007 [Docs] 「Testing a refusal」を、応答が効果を含意しない段の規範として書き直す。
- [x] T008 [Docs] `tools/check/README.md` と Context ごとの 18 件の参照を更新する。

## Verification

- `mise run check-security-controls`
- `mise run check-spec`
- `mise run test-tools`
- `mise run check-links`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

これは検査を緩める方向に見える変更である。

実際には規則の数は変わらず、同じ規則の 2 つ目の実装が消えるだけだが、取り違えれば強制が静かに失われる。

そのため R4 の findings の一致を検証に固定し、統合後の台帳が 191 件であることを数で確認する。

一致しない場合は統合を止める。

`REFUSAL_WORDS` を削除すると、語彙のみで検出されていた 49 件は「拒否である」という属性を失う。

失うのは属性であって台帳上の地位ではない。49 件は id として台帳に残り、同じ規則で同じように扱われる。

ただし報告の重み付けは契約由来になるため、エラー型名を書いていない本物の拒否 — `REQ-AUTHORIZATION-004` の「許可しない」など — は重みを得られない可能性がある。

これは規範の側でエラー型名を書けば解決するが、書式を検査の都合で変えることになるため、統合の時点では重みの欠落として記録するに留め、別途判断する。

Context ごとの 18 件は台帳から行を削除するだけなので、統合が先でも後でも作業は成立する。

`depends_on` は置かない。18 件を tooling の変更の後ろに直列化する利得が無いためである。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の差分は無い。
  変わったのは網羅を数える仕組みで、規則は 1 つ、台帳は 2 つ (シナリオ 191 件、standards 134 件)、
  網羅の検査は `checkNormativeCoverage` 1 つになった。`security-refusal-debt.json`、
  `checkRefusalCoverage`、`refusalScenarioIds`、15 語の `REFUSAL_WORDS`、`accounted` による
  互いに素の不変条件を削除し、`declaredRefusalTypes` は述語を外して
  `errorTypesNamedByScenarios` になった。R1、R2、R4 は残り、判定は変わらない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec` (統合した台帳に対し、検査を変えないまま実行)。
  - **Requirement**: N/A: 規範 id を数える仕組みの変更であり、製品の規範要求を持たない。
  - **Observed Failure**: 102 件の `REQ-... is already listed in tools/check/security-refusal-debt.json. Keep the two lists disjoint.`
  - **Detection Reason**: 互いに素の不変条件が実在し、統合を妨げていることをその場で示す。
    `accounted` の受け渡しを外した後にこの 102 件が消え、他の findings が増えないことで、
    削除したのが不変条件だけであることが分かる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/security-controls.test.ts` の `errorTypesNamedByScenarios` の 4 件。
  - **Requirement**: N/A: 規範 id を数える仕組みの変更であり、製品の規範要求を持たない。
  - **Observed Failure**: `SyntaxError: Export named 'errorTypesNamedByScenarios' not found in module 'security-controls.ts'`
  - **Detection Reason**: 4 件のうち 1 件は、拒否の語彙を 1 つも含まない段
    (`THEN JobClaimExpiredError is returned to the caller`) がエラー型を宣言することを固定する。
    述語を戻せばこのテストが落ちるので、拒否の判定に依存しない抽出であることが assert される。
- **Change-Resistance Results**:
  3 件の障害注入をいずれも検出した。
  (1) テストが無い `REQ-OAUTH2-001` を台帳から削除 → `check-spec` が
  `is declared, but no test names it` を報告。
  (2) テストが名指ししている `REQ-AUDIT-001` を台帳へ追加 → `check-spec` が
  `now has a test that names it. Remove it from the list; the list only shrinks` を報告。
  ラチェットが両方向に効いている。
  (3) `jobs` のシナリオから `AccessDeniedError` の言及をすべて除去 → R4 が
  `CancelJob answer 403 with AccessDeniedError, but no scenario declares that refusal` を報告。
  なお `audit` で同じ注入を行っても R4 は落ちない。audit の状態変更操作が 403 を約束していないためで、
  R4 が契約側から判定していることの裏返しである。
  R4 の入力そのものは、21 Context の `{declared, promised}` を変更前に保存し、変更後の再計算と
  文字列として完全一致することを確認した (73 declared error types / 17 promised types)。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-security-controls` - passed
    (`17 refusal(s) promised by a 403 on a state change, 73 error type(s) named by the scenarios`)
  - `mise run check-spec` - passed (`154 standard(s), 306 scenario(s), 134 id(s) named by a test`)
  - `mise run test-tools` - passed (379 tests)
  - `mise run check-links` - passed (680 documents)

### Left Undone

導出した重みは意図的に狭く、実測で 191 件中 21 件にしか付かない。

非 GET 操作の 403 だけを見るため、WorkloadIdentity のアテステーション拒否、DataKeys のフェイルクローズ、
HTTP の境界を持たない Seeding は重み 0 と出る。

これらは wi-390 が書かれた動機そのものの拒否であり、重み付けとしては明確に不足している。

同じ理由で、エラー型名を書かずに「許可しない」とだけ書いている `REQ-AUTHORIZATION-004` と
`REQ-AUTHORIZATION-005` も重みを得ない。

広げる方向は 2 つある。契約側で 403 以外の拒否も宣言させるか、シナリオと TypeSpec 操作を結ぶ参照を入れて
操作単位で判定するかで、後者は R4 の粒度の問題と同じ根を持つ。どちらもこの項目では判断しない。
