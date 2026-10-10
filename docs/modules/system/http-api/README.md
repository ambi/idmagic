# HTTP API の共通の振る舞い

## 概要

この文書は、複数の API 操作に同じ内容でかかる HTTP の振る舞いの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | エラーの応答の形式、コレクションのページサイズとカーソルとページ送りのヘッダー、リクエストボディの大きさと文字列の長さの上限、セキュリティヘッダー、レートリミットの拒否の形式 |
| 行為者 | APIConsumer（汎用 API を呼ぶ呼び出し元） |
| 扱わないもの | 各操作のパス、パラメーター、状態コード、エラーの型は TypeSpec が、それを返す条件は各機能の要件が扱う。契約の書き方の規則は [API ガイドライン](../../../design/application/api-guidelines.md)が、経路の種類ごとの認可は [API の境界](../api-boundary/README.md)が扱う |

## モデル

| 概念 | 内容 |
| --- | --- |
| 汎用 API | 管理 API（`/api/admin/v1/`）、アカウント API（`/api/account/v1/`）、ブラウザー API（`/api/auth/`）。各標準が形式を定めるプロトコルエンドポイントと、運用エンドポイントは含めない |
| 署名付きカーソル | 発行したテナントと、絞り込みと並び順の同一性を束縛し、署名したページ送りの位置 |
| 総件数を返すコレクション | 絞り込みに一致する正確な件数を、ページとともに返すコレクション |

