# Provisioning

下流の SaaS へユーザーとグループを反映する、外向きのプロビジョニングを所有する。情報の正は IdMagic 側の User と Group であり、下流のリソースはその複製である。接続は Application 1 件につき最大 1 件とし、配信対象の範囲には既存の ApplicationAssignment を利用する。

`Sourcing` が外部から取り込むのに対し、この Context は外部へ送り出す。処理の向き、記録の正の所在、語彙が異なるため、`Tenancy`、`Application`、`IdManagement`、`Jobs` の公開インターフェースを除いてコードを共有しない。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
