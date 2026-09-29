# サインイン履歴

本人が自分のサインイン履歴を参照する API を扱う。
履歴の元になる認証イベントの記録と保持は Authentication のルートが扱う。
コードに独立した機能スライスはなく、`backend/authentication/handlers_http` と `backend/authentication/usecases` が実装する。

| 文書 | 内容 |
|---|---|
| [サインイン履歴のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
