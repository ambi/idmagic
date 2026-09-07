# 要求

IdMagic が満たすシステム要求を、機能、品質、設計上の制約に分ける。
個別の振る舞いは各 Bounded Context の `scenarios.feature.md` と TypeSpec が持ち、このディレクトリは利用目的から詳細仕様への割当を示す。

| 文書 | 責務 |
| --- | --- |
| [機能要求](functional.md) | システム機能の分解、担当 Context、詳細仕様への参照 |
| [品質要求](quality.md) | 品質目標、測定境界、容量受入目標、復旧目標 |
| [システム制約](constraints.md) | 外部規範、運用環境、技術と配備に課される条件 |

要求を実現する構造は[アーキテクチャ](../architecture/)、機構は[設計](../design/)、合否の判定方法は[検証](../verification/)が持つ。
