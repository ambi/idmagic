# システム受入れ設計

## 機能受入れ

ログイン、トークン発行、管理操作、上流連携、下流プロビジョニング、監査の代表経路を、実際のルーティングと依存注入を通して確認する。
拒否ではレスポンスだけでなく、保存、発行、通知が行われなかったことを確認する。

## 品質受入れ

性能と容量は、[品質要求](../requirements/quality.md)の `CAP-*` 条件と[容量設計](../design/performance/capacity.md)の運用プロファイルを使う。
可用性は Pod、ノード、PostgreSQL、外部依存の障害を別々に注入し、受付、縮退、回復、データ整合性を確認する。
復旧は空の対象へバックアップを戻し、テナント分離、参照整合性、鍵への到達性、サービス再開までを測る。

## 現在の証拠の限界

ローカルのスモーク試験と復元訓練は存在するが、ステージングの全運用プロファイル、マルチ AZ のフェイルオーバー、本番相当の RPO と RTO は未検証である。
負荷試験は [wi-282](../../work-items/wi-282-staging-load-testing-and-capacity-validation.md)、高可用性は [wi-165](../../work-items/wi-165-high-availability-and-failover-resilience-topology.md)、異なる版の混在は [wi-450](../../work-items/wi-450-mixed-version-release-acceptance.md) が扱う。
