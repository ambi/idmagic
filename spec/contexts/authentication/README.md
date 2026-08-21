# Authentication

エンドユーザーの資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、アカウントの復旧、ログイン時のフェデレーション、認証イベントを担う。

`User` / `Group` / `Agent` のライフサイクルそのものは `IdManagement` が担う。この Context が扱うのは、そのプリンシパルが本人であることをどう確かめ、確かめた結果をどうセッションとして保つかである。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
