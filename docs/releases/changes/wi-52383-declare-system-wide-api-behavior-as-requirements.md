# wi-52383-declare-system-wide-api-behavior-as-requirements

すべての汎用 API に共通する振る舞いを、[HTTP API の共通の振る舞い](../../modules/system/http-api/README.md)の規則として約束するようになった。
これまで API ガイドラインの散文だけに書かれていた挙動であり、振る舞いは変わらない。

- エラーの応答：RFC 9457 の Problem Details の形式と、エラーコードを割り当てていない失敗の 500（[`REQ-SYSTEM-022`](../../modules/system/http-api/README.md)、`REQ-SYSTEM-023`）
- コレクションの一覧：`limit` のデフォルト 50 件と最大 200 件、テナントと絞り込みに束縛したカーソル、`Link` と `Pagination-*` ヘッダー（[`REQ-SYSTEM-024`](../../modules/system/http-api/README.md)〜`REQ-SYSTEM-026`）
- リクエストの上限：`HTTP_MAX_BODY_BYTES` による 413、コードポイントで数える文字列の長さと 422 の `field_length_exceeded`（[`REQ-SYSTEM-027`](../../modules/system/http-api/README.md)、`REQ-SYSTEM-028`）
- 応答のヘッダー：セキュリティヘッダーと、HSTS と CSP の報告専用の構成（[`REQ-SYSTEM-029`](../../modules/system/http-api/README.md)）
- レートリミット：429 の本文と `Retry-After`、残量を示すヘッダーを返さないこと（[`REQ-SYSTEM-030`](../../modules/system/http-api/README.md)）

監査イベントの検索のページサイズ（[`REQ-AUDIT-004`](../../modules/audit/event-search/README.md)）は、デフォルトのページサイズの例外として位置付けた。
