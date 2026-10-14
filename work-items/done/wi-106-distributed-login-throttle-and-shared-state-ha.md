---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# login throttle を共有ストア化し、複数レプリカでも閾値が正しく効く HA 整合を担保する

ADR-077 を新設し、login throttle の共有ストア化・原子的インクリメント・SHA-256
識別子ハッシュ・fail-closed 縮退・ephemeral state の HA 棚卸しを決定として記録した
(ADR-029 の追補)。SCL: authentication.yaml の LoginThrottlePolicy に counter_scope
(cluster_wide) / identifier_hash (sha256) / shared_store_required_for_multi_replica /
degraded_store_behavior (fail_closed) を追加し、system.yaml に SharedEphemeralStateHA
objective を新設して共有インベントリ (Valkey: auth request/code/PAR/device code/session/
replay/denylist/login throttle、Postgres: refresh/audit/bucket) と memory=単一レプリカ
専用を明文化、派生 HTML/JSON を再生成した。
LoginThrottleConfig / LoginThrottleConfigs を memory adapter から
internal/authentication/ports へ移し、memory / valkey 両 adapter が同一 port 型を消費する。
valkey.LoginAttemptThrottle を新設: recordFailure は Lua script で INCR + 初回 EXPIRE +
閾値到達で DEL & SET-EX(lock) を 1 往復で原子的に行い、tryAcquire は lock キーの PTTL 一発、
recordSuccess は counter/lock を DEL する。キーは tenant_id を含み、識別子 (username/IP) は
adapter 内で SHA-256 ハッシュ化して平文を共有ストアに残さない。now は Redis の TTL に委ね
無視する。
bootstrap: Dependencies に NewLoginAttemptThrottle ファクトリを追加し、memory ランタイムは
memory 版、postgres ランタイムは valkey 共有版を返す。server.go は SCL 由来のしきい値から
deps.NewLoginAttemptThrottle(...) で adapter を生成するよう変更 (memory 固定配線を撤去)。
ログインハンドラの既存 error 伝播により、共有ストア到達不能時は throttle が fail-closed に
倒れる (縮退時も攻撃を素通しにしない)。README に「複数レプリカ運用では Valkey 必須」節を追記し、
共有対象・throttle のクラスタ全体カウント・fail-closed 縮退・memory=単一レプリカ専用を明記した。
HA 棚卸し結論: postgres モードで per-replica に残っていたのは login throttle のみで、本 WI で
共有化したことで ephemeral state は全て共有ストア (Valkey/Postgres) に載る。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
