# Tenancy

Tenant (Realm) の Aggregate、ライフサイクル、HTTP リクエストからのテナント解決、テナント単位の設定・外装・属性スキーマ・通知テンプレート・リソース上限、そして制御面のテナント管理 API を所有する。Tenant は IdMagic のあらゆる Aggregate が属する境界である。

テナントの中に置かれる記録は所有しない。User と Group は `IdManagement`、Application は `Application`、資格情報とセッションは `Authentication` が持つ。この Context が決めるのは、それらがどの境界に属し、その境界にどんな上限と既定が効くかである。

テナント分離の規則そのものは製品全体の関心事なので [spec/authorization.md](../../authorization.md) が正であり、ここはその境界を決める側の記録を持つ。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
