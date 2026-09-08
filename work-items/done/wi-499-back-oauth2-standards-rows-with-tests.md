---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。分割して起票した子 work item は、それぞれが自分のリリース文書を持つ。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md
  typespec: []
  source:
    - backend/oauth2/client/domain/cimd.go
    - backend/oauth2/client/cimd_http/fetcher.go
    - backend/oauth2/client/cimd_http/client_repository.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/oauth2/authorization/domain/redirect_uri.go
    - backend/cmd/internal/bootstrap/memory.go
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-specifications.ts
    - tools/check/src/check-boundaries.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/oauth2/client/domain/cimd_test.go
    - backend/oauth2/client/domain/cimd_fuzz_test.go
    - backend/oauth2/client/cimd_http/fetcher_test.go
    - backend/oauth2/client/cimd_http/client_repository_test.go
    - backend/oauth2/client/cimd_http/refusal_effects_test.go
    - backend/oauth2/authorization/usecases/authorize_test.go
  stop_before_reading:
    - backend/oauth2/token
    - backend/oauth2/device
    - backend/saml
    - backend/wsfederation
    - frontend
---

# OAuth2 が宣言する標準 80 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/oauth2/standards.md` の 80 行を引き取る。

80 行は台帳全体 130 件の 6 割であり、他のどの文書よりひと桁多い。この文書だけが取り残されると、台帳は残り 8 文書を消化しても空にならず、[[wi-495-burn-down-the-standards-coverage-debt]] の最後の台帳削除が実行できない。

80 行は 33 の標準節にまたがる。採用の内訳は `required` 50、`optional` 20、`excluded` 9、`partial` 1 である。`optional` と `excluded` で 29 行、つまり 4 割弱が「提供していること」以外を観測する行である。

## Scope

T002 の判断により、80 行のうち 73 行の消化は子 work item 7 件が持つ。本項目が直接持つのは、`OAuth Client ID Metadata Document` の 7 行の消化と、分割の判断と起票である。

- `docs/contexts/oauth2/standards.md` の `OAuth Client ID Metadata Document` 節が宣言する 7 行を消化する。
- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。
- 最初の標準節を通しで消化した時点で、本項目をさらに分割するかを決めて本節へ書く。**7 件へ割った。内訳は Design の「T002 の判断」節。**

## Out of Scope

- 残る 73 行の消化。Design の分割表が定める 7 件の子 work item が持つ。
- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

80 行を 1 度に読むことはできないので、標準節を単位に進める。節は標準そのものの単位であり、1 つの RFC を 1 度読む文脈で連続した行を判断できる。

節ごとの負債の分布は次のとおりである（2026-09-06 実測）。

| 節 | 負債 |
|---|---:|
| OAuth Client ID Metadata Document | 7 |
| OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 | 6 |
| The OAuth 2.0 Authorization Framework | 4 |
| OAuth 2.0 Token Exchange | 4 |
| OAuth 2.0 Protected Resource Metadata | 4 |
| FAPI 2.0 Security Profile | 4 |
| Best Current Practice for OAuth 2.0 Security | 4 |
| 3 行の節が 6 つ | 18 |
| 2 行の節が 12 個 | 24 |
| 1 行の節が 9 個 | 9 |

`optional` の 20 行は、提供しているならその振る舞いを観測する。提供していないなら、その行の `Adoption` が誤っているということなので、規範の変更として切り出す。この判断が 20 回出てくるため、`optional` の 1 件目で「提供している」と言える根拠の形を決め、残りへ広げる。

`excluded` の 9 行は、提供していないことを観測する。要求が届いたときに製品が拒否するなら、その拒否と、拒否が防いだ効果を観測する。書き方が `required` と逆になるため、1 件目で型を決めてから残りへ広げる。

**本項目をさらに分割するかは未決である。** 最も大きい `OAuth Client ID Metadata Document` の 7 行を通しで消化し、1 行あたりの所要を測ってから決める。測る前に分割の粒度を決めない。

### 着手前に名指しした RED 検査

本項目は製品の振る舞いを変えないので、規範要求に対応する Acceptance RED を持たない。代わりに次の 2 つを置く。

