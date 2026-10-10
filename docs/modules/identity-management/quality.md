# IdManagement の品質要件

この文書は、システムの[品質要件](../../requirements/quality.md)と[キャパシティ設計](../../design/performance/capacity.md)のうち、IdManagement が担う分を扱う。
値を変えると製品が守る境界が変わるものは、規則として機能仕様に置き、ここからは参照するだけにする。

| 品質要件 | 値 | 根拠 | 実現方式 |
| --- | --- | --- | --- |
| CSV の往復の規模 | 10,000 件の User を、すべての組み込み列でエクスポートして再インポートすると、全行が変更なしになる | キャパシティ設計の、テナント当たり利用者数の最大値 | [性能](design/performance.md) |
| CSV の一つの成果物の上限 | データ行 100,000 行、64 MiB、一つの項目 64 KiB | 規則 REQ-IDMANAGEMENT-037 | [性能](design/performance.md) |
| CSV の処理のメモリ使用量 | `worker` のメモリ使用量が、成果物の大きさに比例しない | 上限の大きさの成果物を、通常の `worker` の割り当てで処理するため | [CSV の往復変換](design/csv-transfer.md) |
