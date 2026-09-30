# テナントの解決

HTTP リクエストの Host とパスからテナントを決める規則、正規ロケーションと発行者の導出、正規ロケーションの切り替えを扱う。
解決したテナントの中で何を許可するかは扱わず、[認可設計](../../../design/security/authorization.md)に委ねる。
コードは `backend/shared/http/support_http` のテナント解決ミドルウェアと `backend/tenancy` にあり、機能スライスは持たない。

| 文書 | 内容 |
|---|---|
| [テナントの解決の設計判断](decisions.md) | 設計判断 |
| [テナントの解決の内部設計](internals.md) | 機構の説明 |
| [テナントの解決のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