- **Acceptance RED の代替**: 消化する id を `tools/check/standards-coverage-debt.json` から先に外し、テストを書く前に `mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。この検査は宣言・名指し・台帳の 3 つを突き合わせるので、テストを書かずに台帳を縮めることはできない。
- **Unit RED の代替**: 注記を足した各テストについて、その行が言っている判断を production 側で崩し、当のテストが落ちることを観測する。`checkNormativeCoverage` は文字列の一致しか見ないため、名指しが実在の検証に付いていることの担保はこの故障注入しかない。

### 入口の選択: 解決経路そのもの（T001 で決めた）

7 行のうち 6 行を `ClientRepositoryWithCIMD.FindByID` から、`CIMD00-REDIRECT-VALIDATE` の 1 行を `Authorize` から観測した。テストは `backend/oauth2/client/cimd_http/standards_test.go` の 1 ファイルに置いた。

`FindByID` を選んだのは、**登録簿を外した `client_id` が文書へ問い合わせに行く経路がここしか無い**からである。`cmd/internal/bootstrap` の `memory.go` と `postgres.go` はどちらもこのデコレータを登録簿へ被せており、`/authorize` から入っても `/token` から入っても解決はこの 1 か所を通る。`ParseClientIDMetadataDocument` の単体テストでは代わりにならない。関数が正しくても取得と配線が落ちていれば、行が言っている振る舞いは 1 度も起きないからである。

`CIMD00-REDIRECT-VALIDATE` だけ入口が違うのは、この行が文書の形ではなく**解決したクライアントを使う側の判断**を言っているためである。`redirect_uri` の照合は `Authorize` にあり、`FindByID` からは観測できない。

本番と差し替えたのは TLS の信頼点だけである。文書を配るのは `httptest` の自己署名サーバーなので、その証明書を信頼する `http.Client` を `Fetcher` へ渡した。URL 形状の検査、取得、上限、解析、キャッシュはすべて製品の部品が行う。安全な取得側（SSRF 防護と非公開 IP の拒否）は `NewFetcher` を直接使う既存の `refusal_effects_test.go` が持つので、重ねて観測していない。

### `partial` と `excluded` の観測の型（T003、本文書で決めた）

`CIMD00-CACHE` が本項目唯一の `partial` である。行は「HTTP キャッシュヘッダーに従って文書をキャッシュする」と書いているが、`Fetcher` が採っているのは前半だけで、応答の `Cache-Control` は読まない。採った範囲（解決した文書を再利用する）と採らなかった範囲（`no-store` を無視する）を、同じ 1 つのテストで対にして読む型にした。**この行が `required` へ変わるなら、まずこのテストが落ちる。** `partial` を「採った側だけ観測する」型にすると、採らなかった側が後から実装されても記録は何も変わらない。

`CIMD00-PRIVATE-KEY-JWT` が本項目唯一の `excluded` である。この行の `Statement` は製品の制約ではなく標準側の機能を書いているので、観測は [[wi-500-back-api-tokens-standards-rows-with-tests]] の `RFC6750-API-TOKEN-QUERY` と同じ向きになる。すなわち、その機能を使う文書が丸ごと拒否されることと、**拒否が防いだ結果として何が存在しないか**を対で読む。ここでは、認証方式を宣言しない文書は解決されるが、そこに載った `jwks` と `jwks_uri` はクライアントへ 1 つも入らないことを併せて観測した。前者だけでは、認証方式を `none` へ落として残りを取り込む実装と区別できない。

`optional` の行は本項目には無いので、その型は決めていない。子 work item がそれぞれ 1 件目で決める。

### T001 の測定

| 項目 | 実測 |
|---|---|
| 消化した行 | 7 |
| 注記だけで済んだ行 | 0 |
| 新しく書いたテストファイル | 1（6 テスト、約 470 行） |
| 変えた production コード | 0 行 |
| 故障注入 | 19 件（うち 2 件は最初の形では生き残り、事例または注入点を足した） |
| 切り出した欠陥 | 0 件 |

注記だけで済んだ行が 1 つも無かったのは、既存のテストがいずれも行の入口に立っていなかったからである。`cimd_test.go` は純粋な解析関数を、`client_repository_test.go` は取得を代役へ差し替えたデコレータを見ていた。どちらも「文書を取りに行って組み立てる」という行そのものは観測しない。

**所要を支配したのは行数ではなく、7 行が共有する入口にハーネスを 1 つ組むことだった。** 文書を配る TLS サーバー、製品の `Fetcher`、製品のデコレータをつないだ時点で、以降の 1 行はテーブル 1 つと故障注入 1、2 件で済んだ。[[wi-495-burn-down-the-standards-coverage-debt]] は固定費が文書ごとに立つと測ったが、`oauth2` の中ではその単位はもう一段細かい。

### T002 の判断: 行が共有する製品の入口ごとに 7 件へ割る

残る 73 行を**行が共有する入口**を単位に 7 件へ割り、起票した。

| 子 | 入口 | 件数 |
|---|---|---:|
| [[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]] | 認可エンドポイントと PAR | 15 |
| [[wi-517-back-oauth2-token-issuance-standards-rows-with-tests]] | トークンエンドポイント（発行と交換） | 15 |
| [[wi-518-back-oauth2-token-presentation-standards-rows-with-tests]] | 保護リソース、内省、失効 | 13 |
| [[wi-519-back-oauth2-logout-standards-rows-with-tests]] | `end_session` と front / back-channel の配信 | 9 |
| [[wi-520-back-oauth2-non-interactive-grant-standards-rows-with-tests]] | デバイス認可と CIBA | 8 |
| [[wi-521-back-oauth2-metadata-standards-rows-with-tests]] | `/.well-known` と JWKS | 7 |
| [[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] | クライアント登録とプロファイル選択 | 6 |

**節を単位に割る案は採らなかった。** 節で割ると 33 件になるうえ、同じハーネスを何度も組み直すことになる。`RFC6749-AUTHORIZATION-CODE`、`RFC7636-VERIFY`、`RFC9207-ISS`、`RFC9700-REDIRECT-MATCH` は 4 つの別々の節にありながら、どれも認可リクエストを 1 本送って応答と保存を読む同じ観測をする。逆に `RFC6749` の 4 行は 1 つの節にありながら、認可エンドポイントとトークンエンドポイントに分かれる。

**機能領域を単位に割る案とも違う。** [[wi-495-burn-down-the-standards-coverage-debt]] はこれを「領域の境界を測定ではなく読みで引くことになる」として退けた。入口は読みで引く境界ではない。行の `Statement` が指す経路そのものであり、どの HTTP 経路でその振る舞いが起きるかは製品のルーティングを見れば決まる。T001 の測定が費用の単位として指したのもこの境界である。

割り当ては網羅かつ排他である。7 群の和は 73 件で、`CIMD00-*` を除いた台帳上の `oauth2` の行と 1 件の差も無い（実測）。

## Plan

1. ~~`OAuth Client ID Metadata Document` の 7 行を通しで消化し、1 行あたりの所要を記録する。~~ 完了。測定は Design の「T001 の測定」節。
2. ~~記録をもとに、残る 73 行を本項目で続けるか子 work item へ割るかを決め、本節へ書く。~~ 完了。入口を単位に 7 件へ割った。判断は Design の「T002 の判断」節。
3. `optional` の 1 件目で「提供している」と言える根拠の形を決める。**本項目の 7 行に `optional` は無い。子 work item がそれぞれ決める。**
4. ~~`excluded` の 1 件目で観測の型を決める。~~ 完了。`partial` の型と併せて Design の該当節。
5. ~~節ごとに消化し、解決した id を台帳から外す。~~ 7 件を外した。残る 73 件は子 work item が持つ。
6. ~~宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。~~ 7 行とも満たされていた。切り出しは無い。

## Tasks

- [x] T001 [Baseline] `OAuth Client ID Metadata Document` の 7 行を消化し、1 行あたりの所要を記録する。
  `backend/oauth2/client/cimd_http/standards_test.go` に、製品の `Fetcher` と `ClientRepositoryWithCIMD` を
  文書を配る TLS サーバーの上へ組んだハーネスを置き、6 テストで 7 行を消化した。入口の選択は Design の
  「入口の選択」節、測定は「T001 の測定」節。
  recipe: `mise run test-go-package -- ./backend/oauth2/client/cimd_http`
- [x] T002 [Plan] 残る 73 行の進め方を決めて記録する。
  行が共有する製品の入口を単位に 7 件へ割り、[[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]]
  から [[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] までを起票した。
  [[wi-495-burn-down-the-standards-coverage-debt]] の `depends_on` へ 7 件を足した。
  recipe: `mise run check-work-items`
- [x] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
  `excluded` の型は `CIMD00-PRIVATE-KEY-JWT` で、`partial` の型は `CIMD00-CACHE` で決めた。Design の該当節。
  `optional` は本項目の 7 行に無いので決めていない。
- [x] T004 [Ledger] 節ごとに消化し、解決した id を台帳から外す。
  `CIMD00-*` の 7 件を外した（97 → 90）。
  recipe: `mise run check-spec`
- [x] T005 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  7 行とも満たされていた。切り出しは無い。
- [x] T006 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 7 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 残る 73 件は 7 件の子 work item が網羅かつ排他に持ち、いずれも `tools/check/standards-coverage-debt.json` に残っている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。80 行という数がこの誘惑を最も強くする。注記に「何を固定しているか」を書かせ、崩して落ちることを確かめる。
- **80 行が長期化し、台帳が空にならない。** T002 で分割を判断するまで着手を広げない。分割した場合、本項目は最初の節と分割の記録だけを持つ。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、文書ごとの work item が並行しても衝突はエントリー単位に収まる。各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
- **分割が網羅でも排他でもない。** 73 行のどれかがどの子にも入らなければ、台帳は永久に空にならない。どれかが 2 つの子に入れば、同じ行を 2 度消化して台帳の編集が衝突する。割り当ては 7 群の和を台帳と突き合わせて確かめた。

## Completion

- **Completed At**: 2026-09-09
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。

  本項目は 2 つのことをした。1 つは `docs/contexts/oauth2/standards.md` の
  `OAuth Client ID Metadata Document` 節が宣言する **7 行の消化**である。7 行とも、その行の `Statement` を
  区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から消え、台帳は
  97 件から 90 件になった。テストは 1 ファイル（`backend/oauth2/client/cimd_http/standards_test.go`、
  6 テスト）で、文書を配る TLS サーバーの上に製品の `Fetcher` と `ClientRepositoryWithCIMD` を組み、
  そこへ `client_id` を通す。製品コードは 1 行も変わっていない。7 行とも宣言した採用を満たしており、
  欠陥の切り出しは無い。

  もう 1 つは **T002 の分割**である。残る 73 行を、節でも機能領域でもなく**行が共有する製品の入口**を
  単位に 7 件へ割り、[[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]] から
  [[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] までを起票して
  [[wi-495-burn-down-the-standards-coverage-debt]] の `depends_on` へ足した。この単位を選んだ根拠は
  T001 の測定にある。7 行の所要を支配したのは行数ではなく、7 行が共有する入口にハーネスを 1 つ組むこと
  だった。節で割ると同じハーネスを何度も組み直すことになり、実際
  `RFC6749-AUTHORIZATION-CODE`、`RFC7636-VERIFY`、`RFC9207-ISS`、`RFC9700-REDIRECT-MATCH` は 4 つの別々の
  節にありながら同じ観測をする。割り当ては網羅かつ排他であることを台帳と突き合わせて確かめた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`CIMD00-*` の 7 件を台帳から外し、テストを書く前に）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/oauth2/standards.md` の 80 行目から 86 行目まで 7 件それぞれに
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。つまり「台帳を縮めた」という主張は、テストを
    書かずには通せない。7 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 195 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 注記を足した 7 行それぞれについて、対応する production の判断を崩すとその
    テストが落ちることを観測した。内訳は Change-Resistance Results の表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。したがって本項目ではこれを Unit RED の代わりに置いた。
  - **見つけた不足 1**: `CIMD00-URL-SHAPE` を、TLS の待ち受け 1 つだけで観測しようとしていた。この形では
    `https` の検査を外す変異が生き残る。同じ待ち受けへ `http` で送っても接続が成立しないので、
    「スキームの検査が効いた」と「TLS で失敗した」を区別できないからである。**平文の待ち受けを別に立て、
    そこへ要求が 1 件も届かないことを読む形へ変えたところ、この変異を検出するようになった。**
  - **見つけた不足 2**: `CIMD00-STRUCTURE` を、不正な JSON の一覧だけで観測しようとしていた。この形では
    JSON の妥当性検査を外す変異が生き残る。`json.Unmarshal` は文書全体の妥当性を先に見るので、不正な
    JSON では `client_id` が空のまま残り、直後の一致検査に必ず捕まるからである。**必須の 3 項目は正しく、
    別の項目だけ型が違う文書を足したところ、この変異を検出するようになった。** この 1 事例だけが、
    「JSON として妥当であること」を「読めた値がそろっていること」から分離する。
