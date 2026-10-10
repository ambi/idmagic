# CSV の転送の設計

この文書は、[CSV の転送](README.md)の規則を保証する仕組みを扱う。
User と Group の CSV が共有する仕組みの全体は、モジュールの横断的概念[CSV の往復変換](../design/csv-transfer.md)が扱い、この文書はそこへの索引だけを持つ。

この機能の設計のうち、ほかの文書が扱う内容は次のとおりである。

- アーキテクチャ設計：[CSV の往復変換](../design/csv-transfer.md)の「共有するものと、種別ごとに分けるもの」と「解析と変換」
- 設計判断：モジュールの[重要な設計判断](../design/decisions.md)に従う
- データ設計：成果物ストアはモジュールの[データ](../design/data.md)が扱う
- セキュリティ設計：[CSV の往復変換](../design/csv-transfer.md)の「解析と変換」（シークレットの列の拒否、数式の保護）
- 信頼性設計：モジュールの[信頼性](../design/reliability.md)に従う
- 性能設計：[IdManagement の性能](../design/performance.md)
