---
depends_on:
  - wi-499-back-oauth2-standards-rows-with-tests
  - wi-500-back-api-tokens-standards-rows-with-tests
  - wi-501-back-authentication-standards-rows-with-tests
  - wi-502-back-cross-cutting-standards-rows-with-tests
  - wi-503-back-saml-standards-rows-with-tests
  - wi-504-back-ws-federation-standards-rows-with-tests
  - wi-505-back-sourcing-standards-rows-with-tests
  - wi-506-back-authorization-standards-rows-with-tests
  - wi-507-back-provisioning-standards-rows-with-tests
  - wi-516-back-oauth2-authorization-request-standards-rows-with-tests
  - wi-517-back-oauth2-token-issuance-standards-rows-with-tests
  - wi-518-back-oauth2-token-presentation-standards-rows-with-tests
  - wi-519-back-oauth2-logout-standards-rows-with-tests
  - wi-520-back-oauth2-non-interactive-grant-standards-rows-with-tests
  - wi-521-back-oauth2-metadata-standards-rows-with-tests
  - wi-522-back-oauth2-client-profile-standards-rows-with-tests
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: Git ratchet は検査の内側の仕組みであり、標準の行も製品の振る舞いも変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/standards.md
    - docs/contexts/sharedsignals/standards.md
  typespec: []
  source:
    - tools/check/src/check-documents.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/src/coverage-debt-ratchet.ts
    - tools/check/README.md
    - backend/sharedsignals/usecases/transmit.go
    - backend/sharedsignals/usecases/receive.go
    - backend/shared/security/tokens_jose/security_event_token_verifier.go
  tests:
    - tools/check/src/normative-coverage.test.ts
    - tools/check/src/repository-checks.acceptance.test.ts
    - backend/sharedsignals/usecases/transmit_test.go
    - backend/sharedsignals/usecases/receive_test.go
    - backend/sharedsignals/usecases/receive_subject_identifier_test.go
    - backend/shared/security/tokens_jose/security_event_token_verifier_test.go
  stop_before_reading:
    - docs/contexts/oauth2/standards.md
    - backend/oauth2
    - frontend
---

# 標準の被覆負債 134 件を消化し、`standards-coverage-debt.json` を空にする

## Motivation

`docs/standards.md` は自らの表について「各行は、規範 ID を名指しするテストを製品のテストの中に持つ。まだテストの無い行は `tools/check/standards-coverage-debt.json` に理由付きで残っており、この一覧は縮むだけである」と宣言している。

[[wi-418-normative-coverage-gates]] がこの検査を入れたとき、154 行のうち 134 行が名指しを持っていなかったので、134 件がそのまま台帳へ入った。行は 155 行へ増えたが、台帳は 2026-09-06 時点でも 134 件のままである。`git log -- tools/check/standards-coverage-debt.json` が返すコミットは、台帳を作った 1 件だけである。

**この台帳を縮めることを Scope に書いた work item は、リポジトリのどこにも無い。** 134 件すべての理由が `present when the check was introduced` で揃っており、投入以降に 1 件も判断されていないことが理由の欄からも読める。

負債の性質は `Adoption` 列によって違う。全 155 行に対する内訳は次のとおり（2026-09-06 実測）。

| Adoption | 負債 | 全体 |
|---|---:|---:|
| required | 89 | 96 |
| optional | 25 | 27 |
| excluded | 15 | 19 |
| partial | 5 | 13 |

所有する文書ごとの分布は次のとおりで、`oauth2` の 80 件が全体の 6 割を占める。`OIDC-CORE-CODE-FLOW` だけが 2 つの文書から宣言されているため、行数の合計は 155 ではなく 156 になる。