- **Change-Resistance Results**:
  7 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。
  19 件の変異のうち 17 件は最初の形で検出され、2 件は上記の不足を埋めてから検出された。生き残りは無い。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `CIMD00-URL-SHAPE` | `IsClientIDMetadataDocumentURL` から `https` の検査を外す | `TestClientIDMetadataDocumentIsFetchedOnlyForHTTPSURLClientIDs/http_スキーム`: 平文の待ち受けへ要求が届く |
  | 〃 | 空でないパスの検査を外す | 同テスト `/パスを持たない`: パスの無い `client_id` で取得が走る |
  | 〃 | userinfo の検査を外す | 同テスト `/userinfo_を持つ`: 同上 |
  | 〃 | fragment の検査を外す | 同テスト `/fragment_を持つ`: 同上 |
  | `CIMD00-FETCH` | デコレータが URL 形式を検出しても取得しない | 同テスト: 正しい形の `client_id` が解決されない |
  | 〃 | `client_id` とは別の URL から取得する | 同テスト: 取得先の経路が `client_id` のパスと違う |
  | `CIMD00-CLIENT-ID-MATCH` | 取得元 URL との一致検査を外す | `TestClientIDMetadataDocumentMustNameTheURLItWasFetchedFrom` の 4 事例すべて |
  | 〃 | 一致を `strings.EqualFold` にする | 同テスト `/パスの大文字化`: 大文字化した `client_id` が通る |
  | `CIMD00-STRUCTURE` | JSON の妥当性検査を外す | `TestMalformedClientIDMetadataDocumentIsRefusedFailClosed/必須の_3_項目は正しいが別の項目の型が違う` |
  | 〃 | `client_name` の必須を外す | 同テスト `/client_name_が無い` と `/client_name_が空` |
  | 〃 | `redirect_uris` の必須を、解析側と `Validate` 側の両方から外す | 同テスト `/redirect_uris_が無い` と `/redirect_uris_が空の配列` |
  | `CIMD00-PRIVATE-KEY-JWT` | `token_endpoint_auth_method` の検査を外し、`none` へ落として取り込む | `TestClientIDMetadataDocumentDoesNotOfferPrivateKeyJwtAuthentication` の 3 事例すべて |
  | 〃 | 文書の `jwks` をクライアントへ取り込む | 同テスト: 「文書の jwks がクライアントへ取り込まれた」 |
  | `CIMD00-CACHE` | 解決した文書をキャッシュしない | `TestResolvedClientIDMetadataDocumentIsCachedRegardlessOfHTTPCacheHeaders`: 要求が 2 件になる |
  | 〃 | キャッシュを `client_id` ごとに引かない | 同テスト: 同上 |
  | 〃 | 応答の `Cache-Control: no-store` に従ってキャッシュしない | 同テスト: 同上（採らなかった範囲を採ると落ちる、という `partial` の観測） |
  | `CIMD00-REDIRECT-VALIDATE` | `redirect_uri` の照合を無条件に真にする | `TestAuthorizationRequestRedirectURIMustBeListedInTheFetchedDocument` の 4 事例すべて |
  | 〃 | 照合を前方一致にする | 同テスト `/末尾スラッシュ違い` と `/クエリを足した`、および保存回数の観測 |
  | 〃 | 文書に無い宛先をクライアントの一覧へ足す | 同テスト `/文書に無いパス` ほか |

  この方法の限界: `CIMD00-STRUCTURE` の `redirect_uris` は、解析側の明示的な検査と `OAuth2Client.Validate`
  の「redirect 系グラントは `redirect_uris` を必須とする」規則の 2 つに守られている。**片方だけを外す変異は
  生き残る**（実測）。行としての被覆に穴は無く、両方を外せば検出されるが、行に対応するテスト 1 つが
  production の 1 か所に対応しているわけではない。表の 19 件も、行と 1 対 1 ではなく、行が言っている判断
  ごとに 1 件を当てている。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 195 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-package -- ./backend/oauth2/client/cimd_http` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテスト 1 ファイルと tools の JSON、および work item だけで、ブラウザーへ到達する経路が無い。`
