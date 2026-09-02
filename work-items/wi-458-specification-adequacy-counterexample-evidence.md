---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-02
priority: p1
depends_on:
  - wi-456-retroactive-primary-use-case-evidence
change_kind: tooling
spec_impact: { kind: none, reason: "実装前に仕様の未決定を洗い出す開発時の証拠契約とその検査を追加するだけで、製品の振る舞い、公開契約、配備構成は変更しない。試行で見つかった個別の仕様欠陥は別の work item が是正する。" }
---

# 変更した規範要素ごとに反例を記録し、仕様の未決定を明示する

## Motivation

`risk-based-v3` の証拠契約は、実装が仕様どおりであることを強く検査する。RED、故障注入、変更抵抗性はいずれも「意図した」だけでは満たせない観測であり、この点でこのリポジトリは十分に強い。しかし、それらが答えている問いは一つだけである。**書かれた仕様に対して、実装が正しいか**。

書かれていない仕様に対しては、何も問うていない。`mise run check-spec` は表の形と ID の解決を検査し、`wi-418` はシナリオと規範に対応するテストの存在を検査する予定だが、`wi-418` 自身が Out of Scope で「必要な `REQ` が足りないことの検出」を対象外と明記している。結果として、次の欠陥はどの検査も通過する。

- 仕様が振る舞いを決めていない。したがって、どの実装も仕様に反しない。
- 仕様が二通りに読める。実装は片方を選び、テストは同じ読み方から書かれ、両者は一致する。
- 規範文書どうしが矛盾する。`states.md` の遷移表が禁じる遷移を `scenarios.md` の `ALT` が許す、といった形で、どちらも単独では検査を通る。

これは仮説ではない。`REQ-OAUTH2-009` は PAR の `request_uri` について `THEN その PAR レコードの状態は "Used"` までを書いているが、**同じ `request_uri` を二度目に提示したときに何が起きるか**を書いていない。一度限りの使用は `docs/contexts/oauth2/states.md` の状態表と `docs/contexts/oauth2/internals.md` に書かれており、実装も RFC 9126 に従っている。しかし `SPECIFICATION_FORMAT.md` §6 は「セキュリティ制御が責任を負う拒否は観測可能な振る舞いであり、シナリオに書く」と定めている。二度目の提示を受理してもなお `REQ-OAUTH2-009` に反しない実装が書けるのだから、この規範要素は一度限りの使用を規定していない。`wi-418` の被覆ゲートを入れても、名指しするテストが 1 件あれば通ってしまう。

不足しているのは検査の厳しさではなく、**実装前に一度だけ立てる問い**である。

> いま書いた仕様を満たしたまま、なお誤っている実装が書けるか。

書けるなら、仕様が足りない。書けないなら、その根拠となる規範要素を名指しできる。この問いへの答えは、意図では満たせない観測であり、`risk-based-v3` が RED と故障注入に求めたのと同じ性質を持つ。いま work item に残るのは実装後の検出能力の証拠だけで、実装前に仕様が何を決めていなかったかは、決めなかったという事実ごと残らない。

## Scope

