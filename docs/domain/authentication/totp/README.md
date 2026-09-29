# TOTP

TOTP 認証要素の登録、照合、解除を扱う。
どの場面で第二要素を求めるかは多要素認証が扱う。
コードの機能スライスは `backend/authentication/totp` である。

| 文書 | 内容 |
|---|---|
| [TOTPの設計判断](decisions.md) | 設計判断 |
| [TOTPの内部設計](internals.md) | 機構の説明 |
| [TOTPのシナリオ](scenarios.feature.md) | 受け入れシナリオ |
