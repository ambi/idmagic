# WI-454: Update generated 403 error body types

作業項目は `wi-454-declared-403-body-vs-guard-error-codes` である。

生成クライアントで UserInfo の OAuth エラー本文型 `InsufficientScopeError` を直接参照している場合は、用途が明確な `OAuthInsufficientScopeError` へ型名を変更する。ワイヤー上の `error: insufficient_scope` は変わらない。

管理 API の 403 は、既存の `AccessDeniedError` に加えて `InsufficientScopeError`、`InvalidOriginError`、`CsrfFailedError` など到達可能な RFC 9457 Problem Details 型の union になる。網羅的な型分岐を持つ呼び出し側は、新しい union メンバーを処理する。実行時の拒否条件、設定、永続データの移行はない。

契約上の拒否条件は [REQ-APITOKENS-004](../../contexts/api-tokens/scenarios.feature.md#rule-req-apitokens-004-管理-api-は-api-アクセストークンの粒度スコープでフェイルクローズに認可する) と [REQ-PLATFORM-004](../../scenarios.feature.md#rule-req-platform-004-周囲資格情報による状態変更は同一オリジンと-csrf-トークンを必要とする) が定める。
