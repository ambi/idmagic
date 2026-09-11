---
depends_on: [wi-532-fapi-security-profile-selection-applies-no-constraint]
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-09
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの標準行にテストを対応付けるだけで、利用者が読むリリース情報に変化は無い。切り出した欠陥は自分のリリース文書を持つ。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md#RFC7591-REGISTER
    - docs/contexts/oauth2/standards.md#RFC8176-AMR
    - docs/contexts/oauth2/standards.md#FAPI2-PROFILE-SELECTION
    - docs/contexts/oauth2/standards.md#FAPI2-PAR-PKCE
    - docs/contexts/oauth2/standards.md#FAPI2-CLIENT-AUTH
    - docs/contexts/oauth2/standards.md#FAPI2-SENDER-CONSTRAINT
    - docs/contexts/oauth2/decisions.md
  typespec: []
  source:
    - backend/oauth2/handlers_http/routes.go
    - backend/oauth2/handlers_http/register_handler.go
    - backend/oauth2/client/usecases/register_client.go
    - backend/oauth2/client/domain/client.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/authentication/domain/amr.go
    - backend/shared/spec/policy.go
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/shared/http/server_http/metadata_standards_test.go
  stop_before_reading:
    - backend/oauth2/db_postgres
    - backend/saml
    - backend/wsfederation
    - frontend
---

# クライアントの登録とプロファイル選択が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 6 行を持つ。

6 行は、クライアントごとに制約を変える仕組みを定める。FAPI 2.0 の 4 行は、プロファイルを選んだクライアントにだけ PAR、S256 PKCE、`private_key_jwt` または mTLS、送信者制約を課す。残る 2 行は動的登録と `amr` の記録である。5 行が `optional` である。

## Scope

- 次の 6 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC7591-REGISTER` | optional | OAuth 2.0 Dynamic Client Registration Protocol |
| `RFC8176-AMR` | required | Authentication Method Reference Values |
| `FAPI2-PROFILE-SELECTION` | optional | FAPI 2.0 Security Profile |
| `FAPI2-PAR-PKCE` | optional | FAPI 2.0 Security Profile |
| `FAPI2-CLIENT-AUTH` | optional | FAPI 2.0 Security Profile |
| `FAPI2-SENDER-CONSTRAINT` | optional | FAPI 2.0 Security Profile |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の work item が持つ標準行。[[wi-499-back-oauth2-standards-rows-with-tests]] の Design にある分割表が所属を定める。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**観測の入口は クライアント登録と、クライアントごとのプロファイル選択が効く各エンドポイント である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 6 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は 2 つの対を要する。プロファイルを選んだクライアントが追加の制約を受けることと、選んでいないクライアントが同じ要求で通ることを、同じ 1 つのハーネスの中で対にして読む。片方だけでは、制約が全クライアントに掛かっている実装と区別できない。

`RFC8176-AMR` は「実際に成立した認証方法を記録する」ことを言っているので、成立していない方法が `amr` に載らないことを併せて読む。[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] がこの行に隣接する欠陥を持つので、着手前に読む。**読んだ。完了しており、語彙は `backend/authentication/domain/amr.go` に閉じている。** 本項目が読むのは語彙そのものではなく、成立した方法が `amr` として RP まで届くかである。語彙の閉じ方は `RFC8176-AMR-VOCABULARY` が持ち、そちらは既に消化されている。

### 着手前の判定: 6 行のうち 4 行は製品が宣言した採用を満たしていない

`FAPI2-*` の 4 行は現状のままでは消化できない。`fapi_profile` は保存され、列挙として検証され、admin API と `/register` の応答へ書き戻され、管理 UI へ表示されるが、**この値を読んで制約を掛ける箇所が製品に 1 つも無い**。非テストの Go 全体で `fapi` を探すと、当たるのは認可規則の名前 `par_required_if_fapi` と `authorize.go` の予定を書いたコメントだけである。規則の名前は FAPI を名乗るが、実装が読むのは `RequirePAR` であり、この値は `FapiProfile` からではなく登録入力の同名フラグから来る。送信者制約の `DpopBoundAccessTokens` とクライアント認証方式も同様に `FapiProfile` と無関係に決まる。

したがって「プロファイルを選んだクライアントが追加の制約を受ける」という観測が作れない。Design がこの入口に要求した対 — 選んだ側と選んでいない側 — は、選んだ側が何も変わらないので成立しない。`optional` は「提供しているならその振る舞いを観測する」ことを意味するが、提供されていない。

これは Scope と T006 が想定した「宣言した採用を満たしていない行」であり、[[wi-532-fapi-security-profile-selection-applies-no-constraint]] として切り出した。`depends_on` へ入れた。

**この 4 行は前提 work item の側で消化された。** 実装を入れたテストが 4 つの id を名指した時点で `check-spec` が台帳からの削除を要求したので、境界はそこで動いた。振る舞いを入れたテストと、その行を名指すテストを別々に書く理由は無い。本項目に残るのは `RFC7591-REGISTER` と `RFC8176-AMR` の 2 行である。

**残る 2 行は前提を持たない。** `RFC7591-REGISTER` の `/register` は `routes.go` に配線され、`RFC8176-AMR` の記録は認可コードフローを通って ID トークンまで届く。ハーネスは [[wi-532-fapi-security-profile-selection-applies-no-constraint]] が `backend/shared/http/server_http/fapi_security_profile_e2e_test.go` に組んだものが `/register` まで届いているので、`RFC7591-REGISTER` はその隣に置ける。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

## Plan

1. 6 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。
2. `optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。提供していなければ規範の変更として切り出す。
3. `excluded` があれば、その 1 件目で観測の型を決める。**本項目の 6 行に `excluded` は無い。**
4. 残りを消化し、解決した id を台帳から外す。
5. 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。

## Tasks

- [ ] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
- [ ] T002 [Harness] クライアント登録と、クライアントごとのプロファイル選択が効く各エンドポイント にハーネスを組み、`required` の 1 件目で型を決める。
- [ ] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。`excluded` は本項目に無い。
- [ ] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
- [ ] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
- [x] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  `FAPI2-*` の 4 行が満たされていなかった。[[wi-532-fapi-security-profile-selection-applies-no-constraint]]
  として切り出し、`depends_on` へ入れた。判定は Design の「着手前の判定」節。4 行の消化は
  実装と同時に起きたので、その work item が台帳から外した。
  recipe: `mise run check-work-items`
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 6 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
  `FAPI2-*` の 4 件は [[wi-532-fapi-security-profile-selection-applies-no-constraint]] が実装と同時に外した。
  本項目が直接外すのは `RFC7591-REGISTER` と `RFC8176-AMR` の 2 件である。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **プロファイルを選んだ側だけを読む。** 選んでいないクライアントが同じ要求で通ることを対にしないと、制約が全クライアントに掛かっている実装と区別できない。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