- `docs/development/specification-first-workflow.md` に、仕様変更と Acceptance RED の間に位置する段階として「仕様の妥当性」を追加する。探索の観点（未定義の振る舞い、多義性、文書間の矛盾、境界、到達不能または不正な状態、並行性、再試行、部分障害、セキュリティ上の拒否）を、埋める欄ではなく問いとして示す。
- work item の frontmatter に `specification_adequacy` を追加する。`affected_spec` が名指しする規範要素ごとに、反例、判定、判定の根拠となる解決先を 1 件以上記録する。
- 判定を `strengthened` / `undetermined` / `refuted` の閉じた集合とし、それぞれの解決先が何を指すかを固定する。未知の値、空の解決先、解決しない参照は fail-closed で拒否する。
- 意図して決めない選択の置き場所を、当該 Context の `decisions.md` とする。新しい正本の種類も新しいファイル名も作らない。理由と、再検討の条件を伴う既存の決定の形をそのまま使う。
- 発見した不変条件を既存の所有者へ振り分ける規則を書く。一意性と参照整合性はスキーマ、観測可能な性質は `scenarios.md`、構築と事後条件は `docs/design-rules.md` に従い型または操作。`decisions.md` に不変条件を列挙しないという `SPECIFICATION_FORMAT.md` §3 の規則は維持する。
- `evidence_policy` を `risk-based-v4` へ上げる。完了済みの v1 / v2 / v3 の記録は履歴として再解釈しない。採用時点で `in_progress` の該当項目は新しい計画を追加してから完了できる。
- `tools/check/src/specification-adequacy.ts` と単体検査を追加し、`tools/check/schemas/work-item.schema.json` と `tools/workspace/src/check-workspace.ts` へ接続して `mise run check-work-items` のゲートにする。
- 完了済みの実際の変更 3 件へ遡って本段階を試行し、反例が出たか、出た反例が既存の規範要素で棄却できたか、記録に要した時間を実測する。試行結果は Design へ追記する。
- 被覆を報告するあらゆる出力について、それが**宣言済みモデルの被覆**であって実世界の網羅ではないことを、出力自身が述べる規則を書く。`docs/threat-model.md` が「一覧は網羅的ではない」と述べるのと同じ扱いを、シナリオと規範の被覆へ広げる。

## Out of Scope

- 試行で見つかった個別の仕様欠陥の是正。`REQ-OAUTH2-009` への `ALT` 追加を含め、規範要素を変える作業は規範参照を持つ別の work item が扱う。本項目は欠陥を検出して起票するところまでを対象とする。
- シナリオと規範に対応するテストの存在を検査する被覆ゲート。`wi-418` が扱う。本項目はその逆方向、すなわち規範要素が振る舞いを決めているかどうかだけを対象とする。
- 値ベースの property testing とメタモルフィック関係（`wi-420`）、設計時のモデル検査と決定的シミュレーション（`wi-421`）、変異試験基盤（`wi-457`）。いずれも実装とテストの側から仕様を検証する手段であり、本項目は仕様そのものの未決定を対象とする。反例を性質へ一般化する作業は `wi-420` へ渡す。
- 反例の文章の質を機械が判定すること。検査できるのは判定値の集合、解決先の解決可能性、記録の欠落だけである。反例が本当に妥当かどうかは引き続きレビューの判断とする。
- 形式仕様言語の導入。反例は自然言語で 1 文書けば足り、書けないなら仕様が読めていないという指標として扱う。
- `change_kind` が `refactor`、`docs`、`maintenance` で `spec_impact: none` の項目へ本契約を課すこと。規範要素を変えていない変更には答えるべき問いがない。
- 完了済み work item への遡及適用。`wi-456` が `risk-based-v3` の遡及を扱っており、本項目の遡及可否はその実測結果を見てから別途判断する。

## Design

### 対象母集団

`affected_spec` を持つ項目、すなわち `feature`、`bugfix`、`operations`、および `standards.md` の規範を参照する任意の `change_kind` を対象とする。`primary_use_cases` が使う母集団の判定をそのまま再利用し、第二の母集団定義を作らない。`spec_impact: { kind: none }` の項目は対象外である。

対象は `affected_spec` の各要素であって、変更行ではない。退役して後継へ差し替えた見出しは対象から除く。後継が同じ問いに答えるからである。

### 記録の形

```yaml
specification_adequacy: # 着手後に必須。affected_spec の各要素へ 1 件以上
  - element: REQ-OAUTH2-009
    counterexample: 使用済みの request_uri を二度目に受理して認可コードを再発行する実装も、この scenario に反しない。
    disposition: strengthened
    resolution: docs/contexts/oauth2/scenarios.md#REQ-OAUTH2-009
  - element: Product.OAuth2.Operations.PushedAuthorizationRequest
    counterexample: expires_in を 600 ちょうどで返し続ける実装も、TypeSpec の制約に反しない。
    disposition: refuted
    resolution: docs/contexts/oauth2/states.md#par-request-uri
```

`element` は `affected_spec` が名指しした規範 ID、標準 ID、または TypeSpec シンボルと一致する。`counterexample` は「仕様を満たしたまま誤っている実装」を 1 文で述べる。`disposition` と `resolution` の対応は次のとおり固定する。

