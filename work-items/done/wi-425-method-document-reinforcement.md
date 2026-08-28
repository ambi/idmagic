---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-27
priority: p2
change_kind: docs
evidence_policy: risk-based-v2
spec_impact: { kind: none, reason: "方法論文書（docs/development/specification-first-workflow.md、docs/development/testing.md、WORK_ITEM_FORMAT.md）の補強であり、製品の振る舞いと外部契約を変えない。" }
initial_context:
  source:
    - docs/development/specification-first-workflow.md
    - docs/development/testing.md
    - docs/development/README.md
    - WORK_ITEM_FORMAT.md
    - tools/check/schemas/work-item.schema.json
  tests:
    - tools/check/src/lib.test.ts
    - frontend/src
  stop_before_reading:
    - spec
    - docs/contexts
---

# 方法論文書に、判断の可逆性・テストダブルの方針・記録の根拠を足す

## Motivation

`docs/development/specification-first-workflow.md` と `WORK_ITEM_FORMAT.md` は、いくつかの点で規律が非常に厚い一方、隣接する判断を読み手の裁量に委ねたままにしている。前 2 つの分析で見つかった、方法論文書の側に属する欠落をここにまとめる。

第一に、`risk` が結果の重大さという一軸しか持たない。低・中・高・重大は「破れたときにどれだけ困るか」を表すが、「元に戻せるか」を表さない。現実には、重大だが元に戻せる判断（レプリカ構成、キャッシュ方針、UI の配置）と、軽微だが元に戻せない判断（ワイヤ形式、識別子の意味、公開スキーマ、鍵の破棄、`REQ` 番号の割り当て）があり、後者のほうが慎重さを要する。`SPECIFICATION_FORMAT.md` が `REQ` の不変性と退役をあれほど厳密に扱っているのは、この直感が既に部分的に入っている証拠だが、軸として明示されていないため一貫して適用できない。

第二に、リファクタリングの規範が 3 行しかない。`docs/development/specification-first-workflow.md` は「GREEN のままリファクタリングする」と述べるだけで、何をリファクタリングとみなすか、いつ止めるかが無い。テスト先行の規律がこれだけ厚い文書で、赤緑リファクタリングの 3 番目だけが薄いのは不均衡である。

第三に、テストダブルの方針が無い。永続化を持つパッケージの多くが `db_memory` と `db_postgres` を対で持つため、バックエンドは事実上フェイクを優先する設計になっているが、それは慣習であって規則ではない。どこで呼び出しの表明を使ってよいか、`db_memory` が `db_postgres` と同じ振る舞いをすることを何が保証するかが書かれていない。

第四に、独立検証の位置づけが手法として説明されていない。`docs/development/specification-first-workflow.md` の「実装していない人または新しい文脈のエージェントが検証する」は明らかに Extreme Programming のペアレビューの代替物だが、そう書かれていないため、なぜその形なのかが読み取れない。

第五に、実装前に RED を観測して記録するという規律の根拠が書かれていない。この規律は、現実には探索的に進んだ作業を、あとから合理的な過程として提示することと必ず緊張を持つ。Parnas がこの緊張を正面から扱っており、「合理的な過程の記録は不誠実な偽装ではなく、読み手のために必要な整形である」という論拠を与える。根拠が無いまま記録だけを要求すると、規律が形式的な作業として扱われる。

## Scope

- **可逆性の軸**：`risk` の隣に、判断が元に戻せるかどうかの軸を導入する。両軸の組み合わせが証拠の要求をどう変えるかを定め、検証のはしご・完了記録の書式・スキーマの 3 箇所へ同じ規則を通す。
- **リファクタリングの規範**：何をリファクタリングとみなすか（振る舞いを変えずに構造を変えること）、いつ行い、いつ止めるかを書く。
- **テストダブルの方針**：フェイク・記録するダブル・スタブをどう選び分けるか、呼び出しの表明が正しい場合、`db_memory` と `db_postgres` の同値性を何が保証するかを `docs/development/testing.md` へ書く。
- **独立検証の位置づけ**：ペアレビューの代替であることと、その帰結（新しい文脈であることが本質であり、人かエージェントかは本質ではない）を書く。
- **記録の根拠**：RED を実装前に観測して記録する規律の根拠を、`docs/development/specification-first-workflow.md` の「Influences and references」へ加える。

## Out of Scope

