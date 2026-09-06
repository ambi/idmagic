# WI-454: Declare every 403 body written by request guards

作業項目は `wi-454-declared-403-body-vs-guard-error-codes` である。

WI-454 は、管理 API の 403 契約に `access_denied` だけでなく、API アクセストークンの `insufficient_scope` と、ブラウザー要求の `invalid_origin` / `csrf_failed` を実装どおり宣言する。

同じ `insufficient_scope` でも、管理 API は RFC 9457 Problem Details、UserInfo は OAuth エラー本文を返すため、契約モデルを分離する。実行時の拒否条件とワイヤー本文は変わらない。

規範上の条件は [REQ-APITOKENS-004](../../contexts/api-tokens/scenarios.feature.md#rule-req-apitokens-004-管理-api-は-api-アクセストークンの粒度スコープでフェイルクローズに認可する) と [REQ-PLATFORM-004](../../scenarios.feature.md#rule-req-platform-004-周囲資格情報による状態変更は同一オリジンと-csrf-トークンを必要とする) が定める。