| 文書 | 負債 / 行 |
|---|---:|
| `docs/contexts/oauth2/standards.md` | 80 / 83 |
| `docs/contexts/api-tokens/standards.md` | 11 / 12 |
| `docs/contexts/authentication/standards.md` | 9 / 10 |
| `docs/standards.md` | 7 / 8 |
| `docs/contexts/saml/standards.md` | 6 / 7 |
| `docs/contexts/ws-federation/standards.md` | 6 / 6 |
| `docs/contexts/authorization/standards.md` | 5 / 5 |
| `docs/contexts/sourcing/standards.md` | 5 / 8 |
| `docs/contexts/sharedsignals/standards.md` | 4 / 4 |
| `docs/contexts/provisioning/standards.md` | 1 / 13 |

`provisioning` だけが 13 行中 12 行の名指しを持っている。SCIM の適合作業（[[wi-238-scim-inbound-list-query-conformance]]）が id を名指しするテストを書いたからであり、消化が可能であることの実例である。裏を返せば、他の文書は適合作業を経ていないというだけで負債になっている。

**台帳の宣言と検査の食い違いは解消した。** Git ratchet が標準と具体例の台帳を各基準 revision と比較し、新しい id と解消済み id の再流入を同じく拒否する。「この一覧は縮むだけである」は検査された性質である。

## Scope

T003 の判断により、134 件のうち 130 件の消化は所有文書ごとの子 work item が持つ。本項目が直接持つのは、Git ratchet の維持、`sharedsignals` の 4 件による測定、分割の判断、そして子 work item がすべて完了した後の台帳削除である。以下の各項は、本項目と子 work item の双方に効く規則として残す。

- Git ratchet が標準の台帳への新規 id の追記を拒否する。
- 134 件を 1 件ずつ確認し、次のいずれかに解決して台帳から外す。
  - 当の行を検証しているテストが実在する → そのテストに `// <ID>: <この行の何を固定しているか>` の注記を足す。
  - 当の行を検証しているテストが無い → 書く。観測は `Adoption` 列に応じた形（Design を参照）にする。
  - 行が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 名指しの対象が `docs/standards.md`（横断）の 7 件について、どのパッケージのテストが所有するかを決める。
- 実装が宣言した採用を満たしていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。テストの追加と実装の修正を同じ変更に混ぜると、どちらが何を意味するのか後から読めない。
- 134 件が 0 になった時点で `standards-coverage-debt.json` を落とし、`checkNormativeCoverage` へ標準側から `debt` を渡すのをやめる。例外を持たない検査にする。

## Out of Scope

- `tools/check/example-coverage-debt.json` の 614 件。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 既存の拒否テストが防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- 「id を名指しする文字列がある」だけを根拠にした台帳からの削除。

## Design

台帳を縮める根拠は、id が書かれていることではなく、そのテストの入力と観測が当の行の `Statement` を区別できることである。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しさえすれば検査は通る。注記に「何を固定しているか」を書かせることが、名指しが実在の検証に付いていることの唯一の担保になる。id だけの注記は禁止する。

`Adoption` 列は、テストが何を観測すべきかを決める。同じ「テストが名指しする」でも、次の 4 通りは中身が違う。

| Adoption | テストが観測するもの |
|---|---|
| `required` | 宣言した振る舞いが、製品の正式な入口から到達できること。 |
| `partial` | 採用した範囲の振る舞いと、採用していない範囲がどう扱われるか（拒否するのか、単に提供しないのか）。 |
| `optional` | 提供している場合はその振る舞い。提供していないなら、行の `Adoption` が誤っているので規範の変更として切り出す。 |
| `excluded` | 提供していないこと。要求が届いたときに製品が拒否するなら、その拒否と、拒否が防いだ効果。 |

`excluded` の 15 件は「Implicit Grant を提供する」のように、行の `Statement` が製品の振る舞いではなく標準側の機能を書いている。これらのテストは `Statement` を満たすことではなく、満たさないことを観測する。書き方が `required` と逆になるため、最初の 1 件で型を決めてから残りへ広げる。

Git ratchet は消化より先に実行する。順序を逆にすると、消化している間に新しい行が台帳へ流れ込み、件数が減らない理由が消化の遅さなのか流入なのか区別できなくなる。

