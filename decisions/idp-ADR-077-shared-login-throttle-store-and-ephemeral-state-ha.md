# idp-ADR-077: login throttle を共有ストア化し ephemeral state の HA 前提を明文化する

## ステータス
採用。`spec/contexts/authentication.yaml` の `objectives.LoginThrottlePolicy`（`counter_scope` / `degraded_store_behavior` 等）と `spec/contexts/system.yaml` の `objectives.SharedEphemeralStateHA`、`internal/authentication/ports`（`LoginThrottleConfig` / `LoginThrottleConfigs`）、`internal/shared/adapters/persistence/valkey`（`LoginAttemptThrottle`）、`internal/bootstrap`（postgres ランタイムでの adapter 選択）、README の HA 運用節に反映。ADR-029（ログイン throttle）の追補。

## コンテキスト
ADR-029 は login throttle の port と閾値（per-account 10/15分、per-IP 30/15分）を定め、「影響」節で本番 adapter として `ValkeyLoginAttemptThrottle`（`INCR` + `EXPIRE` + `SET NX EX`）を想定していた。しかし実装は postgres/valkey ランタイムでも `memory.NewLoginAttemptThrottle` を常に配線しており、throttle だけがプロセスメモリに残っていた。

authorization request / authorization code / PAR / device code / login session / DPoP・client-assertion replay / access-token denylist といった他の ephemeral state は postgres モードで Valkey adapter に載っているのに、ブルートフォース防御の要である login throttle だけが per-replica だった。N レプリカで水平スケールすると攻撃者の失敗試行が各レプリカに分散カウントされ、ADR-029 が「クラスタ全体で N 回」を意図した閾値が実効的に最大 N 倍まで緩む。10 レプリカなら per-account 100 回まで許してしまう silent なセキュリティ劣化である。Keycloak が brute-force 検知を Infinispan の分散キャッシュで共有するのと同様、idmagic も throttle 状態を共有ストア化して HA 構成で閾値が意図どおり効くことを保証すべきである。

throttle を共有化すると Valkey が新たな critical path になり、その障害時挙動がログイン可用性と防御の双方に効く。縮退方針を fail-open にすると攻撃を素通しさせ、fail-closed にすると Valkey 障害で全ログイン不能になり得るため、方針を明示的に決める必要がある。

## 決定
1. **login throttle を共有ストア化する**。postgres/valkey ランタイムでは Valkey adapter を選択し、カウンタとロックをレプリカ間で共有する。memory adapter は単一レプリカ / テスト専用として残す。adapter 選択は他の ephemeral state と同じく `internal/bootstrap` の assemble で行い、throttle 閾値（SCL `LoginThrottlePolicy` 由来）を注入する。
2. **原子的インクリメント**。per-account / per-IP のカウントは Valkey の原子操作で行う。失敗記録は Lua script で `INCR`（初回のみ `EXPIRE` で window を張る）→ 閾値到達なら counter を `DEL` しロックキーを `SET EX`（lockout）、を 1 往復で原子的に実施する。read-write が分割されて複数レプリカで競合し二重カウント / 取りこぼしが起きないようにする。取得判定（tryAcquire）はロックキーの `PTTL` 一発で allowed / retryAfter を返す。fixed-window セマンティクスは memory adapter と揃える。
3. **識別子は SHA-256 で hash 化し平文を残さない**。共有ストアに載るキーの識別子（username / IP）は adapter 内で SHA-256 でハッシュ化してから Redis キーへ埋める。キーは tenant_id を含め（`tenant:{tid}:login_throttle:...`）、テナント間で counter が混ざらないようにする。共有ストアを直接覗いても平文の username / IP が読めないようにする（監査イベントの keyHash と同じ hash 方針）。
4. **縮退方針は fail-closed**。共有ストアが到達不能な縮退時、throttle は閾値を保証できない。このとき adapter はエラーを返し、ログインハンドラは既存の error 伝播どおり当該ログイン試行を成功させない（素通しさせない）。fail-open（エラー時 allow）は攻撃をそのまま通すため採らない。可用性より防御を優先する secure-by-default の選択であり、複数レプリカ本番は Valkey を HA 構成（レプリケーション / フェイルオーバ）で運用することを前提とする。この前提を README に明記する。
5. **HA 前提の棚卸しと明文化**。SCL `SharedEphemeralStateHA` objective として、複数レプリカ運用でどの ephemeral state が共有ストア（Valkey）に載り、どれが Postgres の durable な共有ストアが所有するかを列挙する。本 ADR 時点で per-replica に残っていたのは login throttle のみで、これを移せば postgres モードの ephemeral state は全て共有される。memory adapter は共有ストアを持たないため単一レプリカ / テスト専用であることを明文化する。

## 却下した代替案
- **fail-open（Valkey 障害時に throttle を無効化して allow）**: 障害の窓で brute-force / credential stuffing がそのまま通る。ADR-029 が閉じた攻撃面を可用性のために開き直すことになり、IdP としての基本防御に反する。「縮退時も攻撃を素通しにしない」を満たさない。
- **障害時に per-replica の in-memory throttle へフォールバック**: 縮退中も何らかの絞りは残るが、実効閾値がレプリカ数倍に緩む（まさに本 ADR が塞ぐ劣化）状態へ意図的に戻ることになり、挙動が二段構えで複雑化する。共有ストアを HA で運用する運用契約（README）で代替できるため採らない。
- **throttle を Postgres に載せる**（他 ephemeral state と別ストアにする）: 高頻度の INCR / TTL 失効を RDB で扱うのは非効率で、既存 ephemeral state（session/PAR/code 等）と別のストアに散らばる。Valkey の `INCR` + TTL が用途に最適で、ストアも他 ephemeral state と揃う。
- **識別子を平文のままキーにする**（memory adapter 同様）: memory はプロセス内に閉じるが Valkey は共有・運用ツールから覗け得る。平文 username / IP を残すのは監査 keyHash の hash 方針と不整合。SHA-256 で揃える。

## 影響
- SCL: `LoginThrottlePolicy.value` に `counter_scope: cluster_wide` / `identifier_hash: sha256` / `shared_store_required_for_multi_replica: true` / `degraded_store_behavior: fail_closed` を追加。`SharedEphemeralStateHA` objective を新設し ephemeral state の共有インベントリと縮退方針を記録。
- ports: `LoginThrottleConfig` / `LoginThrottleConfigs` を memory adapter から `internal/authentication/ports` へ移し、memory / valkey 両 adapter が同一の port 型を消費する。
- adapter: `valkey.LoginAttemptThrottle` を新設（Lua script による原子的 recordFailure、`PTTL` による tryAcquire、tenant scoping + SHA-256 キー）。memory adapter は port 型を参照するよう更新。
- runtime: `internal/bootstrap` の Dependencies に throttle 生成ファクトリを追加し、memory ランタイムは memory adapter、postgres ランタイムは Valkey adapter を選択する。
- 運用: 複数レプリカ運用では Valkey（PERSISTENCE=postgres）が必須で、throttle を含む共有対象と縮退が fail-closed であることを README に明記。単一レプリカ / テストは memory で従来どおり。
