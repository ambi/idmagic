---
status: completed
archived_from: 4af17a0996ed7412344dd546acbaa9f4d1b2837c
---

# 監査イベントに汎用検索属性 registry と filter 式を導入し、tenant salt で相関ハッシュを統一する

監査イベント検索の汎用基盤を追加した。SCL に検索属性 registry / filter 式 / `q` を追加し、Go 側に registry twin、filter parser / validator、非 PII raw id 抽出器、tenant salt store、相関 hash / IP 丸めヘルパを実装した。HTTP handler は `filter=field:op:value[,value2]` と `q` を受け、memory / PostgreSQL repository は sidecar 検索属性で照合する。
- `audit_event_search_attributes` sidecar と `tenant_correlation_salts` を PostgreSQL schema に追加。
- bootstrap の audit emit 経路で `ExtractSearchAttributes` を接続し、memory / postgres の両 persistence mode へ `TenantSaltStore` を注入。
- throttle / authentication event bucket の外部公開 `keyHash` は `SaltedHash` helper 経由へ統一し、内部 throttle/bucket namespacing は変更しないまま維持。
- username / IP の emit 値 populate、平文入力からの PII 検索、UI 検索ビルダーは予定どおり wi-46 に残置。

全文は `git show 4af17a0996ed7412344dd546acbaa9f4d1b2837c:<このファイルのパス>` で読む。
