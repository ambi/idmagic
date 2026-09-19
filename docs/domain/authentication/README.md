# Authentication

エンドユーザーの資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、アカウントの復旧、ログイン時のフェデレーション、認証イベントを担う。

`User` / `Group` / `Agent` のライフサイクルそのものは `IdManagement` が担う。この Context が扱うのは、そのプリンシパルが本人であることをどう確かめ、確かめた結果をどうセッションとして保つかである。

| 文書 | 内容 |
|---|---|
| [Authentication の用語集](glossary.md) | この Context での語義 |
| [Authentication の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Authentication の状態遷移](states.md) | 状態と遷移 |
| [Authentication の設計判断](decisions.md) | 設計判断 |
| [Authentication の内部設計](internals.md) | 機構の説明 |
| [Authentication のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
