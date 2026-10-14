---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 利用者向けコピーと self-service 画面の体裁を整える (属性の日本語名 / 内部表現の排除 / プロモ枠の除去)

7 件の利用者向けコピー/体裁の不整合を 6 コミットで解消した。
UserAttributeDef に日本語表示名 label を追加し組み込み属性すべてに付与、
管理・self の各画面で「表示名 (key)」を表示 (address_* も区別可能に)。
AuthShell に aside=false の単一カラムモードを足し profile / change-password
からプロモ枠を除去。監査ログの (Audit Events)/DomainEvent、グループの
ロール束/group.roles、ロール権限の内部英語 (User.roles 等) を user-facing
日本語に置換。UI の "idmagic" を "RA Identity" に統一。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
