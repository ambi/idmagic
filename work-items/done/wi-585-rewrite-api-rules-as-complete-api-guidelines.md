---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-16
priority: p1
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "既存の規則の意味を変えず、未記述の観点を決定として書き足す。個々の操作の形は TypeSpec が持ち、この変更では触らない。"
documentation_impact:
  level: none
  reason: "変わるのは開発者が読む設計文書だけで、利用者が観測する振る舞い、API、設定キーはどれも変わらない。"
  references: []
initial_context:
  specification:
    - docs/design/application/api-guidelines.md
    - DOCUMENTATION_GUIDE.md
    - docs/design/data/database.md
  typespec:
    - spec/generated/openapi/idmagic.openapi.json
    - spec/contexts/audit/main.tsp
    - spec/contexts/identity-governance/main.tsp
  source:
    - backend/shared/http/support_http/pagination.go
    - backend/shared/http/support_http/pagination_request.go
    - backend/shared/http/support_http/response.go
    - backend/shared/http/support_http/rate_limit.go
    - backend/shared/http/support_http/deprecation.go
    - backend/shared/spec/runtime_contract.go
    - backend/shared/spec/uuid.go
    - backend/jobs/handlers_http/admin_job_handler.go
    - backend/idmanagement/domain/csv.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/branding_handler.go
    - tools/check/src/contract-drift.ts
    - tools/check/src/terminology.ts
    - infra/schema/postgres.sql
  tests: []
  stop_before_reading:
    - frontend/
    - work-items/done/
---

# API 規則を、基礎から観点の揃った API ガイドラインへ書き直す

## Motivation

[API 規則](../../docs/design/application/api-guidelines.md)は、カーソルページング、エラーレスポンス、セキュリティヘッダー、宣言する状態コード、安定性と版管理、廃止、ワイヤ本体、文字列長の八つの節からなる。
どの節も深いが、`DOCUMENTATION_GUIDE.md` §4.5 が挙げる観点のうち次が書かれていない。

- リソースの命名。パスの形、コレクションと単一リソース、プロパティの記法
- 値の表現。日時、期間、数値、列挙、識別子のフォーマット、`null` と省略の使い分け
- HTTP メソッドの割り当てと、選んだメソッドが負う義務の果たし方
- コレクションの操作のうち、絞り込み、並べ替え、部分レスポンス
- 再送と重複防止。冪等キーの受け付けと保持期間
- 条件付きリクエスト。`ETag` と `If-Match`
- 長時間かかる操作の表し方
- CRUD に収まらない操作で、動詞をどこに置くか
- 一括リクエストの最大件数と、レート制限の伝え方

同じ §4.5 は「採らないと決めた観点は、決めたと分かる形で残す」と定めている。
現状は、書かれていない観点が判断の結果なのか見落としなのかを後から区別できない。

読みやすさにも問題がある。
文字列長の節は 80 行近い散文と表が混ざり、状態コードの節は例外の説明が段落で続く。
規則そのものと、その規則が守っているものと、どこが強制点かが、同じ段落に溶けている。

## Scope

- [API 規則](../../docs/design/application/api-guidelines.md)を、観点ごとの節に再構成し、各節を「規則」「守っているもの」「強制点」「適用状況」で書く。
- 上に挙げた未記述の観点について、採用する規則を決めて書く。採らない観点は、採らない決定として残す。
- 既存の八つの節を、意味を変えずに表中心へ組み直す。
- 文書の題名を「API ガイドライン」にする。
- [用語集](../../docs/domain/glossary.md)の `InterfaceStability` など、この文書を名指す記述の参照を追従させる。

## Out of Scope

- エンドポイントの改修。この文書が決めた規則に現行実装を合わせる作業は別に扱う。
- TypeSpec の変更。個々の操作の形、ステータスコード、エラーの直和は TypeSpec が正本である。
- 認可とスコープの規則。[認可設計](../../docs/design/security/authorization.md)が持つ。
- プロトコルエンドポイント（OAuth 2.0、OIDC、SAML、WS-Federation、SCIM、Shared Signals）の契約。各標準が定める。
- ファイル名の変更。`api-guidelines.md` のままとする。

## Design

各規則を次の四つで書く。
散文を表へ移すのは見た目のためではなく、この四つが揃っているかを行単位で確かめられるようにするためである。

| 列 | 内容 |
| --- | --- |
| 規則 | 何をどう揃えるか |
| 守っているもの | この規則が無いと何が壊れるか。緩めてよいかを後から判断する根拠 |
| 強制点 | TypeSpec、Go の検証、PostgreSQL の制約、検査タスクのどれが拒否するか |
| 適用状況 | 全接点で適用済みか、適用していない接点があるか |