| disposition | 意味 | resolution が指すもの |
|---|---|---|
| `strengthened` | 反例を排除するため規範要素を変更した | 変更後の規範要素。完了時の Summary に同じ要素が現れること |
| `undetermined` | 意図して決めないと判断した | 当該 Context の `decisions.md` の見出し。理由と再検討の条件を持つこと |
| `refuted` | 反例は既存の規範要素が既に排除していた | その規範要素の ID または見出し。解決すること |

`refuted` に根拠の名指しを求めるのが、この設計の要である。根拠なしの `refuted` を許すと、契約全体が「考えました」の一語へ縮む。名指しされた ID の解決は機械が確かめられるので、この一点だけは意図では満たせない。

### 判定の状態と遷移

一つの規範要素に対する妥当性判定の状態機械。

| State | Kind | Meaning |
|---|---|---|
| Unexamined | initial | 反例を立てていない。着手時の既定 |
| Strengthened | terminal | 反例を排除するよう規範要素を変更した |
| Undetermined | terminal | 反例を残すと決め、決定と再検討の条件を記録した |
| Refuted | terminal | 反例が既存の規範要素で排除されることを確認した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Unexamined | 反例が書けた | 規範要素を変更する | Strengthened | 規範要素を変更し、完了 Summary へ現れる |
| Unexamined | 反例が書けた | 決めないと判断する | Undetermined | `decisions.md` へ理由と再検討の条件を追加する |
| Unexamined | 反例が書けなかった | 排除する規範要素を名指しできる | Refuted | 名指しした ID を `resolution` へ記録する |
| Unexamined | 反例が書けなかった | 排除する規範要素を名指しできない | Unexamined | 検査が完了を拒否する |
| Strengthened | 変更後の要素へ再び反例が書けた | — | Unexamined | 同じ要素へ 2 件目の記録を追加する |

最後の行は反復を許すためにある。1 回の反例で仕様が十分になるとは限らず、記録は要素あたり 1 件以上であって 1 件ちょうどではない。

`Unexamined` のまま完了できないこと、および終端が 3 つあることが、この機械の全体である。「検討したが該当なし」に相当する第 4 の終端は作らない。それは `refuted` から根拠を抜いたものであり、契約を無効にする。

### 具体例

実在する要素で 3 件挙げる。網羅ではなく、判定の 3 値がそれぞれどう決まるかを示すためのものである。

**`REQ-OAUTH2-009`（`strengthened` になる例）。** 反例は Motivation に述べたとおり、使用済みの `request_uri` を二度目に受理する実装である。`states.md` の状態表と `internals.md` は一度限りの使用を述べているが、`SPECIFICATION_FORMAT.md` §6 はセキュリティ制御の拒否をシナリオに書くよう求めている。したがって `refuted` にはならず、`ALT` の追加、すなわち `strengthened` が正しい判定である。是正そのものは別項目へ分ける。

**`REQ-JOBS-003`（`refuted` になる例）。** 反例の候補は「`dedup_key` を見ずに毎回通知する実装」だが、これは `THEN ハンドラーは dedup_key を用いて冪等に判定し、重複した通知を送らない` が直接排除している。`resolution` はその `THEN` を持つシナリオ ID 自身になる。反例が立たない要素が存在すること自体は健全であり、契約は反例の存在を強制しない。強制するのは、立たなかったときに根拠を名指しすることである。

**並行性の例（`undetermined` になる例）。** リースの期限切れと完了報告が同時に起きたときの順序は、`docs/contexts/jobs/decisions.md` が「停止時の回復は明示的な再投入ではなくリースの自然な期限切れに委ねる」と決めており、その帰結として一部の順序を規定しないままにしている。反例は書けるが、規定しないことが決定である。この形が `undetermined` であり、`decisions.md` に再検討の条件を伴って残る。work item の Out of Scope に書いて完了ファイルへ移動させると、仕様を読む人には二度と見えない。

### 性質と反証

記録の集合について、実装前に次を成り立たせる。

- **完全性**：`affected_spec` の各要素に、少なくとも 1 件の記録がある。
- **解決可能性**：すべての `resolution` が解決する。`undetermined` の解決先は `decisions.md` の見出しに限る。
- **非空性**：`counterexample` は空でない。

