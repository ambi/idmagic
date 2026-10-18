---
status: cancelled
---

# WS-Federation アプリケーションの管理を Application のポートで要求する

ApplicationCatalog の分離を行う wi-28013 により、WS-Fed の管理依存を状態所有とアクセス判定から分離する。
独立した要求型と変換層の追加は中止する。
RP の作成、参照、更新、関連付けと削除の既存処理を Catalog に移し、公開する所有者の型と操作を利用する。
依存の再計測で Application と WsFederation の相互参照が循環から外れたことを確認した。
HTTP、保存状態、クレームの公開制限と確定順の維持は wi-28013 の既存テストと検証で扱う。

全文は `git log -1 -- work-items/active/wi-61016-administer-wsfed-applications-through-application-ports.md` が示すコミットの親、またはそのコミットで読む。
