---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ユーザ削除を「Soft-delete + 復元期間 → 確定 anonymize」の 2 段階にする

ユーザー削除を「soft-delete (削除予約) → 30 日以内の復元 → 完全削除
(匿名化 cascade)」の 3 段階にした。`UserStatus` に `PendingDeletion` を追加し、
状態機械 UserLifecycle に SoftDelete / Restore / Purge 遷移を加えた。既定の
DELETE は soft-delete、`?purge=true` (または body force=true) で完全削除に分岐し、
`POST /api/admin/users/{sub}/restore` を追加した。soft-delete は PII / Consent /
RefreshToken / Session を温存し、認証系は既存の `IsActive()` ゲートにより
Disabled / Deleted と同様に fail-close で拒否する。復元可能期間は SCL objective
`UserSoftDeleteGracePeriod` (30 日) に固定し、経過分は一覧取得時の lazy-on-access で
Purge (reason=auto_purge) する。SoftDelete / Restore / Purge の全経路に自爆防止を
適用した。Admin UI に「削除予約」フィルタ・amber StatusBadge・猶予残日数表示・
復元 / 完全削除の動線を追加。決定は ADR-072 に記録した。
実装差分 (WI 起票時の go 計画からの適応):
- WI の `pending_deletion_at` カラム + migration 0008 は、その後 status を唯一の
真実に統合した現行モデル (ADR-039) に合わせ、`UserStatus` enum 値 +
`status_changed_at` で表現した。JSONB `lifecycle.status` に enum 値が増えるだけ
なので schema migration は不要 (ADR-071 の宣言的 schema と整合)。
- 認証経路は既に `IsDeleted` ではなく `IsActive()` ゲートに移行済みだったため、
PendingDeletion (≠ Active) は追加改修なしで fail-close になった。WI が想定した
全経路の書き換えは不要だった。
- 一覧の既定フィルタは既存の「すべて」を維持し、「削除予約」タブと amber バッジで
識別する方式にした (既定を「有効」に変える既存挙動変更は避けた)。
- ADR-036 は改定せず、2 段階化の決定を新規 ADR-072 として追加した
(リポジトリ慣習は ADR を新番号で足す)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
