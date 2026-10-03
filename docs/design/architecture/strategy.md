# 解決戦略

この文書は、IdMagic が要求と[制約](constraints.md)に応える基本方針を要約する。
各方針の詳細と、代替案を比べた判断は、表の右の列の文書が扱う。

| 方針 | 要約 | 詳細 |
| --- | --- | --- |
| Modular Monolith | 一つの Go モジュールの中に Bounded Context の境界を保ち、API、Worker、Batch が同じ実装を共有する。Context ごとの独立したデプロイが要るまでは、サービスに分けない | [論理アーキテクチャ](logical.md#アーキテクチャ様式)、[判断](decisions.md#modular-monolith-の採用) |
| ポートによる Context の分離 | ドメイン層とユースケース層は外部の技術に依存せず、ポートを通して PostgreSQL、HTTP、通知などのアダプターにつながる。Context 間のイベントは、組み立ての地点にある一つの配信点を通して渡し、Context の間に import を作らない | [論理アーキテクチャ](logical.md#context-map)、[設計ガイドライン](../application/design-guidelines.md) |
| PostgreSQL への状態の集約 | 永続状態と、レプリカの間で共有する短命の状態を PostgreSQL に集める。正しさをレプリカの数から独立させ、二つ目のステートフルな基盤を運用しない | [判断](decisions.md#共有状態としての-postgresql) |
| 同期と非同期の分離 | 長時間の処理と再試行を、HTTP の要求から Worker と Batch へ分ける。ジョブの状態は PostgreSQL に置く | [ランタイムアーキテクチャ](runtime.md#実行単位) |
| 同一オリジンの境界 | ブラウザー向けの UI と API を、ゲートウェイで同一オリジンにする。Cookie のスコープと `Origin` の検証をこの境界に頼る | [判断](decisions.md#ブラウザーの同一オリジン境界) |
| テナント単位の分離 | すべての業務データをテナントに属させ、テナントの境界を越える操作は制御面のテナントの主体に限る | [データベース設計](../data/database.md)、[認可設計](../security/authorization.md#テナント境界) |
| 仕様を先に書く | 外から観測できる振る舞いは、TypeSpec と機能仕様の規則に先に書き、テストが規則を引く | [仕様フォーマット](../../../SPECIFICATION_FORMAT.md) |
