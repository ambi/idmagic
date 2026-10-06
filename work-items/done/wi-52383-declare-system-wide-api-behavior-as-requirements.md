---
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-10-06
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 汎用 API の応答とヘッダーは変わらないが、利用者が依存してよい共通の振る舞いを新しい規則として約束するので、リリースの読者へ規則の所在を知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-52383-declare-system-wide-api-behavior-as-requirements.md }
initial_context:
  specification:
    - docs/design/application/api-guidelines.md
    - docs/design/security/authorization.md
    - docs/domain/system/README.md
    - docs/domain/system/api-boundary/README.md
    - docs/domain/system/localization/README.md#REQ-SYSTEM-013
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-005
    - docs/domain/api-tokens/authentication/README.md#REQ-APITOKENS-004
    - docs/domain/identity-management/admin-access/README.md
    - docs/domain/audit/event-search/README.md#REQ-AUDIT-004
  typespec: [IdMagic.Contract.ProblemDetails]
  source:
    - backend/shared/http/support_http/problem.go
    - backend/shared/http/support_http/error_handler.go
    - backend/shared/http/support_http/pagination.go
    - backend/shared/http/support_http/pagination_request.go
    - backend/shared/http/support_http/security_headers.go
    - backend/shared/http/support_http/response.go
    - backend/shared/http/support_http/admin_scope.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/spec/length.go
  tests:
    - backend/shared/http/support_http
    - backend/shared/spec/length_test.go
    - backend/cmd/internal/bootstrap/hardening_test.go
  stop_before_reading: [frontend, spec/contexts]
affected_spec:
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-022 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-023 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-024 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-025 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-026 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-027 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-028 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-029 }
  - { path: docs/domain/system/http-api/README.md, requirement: REQ-SYSTEM-030 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-004 }
---

# すべての API に共通する外部の振る舞いを、規範 ID を持つ要件として宣言する

## 動機

`REQ-*` を宣言できるのは機能スライスの仕様だけである（`SPECIFICATION_FORMAT.md` の仕様の木）。
一方、複数の API 操作に共通する外部の振る舞いは、`docs/design/application/api-guidelines.md` と `docs/design/security/authorization.md` の散文に書かれている。
ページサイズの既定値と上限、エラーレスポンスの形式、冪等キーの扱い、ページングのヘッダー、テナント境界での拒否などがこれにあたる。

これらは外部から観測できる振る舞いなのに、規範要素になっていない。
そのため、次の仕組みがどれも働かない。

| 仕組み | 働かない理由 |
| --- | --- |
| `mise run spec-diff` | REQ、EX、標準仕様の行、状態遷移、TypeSpec だけを規範要素として読む |
| 仕様影響の宣言 | すべての API のページサイズの既定値を変えるコミットに `Spec-Impact: none` を付けても、仕様差分がないので矛盾にならない |
| `//spec:covers` | 引く ID がないので、どのテストがその規則を固定しているかをたどれない |
| 上位の要件の欄 | 機能仕様の例外は見出しへのリンクで上位を指すが、上位の側に規範 ID がない |

## 対象範囲

