# システム設計

アーキテクチャが構成要素へ割り当てた要求を、領域ごとの機構へ具体化する。
品質要求は[品質要求](../requirements/quality.md)を正本とし、各設計は対象 ID と実現方法を結び付ける。

| 領域 | 答える問い |
| --- | --- |
| [application/](application/) | 機能、API、UI、モジュールをどう実装するか |
| [data/](data/) | データの正、整合性、保存、移行、廃棄をどう扱うか |
| [infrastructure/](infrastructure/) | 計算資源、ストレージ、ネットワークをどう構成するか |
| [security/](security/) | 資産と境界をどの制御で守るか |
| [reliability/](reliability/) | 障害、冗長性、縮退、復元をどう扱うか |
| [performance/](performance/) | 負荷、資源、待ち行列、拡張をどう扱うか |
| [observability/](observability/) | 状態をどの信号で観測し、検知するか |