進める単位は所有文書とする。分類（named / nearby）順に進める案は却下した。`docs/standards.md` の行は分類の材料になる `report-coverage-debt` の対象外であり（同ツールは `example-coverage-debt.json` しか読まない）、標準側には機械的な分類がそもそも存在しない。文書単位なら、標準そのものを 1 度読む文脈で連続した行を判断できる。

`oauth2` の 80 件を 1 つの work item で扱うかは未決である。**最初の 1 文書（`ws-federation` の 6 件、または `sharedsignals` の 4 件）を通しで消化して 1 件あたりの所要を測り、そこで決める。** 測る前に分割の粒度を決めない。

### 測定の結果（T002、`sharedsignals` の 4 件）

| ID | Adoption | 既存テストの状態 | やったこと |
|---|---|---|---|
| `RFC9493-SUBID-FORMAT` | partial | 行を区別できていた | 注記。`format` の値だけで種別を決めることを区別する事例を 1 件追加 |
| `RFC9493-SUBID-ISS-SUB` | partial | 行を区別できていた | 注記のみ |
| `RFC8417-SET-SIGNED` | required | `iat` を観測せず、署名も検証していなかった | 注記。`iat` の観測と、テナントの公開鍵での署名検証を追加 |
| `RFC8417-SET-VERIFY` | required | 未知の鍵と改ざんを観測せず、「反映しない」効果も観測していなかった | 注記。3 つとも追加。名指しは 2 ファイルに置いた |

4 件のうち 2 件は注記だけで済み、2 件は新しい観測を要した。つまり**注記だけの作業ではない**。所要の内訳は、1 件あたりの手数よりも、その Context の domain・use case・adapter・既存テストを 1 度読む固定費が支配的だった。この固定費は 1 文書の中では償却されるが、文書をまたぐと償却されない。

`RFC8417-SET-VERIFY` は、`Statement` が「反映しない」と「監査する」の 2 つを言っているのに、既存テストは後者しか観測していなかった。**`Statement` に動詞が 2 つあれば観測も 2 つ要る**というのが、この 4 件から出た最も一般的な読み方である。

### T003 の判断: 所有文書ごとに子 work item へ割る

固定費が文書ごとに立つという測定結果から、残る 130 件は**所有文書ごとに 1 件の子 work item へ割る**。9 件を起票した。

| 子 | 文書 | 件数 |
|---|---|---:|
| [[wi-499-back-oauth2-standards-rows-with-tests]] | `docs/contexts/oauth2/standards.md` | 80 |
| [[wi-500-back-api-tokens-standards-rows-with-tests]] | `docs/contexts/api-tokens/standards.md` | 11 |
| [[wi-501-back-authentication-standards-rows-with-tests]] | `docs/contexts/authentication/standards.md` | 9 |
| [[wi-502-back-cross-cutting-standards-rows-with-tests]] | `docs/standards.md` | 7 |
| [[wi-503-back-saml-standards-rows-with-tests]] | `docs/contexts/saml/standards.md` | 6 |
| [[wi-504-back-ws-federation-standards-rows-with-tests]] | `docs/contexts/ws-federation/standards.md` | 6 |
| [[wi-505-back-sourcing-standards-rows-with-tests]] | `docs/contexts/sourcing/standards.md` | 5 |
| [[wi-506-back-authorization-standards-rows-with-tests]] | `docs/contexts/authorization/standards.md` | 5 |
| [[wi-507-back-provisioning-standards-rows-with-tests]] | `docs/contexts/provisioning/standards.md` | 1 |

`oauth2` の 80 件をここでさらに割らないのは、割る根拠がまだ無いからである。この文書は 33 の標準節にまたがり、節ごとの負債は最大でも 7 件しかないので、節を単位に割ると 33 件の work item になる。機能領域を単位に割る案は、領域の境界を測定ではなく読みで引くことになる。したがって**分割の判断そのものを [[wi-499-back-oauth2-standards-rows-with-tests]] へ渡す**。同項目は最大の節（`OAuth Client ID Metadata Document` の 7 件）を通しで消化してから決める。本項目が `sharedsignals` で採ったのと同じ順序である。

### `oauth2` の分割結果（[[wi-499-back-oauth2-standards-rows-with-tests]] が決めた）

