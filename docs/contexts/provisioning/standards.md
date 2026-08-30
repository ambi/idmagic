# Provisioning Standards

## System for Cross-domain Identity Management Core Schema

RFC 7643 — https://www.rfc-editor.org/rfc/rfc7643.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7643-OUT-CORE-RESOURCES | partial | MUST | 下流へ送る User リソースの本文は、必須属性 `schemas` と、接続の属性対応付けが解決した属性で組み立てる。`schemas` は `urn:ietf:params:scim:schemas:core:2.0:User` の 1 要素で、対応付けの外側で載せる —— リソース表現の必須属性であり、管理者が編集できる表に置いて消せてよいものではない。載せるのは作成 (POST) と置換 (PUT) の本文だけで、PATCH の本文は `PatchOp` のメッセージ URN を持ち、その `Operations[].value` は部分断片なので持たない。既定の対応付けは `externalId`、`userName`、`active`、`name.givenName`、`name.familyName`、`displayName`、`emails[type eq "work"].value` である。`required` の対応付けが値を解決できない配信は、部分的な本文を送らずに失敗させる。 |
| RFC7643-OUT-EXTERNAL-ID | required | MUST | User の作成要求に、IdMagic 側の識別子を `externalId` として送る。更新要求では送り直さず、以後の相関は保存した対応関係が担う。 |
| RFC7643-OUT-GROUP-RESOURCES | partial | MUST | 接続の `push_groups` が有効なとき、Group を SCIM の Group リソースとして送る。本文は必須属性 `schemas` (`urn:ietf:params:scim:schemas:core:2.0:Group` の 1 要素) と、接続の属性対応付けが解決した属性で組み立てる。`displayName` の取得元は `GroupPushConfig.display_name_source` が選び、既定は Group の名前である。メンバーシップはリソース本文には載せず、`members` に対する増分 `add` の PATCH として別に送る。送るのは既に下流へ provision 済みのメンバーだけで、対応関係を持たないメンバーの識別子をこちら側で作ることはしない。メンバーの除去は送らない —— 下流の現在のメンバー集合を読み戻していないので、除くべき相手を知る手段が無く、全置換はこの接続が追加していないメンバーを消す。 |
| RFC7643-OUT-SCHEMA-EXTENSIONS | excluded | MUST NOT | 拡張スキーマの属性を送らない。属性対応付けの対象パスとして書けるのは `.` で区切った単純パスと 1 段の多値フィルターパスだけであり、拡張スキーマの URN で修飾したパスは表現できない。Enterprise 拡張 (`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`) も対象外である。 |

## System for Cross-domain Identity Management Protocol

RFC 7644 — https://www.rfc-editor.org/rfc/rfc7644.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-OUT-RESOURCE-OPERATIONS | partial | MUST | User リソースに対して、作成 (`POST /Users`)、更新 (`PATCH` または `PUT /Users/{id}`)、削除 (`DELETE /Users/{id}`)、および衝突解決のための照会 (`GET /Users`) を送る。要求は `Accept: application/scim+json` を持ち、本文を伴う要求は `Content-Type: application/scim+json` を持つ。削除に対する 404 は、目的の状態に既に到達しているものとして成功に数える。 |
| RFC7644-OUT-PATCH | optional | MUST | 更新を PATCH で送るのは、接続テストが取得した下流の `patch.supported` が真のときだけである。偽のとき、および対応可否を取得していないときは PUT でリソース全体を置換する。送る PATCH は、対象パスを持たない `replace` 操作 1 件だけで構成する。 |
| RFC7644-OUT-DISCOVERY | partial | MUST | 接続テストで取得するのは `/ServiceProviderConfig` だけであり、`patch`、`bulk`、`filter`、`etag`、`sort` の対応可否を接続に保存する。`/ResourceTypes` と `/Schemas` は取得せず、下流が広告するスキーマに送出内容を合わせることはしない。 |
| RFC7644-OUT-FILTERING | partial | MUST | 組み立てるフィルターは `<属性> eq "<値>"` という比較 1 つだけである。論理演算子、グループ化、複合値のブラケット構文は組み立てない。フィルターを使うのは、作成が 409 で衝突したときに照合属性で既存リソースを探す場面に限る。 |
| RFC7644-OUT-ERROR-RESPONSE | partial | MUST | 下流の応答のうち、409 は既存リソースへの関連付けとして、404 は再作成として、429 と 5xx は `Retry-After` に従う再試行として扱い、それ以外の 2xx 以外の応答は再試行しない失敗として扱う。SCIM のエラー本文からは `detail` だけを読み、`scimType` は解釈しない。 |
| RFC7644-OUT-AUTHENTICATION | required | MUST | 下流への要求は `Authorization: Bearer` ヘッダーで認証する。ヘッダーに載せる値は接続の認証方式が決める。`bearer_token` は保存した資格情報をそのまま載せ、資格情報を得るための別の要求は送らない。`oauth2_client_credentials` はクライアント資格情報フロー (RFC 6749 §4.4) で取得したアクセストークンを載せる —— 保存した `client_secret` をそのまま提示することはない。取得したトークンは失効の 60 秒手前まで再利用し、期限切れと下流の 401 で取り直す。401 による取り直しは 1 度だけで、2 度目も 401 なら失敗として扱う。 |
| RFC7644-OUT-BULK | excluded | MUST NOT | `/Bulk` を使わない。下流が `bulk.supported` を広告していても、配信 1 件につき 1 リソースの要求を送る。 |
| RFC7644-OUT-SORT | excluded | MUST NOT | 照会に `sortBy` と `sortOrder` を送らない。 |
| RFC7644-OUT-ETAG | excluded | MUST NOT | 条件付き要求を送らない。更新にも削除にも `If-Match` と `If-None-Match` を付けず、応答の `ETag` を後続の要求の前提条件に使わない。 |
