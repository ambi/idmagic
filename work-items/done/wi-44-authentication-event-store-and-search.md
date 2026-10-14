---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 認証イベントを永続ストア・属性拡張・admin 検索 UI まで実運用相当にする

認証イベントの永続ストア・bucket 集約・retention sweep・admin 検索を、専用 認証イベント API/UI ではなく既存監査ログ (`audit_events` / `/admin/audit_events`) へ統合する方針で完了した。Postgres の `authentication_event_buckets` は攻撃時の個別 INSERT 爆発を抑え、admin は `category` フィルタで認証成功・失敗・集約イベントを調査できる。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