これらへの反証を試みると、次の穴が残る。完全性は、要素ごとに 1 件あることしか言わない。無内容な 1 文と正しい ID を書けば通る。解決可能性は、名指しされた要素が**本当に**その反例を排除するかを見ない。この 2 つは機械では閉じられないので、閉じられないことを文書へ書く。`risk-based-v3` が「検査はソースの文面から表明の質を推論しない。故障注入の観測が検出能力の証拠である」と書いているのと同じ扱いである。

一方で、`undetermined` の解決先を `decisions.md` に限る規則は、逃げ道の一つを実際に塞ぐ。最も安易な回避は「決めないことにした」と書いて work item の中で完結させることだが、`decisions.md` の見出しは正本に残り、レビューと `check-spec` の対象になる。

### 却下した代替案

**観点ごとのチェックリストを frontmatter に持たせる案。** 多義性、境界、並行性、部分障害などを真偽値の欄にする。却下する。`docs/development/specification-first-workflow.md` §4 が「承認欄は考えずに署名されるが、下の検査は観測でしか満たせない」と述べており、観点の真偽値はまさに前者である。加えて `SPECIFICATION_FORMAT.md` §3 が、`Invariants` や `Concurrency` のような観点名の見出しは「埋める箱として読まれ、該当しない観点に散文をでっち上げるか、一つの決定を複数へ割る」と警告している。観点は問いとして workflow 文書に置き、記録は反例という成果物だけにする。

**`spec-diff` との突き合わせを機械で行う案。** `strengthened` の記録に対し、その要素が `mise run spec-diff` の出力に現れることを検査する。却下する。完了時点の diff の基準点が記録されておらず、基準点を新たな必須項目にすると、遅れて完了した項目や rebase 後の項目で偽陽性を出す。代わりに、完了 Summary が既に `spec-diff` を読んで書かれる契約になっているので、`strengthened` の要素が Summary に現れることを検査する。弱いが、偽陽性でゲートを壊さない。

**新しい正本ファイル（`open-questions.md` など）を作る案。** 却下する。`SPECIFICATION_FORMAT.md` §1 の正本ファイル名は閉じた集合であり、未決定は理由と再検討の条件を持つ決定として `decisions.md` が既に扱える。第二の置き場所を作れば、決定と未決定が別ファイルに分かれて同じ監査を二度行うことになる。

**`risk-based-v3` を据え置いて欄だけ追加する案。** 却下する。`WORK_ITEM_FORMAT.md` は完了済みの vN 記録を再解釈しないと定めており、v3 の意味を後から変えるとその約束を破る。版を上げるのが既存の設計に沿う。

## Plan

1. 反例が実際に見つかるかを先に測る。完了済みの変更 3 件（規範シナリオの追加、`standards.md` の行の追加、状態遷移の変更を 1 件ずつ）へ本段階を手作業で適用し、要素あたりの所要時間、反例の有無、判定の内訳を記録して Design へ追記する。反例が 1 件も出ないなら契約を導入しない。
2. `specification_adequacy` の型、判定の閉集合、解決先の規則を確定し、不正入力（未知の判定値、空の反例、解決しない参照、`decisions.md` 以外を指す `undetermined`、要素の欠落、`affected_spec` に無い要素）を RED で固定する。
3. `tools/check/src/specification-adequacy.ts` を実装し、`primary-use-case-evidence.ts` の構成に合わせて純粋な判定と読み取りを分離する。
4. `tools/check/schemas/work-item.schema.json` と `tools/workspace/src/check-workspace.ts` へ接続し、`mise run check-work-items` から実行できるようにする。
5. `WORK_ITEM_FORMAT.md`、`docs/development/specification-first-workflow.md` §3 のループ表と §4、`SPECIFICATION_FORMAT.md` §6 の未決定の置き場所を更新する。`new-work-item` と `implement-work-item` の skill を同じ内容へ揃える。
6. `evidence_policy: risk-based-v4` を定義し、採用時点で `in_progress` の該当項目の移行規則を書く。
7. 試行で見つかった仕様欠陥を、規範参照を持つ個別の work item として起票する。

