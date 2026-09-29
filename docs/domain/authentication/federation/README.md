# 外部 IdP との連携

テナント単位の上流 OIDC / SAML 接続、外部 subject とローカル User の関連付け、関連付けと JIT のポリシー、ログインセッションへの引き渡しを扱う。
下流向けの SAML IdP と WS-Federation の発行は、各プロトコルの Context が扱う。
コードの機能スライスは `backend/authentication/federation` である。

| 文書 | 内容 |
|---|---|
| [外部 IdP との連携の状態遷移](states.md) | 状態と遷移 |
| [外部 IdP との連携の設計判断](decisions.md) | 設計判断 |
| [外部 IdP との連携の内部設計](internals.md) | 機構の説明 |
| [外部 IdP との連携のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
