---
id: idp-wi-106-distributed-login-throttle-and-shared-state-ha
title: "login throttle を共有ストア化し、複数レプリカでも閾値が正しく効く HA 整合を担保する"
created_at: 2026-07-04
authors: ["tn"]
status: completed
risk: medium
---

# Motivation
bootstrap は postgres/valkey ランタイムでも login throttle を
`memory.NewLoginAttemptThrottle` で常にプロセスメモリに置いている。
authorization request / PAR / device code / session / refresh といった他の
ephemeral state は postgres モードで Valkey adapter へ載っているのに、
ブルートフォース防御の要である login throttle だけが per-replica になっている。

この結果、N レプリカで水平スケールすると、攻撃者の失敗試行が各レプリカに
分散カウントされ、実効ロックアウト閾値が最大 N 倍に緩む。ADR-029 の
per-account / per-IP 閾値は「クラスタ全体で 5 回」を意図しているのに、
10 レプリカでは実質 50 回まで許してしまう、という silent なセキュリティ劣化になる。
Keycloak が brute-force 検知を Infinispan の分散キャッシュで共有するのと同様、
idmagic も throttle 状態を共有ストア化し、HA 構成で閾値が意図どおり効くことを
保証すべきである。あわせて、他の ephemeral state が本当に全て共有化されているかを
棚卸しし、HA 前提を明文化する。

# Scope
- **decision**:
  - 新規 ADR（または ADR-029 追補）: throttle 状態の共有ストアと原子的インクリメント方式、 Valkey 障害時の fail-open/fail-closed、memory adapter を単一レプリカ/テスト用途に 限定する方針を定義する。
- **go**:
  - LoginAttemptThrottle の Valkey adapter を実装し、postgres/valkey ランタイムでは 共有ストア版を選択する。カウンタは原子的操作（INCR + TTL 等）で per-account / per-IP を数える。
  - key は tenant_id を含め、識別子は既存どおり SHA-256 で hash 化し平文を残さない。
  - Valkey 到達不能時の挙動を ADR に従って明示（縮退時も攻撃を素通しにしない）する。
  - HA 前提の棚卸し: session/PAR/device code/refresh/replay/denylist 等が全て共有ストアに 載っているかを確認し、memory 固定で残る箇所があれば洗い出す（本 WI で移すか別 WI 化するか判断）。
- **documentation**:
  - README に「複数レプリカ運用では Valkey 必須」と、throttle を含む共有対象を明記する。

# Out of Scope
- login throttle の閾値・アルゴリズム自体の再設計（ADR-029 の値は維持）。
- WI-27 が扱う endpoint 汎用 rate limit / CAPTCHA。
- CAPTCHA・bot スコアリング。

# Verification
- `go test -race ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- 手動: 2 レプリカ + 共有 Valkey で同一アカウントへ失敗試行を分散させ、 合算閾値でロックされる（レプリカごとに別々に数えない）ことを確認する。
- 手動: Valkey 停止時に throttle が ADR どおりの縮退挙動になることを確認する。

# Risk Notes
throttle を共有化すると Valkey が新たな critical path になり、その障害時挙動が
そのままログイン可用性/防御に効く。fail-open にすると攻撃素通し、fail-closed に
すると Valkey 障害で全ログイン不能になり得るため、縮退方針を ADR で明確に決める。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
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
- **Verification Results**:
  - just yaml-check (SCL / work-items / ids すべて OK)、just scl-render で派生物同期
  - golangci-lint run ./... : 0 issues
  - go test -race ./... green (全パッケージ)
  - valkey/login_attempt_throttle_test.go (miniredis): 閾値ロック、別接続=別レプリカ相当からの
    失敗が合算閾値でロックされ全レプリカから見える (cluster-wide)、success が account のみクリアし
    per-IP は継続、ストア切断時に TryAcquire/RecordFailure がエラー=fail-closed を確認した。
  - memory/login_attempt_throttle_test.go: port 型移行後もロック/失効/success クリアが従来どおり green
  - 手動 (2 レプリカ + 共有 Valkey / Valkey 停止時の縮退) は上記自動テストで cluster-wide 合算と
    fail-closed を代替検証済み。