- `api-guidelines.md` と `authorization.md` の規則を、外部から観測できる振る舞いと、設計の規則（目的、担保手段、適用状況）に分ける。
- 外部から観測できる振る舞いを、共有の仕組みの機能スライスに `REQ-*` として宣言する。
  宣言するのは、[仕様として書く実装上の細部](../../SPECIFICATION_FORMAT.md#仕様として書く実装上の細部)の基準を満たし、すでに文書に書かれている規則だけとする。コードを読んで規則を新しく書き足すことはしない。
- `api-guidelines.md` と `authorization.md` は、宣言した要件へリンクし、目的と担保手段と適用状況を書く文書として残す。
- 既存の機能仕様の「上位の要件」の欄のうち、`api-guidelines.md` の見出しを指すものを、新しい要件の見出しへ付け替える。
- 宣言した要件を引くテストを `//spec:covers` で結ぶ。テストがない要件は、テストを追加する。

## 対象外

- `docs/design/application/user-interface.md` の規則。
  画面を跨ぐ表示の規則は、フロントエンドの仕様の扱いと合わせて別に判断する。
- 規則の内容の変更。
  現在の振る舞いを要件として書き、意図と異なる挙動は別の work item として起票する。
- 未適用の API 操作を規則に合わせること。
  適用状況の未適用の行は、既存の work item（wi-596、wi-597、wi-604、wi-605 など）が扱う。
- 別のテナントで発行したトークンを 401 `invalid_token` で拒否する要件が、Application、Saml、Sourcing、Authentication、Provisioning、WsFederation、OAuth2 の各機能に同じ内容で重複していること。
  一か所へ集めると各機能の要件の廃止を伴うため、この記録では扱わない。
- `REQ-PLATFORM-005` を System の機能スライスへ移すこと。
  移すと ID の接頭辞が変わり、参照済みの ID を変えることになる。

## 設計

### 宣言する場所

[仕様の木](../../SPECIFICATION_FORMAT.md#仕様の木)の「実装が共有の仕組みであれば、その仕組みを一つの機能として機能スライスを設ける」に従う。
汎用 API の共通の振る舞いを実装する `backend/shared/http/support_http` は System の担当なので、System の下に機能スライス `docs/domain/system/http-api/` を置く。
コードのディレクトリ `backend/system/httpapi/` は存在しないので、System の設計の構成要素の表に、実装するコードを書く。

認可の要件の置き場所（計画で唯一の未解決の問い）は、新しい機能スライスを設けないと決めた。
`authorization.md` の規則のうち外部から観測できるものは、次の分類のとおり既存の機能スライスがすでに宣言している。
共有の仕組みが実装していて未宣言なのは、ページ送りのカーソルのテナントへの束縛だけであり、これは `http-api` の REQ-SYSTEM-025 に置く。

### 規則の分類

分類の記号は次のとおりとする。

| 記号 | 意味 | 扱い |
| --- | --- | --- |
| 新規 | 外部から観測でき、共有の仕組みが実装し、どの要件も宣言していない | `http-api` に要件を宣言し、システム文書からリンクする |
| 宣言済み | 外部から観測でき、既存の要件が宣言している | システム文書から既存の要件へリンクする |
| 契約 | API 契約の書き方の規則。結果は TypeSpec の各操作の宣言として観測される | 要件にしない。一次情報は TypeSpec |
| 設計 | 方式の選択、内部の仕組み、開発の手順 | 要件にしない |
| 未適用 | 規則に従う振る舞いがまだ存在しない | 要件にしない。現在の振る舞いがないので書き起こせない |

`api-guidelines.md` の規則の分類は次のとおりである。

| 規則 | 分類 | 扱い |
| --- | --- | --- |
| API 区分 | 設計 | 区分の定義 |
| パスの構成、静的パスセグメントの記法、パラメーターとプロパティの記法 | 契約 | TypeSpec のルートとモデル |
| 日時と日付、期間、数値、列挙値、識別子、値の不在 | 契約 | TypeSpec の型。`PATCH` の省略の扱いは各操作の契約 |
| GET の安全性、PUT の冪等性、PATCH のメディアタイプ、作成のレスポンス、URI をキーとするリソース | 契約 | TypeSpec のメソッドと応答 |
| ステータスコードの宣言、400 と 422 の区別、一意性違反と状態の競合、401 と 403 | 契約 | 状態コードとエラーの型は TypeSpec、返す条件は各機能の要件 |
| エラーレスポンスの形式 | 新規 | REQ-SYSTEM-022 |
| 宣言しないステータスコードの 500 | 新規 | REQ-SYSTEM-023 |
| 宣言しないステータスコードの 404 `tenant_not_found` | 宣言済み | REQ-TENANCY-006 |
| 宣言しないステータスコードの 503 `service_overloaded` | 宣言済み | REQ-SYSTEM-018 |
| 標準準拠の例外、独自形式のエラー | 契約 | TypeSpec のエラーのモデル |
| エラーメッセージの言語 | 宣言済み | REQ-SYSTEM-013 |
| ページング方式 | 設計 | 方式の選択。適用状況は未適用の行のまま |
| カーソル | 新規 | REQ-SYSTEM-025 |
| ページングのレスポンスヘッダー | 新規 | REQ-SYSTEM-026。TypeSpec への宣言は契約で、未適用のまま |
| ページサイズ | 新規 | REQ-SYSTEM-024。監査イベントの例外は REQ-AUDIT-004 |
| フィルタリング、ソート、部分レスポンス | 契約 | TypeSpec のクエリパラメーター |
| 冪等キー | 未適用 | 汎用 API は `Idempotency-Key` を受け付けない |
| 自然キーによる重複の拒否 | 宣言済み | 各機能の作成の要件 |
| 非同期処理の冪等性 | 設計 | 内部の仕組み |
| 再試行の指示 | 宣言済み | REQ-SYSTEM-018、REQ-OAUTH2-040、REQ-AUTHENTICATION-007、REQ-SYSTEM-030 |
| ETag と If-Match による競合検出 | 未適用 | 汎用 API の更新は `If-Match` を受け付けない |
| ドメインのリビジョン | 宣言済み | IdGovernance のライフサイクルワークフローの要件 |
| キャッシュ検証 | 宣言済み | REQ-TENANCY-034 |
| SCIM のバージョン | 宣言済み | Sourcing の `standards.md` |
| 長時間実行操作の 4 規則、カスタムメソッドの 3 規則 | 契約 | TypeSpec の操作と応答 |
| リクエストボディのサイズ | 新規 | REQ-SYSTEM-027。JSON の 64 KiB は REQ-PLATFORM-005 |
| 配列の要素数 | 契約 | TypeSpec の `@maxItems` |
| CSV の一括転送 | 宣言済み | REQ-IDMANAGEMENT-037 |
| レートリミットの適用範囲 | 宣言済み | REQ-OAUTH2-040、REQ-AUTHENTICATION-007 |
| レートリミットの残量ヘッダー | 新規 | REQ-SYSTEM-030 |
| 計数単位、上限違反のエラー | 新規 | REQ-SYSTEM-028。プロトコルのエラーで返す区分は各機能の要件 |
| 上限値の区分、インデックスキーの構成カラム、各箇所の役割、CHECK 制約の範囲 | 設計 | 上限の値は TypeSpec の `@maxLength` |
| 共通のセキュリティヘッダー、HSTS、CSP の段階的な強化 | 新規 | REQ-SYSTEM-029 |
| インラインスクリプト | 設計 | 自動送信の応答を返す Saml と WsFederation の仕組み |
| CSP の設定主体 | 設計 | 担当の配置 |
| 安定性区分の宣言、後方互換性、URI バージョニング、破壊的変更の検出 | 設計 | 契約の運用と検査の手順 |
| 未知のプロパティ | 宣言済み | REQ-PLATFORM-005 |
| API アクセストークンのスコープ宣言 | 宣言済み | REQ-APITOKENS-004 |
| 非推奨の宣言 | 宣言済み | REQ-SYSTEM-014 |
| HTTP ボディの宣言、列挙値と任意の値の宣言 | 契約 | TypeSpec の `@body` と enum |
| 採用しない方式 | 設計 | 方式の選択 |

`authorization.md` の規則の分類は次のとおりである。

| 規則 | 分類 | 扱い |
| --- | --- | --- |
| 主体の種類 | 宣言済み | 管理 API が受け入れる主体は REQ-IDMANAGEMENT-014 と REQ-APITOKENS-004 |
| スコープの語彙、account リソースサーバーの `audience` の検査 | 契約、宣言済み | 語彙は TypeSpec の `ApiTokenScope`、`audience` の検査は OAuth2 の `standards.md` の `RFC9068-DEFAULT-AUDIENCE` |
| 対話セッション限定の操作 | 宣言済み | REQ-APITOKENS-004 |
| テナント境界の認証 | 宣言済み | テナントの解決は REQ-TENANCY-006 から REQ-TENANCY-009、別テナントのトークンの拒否は各機能の要件 |
| テナント境界の認可 | 宣言済み | REQ-IDMANAGEMENT-014 |
| テナント境界の実行 | 設計 | Jobs の実行コンテキストの仕組み |
| テナント境界のページング | 新規 | REQ-SYSTEM-025 |
| テナント境界のレスポンス | 宣言済み | 各機能の参照と更新の要件 |
| テナントを越える操作、`system_admin` の割り当て、テナント解決 | 宣言済み | REQ-SYSTEM-020、IdManagement のロールの要件、REQ-TENANCY-009 |
| 判断不能なときの拒否、権限の有無を漏らさない、権限の伝播 | 宣言済み | 各機能の拒否の要件（例：REQ-OAUTH2-040 の共有カウンターへ到達できない場合） |
| CSRF と `Origin` の検証 | 宣言済み | REQ-PLATFORM-004 |
| UI での非表示は認可判定ではない | 設計 | 認可の置き場所の原則 |
| レスポンスが決して含まないもの | 宣言済み | 各機能の参照の要件 |

### 要件の差分

`docs/domain/system/http-api/README.md` に次の要件を追加する。

- REQ-SYSTEM-022 汎用 API の拒否は RFC 9457 の Problem Details で返る
  - 汎用 API の操作がレートリミット以外の理由で要求を拒否するとき、System は、`Content-Type: application/problem+json` で、`type`、`title`、`status`、`detail`、`instance` を持つ本文を返す。
  - 汎用 API の操作が要求を拒否するとき、System は、`type` に `urn:idmagic:error:` の後にエラーコードを続けた URN を、`status` に応答の状態コードを入れる。
  - 汎用 API の操作が要求を拒否するとき、System は、`instance` に要求の相関 ID を入れる。
- REQ-SYSTEM-023 汎用 API は、エラーコードを割り当てていない失敗を、内容を明かさない 500 で返す
  - 汎用 API の操作がエラーコードを割り当てていないエラーで失敗した場合、System は、500 と `internal_server_error` の Problem Details で応答し、元のエラーの文を `detail` に含めない。
- REQ-SYSTEM-024 汎用 API のコレクションのページサイズは、デフォルト 50 件、最大 200 件である
  - 署名付きカーソルでページングするコレクションを一覧したとき、System は、`limit` の件数（デフォルト 50 件、最大 200 件）まで返す。
  - 最大の件数を超える `limit` を受けたとき、System は、拒否せずに最大の件数まで返す。
  - 0 以下か整数でない `limit` を受けた場合、System は、400 と `invalid_request` で拒否する。
- REQ-SYSTEM-025 ページ送りのカーソルは、発行したテナントと絞り込みの外では使えない
  - 別のテナントで発行したカーソルか、異なる絞り込みで発行したカーソルを受けた場合、System は、400 と `invalid_request` で拒否する。
  - 署名と一致しないカーソルか、形式の壊れたカーソルを受けた場合、System は、400 と `invalid_request` で拒否する。
- REQ-SYSTEM-026 コレクションの一覧は、ページ送りの URL を `Link` ヘッダーで返す
  - 前後のページのあるコレクションを一覧したとき、System は、`rel="prev"` と `rel="next"` の URL を RFC 8288 の `Link` ヘッダーで返す。
  - ページ送りの URL を返すとき、System は、要求のクエリパラメーターを保ち、`cursor` だけを置き換える。
  - 総件数を返すコレクションを一覧したとき、System は、`Pagination-Total-Items`、`Pagination-Total-Pages`、`Pagination-Current-Page`、`Pagination-Page-Size` ヘッダーを返す。
  - 総件数を返すコレクションの先頭以外のページを一覧したとき、System は、`cursor` を除いた `rel="first"` の URL も返す。
  - 総件数を返すコレクションの末尾以外のページを一覧したとき、System は、`rel="last"` の URL も返す。
- REQ-SYSTEM-027 起動時設定の上限を超えるリクエストボディは 413 で拒否する
  - `HTTP_MAX_BODY_BYTES`（デフォルト 1 MiB）を超えるリクエストボディを受けた場合、System は、ハンドラーへ渡さずに 413 で拒否する。
- REQ-SYSTEM-028 汎用 API の文字列の長さはコードポイントで数え、上限を超えると 422 で拒否する
  - 汎用 API の要求の文字列のフィールドの長さを検証するとき、System は、UTF-8 のバイト数ではなく Unicode のコードポイントの数を数える。
  - 汎用 API の要求の文字列のフィールドが長さの上限を超える場合、System は、422 と `field_length_exceeded` で拒否し、`detail` に契約上のフィールド名と上限を示す。
- REQ-SYSTEM-029 バックエンドの応答はセキュリティヘッダーを伴う
  - 自動送信のフォーム以外の応答を返すとき、System は、`X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`、`X-Frame-Options: DENY` と、`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'` を方針とする `Content-Security-Policy` を付ける。
  - HSTS を有効にした構成では、System は、`max-age` に `HSTS_MAX_AGE_SECONDS` を入れた `Strict-Transport-Security` を付ける。
  - HSTS とサブドメインへの適用を有効にした構成では、System は、`Strict-Transport-Security` に `includeSubDomains` を加える。
  - HSTS を無効にした構成（デフォルト）では、System は、`Strict-Transport-Security` を付けない。
  - CSP の報告専用を有効にした構成では、System は、`Content-Security-Policy` の代わりに、同じ方針の `Content-Security-Policy-Report-Only` を付ける。
  - CSP の報告先を設定した構成では、System は、方針に `report-uri` と `CSP_REPORT_URI` の値を加える。
- REQ-SYSTEM-030 レートリミットの拒否は待機の秒数だけを示し、残量を示さない
  - レートリミットで要求を拒否するとき、System は、429 と、`error` に `rate_limited`、`retry_after_seconds` に待機の秒数、`message` に英語の文を持つ JSON を、同じ秒数の `Retry-After` ヘッダーとともに返す。
  - レートリミットを適用する経路で応答するとき、System は、`RateLimit`、`RateLimit-Policy`、`X-RateLimit-*` のヘッダーを付けない。

`REQ-AUDIT-004` は要件文を変えず、上位の要件の欄に REQ-SYSTEM-024 へのリンクを加える。
監査イベントのページサイズ（デフォルト 100 件、最大 999 件）は REQ-SYSTEM-024 の例外である。

### デフォルトのページサイズの所在

REQ-SYSTEM-024 の 50 件と 200 件は、現在は各ハンドラーが同じ値の定数を個別に持つ。
このままでは、テストで固定できるのは各ハンドラーの定数だけであり、要件の値をどのテストが固定するかをたどれない。
そこで、構造変更として `support_http.DefaultPageLimit = 50` と `support_http.MaxPageLimit = 200` を置き、値が同じハンドラーの定数をこれに置き換える。
例外の値を持つ監査イベントとインポートの行エラーの定数はそのまま残す。
`authentication/usecases` の認証イベントのバケットは、ユースケースの層から `support_http` を参照できないので定数を残す。
この構造変更は応答を変えないので、`Spec-Impact: none` のトレーラーで別のコミットにする。

### 採用しない案

| 案 | 採らない理由 |
| --- | --- |
| システム文書に別の ID 体系（`API-NNN` など）を作る | 検査、`spec-diff`、`spec-route`、work item の `affected_spec` をすべて二つ目の文法に対応させる必要がある。機能スライスに置けば既存の仕組みがそのまま働く |
| `api-guidelines.md` で `REQ-*` の宣言を許す | 「要件は機能スライスの仕様でだけ宣言する」という検査済みの規則を崩し、要件の置き場所の判断がまた文書ごとに分かれる |
| 散文のまま、テストに見出しの名前を書く | 検査できず、仕様影響の宣言とも結び付かない |
| 契約の規則（命名、値の表現、メソッドの割り当て）も要件にする | 結果は TypeSpec の各操作の宣言として観測され、一次情報は TypeSpec である。要件文にすると同じ事実を二か所に書く |
| 認可の要件のために `tenant-boundary` の機能スライスを設ける | 観測できる規則は各機能がすでに宣言しており、共有の仕組みが実装する未宣言の規則はカーソルの束縛だけである |

## 計画

1. `api-guidelines.md` と `authorization.md` の各規則を、外部の振る舞いと設計の規則に分類し、この work item に一覧を書く。
2. 機能スライスの置き場所を決め、要件の差分（追加する要件と要件文）をこの work item に書く。
3. デフォルトのページサイズの定数を `support_http` へ集める構造変更を、別のコミットにする。
4. 要件を宣言し、システム文書からリンクする。
5. 上位の要件の欄を付け替える。
6. 要件を引くテストを結び、ないものは追加する。

## タスク

- [x] T001 [Plan] 二つの文書の規則を分類した一覧と、要件の差分をこの work item に書く。
- [x] T002 [Refactor] `support_http.DefaultPageLimit` と `MaxPageLimit` を置き、値が同じハンドラーの定数を置き換える。代替検査：`mise run test-go-changed` が GREEN のまま通ること。
- [x] T003 [Spec] 機能スライスを作り、要件を宣言する。システム文書を要件へのリンクと設計の規則に書き直す。検査：`mise run check-spec`。
- [x] T004 [Spec] 既存の機能仕様の上位の要件の欄を付け替える。`api-guidelines.md` の見出しを指す上位の要件の欄は存在しないので、例外である REQ-AUDIT-004 に上位の要件の欄を加える。
- [x] T005 [Acceptance] 宣言した要件を引くテストを結び、ないものは追加して、挙動を変える誤実装の注入で失敗することを確認する。単体の検査：`mise run test-go-package -- ./backend/shared/http/support_http`、`./backend/shared/spec`、`./backend/cmd/internal/bootstrap`。
- [x] T006 [Docs] `SPECIFICATION_FORMAT.md` の横断的要件の節（「システム全体にかかる要件は api-guidelines.md などのシステム文書に置く」）を、この置き場所に合わせて書き直す。
- [x] T007 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run spec-diff`
- `mise run check`
- `mise run verify`

## リスク

- 割り当てた規範 ID は取り消せない。
  規則の単位を決めてから ID を割り当てる。
- 現在の偶然の挙動を仕様として固定する危険がある。
  維持する義務のない細部は要件にしない。
  空の `instance` の省略、`HEAD` の本文の省略、`title` の生成規則は (b) として要件にしない。

## 完了

- **Completed At**: 2026-10-07
- **Summary**:
  `mise run spec-diff` の結果は、REQ-SYSTEM-022 から REQ-SYSTEM-030 の追加と、REQ-AUDIT-004 の変更である。
  汎用 API に共通する振る舞いを、新しい機能スライス「HTTP API の共通の振る舞い」の要件として宣言した。
  宣言したのは、Problem Details の形式、エラーコードのない失敗の 500、ページサイズのデフォルトと上限、カーソルのテナントと絞り込みへの束縛、ページ送りの `Link` と `Pagination-*` ヘッダー、`HTTP_MAX_BODY_BYTES` による 413、コードポイントで数える文字列長と 422、セキュリティヘッダー、レートリミットの拒否の形式である。
  REQ-AUDIT-004 は、要件文を変えずに上位の要件の欄で REQ-SYSTEM-024 の例外として位置付けた。
  `api-guidelines.md` と `authorization.md` は、宣言済みの要件へリンクし、契約の書き方、目的、担保手段、適用状況を書く文書になった。
  デフォルトのページサイズは `support_http.DefaultPageLimit` と `MaxPageLimit` へ集め、値が 50 件と 200 件の 8 つのコレクションのハンドラーがこれを参照する。
  `SPECIFICATION_FORMAT.md` は、システム全体にかかる要件も共有の仕組みの機能スライスで宣言すると定めた。
- **Acceptance RED Evidence**:
  - **Test**: `TestParsePageRequestRejectsCursorFromDifferentQuery`（`backend/shared/http/support_http`）
  - **Requirement**: REQ-SYSTEM-025
  - **Observed Failure**: 異なる絞り込みのカーソルの照合を無効にする誤実装を注入しても、テストは GREEN のままだった。テストのカーソルの keyset が base64 として壊れており、照合より前に別の理由で拒否されていたためである。keyset を `joinKeyset("x", "y")` で正しく作ると、同じ注入でテストが失敗した。
  - **Detection Reason**: 照合以外の理由で拒否される入力では、照合の欠落と正しい実装を区別できない。keyset を正しくしたことで、拒否の理由が絞り込みの不一致だけになった。
- **Unit RED Evidence**:
  - **Test**: `TestParsePageRequestAppliesSharedPageSize`（追加したほかの 3 つのテストは Change-Resistance Results に記す）
  - **Requirement**: REQ-SYSTEM-024
  - **Observed Failure**: N/A: 既存の挙動を書き起こす作業なので、本番コードを変える前に失敗するテストはない。代替検査として、下の故障注入で各テストが失敗することを観測した。
  - **Detection Reason**: 要件の値（50 件、200 件、422、first と last の省略、残量のヘッダーの不在）を変える注入を、それぞれのテストが検出した。
- **Change-Resistance Results**:
  一つずつ注入し、対象のテストを `mise run test-go-test` で実行して、注入を戻した。
  - REQ-SYSTEM-022：`type` の URN の接頭辞を変える → `TestWriteProblem_RFC9457Fields` が検出した。
  - REQ-SYSTEM-023：元のエラーの文を `detail` へ写す → `TestErrorHandler_FallbackWritesProblemDetailsForPlainError` が検出した。
  - REQ-SYSTEM-024：デフォルトを 51 にする、上限への丸めを外す、0 を受け付ける → `TestParsePageRequestAppliesSharedPageSize` がいずれも検出した。
  - REQ-SYSTEM-025：v3 の署名からテナントを外す、絞り込みを外す → `TestV3CursorRejectsTamperTenantQueryAndUnknownVersion` が検出した。旧形式の絞り込みの照合を外す → 上の Acceptance RED のとおり、テストを直した後に検出した。旧形式のテナントの照合を外す注入は、v3 のカーソルを使うテストでは到達しないので生き残った。旧形式のカーソルは移行のために読むだけで、新たに発行しない。
  - REQ-SYSTEM-026：`cursor` 以外のクエリパラメーターを落とす、末尾のページでも `last` を返す、先頭のページでも `first` を返す、`Pagination-Page-Size` を消す → 対応するテストがいずれも検出した。
  - REQ-SYSTEM-027：`HTTP_MAX_BODY_BYTES` のデフォルトを 2 MiB にする → `TestLoadHTTPServerHardeningDefaults` が検出した。`TestBodyLimitEnforcesConfiguredMax` はミドルウェアを直接構築するため、起動処理の配線を外す故障は検出できない。
  - REQ-SYSTEM-028：文字列長の違反を 400 にする → `TestErrorHandlerWritesFieldLengthViolationAs422`、上限の判定をバイト数にする → `TestCharsCountsCodePointsNotBytes` が検出した。
  - REQ-SYSTEM-029：`X-Frame-Options` を `SAMEORIGIN` にする → `TestSecurityHeadersMiddleware_DefaultsAreSecure` が検出した。
  - REQ-SYSTEM-030：拒否に `X-RateLimit-Remaining` を加える → `TestRateLimitDisclosesOnlyRetryDelay` が検出した。
  本番コードの変更は定数の集約だけなので、`test-go-mutation` は実行せず、定数の値を変える注入で代えた。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-links` - 成功
  - `mise run check-work-items` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
