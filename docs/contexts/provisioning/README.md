# Provisioning

下流の SaaS へユーザーとグループを反映する、外向きのプロビジョニングを担う。情報の正は IdMagic 側の User と Group であり、下流のリソースはその複製である。接続は Application 1 件につき最大 1 件とし、配信対象の範囲には既存の ApplicationAssignment を利用する。

`Sourcing` が外部から取り込むのに対し、この Context は外部へ送り出す。処理の向き、記録の正の所在、語彙が異なるため、`Tenancy`、`Application`、`IdManagement`、`Jobs` の公開インターフェースを除いてコードを共有しない。

| 文書 | 内容 |
|---|---|
| [Provisioning の用語集](glossary.md) | この Context での語義 |
| [Provisioning の採用規範](standards.md) | 準拠する外部規範 |
| [Provisioning の状態遷移](states.md) | 状態と遷移 |
| [Provisioning の設計判断](decisions.md) | 設計判断 |
| [Provisioning の内部設計](internals.md) | 機構の説明 |
| [Provisioning Scenarios](scenarios.feature.md) | 受け入れシナリオ |
