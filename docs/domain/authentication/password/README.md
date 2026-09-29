# パスワード

パスワードのポリシー、変更、有効期限、リセットの要求と引き換えを扱う。
パスワードによるログインそのものは複数の機能をまたぐので、Authentication のルートが扱う。
コードの機能スライスは `backend/authentication/password` である。

| 文書 | 内容 |
|---|---|
| [パスワードの設計判断](decisions.md) | 設計判断 |
| [パスワードの内部設計](internals.md) | 機構の説明 |
| [パスワードのシナリオ](scenarios.feature.md) | 受け入れシナリオ |