- 既存 work item の `risk` の再評価。可逆性の軸は以後の work item に適用する。
- テストコードの書き換え。`db_memory` と `db_postgres` の共有契約テストの新設は、本 work item が記録する負債であって、本 work item が返済するものではない。
- テストの配分方針の新設。調査の結果、`docs/development/testing.md` が既に水準ごとの目的・境界・使いどころと E2E 追加時の要求を持つため、本 work item は同じことを二度書かない（Design 参照）。
- `AGENTS.md` の拡張。同ファイルは意図的に薄く保つ方針であり、`reversibility` は `WORK_ITEM_FORMAT.md` が持つ frontmatter 項目であって、同ファイルは既にそこを参照している。
- `DOCUMENTATION_GUIDE.md` の改訂。同文書はリポジトリに依存しない一般論を扱う。

## Design

### 起票時からの前提の変化

起票後に `docs/development/testing.md` が新設され、開発文書の平面が整理された。これが本 work item の 2 点を変える。

第一に、起票時の Scope が挙げていた「テストの配分」は既に満たされている。`docs/development/testing.md` は単体・アダプター統合・受け入れ・E2E・運用検証の 5 水準について、目的、境界と実物にするもの、使いどころを表で持ち、さらに「E2E を追加するときは、単体、アダプター統合、受け入れのいずれでも同じ失敗を検出できない理由をテストまたは work item に残す」と定めている。これは起票時に書こうとした「E2E で確かめるものを、他の層で確かめられないものに限る」そのものであり、かつ既存テストの遡及的な削減を要求しない形になっている。二重に書けば二つの文書が乖離するだけなので、本 work item はこの項目を Out of Scope へ移す。

第二に、テストダブルの方針の置き場所が変わる。起票時は `docs/development/specification-first-workflow.md` を想定していたが、テスト水準と実行境界の正本は `docs/development/testing.md` になった。「最も小さい所有文書へ書く」という同ワークフローの規則に従い、テストダブルの方針は `docs/development/testing.md` が持つ。`docs/development/specification-first-workflow.md` が持つのは、赤緑リファクタリングの輪という工程に属するものだけになる。

### 可逆性の軸

表し方には 2 案ある。`risk` の値を増やす案（`low-irreversible` のような複合値）と、独立した項目を足す案である。採るのは後者で、frontmatter に `reversibility` を足す。複合値は組み合わせの数だけ値が増え、既存の `risk` の表が読めなくなる。独立した項目なら、既存の risk 別の証拠契約はそのまま残り、可逆性は追加の要求として重ねられる。

値は `reversible` と `irreversible` の 2 値とする。中間の値（「費用をかければ戻せる」）は、判断を裁量へ戻すだけで、どちらの要求が適用されるかを決められない。

両軸の組み合わせが何を変えるかは、慎重に決める必要がある。要求を増やしすぎると、低リスクの変更まで重くなる。採る形は、`reversibility: irreversible` の場合に `risk` が `low` でも独立検証を要求する、というものである。逆方向、すなわち可逆であることを理由に `risk` の要求を緩めることはしない。緩和の経路を作ると、可逆性の申告が要求を回避する手段になる。

起票時の Design は「承認を必須とする」とも書いていたが、これは採らない。`docs/development/specification-first-workflow.md` の証拠契約が「起票そのものが作業を認可するので、別の承認記録は持たない。承認欄は考えずに署名されるが、下の検査は観測によってしか満たせない」と明示的に述べている。不可逆性を理由に承認欄を復活させると、その論拠と正面から矛盾する。不可逆な判断に対して実際に効くのは、観測によってしか満たせない要求、すなわち独立検証のほうである。

スキーマ上は任意項目とする。既存の 90 件を超える work item に遡って書かせることはしないが、値を書いたときに不正な値が通ってはならない。

既存の類似規則（認証・暗号・移行は risk を上げよ）は、追加の要求を `risk` の値へ合流させることで、検証のはしご・完了記録の書式・スキーマの下流 3 箇所を自動的に整合させている。可逆性の軸はあえてこの形を採らず、`risk` とは独立に重ねる。したがって下流 3 箇所を手で貫通させる必要があり、これを怠ると「新しい規則がどこからも強制されない」状態になる。独立検証の指摘で実際にこの状態が見つかったため、3 箇所すべてを本 work item の範囲に含める。

- `docs/development/specification-first-workflow.md` の検証のはしご 4 段目
- `WORK_ITEM_FORMAT.md` の完了記録の Independent Verification 欄
- `tools/check/schemas/work-item.schema.json` の完了時条件

### リファクタリングの規範

