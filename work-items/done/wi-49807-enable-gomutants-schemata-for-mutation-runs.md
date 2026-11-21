---
status: cancelled
---

# gomutants の mutant schemata を有効にし、変異体ごとのビルドをなくす

gomutants の schemata 提供を待つ方針を中止する。
2026-11-21 に、wi-15163 で gamut 0.1.0 への移行を実装した。
gamut は全変異体を一度の構築へ埋め込む方式を備え、変異体ごとの構築費用と共有ビルドキャッシュへの書き込みを減らすという本項目の目的を引き継ぐ。