同項目は `OAuth Client ID Metadata Document` の 7 件を消化して測り、残る 73 件を**行が共有する製品の入口**を単位に 7 件へ割った。節でも機能領域でもない。測定が示したのは、費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点であり、入口はそこで読み取る境界ではなく行の `Statement` が指す経路そのものだからである。7 件の内訳と各件の担当行は同項目の Design にある。

これにより本項目の `depends_on` は 9 件から 16 件になった。

本項目はこれ以降、Git ratchet の維持と、16 件がすべて完了した後の台帳削除だけを持つ。`depends_on` がその順序を機械で拘束する。

## Plan

1. ~~`standards-coverage-debt-baseline.json` を現在の 134 件で作り、`check-specifications.ts` から `debtBaseline` として渡す。受入集合に無い id を足した fixture が、規則を入れる前は通り、入れた後に落ちることを観測する。~~ 完了。観測は Verification の「T001 の観測」節。
2. ~~`sharedsignals` の 4 件を通しで消化し、注記の型と、1 件あたりの所要を記録する。~~ 完了。記録は Design の「測定の結果」節。
3. ~~記録をもとに、残る文書を本 work item で続けるか子 work item へ割るかを決め、本節へ書く。~~ 完了。所有文書ごとに 9 件へ割った。
4. 16 件の子 work item の完了を待つ。`excluded` の行の観測の型は、各子がその文書の 1 件目で決める。文書をまたいで型を先に揃えることはしない。`sharedsignals` に `excluded` の行が無かったので、本項目はその型を決めていない。
5. 134 件が 0 になったら、台帳と標準側の `debt` 引数を落とす。

## Tasks

- [x] T001 [Tooling] 標準の台帳に受入集合を導入し、新規の追記を拒否する。
  `tools/check/standards-coverage-debt-baseline.json` を投入時点の 134 件で作り、`check-specifications.ts` から `debtBaseline` として渡した。検査の型は増やしていない。再実行の recipe は `mise run check-spec`。
- [x] T002 [Baseline] `sharedsignals` の 4 件を消化し、注記の型と所要を本 work item へ記録する。
  4 件とも消化し、台帳は 134 → 130 件。測定は Design の「測定の結果」節。再実行の recipe は `mise run test-go-package -- ./backend/sharedsignals/usecases` と `mise run test-go-package -- ./backend/shared/security/tokens_jose`。
- [x] T003 [Plan] 残る 130 件の進め方（本 work item で続けるか分割するか）を決めて記録する。
  所有文書ごとに 9 件へ割った。判断と根拠は Design の「T003 の判断」節。
- [x] T004 [Ledger] 文書ごとに消化し、解決した id を台帳から外す。
  16 件の子 work item がすべて完了し、`untested` は空になった。本項目が直接消化したのは T002 の
  `sharedsignals` の 4 件だけである。
- [x] T005 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  各子 work item が自分の文書について持った。`sharedsignals` の 4 件では 1 件も見つからなかった。
- [x] T006 [Tooling] 台帳が空になったら、台帳と標準側の `debt` 引数を落とす。
  `tools/check/standards-coverage-debt.json` を削除し、`checkNormativeCoverage` の台帳引数を
  `ledger?: { entries, path }` の 1 つの省略可能な値にまとめて、標準側では渡さないようにした。
  `COVERAGE_DEBT_PATHS` からも標準の台帳を外した。recipe: `mise run test-tools`、`mise run check-spec`。
- [x] T007 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が標準の被覆について例外を持たずに通る。
- 基準 revision に無い id を台帳へ足すと `mise run check-coverage-debt-ratchet` が落ちる。
- `mise run verify`

### T006 の観測（免除の消滅）

台帳を消しただけでは、消えたのが一覧なのか免除の仕組みなのかを区別できない。区別するために、
`docs/contexts/sharedsignals/standards.md` へ `RFC8417-SET-FIXTURE` の行を 1 行足し、2 つの状態で
`mise run check-spec` を走らせた。

