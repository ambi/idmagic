# 多要素認証

第二要素を求める条件、MFA の強制と登録専用フロー、管理者による認証器のリセットを扱う。
個々の認証要素の登録と照合は TOTP、WebAuthn、復旧コードの各機能が扱う。
コードの機能スライスは `backend/authentication/mfa` である。

| 文書 | 内容 |
|---|---|
| [多要素認証の設計判断](decisions.md) | 設計判断 |
| [多要素認証の内部設計](internals.md) | 機構の説明 |
| [多要素認証のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
