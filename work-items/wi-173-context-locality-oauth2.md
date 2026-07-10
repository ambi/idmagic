---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-11
---

# oauth2 コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]]（application context パイロット）で [[ADR-089]]（ドメイン型の per-context 化）・
[[ADR-090]]（永続化同居＋sqlc）・[[ADR-091]]（Module パターン DI/ルーティング）の 3 ADR を
貫通実装し型紙を確立した。本 WI はその型紙を oauth2 context へ適用する第一弾横展開である。

oauth2 は `spec/scl.yaml` context_map 上で他 context からの被依存が 0（leaf）だが、
`internal/shared/spec` / `internal/shared/adapters/persistence` 双方で最大規模を占める
context であり、横展開の中でも最も工数の大きい 1 件になる見込み。低被依存のうちに最大
規模を消化し、後続 WI の見積り精度を上げる狙いもある。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/oauth2.yaml` を
正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/oauth2.go`（292 行）ほか oauth2 固有の業務型
  （client / consent / authorization detail / refresh token / grant / authorization
  code・device code の state machine 等）を `internal/oauth2/domain/` へ移設。
- oauth2 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `clients.go` / `consents.go` / `authorization_detail_types.go` / `refresh_tokens.go` /
  `audit_events.go` / `outbox.go`、および valkey backed の authorization request / code /
  PAR / device code / DPoP replay / client assertion replay store）を
  `internal/oauth2/adapters/persistence/{postgres,memory,valkey}` へ同居。
- oauth2 の postgres 実装を sqlc 生成へ置換（動的クエリはエスケープハッチ、[[ADR-090]] 準拠）。
- `internal/oauth2/module.go` を新設し、`Deps`/`bootstrap` から oauth2 分を Module へ移す。

## Out of Scope

- SigningKeys 関連（`shared/adapters/persistence/postgres/keys.go`、`ports.SigningKey`）。
  SigningKeys はまだ独立 context（`internal/signingkeys/` 相当）を持たないため、本 WI では
  shared に残置する。SigningKeys 自身を context 化するかは別途評価する。
- ClaimMapping・Authentication・IdentityManagement・Tenancy 等、他 context の型移設
  （[[wi-174]]〜[[wi-179]] で扱う）。
- memory 二重実装の解消（testcontainers 退役の是非は別評価、[[ADR-090]] 決定 6 を参照）。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序（domain → ports → persistence → sqlc → module.go →
   中央 Deps/bootstrap 撤去）で進める。
2. **規模が大きいため、T001（domain 移設の実測）の時点で 1 WI では収まらないと判明した
   場合、client / token / audit・outbox の 2〜3 分割に切り出す**。分割時は本 WI を
   `cancelled` にせず、後続 WI へ Tasks を移管し Scope を縮小したうえで完了させる。
3. sqlc 化にあたり、client 一覧の admin filter・audit event 検索など可変 WHERE を持つ
   クエリが oauth2 に含まれる可能性が高い。[[ADR-090]] の「動的比率が支配的なら bob へ
   切替」の再評価トリガーになりうるため、実測値を [[ADR-090]] へ追記する。
4. DPoP replay / client assertion replay / device code / PAR / authorization code /
   session store は valkey backed（`shared/adapters/persistence/valkey`）であり sqlc の
   対象外。これらは同居のみ行い、[[ADR-090]] の対象外である旨を明記する。

## Tasks

- [ ] T001 [Domain] `shared/spec/oauth2.go` ほか oauth2 業務型を `oauth2/domain/` へ移設し参照更新。
  移設規模を実測し、1 WI で収まらない場合はここで分割方針を確定する。
- [ ] T002 [Kernel] oauth2 が他 context と共有する型を選別（context-map の publishes 基準）。
- [ ] T003 [Persistence] oauth2 固有 repo 実装を `oauth2/adapters/persistence/{postgres,memory,valkey}` へ同居。
- [ ] T004 [Persistence] oauth2 postgres 実装を sqlc 生成へ置換（動的はエスケープハッチ）。
- [ ] T005 [DI] `oauth2/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から oauth2 分を撤去。
- [ ] T007 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T008 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go`（format-check / lint / typecheck / build）が green。
- `just test-go` で回帰なし。oauth2 の E2E / 単体（authorization code / client
  credentials / refresh / DPoP / PAR 等）が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/oauth2 | wc -l` が SigningKeys
  等の scope 外参照を除きゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と `just dev` でスモーク。

## Risk Notes

- **risk: high**。oauth2 は shared 内最大規模の context であり、(a) 移設対象の型数・
  ファイル数が多く見積りが不確実、(b) token/grant 周りは認可の根幹でありバグの影響が
  大きい、(c) valkey backed store が持ち込み対象に含まれ sqlc 対象外の同居作業が別途発生
  する、の 3 点が主リスク。
- 軽減：[[wi-172]] で確立した型紙をそのまま踏襲し、各タスクを小さくコミット可能な粒度に
  保つ。T001 の実測結果次第で早期に分割判断する（Plan 参照）。振る舞い不変を
  `just test-go`（既存 E2E 含む）で都度確認する。