| 状態 | `mise run check-spec` |
|---|---|
| 台帳を置かない | exit 1。`docs/contexts/sharedsignals/standards.md:20: RFC8417-SET-FIXTURE is declared, but no test names it. Cite the id from the test that exercises it.` |
| 同じ名前の台帳を作り直し、その id を理由付きで載せる | exit 1。**文言も同じ**。台帳は読まれない |

2 行目が本題である。免除の一覧が消えただけなら、作り直せば通ってしまう。文言から
`tools/check/standards-coverage-debt.json` への案内が消えたことも、同じ観測に含まれる。存在しない
ファイルへ読み手を送らないためであり、そのファイルを作れば通ると誤解させないためでもある。

この観測は `tools/check/src/repository-checks.acceptance.test.ts` の
`admits no standards row through a recreated debt ledger` として残した。手で 1 度観測しただけでは、
標準側へ台帳を配線し直す変更を誰も止められない。

### T001 の観測（受入集合の RED / GREEN）

固定具は 2 つを対にして足す。`docs/contexts/sharedsignals/standards.md` へ `RFC8417-SET-FIXTURE` の行を 1 行足し、同じ id を `tools/check/standards-coverage-debt.json` へ理由付きで追記する。宣言だけ、または台帳だけでは既存の別の規則が拒否するので、受入集合の有無を分離できない。

| | `mise run check-spec` |
|---|---|
| 受入集合を渡す前 | exit 0。`ok normative coverage (157 standard(s), …)` |
| 受入集合を渡した後 | exit 1。`tools/check/standards-coverage-debt.json: RFC8417-SET-FIXTURE was not in the migration baseline. Add a test instead of growing the debt list.` |

規則そのものの単体検査は `tools/check/src/normative-coverage.test.ts` の `rejects admitting a new id to a ratcheted debt ledger` が既に持っていた。落ちていたのは `check-specifications.ts` の配線だけであり、上の観測はその配線を対象にしている。

### T002 の観測（変異による確認）

注記を足した観測が実際に効くことを、production を崩して確かめた。

| 崩した箇所 | 落ちたテスト |
|---|---|
| `subjectFromIdentifier` の `default` を、`format` を見ずに `id` メンバーで解決する形へ | `TestReceiveSecurityEvent_Rfc9493SubjectIdentifiers/an_uninterpreted_format_carrying_a_resolvable_identifier_anyway` のみ |
| `BuildAndSignSecurityEventToken` の claims から `iat` を削除 | `TestBuildAndSignSecurityEventToken` |
| `VerifySecurityEventToken` の署名検証を無効化 | `TestVerifySecurityEventToken` の `rejects_spoofed_signature` / `rejects_a_kid_the_JWKS_does_not_hold` / `rejects_a_payload_tampered_with_after_signing` |