未解決の問いのうち、着手前に決めるべきものは 1 つだけである。手順 1 の試行で反例が 1 件も出なかった場合に、契約を導入せず本項目を却下として `done/` へ移すのか、対象母集団を狭めて再試行するのか。前者を既定とする。ゲートを増やす前に、そのゲートが何かを止めることを確かめる。

## Tasks

- [ ] T001 [Pilot] 完了済みの変更 3 件へ本段階を手作業で適用し、所要時間、反例の有無、判定の内訳を Design へ追記する。
- [ ] T002 [Design] `specification_adequacy` の型、判定の閉集合、解決先の規則を確定し、不正入力を列挙する。
- [ ] T003 [Core] 判定検査を純粋操作として実装し、各不正入力を RED で固定してから GREEN にする。
- [ ] T004 [Tooling] JSON Schema と `check-workspace.ts` へ接続し、`mise run check-work-items` のゲートにする。
- [ ] T005 [Doc] `WORK_ITEM_FORMAT.md`、`specification-first-workflow.md`、`SPECIFICATION_FORMAT.md` を更新する。
- [ ] T006 [Doc] 被覆の出力が「宣言済みモデルの被覆」であることを述べる規則を追加する。
- [ ] T007 [Policy] `risk-based-v4` を定義し、`in_progress` 項目の移行規則と v1〜v3 の非再解釈を明記する。
- [ ] T008 [Skill] `new-work-item` と `implement-work-item` の skill を更新する。
- [ ] T009 [Followup] 試行で見つかった仕様欠陥を個別の work item として起票する。
- [ ] T010 [Verify] 標準検証を通し、既存の pending 項目が新しいゲートで壊れないことを確認する。

## Verification

- `mise run test-tools`
- `mise run check-work-items`
- `mise run check-spec`
- `mise run verify`
- `affected_spec` に要素があり `specification_adequacy` が空の `in_progress` 項目を拒否する。
- 未知の `disposition`、空の `counterexample`、解決しない `resolution` を拒否する。
- `undetermined` で `decisions.md` 以外を指す `resolution` を拒否する。
- `refuted` で `resolution` を省いた記録を拒否する。
- `strengthened` の要素が完了 Summary に現れない記録を拒否する。
- 退役して後継へ差し替えた見出しを対象から除く。
- `spec_impact: { kind: none }` の項目へは本契約を課さない。
- 完了済みの `risk-based-v1` / `v2` / `v3` の記録が、新しい検査で失敗しない。
- 既存の pending 項目が、着手前の段階では新しい検査で失敗しない。

## Risk Notes

リスクは medium。誤ると製品ではなく開発の側が壊れる。最も可能性の高い失敗は、契約が儀式になることである。要素ごとに 1 文を書けば通る検査は、無内容な 1 文を量産する誘因になる。これを完全には防げないので、手順 1 の試行で反例が実際に出ることを確かめてから導入し、出ないなら導入しない。導入後も、`refuted` の割合が上がり続けるなら契約が形骸化した指標として扱う。

第二の失敗は、`undetermined` が debt の置き場になることである。`docs/threat-model.md` が「再検討の条件を持たない受容は、ラベルを貼っただけの放置である」と述べているのと同じ危険があるため、`undetermined` の解決先を `decisions.md` に限り、理由と再検討の条件を伴う既存の決定の形に載せる。それでも条件の質は機械では測れない。

第三に、`evidence_policy` の版を上げると、採用時点で `in_progress` の項目すべてに移行作業が生じる。`wi-456` が `risk-based-v3` の遡及を進めている最中に版を上げると二重の移行になるため、`depends_on` で順序を固定する。

`reversibility` は irreversible とする。検査は取り外せるが、`risk-based-v4` という版の名前と、完了記録に残る反例は取り消せない。

最後に、本項目は仕様の完全性を保証しない。保証するのは、**宣言済みモデルに対して反例を一度探したという記録**だけである。モデルそのものに含まれていない振る舞い——誰も状態として書かなかった状態、誰も要素として書かなかった規範——は、この段階を通しても見つからない。`docs/threat-model.md` が脅威の一覧を網羅的でないと明示しているのと同じ限界であり、同じ形で文書へ書く。
