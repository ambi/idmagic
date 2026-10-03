# CSV の転送の設計

この文書は、[CSV の転送](README.md)の規則を保証する仕組みを扱う。
User と Group の CSV が共有する仕組みの全体は、Context の横断的概念[CSV の往復変換](../../design/csv-transfer.md)が扱い、この文書はそこへの索引だけを持つ。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [CSV の往復変換](../../design/csv-transfer.md)の「共有するものと、種別ごとに分けるもの」と「解析と変換」 |
| 設計判断 | 該当なし：Context の[重要な設計判断](../../design/decisions.md)に従う |
| アプリケーション | 該当なし：Context の設計に従う |
| データ | 該当なし：成果物ストアは Context の[データ](../../design/data.md)が扱う |
| セキュリティ | [CSV の往復変換](../../design/csv-transfer.md)の「解析と変換」（シークレットの列の拒否、数式の保護） |
| 信頼性 | 該当なし：Context の[信頼性](../../design/reliability.md)に従う |
| 性能 | [IdManagement の性能](../../design/performance.md) |
| オブザーバビリティ | 該当なし：Context の設計に従う |
| 検証 | 該当なし：Context の設計に従う |
| インフラストラクチャ | 該当なし：Context の設計に従う |
| リスク | 該当なし：Context の設計に従う |