カタログを持ち込まない。既存の文書はいずれも「規則と、それが破られている状態の見え方」という形で書かれており、リファクタリングの手法一覧はその形に合わない。書くのは、振る舞いを変えないことの意味（テストを変えずに済むこと）、いつ行うか（GREEN の直後）、いつ止めるか（次の振る舞いを足す準備ができたとき）である。

### テストダブルの方針と、同値性の負債

調査の結果、`db_memory` と `db_postgres` の同値性を保証する共有の契約テストは存在しない。2 実装が対で揃っているのは 33 箇所で、`db_memory` のみが 3 箇所（`backend/oauth2/approval`、`backend/oauth2/authorization`、`backend/oauth2/device`）、`db_postgres` のみが 1 箇所（`backend/signingkeys`）である。文書へはこの実態に合わせた表現を書く。

両者のテストは独立に書かれている。例えば `backend/authentication/session/db_memory/sessions_test.go` は `TestSessionStore` 1 件、`backend/authentication/session/db_postgres/sessions_test.go` は `TestSessionRepositoryRoundTrip` と `TestSessionResolutionSurvivesProcessRestart` の 2 件で、名前も対象も対応していない。テストファイルの数もメモリ側 43、PostgreSQL 側 86 と食い違う。

したがってフェイク優先だけを書くと、「メモリ版では通るが PostgreSQL では通らない」という失敗の型を、規範が是認したように見えてしまう。この欠落は `docs/development/testing.md` の中へ、フェイク優先の規範のすぐ隣に負債として書く。別ファイルへ切り出さないのは `docs/threat-model.md` と同じ理由で、一覧を読んだ人が欠落に気づかずに読み終えてしまうためである。

モックについて、当初は「モック生成ライブラリが一つも入っていないので、テストダブルは全て手書きであり、モック禁止は実態の追認になる」と書いた。これは Go 側だけを見た結論で、誤りだった。フロントエンドは `bun:test` の `mock()` と `spyOn()` を使い、66 のテストファイルが呼び出し回数または引数を表明している。`stop_before_reading: [frontend/src]` が、主張を反証する唯一の場所を調査の外へ置いていた。

したがって規範は「モックを使わない」ではなく、効果を読み戻せるかで選び分ける形にする。ポートと永続化のように状態を読み戻せる依存はフェイクで置き換えて状態を表明し、送り出した呼び出しが唯一の観測可能な効果になる境界では、回数と引数の表明を正しい形として認める。

この書き直しは 2 段階を要した。最初の版は後者を「テスト処理系の中で走らせられない境界、すなわち `fetch`、`navigator.credentials`、`window.location`、SMTP」と書いたが、これは 2 つの点で狭すぎた。第一に「すなわち」が列挙を閉じた集合に見せ、第二に「処理系の中で走らせられない」という第二の基準が、`onSubmit` のようなコールバック引数を除外してしまう。コールバックは処理系の中で問題なく走るが、効果は読み戻せない。66 ファイルのうち 23 ファイルがこの型で、依然として非適合のままだった。判定基準を読み戻せるかの一本に戻し、列挙を例示に変え、`navigator.clipboard` と `URL.createObjectURL` を足した。

同時に、逆向きの行き過ぎも見つかった。読み戻せる依存でも、回数そのものが仕様である場合がある。再試行の打ち切り、バッチの分割、遮断器の閾値がこれで、`backend/shared/resilience/circuitbreaker_test.go` と `backend/datakeys/usecases/reencrypt_test.go` が該当する。ここでの `calls != 1` は実装の呼び出し方ではなく、確かめるべき振る舞いそのものである。第三の枠として明示し、「仕様に回数が書かれているか」を寄りかかる前の判定条件にした。

### 記録の根拠

`docs/development/specification-first-workflow.md` の「Influences and references」が既に「各項目は 1 つの影響につき代表的な出典を 1 つ挙げる。この一覧は出自を説明するものであって、追加の正本や完全な準拠を意味しない」という書式を持つ。同じ書式で 1 項目を足す。同時に、Ousterhout と Parnas の設計面の影響は wi-415 が扱うため、本 work item が足すのは記録と過程に関する項目に限る。

## Plan

