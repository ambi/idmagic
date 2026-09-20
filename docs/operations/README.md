# 運用文書

この文書は、`docs/operations/` と `docs/runbooks/` に収めた運用文書の入口である。
稼働後のサービス管理と保守の責任、判断基準、そして人が手を動かす運用手順を定める。
品質目標、可用性と復旧の機構、デプロイの段取りはここに複製せず、それぞれの一次情報を参照する。

| 文書 | 定めるもの |
| --- | --- |
| [サービス管理](service-management.md) | SLO 評価、当番、障害対応、変更、サービス継続 |
| [保守](maintenance.md) | 定期作業、更新、キャパシティ確認、廃止と引渡し |
| [運用手順](../runbooks/) | 人が手で実行する運用作業と障害対応の手順。作業一つにつき一つ置く |

## 関連する一次情報

| 知りたいこと | 一次情報 |
| --- | --- |
| 目標値と測定境界 | [品質要求](../requirements/quality.md) |
| 可用性と復旧の機構 | [可用性設計](../design/reliability/availability.md)、[リカバリ設計](../design/reliability/recovery.md) |
| 監視、ログ、トレースの設計 | [オブザーバビリティ設計](../design/observability/README.md) |
| デプロイと後退の段取り | [リリース](../development/release.md) |
| 要求を満たしたと判断する証拠 | [検証設計](../verification/README.md) |

警報が `runbook_url` で指す手順は[運用手順](../runbooks/)にある。
恒久対策が規則になったときは、対策の記録ではなく、その規則を扱う一次情報文書または仕様へ反映する。