1 つ目は、既存の「解釈しない `format`」の事例が生き残り、新しく足した事例だけが落ちた。この 1 件が既存の観測では区別できない範囲を埋めていることが、これで読める。

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。減った件数は何も意味しない。注記に「何を固定しているか」を書かせること、および `Adoption` ごとの観測の型を先に決めることで、貼るだけの作業と区別する。
- **`excluded` の行に書けるテストが無い。** 製品がその機能をそもそも実装していないなら、観測できるのは「入口が存在しない」ことだけになりうる。この場合に何を観測とするかは T002 の前に決める。決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。理由が更新されていれば、判断済みであることが後から読める。
- **`oauth2` の 80 件が長期化する。** T003 で分割を判断するまで着手を広げない。分割した場合、親である本 work item は Git ratchet の維持と最後の台帳削除だけを持つ。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。標準の行も製品の
  振る舞いも変わっていない。変わったのは、標準の被覆に免除の仕組みが無くなったことである。
  `tools/check/standards-coverage-debt.json` を削除した。投入時点の 134 件は 0 件になっており、
  内訳は本項目が T002 で消化した `sharedsignals` の 4 件と、16 件の子 work item が消化した 130 件である。
  宣言した標準の行は 157 件あり、そのすべてが id を名指すテストを持つ。
  **消えたのは一覧ではなく免除である。** 台帳を消しただけなら、同じ名前のファイルを作り直せば元へ戻る。
  `checkNormativeCoverage` の台帳引数は省略可能になり、標準側はそれを渡さない。渡さない呼び出しでは、
  行が未消化であることを報告する規則だけが残り、台帳そのものを検査する規則（重複、id 順、理由、
  テストが付いた id の残留）は動かない。検査する台帳が無いからである。
  **台帳引数は 2 つの省略可能な値ではなく 1 つの組にした。** `debt` と `debtPath` を別々に省略可能に
  すると、`debt` だけを渡して `debtPath` を書き忘れた呼び出しが型検査を通り、そのとき台帳側の 4 つの
  規則が黙って消える。残る `example-coverage-debt.json` は 614 件を抱えているので、その 4 つが黙って
  外れる形は作れない。`ledger?: { entries, path }` にすれば、片方だけを渡すことがそもそも書けない。
  **`COVERAGE_DEBT_PATHS` からも標準の台帳を外した。** ratchet は「基準 revision に無い id の追加を
  拒否する」検査であり、守る対象が無くなった。標準の行は、名指すテストを持つか `check-spec` に落ちるか
  のどちらかなので、流入を測る基準そのものが要らない。
  **検査自身が記録の嘘を止めた。** `initial_context` が削除した台帳を指したままだったので
  `check-work-items` が落ちた。読んだファイルの一覧は、消えたファイルを指し続けられない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`docs/contexts/sharedsignals/standards.md` へ
    `RFC8417-SET-FIXTURE` の行を 1 行足した状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/sharedsignals/standards.md:20: RFC8417-SET-FIXTURE is
    declared, but no test names it. Cite the id from the test that exercises it.` 同じ id を載せた台帳を
    作り直しても、exit 1 と文言は変わらなかった。
  - **Detection Reason**: 免除が残っているなら、台帳へ載せた 2 回目は exit 0 になる。2 回とも同じ
    文言で落ちることが、読まれる台帳がもう無いことの観測である。文言から台帳への案内が消えたことは、
    存在しない手順へ読み手を送らないことの観測でもある。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/normative-coverage.test.ts` の
    `names no escape hatch when the caller has no debt ledger`
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: `TypeError: undefined is not an object (evaluating 'debt.map')`
    （`normative-coverage.ts:76`）。型検査も
    `Type '{ declared: …; cited: Set<string>; }' is missing the following properties from type
    'CoverageInput': debt, debtPath` で落ちた。台帳を持たない呼び出しがそもそも書けなかった。
  - **Detection Reason**: 台帳を渡さない呼び出しが型として存在しなければ、「例外を持たない検査」は
    書けない。実行時の失敗と型の失敗が同じ 1 つの欠落を指している。
- **Change-Resistance Results**:
  3 件の故障を注入し、すべて検出された。生存はゼロ。注入は Edit で当て、毎回 `mise run test-tools` の
  pass/fail 件数で着弾を確かめている。

  | 注入した故障 | 落ちたテストと観測 |
  |---|---|
  | M1 標準側へ台帳を配線し直す | `admits no standards row through a recreated debt ledger`。作り直した台帳に載せた行が通ってしまう |
  | M2 台帳が無くても逃げ道を案内する（文言を元へ戻す） | 上と `names no escape hatch when the caller has no debt ledger` の 2 件。免除は消えているのに案内だけが残る状態を、受入側と単体側の両方が落とす |
  | M3 `COVERAGE_DEBT_PATHS` に標準の台帳を戻す | `rejects additions to the example ledger beyond the named Git revision`。消えたはずの台帳への追加を ratchet が報告する |

  **方法の限界。** 注入は台帳の配線と文言だけを崩しており、`citedNormativeIds` の一致規則は動かして
  いない。id の照合そのものが壊れる形（部分一致を許すなど）は、本項目ではなく
  [[wi-418-normative-coverage-gates]] が入れた既存の検査が持つ。
- **Verification Results**:
  - `mise run verify` - passed