1. `reversibility` に不正な値を書いた work item を `mise run check-work-items` が受け入れてしまうことを観測する（RED）。
2. `tools/check/schemas/work-item.schema.json` へ `reversibility` を任意項目として加え、値の集合を検査する。RED が落ちることを確認する。
3. `WORK_ITEM_FORMAT.md` へ `reversibility` の意味と、何が不可逆かの具体例を書く。
4. `docs/development/specification-first-workflow.md` の証拠契約へ、不可逆な判断に重ねる要求を書く。
5. 同文書へリファクタリングの規範を書き、独立検証の位置づけを補い、「Influences and references」へ記録の根拠を 1 項目足す。
6. `docs/development/testing.md` へテストダブルの方針と、同値性の保証が無いという負債を書く。

## Tasks

- [x] T001 [Baseline] E2E の現在の対象と、メモリ版と PostgreSQL 版の同値性の保証の有無を調査する。E2E は `frontend/tests/e2e/` の 4 ファイル 24 件で、方針は `docs/development/testing.md` が既に持つ。同値性を保証する共有契約テストは存在しない。
- [x] T002 [Acceptance] 不正な `reversibility` を `mise run check-work-items` が受け入れることを観測する（RED）。
- [x] T003 [Tooling] work item スキーマへ `reversibility` を加え、値の集合を検査する。T002 が落ちることを確認する。
- [x] T004 [Spec] `WORK_ITEM_FORMAT.md` と `docs/development/specification-first-workflow.md` へ可逆性の軸を足す。
- [x] T005 [Spec] リファクタリングの規範、独立検証の位置づけ、記録の根拠を `docs/development/specification-first-workflow.md` へ書く。
- [x] T006 [Spec] テストダブルの方針と同値性の負債を `docs/development/testing.md` へ書く。
- [x] T007 [Verify] `mise run check-work-items` が、可逆性を持つ work item と持たない既存 work item の両方で通ることを確認する。

## Verification

- `mise run check-work-items` が、`reversibility` を持つ新しい work item と、持たない既存 work item の両方で通る。
- `reversibility` に不正な値を書くと落ちる。
- `reversibility: irreversible` かつ `risk: low` の完了記録が、独立検証の記録が無いと落ちる。同じ記録が `reversibility: reversible` なら通る。
- `mise run check-agent-guidance` が通る。
- `mise run check-links` が通る。
- `mise run verify`

## Risk Notes

方法論文書は増やすほど読まれなくなる。`docs/development/specification-first-workflow.md` は既に 292 行あり、ここへ話題を足すと、必要な節を探す費用が上がる。同文書自身が「必要な節を読め。全体を読む必要はない」と述べているため、節の見出しが内容を正確に言い当てていることが、追加の条件になる。テストダブルの方針を `docs/development/testing.md` へ置き、テストの配分を書かないと決めたのは、この費用を下げる判断でもある。

可逆性の軸を導入すると、申告が形式化して全ての work item が `reversible` になるおそれがある。何が不可逆かの具体例（ワイヤ形式、識別子の意味、公開スキーマ、鍵の破棄、`REQ` 番号）を列挙し、判断を裁量に委ねすぎない。

既存 work item に `reversibility` が無い状態を許すため、スキーマ上は任意項目になる。任意のまま誰も書かない状態が続く可能性があり、導入後しばらくして使われているかを確認する必要がある。使われていなければ、必須にするか撤回するかを判断する。