**未適用の規則を「将来の案」として書かない。** 現在状態の正本に前方参照を置くと、実装が進んだ時点で腐る。
規則そのものは今日有効な決定として書き、まだ従っていない接点を適用状況の列が名指す。
この形なら、決定は現在の事実であり、差分は観測できる事実になる。
`DOCUMENTATION_GUIDE.md` §2 の「現在の目標、設計上の仮定、実装済みの構成を区別する」を、列で区別する形に落とす。

観点の抜けの点検には、[Zalando RESTful API Guidelines](https://opensource.zalando.com/restful-api-guidelines/) と [Future Architect の Web API ガイドライン](https://future-architect.github.io/arch-guidelines/documents/forWebAPI/web_api_guidelines.html) を通す。
両者を通したうえで、IdMagic が採らない観点（HAL や JSON:API のハイパーメディア、GraphQL、`Accept-Language` によるエラー本文の切り替えなど）は、採らない決定として理由付きで残す。
既に `DOCUMENTATION_GUIDE.md` §11 が Zalando、Azure、Google AIP、RFC 9110、RFC 9457、RFC 8594 を出典として挙げているので、参照先を新しく増やすのではなく、挙がっている出典の観点を全部通したかどうかを点検の基準にする。

決めるべき論点のうち、書く前に結論を出すものを挙げる。

- **識別子とプロパティの記法**。現行の管理 API は `snake_case` のプロパティを返す。OAuth 2.0 と SCIM がそれぞれ別の記法を定めるため、記法は接点ごとに決まる。この事実を規則として書く。
- **日時と期間**。既存の契約が RFC 3339 の日時を使うことを確認し、期間を秒の整数で表すか ISO 8601 の期間で表すかを、現行の `retry_after_seconds` などから決める。
- **冪等キー**。`POST` を使う経路のうち再送で二重作用が起きるものを洗い出し、冪等キーを受け付ける規則を書くか、経路ごとの再送防止で足りるとする決定を書く。現行に `Idempotency-Key` の受け付けは無いため、後者ならその理由を残す。
- **条件付きリクエスト**。`ETag` と `If-Match` は [[wi-250-scim-sort-and-etag]] が SCIM について扱う。管理 API 全体へ広げるかを決め、広げないなら更新の取りこぼしを何が防ぐかを書く。
- **長時間かかる操作**。現行は [Jobs](../../docs/domain/jobs/README.md) が非同期処理を持ち、管理 API には操作リソースがない。Google AIP-151 の `Operation` 相当を採るか、ジョブ資源をそのまま公開するかを決める。
- **レート制限の伝え方**。`EndpointRateLimitPolicy` の 429 が `retry_after_seconds` と `Retry-After` を返すことは決まっている。残枠を伝えるヘッダーを持つかどうかを決める。

採らない案は、新しい「API ガイドライン」文書を作って `api-guidelines.md` を残す案である。
同じ観点の正本が二つになり、片方だけが更新される。

## Plan

1. 現行の八つの節を四つの列へ分解し、意味が落ちないことを確認する。
2. `DOCUMENTATION_GUIDE.md` §4.5 の観点表と二つの外部ガイドラインを通し、未記述の観点を列挙する。
3. 上に挙げた論点を、現行の契約と実装を観測して決める。
4. 観点ごとの節として書き直し、採らない観点を決定として残す。
5. この文書を名指している記述の参照とアンカーを追従させる。

## Tasks

- [x] T001 [Design] 現行の規則を四つの列へ分解する。
- [x] T002 [Design] 観点の抜けを外部ガイドラインで点検し、未記述の観点を列挙する。
- [x] T003 [Decision] 記法、値の表現、冪等キー、条件付きリクエスト、長時間操作、レート制限の各論点を決める。
- [x] T004 [Docs] 未記述の観点を規則として書く。
- [x] T005 [Docs] 既存の八つの節を表中心へ組み直す。
- [x] T006 [Docs] 採らない観点を決定として残す。
- [x] T007 [Docs] 参照元のリンクとアンカーを追従させる。
- [x] T008 [Verify] リンク、仕様、契約の差分検査、全体検証を通す。

## Verification

- `mise run check-links`
- `mise run check-spec`
- `mise run check-status-drift`
- `mise run check-contract-drift`
- `mise run verify`

## Risk Notes

観点を埋めるために、実装が従っていない規則を現在の規則として書く危険がある。適用状況の列を必ず埋め、未適用の接点を名指す。列が空の行を残さない。

外部ガイドラインの観点をそのまま輸入すると、IdMagic に存在しない問題への規則が増える。各観点について、対応する接点がこのプロダクトに実在するかを先に確かめ、実在しないものは採らない決定として一行で残す。

既存の節を表へ組み直すときに、散文が持っていた条件や例外が落ちる。文字列長と状態コードの節は特に条件が多いため、組み直した後に元の節と突き合わせる。

節の再構成はアンカーを変える。`docs/domain/glossary.md`、`docs/design/data/database.md`、各 Context の文書がこの文書の節を名指しているため、`mise run check-links` で確かめる。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  API ガイドラインが、`DOCUMENTATION_GUIDE.md` §4.5 の全観点をルール単位の小見出しで持ち、各ルールに目的、担保手段、適用状況を記載するようになった。
  当初は 4 列の表で書いたが、ルール本文が長く表では横に伸びて読めなかったため、小見出しと 3 項目の箇条書きに改めた。「接点」「強制点」「ワイヤ本体」などの造語も、「API 区分」「担保手段」「HTTP ボディの宣言」などの用語に改めた。
  リソースの命名、値の表現、HTTP メソッドの割り当て、コレクション操作、冪等性と再試行、条件付きリクエスト、長時間実行操作、カスタムメソッド、上限とレートリミットを新しく書き、既存の 8 節は意味を変えずに組み直した。
  決めた論点は次のとおりである。
  パラメーターとプロパティはスネークケース、静的パスセグメントはケバブケース、日時は RFC 3339 の UTC、期間は単位付きの整数、列挙値は小文字のスネークケース、値の不在は省略で表す。
  管理 API の作成、長時間実行操作の開始、クレデンシャルを発行する `POST` は、任意指定の `Idempotency-Key` を受け付ける。自動化クライアントのタイムアウト後の再送で、管理されないクレデンシャルが二重に作成されることを防ぐためである。
  管理 API の更新は `ETag` と `If-Match` で競合を検出し、不一致を 412 で返す。`PUT` で全体を置換するセキュリティ設定が、同時編集で気付かれないまま失われることを防ぐためである。
  長時間実行操作は、AIP-151 の汎用 `Operation` ではなく、操作ごとのリソースを 202 で返す。
  レートリミットの残量ヘッダーは返さない。制限の対象が認証とトークン発行の経路であり、残量は攻撃者に制限を回避する送信間隔を教えるためである。
  実装が従っていない箇所は適用状況に記載し、すべてに work item を起票した。
  [[wi-593-kebab-case-static-path-segments]]、[[wi-594-lower-snake-case-enum-values]]、[[wi-595-declare-uniqueness-conflict-responses]]、[[wi-596-paginate-remaining-collections-with-shared-cursor]]、[[wi-597-declare-pagination-response-headers]]、[[wi-598-declare-audit-event-query-parameters]]、[[wi-599-decode-admin-request-bodies-strictly]]、[[wi-600-bound-relation-tuple-write-batch-size]]、[[wi-601-declare-date-format-for-attribute-values]]、[[wi-602-carry-deprecation-dates-into-runtime-contract]]、[[wi-603-declare-interface-stability-in-typespec]]、[[wi-604-accept-idempotency-key-on-admin-post]]、[[wi-605-detect-lost-updates-with-etag-and-if-match]]、[[wi-606-verify-null-omission-in-responses-and-patch]] である。
  コードのコメントが英語の旧節名で指していた箇所を、現在の見出しに合わせた。
  外部ガイドラインのページは取得しておらず、観点の点検は §4.5 の観点表と §11 が挙げる出典の既知の観点で行った。
  `mise run spec-diff` は規範の変更を報告しない。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 設計文書の書き直しであり、観点の抜けを拒否する検査はない
  - **Requirement**: N/A: 規範のプロダクト要求を持たない
  - **Observed Failure**: 書き直す前の文書に対して、§4.5 の観点表 13 項目のうち、リソースの命名、値の表現、HTTP メソッドの割り当て、再送と重複防止、条件付きリクエスト、長時間かかる操作、CRUD に収まらない操作に対応する節がなかった
  - **Detection Reason**: 観点ごとに節を持つので、観点表と見出しを突き合わせれば抜けが分かる
- **Unit RED Evidence**:
  - **Test**: N/A: コードの振る舞いを変えない
  - **Requirement**: N/A: 規範のプロダクト要求を持たない
  - **Observed Failure**: N/A: 単体の境界がない
  - **Detection Reason**: N/A: 単体の境界がない
- **Change-Resistance Results**:
  文書だけの変更であり、変異テストの対象になるコードはない。
  コメントの変更は `mise run lint-go` と `mise run verify` で確認した。
- **Verification Results**:
  - `mise run check-links` - passed（833 文書）
  - `mise run check-terminology` - passed（224 文書）
  - `mise run check-spec` - passed
  - `mise run check-work-item-references` - passed
  - `mise run check-status-drift` - passed（0 件。342 操作のうち 83 を完全に、258 を部分的に読んだ）
  - `mise run check-contract-drift` - passed（0 件。184/342 操作を比較）
  - `mise run lint-go` - passed
  - `mise run verify` - passed
  - `mise run spec-diff` - 規範の変更なし
