---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-07
priority: p2
depends_on: [wi-87724-fix-the-device-flow-refresh-token-and-expired-denial]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/token/README.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/modules/oauth2/authorization/README.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/device/README.md, requirement: REQ-OAUTH2-027 }
---

# refresh_token のグラントを許可していないクライアントにリフレッシュトークンを発行しない

## 動機

認可コードと `device_code` の交換は、スコープに `offline_access` を含めば、クライアントが `refresh_token` のグラントを許可しているかを確かめずにリフレッシュトークンを発行する。
発行の判定はドメインの `GrantsRefreshToken` に一つだけあり、スコープだけを見る。

一方、トークンエンドポイントは、クライアントの `grant_types` にないグラントの要求を 400 と `unauthorized_client` で拒否する。
したがって、`refresh_token` のグラントを許可していないクライアントへ発行したリフレッシュトークンは、一度も使えない。
使えないのに、ベアラーの秘密として応答で渡り、記録として保存され、`RefreshTokenIssued` が発行される。
漏えいしたときの影響だけがあり、得るものがない。

## 対象範囲

- 認可コードと `device_code` の交換で、クライアントが `refresh_token` のグラントを許可し、かつスコープが `offline_access` を含むときだけ、リフレッシュトークンを発行する。
- 条件を満たさない交換は成功させ、アクセストークンと ID トークンだけを返す。

## 対象外

- 認可要求やデバイス認可の要求の時点で `offline_access` を拒否すること。下の「採用しない案」を参照する。
- `refresh_token` のグラントの要求そのもの（ローテーション、ファミリーの失効）。
- 既に発行済みの、使えないリフレッシュトークンの失効。製品は未リリースであり、移行は要らない。

## 設計

### 要件の差分

| 要件 | 変更 |
| --- | --- |
| REQ-OAUTH2-021 | 発行の条件を「`offline_access` を含むスコープ」から「`refresh_token` のグラントを許可したクライアントの、`offline_access` を含むスコープ」へ変える。`refresh_token` のグラントを許可していないクライアントの交換では、リフレッシュトークンを返さないという要件文を足す |
| REQ-OAUTH2-005、REQ-OAUTH2-027 | 交換の条項の「`offline_access` のスコープにはリフレッシュトークンを返し」を、REQ-OAUTH2-021 の条件に合わせる |

### 実装

`GrantsRefreshToken(scopes []string) bool` を、クライアントのグラントも受け取る判定に変える。
両方の交換は判定の呼び出しを替えるだけである。
シグネチャは着手時に決める。たとえば `GrantsRefreshToken(grants []spec.GrantType, scopes []string) bool` である。

### 決めること

- 応答の `scope` と、アクセストークンの `scope` に `offline_access` を残すか。リフレッシュトークンを発行しないのに `offline_access` を付与したと示すと、応答が実際に付与したものと食い違う。RFC 6749 の 5.1 は、付与したスコープが要求と異なるときに `scope` を返すことを求める。着手時に、外すか残すかを決め、要件文へ書く。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| 認可要求の時点で `offline_access` を `invalid_scope` で拒否する | クライアントの許可するスコープに `offline_access` があれば要求は正当であり、拒否すると、登録の不整合が利用者のログインの失敗として現れる。リフレッシュトークンは RFC 6749 の 5.1 で任意であり、発行しないだけで足りる |
| クライアントの登録の時点で、`offline_access` のスコープと `refresh_token` のグラントの組を強制する | 既存の登録の検証を広げることになり、この不具合より大きい変更になる。交換の判定を直せば、登録の組が何であっても使えない資格情報は発行されない |

## 計画

1. 二つの交換で、`refresh_token` のグラントのないクライアントに `offline_access` を付与したときの振る舞いを RED で確かめる。
2. 要件の差分を仕様へ反映する。
3. 発行の判定にクライアントのグラントを加える。

## タスク

- [ ] T001 [Acceptance] 二つの交換で、`refresh_token` のグラントのないクライアントへリフレッシュトークンを発行することを RED で確かめる。
- [ ] T002 [App] 発行の判定にクライアントのグラントを加え、GREEN にする。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run test-go-test -- ./backend/oauth2/token/usecases <test>` と `./backend/oauth2/device/usecases <test>`
- `mise run test-go-mutation -- backend/oauth2/token/domain`
- `mise run verify`

## リスク

- `refresh_token` のグラントを許可せず `offline_access` を要求しているクライアントは、受け取っていた（使えない）リフレッシュトークンを受け取らなくなる。トークンエンドポイントがそのグラントを拒否するので、動いていた利用経路は壊れない。