同値性の負債を文書へ書くことは、それ自体は何も直さない。負債を書いた上で返済しない状態が続けば、規範が欠落を追認する装置になる。返済は別の work item が担う。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。製品の規範シナリオと TypeSpec は変わっていない。変わったのは方法論の平面で、`risk` から独立した `reversibility` 軸が work item の frontmatter に加わり、`irreversible` は risk が `low` でも完了前の独立検証を要求する。この規則は証拠契約・検証のはしご 4 段目・完了記録の Independent Verification 欄・スキーマの完了時条件の 4 箇所を貫通し、最後のものだけが機械検査になる。`docs/development/specification-first-workflow.md` は Refactoring 節を得て、リファクタリングの定義（テストが動かないこと）、行う時（GREEN 直後）、止める時（次の振る舞いを足せる形になった時）と、リファクタリングのみの work item が not-applicable 経路で何を代替検査として記録するかを持つ。独立検証がペアレビューの代替であることと、Parnas と Clements の合理的再構成が記録の順序の根拠であることが明記された。`docs/development/testing.md` はテストダブル節を得て、フェイク・記録するダブル・スタブの選択を「効果を読み戻せるか」の一軸で決め、回数が仕様そのものである場合を第三の枠として持つ。同節は `db_memory` と `db_postgres` の同値性を保証する共有契約テストが存在しないことを負債として明記する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`（`reversibility: nonsense` を持つ probe work item を `work-items/` に置いた状態）
  - **Requirement**: N/A: 方法論文書とその検査ツールの変更であり、製品の規範シナリオを持たない。
  - **Observed Failure**: probe が `ok ../work-items/wi-9999-red-probe.md` として通過した。実装後は `FAIL ../work-items/wi-9999-red-probe.md: schema: /reversibility must be equal to one of the allowed values (allowed: reversible, irreversible)` を出して落ちる。
  - **Detection Reason**: スキーマの根に `additionalProperties: false` が無いため、未知の項目は名前も値も検査されずに通る。この観測は「項目を書けば検査される」という思い込みと、実際に列挙が効いている状態とを区別する。可逆性の軸を導入しても値が検査されなければ、申告は任意の文字列になり、下流の規則が参照する対象が定まらない。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/lib.test.ts` の `requires independent verification for an irreversible item at low risk`（および `accepts either reversibility value and rejects anything else`）
  - **Requirement**: N/A: work item スキーマの内部規則であり、対応する規範的な製品要件を持たない。
  - **Observed Failure**: `(fail) validateAgainstSchema — work-item > requires independent verification for an irreversible item at low risk` — `risk: low` かつ `reversibility: irreversible` の完了記録が、`independent_verification` を欠いたまま検査を通過した。
  - **Detection Reason**: このテストは両方向を表明する。独立検証が無い `irreversible` を落とすだけでなく、同じ記録が `reversible` なら通ることも確かめる。前者だけなら「完了記録には常に independent_verification が必要」という行き過ぎた実装でも通ってしまい、`reversibility` を読んでいる証拠にならない。既存の risk 別条件を壊していないことは、既存 224 件のテストが通ることが示す。
- **Independent Verification**:
  実装していない新しい文脈のエージェントが 2 巡実施した。1 巡目は重大な矛盾を 2 件検出した。第一に、`docs/development/testing.md` に書いた「呼び出しの順序と回数を宣言するモックは使わない」が、呼び出し回数または引数を表明しているフロントエンドの 66 テストファイルを、着地した日に非適合にしていた。work item の `stop_before_reading` に `frontend/src` を入れたことで、主張を反証する唯一の場所が調査範囲の外にあった。第二に、新しく書いた「不可逆なら独立検証」が、検証のはしご 4 段目・完了記録の書式・スキーマの完了時条件のいずれとも矛盾し、どこからも強制されていなかった。既存の類似規則が追加要求を `risk` の値へ合流させて下流を自動整合させるのに対し、可逆性の軸は独立に重ねる設計であるため、下流を手で貫通させる必要があった。他に中程度以下の指摘が 9 件（例示列挙の二重記載、節内参照が別の段落を指していた点、リファクタリングの定義と not-applicable 経路の緊張、スキーマ説明の言語違反、変更固有の理由づけの正本への混入、テンプレートが `reversible` を無注釈の既定値として刷り込む点、`代役` という場当たりの語、Scope の陳腐化、影響一覧の項目の位置と長さ）。全 11 件に対応した。2 巡目は 2 件を検出した。書き直したテストダブルの規範が、「テスト処理系の中で走らせられない境界」という第二の基準と閉じた列挙を残していたため、`onSubmit` のようなコールバック引数を表明する 23 ファイルが依然として非適合だった。また逆向きに、回数そのものが仕様である場合（再試行の打ち切り、バッチ分割、遮断器の閾値）を禁じており、`backend/shared/resilience/circuitbreaker_test.go` と `backend/datakeys/usecases/reencrypt_test.go` の 2 件が該当した。判定基準を読み戻せるかの一本に戻し、列挙を例示に変え、回数が仕様である場合を第三の枠として明示した。2 巡目は不可逆規則の 5 箇所の整合と Refactoring 節に新たな矛盾が無いことを確認している。
- **Change-Resistance Results**:
  スキーマの 2 条件それぞれについて、代表的な誤実装が検出されることを確認した。`reversibility` の列挙を外すと `accepts either reversibility value and rejects anything else` が落ちる。完了時条件を外すと `requires independent verification for an irreversible item at low risk` が落ちる。条件を `reversibility` ではなく `status` のみで判定する誤実装は、同じテストの後半（`reversible` かつ独立検証無しが通ること）が落として区別する。文書側の変更は、独立検証 2 巡が検出器として働き、規範と実コードの矛盾を計 4 件検出した。
- **Verification Results**:
  - `mise run check` - passed
  - `mise run test-tools` - passed（224 件）
  - `mise run verify` - passed