この機能に対応するコードのディレクトリはない。
実装する構成要素は [System のアーキテクチャ](../design/architecture.md#構成要素)が示す。

## 操作

### API の利用者による要求の拒否

#### REQ-SYSTEM-022 汎用 API の拒否は RFC 9457 の Problem Details で返る

- 汎用 API の操作がレートリミット以外の理由で要求を拒否するとき、System は、`Content-Type: application/problem+json` で、`type`、`title`、`status`、`detail`、`instance` を持つ本文を返す。
- 汎用 API の操作が要求を拒否するとき、System は、`type` に `urn:idmagic:error:` の後にエラーコードを続けた URN を、`status` に応答の状態コードを入れる。
- 汎用 API の操作が要求を拒否するとき、System は、`instance` に要求の相関 ID を入れる。
- **判断**：クライアントが処理するエラーの形式を一つにし、`type` のエラーコードから TypeSpec のエラーのモデルを特定できるようにする。

#### REQ-SYSTEM-023 汎用 API は、エラーコードを割り当てていない失敗を、内容を明かさない 500 で返す

- 汎用 API の操作がエラーコードを割り当てていないエラーで失敗した場合、System は、500 と `internal_server_error` の Problem Details で応答し、元のエラーの文を `detail` に含めない。
- **判断**：元のエラーの文には、依存先の名前や SQL の断片のような内部の情報が含まれ得る。

### API の利用者によるコレクションの一覧

#### REQ-SYSTEM-024 汎用 API のコレクションのページサイズは、デフォルト 50 件、最大 200 件である

- 署名付きカーソルでページングするコレクションを一覧したとき、System は、`limit` の件数（デフォルト 50 件、最大 200 件）まで返す。
- 最大の件数を超える `limit` を受けたとき、System は、拒否せずに最大の件数まで返す。
- 0 以下か整数でない `limit` を受けた場合、System は、400 と `invalid_request` で拒否し、一覧を返さない。
- **判断**：過大な `limit` は内容の誤りではなく、応答の大きさの問題なので、拒否せずに上限へ丸める。

#### REQ-SYSTEM-025 ページ送りのカーソルは、発行したテナントと絞り込みの外では使えない

- 別のテナントで発行したカーソルか、異なる絞り込みで発行したカーソルを受けた場合、System は、400 と `invalid_request` で拒否し、一覧を返さない。
- 署名と一致しないカーソルか、形式の壊れたカーソルを受けた場合、System は、400 と `invalid_request` で拒否し、一覧を返さない。
- **判断**：カーソルを別のテナントや条件を変えた後のクエリへ流用させない。

#### REQ-SYSTEM-026 コレクションの一覧は、ページ送りの URL を `Link` ヘッダーで返す

- 前後のページのあるコレクションを一覧したとき、System は、`rel="prev"` と `rel="next"` の URL を RFC 8288 の `Link` ヘッダーで返す。
- ページ送りの URL を返すとき、System は、要求のクエリパラメーターを保ち、`cursor` だけを置き換える。
- 総件数を返すコレクションを一覧したとき、System は、`Pagination-Total-Items`、`Pagination-Total-Pages`、`Pagination-Current-Page`、`Pagination-Page-Size` ヘッダーを返す。
- 総件数を返すコレクションの先頭以外のページを一覧したとき、System は、`cursor` を除いた `rel="first"` の URL も返す。
- 総件数を返すコレクションの末尾以外のページを一覧したとき、System は、`rel="last"` の URL も返す。
- **判断**：クライアントが URL を組み立て直さずにページを送れるようにし、応答の本文の構造を変えずにページングの情報を足せるようにする。

### API の利用者によるリクエストボディの送信

#### REQ-SYSTEM-027 起動時設定の上限を超えるリクエストボディは 413 で拒否する

- `HTTP_MAX_BODY_BYTES`（デフォルト 1 MiB）を超えるリクエストボディを受けた場合、System は、ハンドラーへ渡さずに 413 で拒否し、状態を変えない。
- **判断**：汎用 API の JSON のリクエストボディにかかる 64 KiB の上限は、[REQ-PLATFORM-005](../../../requirements/scenarios.feature.md) が定める。

#### REQ-SYSTEM-028 汎用 API の文字列の長さはコードポイントで数え、上限を超えると 422 で拒否する

- 汎用 API の要求の文字列のフィールドの長さを検証するとき、System は、UTF-8 のバイト数ではなく Unicode のコードポイントの数を数える。
- 汎用 API の要求の文字列のフィールドが長さの上限を超える場合、System は、422 と `field_length_exceeded` で拒否し、`detail` に契約上のフィールド名と上限を示し、対象を変えない。
- **判断**：バイト数で数えると、上限 100 の名前が英字なら 100 文字、日本語なら 33 文字になり、契約に書いた値が意味を失う。

### API の利用者への応答

#### REQ-SYSTEM-029 バックエンドの応答はセキュリティヘッダーを伴う

- 自動送信のフォーム以外の応答を返すとき、System は、`X-Content-Type-Options: nosniff`、`Referrer-Policy: no-referrer`、`X-Frame-Options: DENY` と、`default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'` を方針とする `Content-Security-Policy` を付ける。
- HSTS を有効にした構成では、System は、`max-age` に `HSTS_MAX_AGE_SECONDS` の値を入れた `Strict-Transport-Security` を付ける。
- HSTS とサブドメインへの適用を有効にした構成では、System は、`Strict-Transport-Security` に `includeSubDomains` を加える。
- HSTS を無効にした構成では、System は、`Strict-Transport-Security` を付けない。
- CSP の報告専用を有効にした構成では、System は、`Content-Security-Policy` の代わりに、同じ方針の `Content-Security-Policy-Report-Only` を付ける。
- CSP の報告先を設定した構成では、System は、方針に `report-uri` と `CSP_REPORT_URI` の値を加える。
- **判断**：HSTS のデフォルトは無効である。平文の `http` を使う開発環境に影響させず、TLS を終端する側が有効にする。
- **判断**：SAML と WS-Federation の自動送信のフォームの方針は、送信先と固定のスクリプトのハッシュだけを許す形で、それぞれの応答が上書きする。

### API の利用者によるレートリミットを超えた要求

#### REQ-SYSTEM-030 レートリミットの拒否は待機の秒数だけを示し、残量を示さない

- レートリミットで要求を拒否するとき、System は、429 と、`error` に `rate_limited`、`retry_after_seconds` に待機の秒数、`message` に英語の文を持つ JSON を、同じ秒数の `Retry-After` ヘッダーとともに返す。
- レートリミットを適用する経路で応答するとき、System は、`RateLimit`、`RateLimit-Policy`、`X-RateLimit-` で始まるヘッダーを付けない。
- **判断**：レートリミットを受ける相手には攻撃者も含まれ、残量を返すと制限に達しない送信の間隔を教えることになる。どの経路にどの閾値をかけるかは、[プロトコルエンドポイント](../../oauth2/protocol-endpoints/README.md)と[サインイン](../../authentication/sign-in/README.md)が定める。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| テナント境界 | カーソルは発行したテナントに束縛するので、別のテナントの一覧の位置を推測して読ませない |
| 情報の開示 | エラーコードを割り当てていない失敗は、内部のエラーの文を応答へ写さない。レートリミットの残量は返さない |
| 埋め込み | `frame-ancestors 'none'` と `X-Frame-Options: DENY` により、ログイン、同意、ポータルの画面の埋め込みを拒む |
