---
status: pending
authors: [claude]
risk: medium
created_at: 2026-08-08
depends_on: [wi-58-continuous-access-evaluation-agent-revocation]
---

# SharedSignals (CAEP/SSF) を人間 User セッションへ拡張し、ローカル失効を生態系へ伝播する

## Motivation

[[wi-58-continuous-access-evaluation-agent-revocation]] / [[ADR-057]] は Motivation・specification
(`spec/contexts/sharedsignals/SPECIFICATION.md`)・実装のいずれも Agent プリンシパル限定でスコープされている。
`AgentRevocationEpoch` は `agent_id` をキーに `agents` テーブルへ直接 FK する専用テーブルで、
OAuth2 `Introspect` への `ensures` 節も `access_token_subject_is_agent(...)` というガード付きで
Agent 主体の token にしか revocation epoch 判定をかけない。

一方 `SsfSubject` (CAEP イベント/SET の subject 表現) には元々 `subject_type: "Agent" | "User"` が
定義されており、`SsfStream`/`SsfTransmitterConfig`/`SsfReceiverConfig`/`SecurityEventToken`/
`SecurityEventDelivery`/`ReceivedSecurityEvent`/`CaepEvent` という transmitter/receiver パイプライン
自体は Agent 専用ではなく principal-agnostic に作られている。つまり**配送基盤は既にできているが、
User 側のローカル revocation epoch 機構と、それを既存の human 側失効トリガーに接続する部分が
丸ごと欠けている**。README ロードマップ (Phase 3) は元々 CAEP/SSF を汎用機能として位置づけていた。

人間ユーザーは RFC 7009 Token Revocation・refresh token family revocation ([[ADR-004]])・`sid`
ベースのセッション失効 ([[wi-28-session-management-and-oidc-logout-completion]]) というローカルの
失効プリミティブは既に持っているが、これらは当該 IdP 内で完結し、外部 resource server / 別 IdP へ
CAEP イベントとして伝播しない (EcosystemPropagation が無い)。また外部から届く User 主体の inbound
SET も、ローカルセッション失効として反映する経路が無い。本 WI は、管理者による強制ログアウト・
アカウント無効化/削除・パスワード漏洩時の強制失効・通常のログアウトといった人間ユーザーの失効を
SharedSignals の既存パイプラインに乗せて生態系へ伝播し、外部由来の User 向け signal も受理できる
ようにする。

## Scope

- `spec/contexts/sharedsignals/SPECIFICATION.md`:
  - `AgentRevocationEpoch` と対称な新規 model `UserRevocationEpoch` (`user_id` をキーに `users`
    テーブルへ FK する専用テーブル) を追加する。
  - `CheckUserRevocationEpoch` / `AdvanceUserRevocationEpoch` (`access: internal`) interface を
    `AgentRevocationEpoch` の対と同型で追加する。
  - `RevocationReason` 相当の理由 enum を User 向けに追加する (例: `UserDisabled` /
    `UserSoftDeleted` / `UserDeleted` / `SessionRevoked` / `PasswordCompromised` / `ManualAdmin` /
    `InboundSecurityEvent`)。既存の `RevocationReason` に統合するか別 enum
    (`UserRevocationReason`) にするかは `## Design` で判断する。
  - `ReceiveSecurityEvent` の subject 解決を Agent だけでなく User にも対応させる (現状は
    `security_event_subject_resolves_to_tenant_local_principal` が Agent のみを解決する前提)。
- `spec/contexts/oauth2/SPECIFICATION.md`:
  - `Introspect` の `ensures` 節に、`access_token_subject_is_user(...)` の場合も同様に revocation
    epoch を判定する節を追加する (現状の Agent 向け節と対称)。
- specification 変更なしで既存イベントをトリガーとして利用する対象 (`SharedSignals` が構造的に反応する、
  wi-58 と同じパターン):
  - `identity-management.yaml`: `UserDisabled` (`identity-management.yaml:1304`)、
    `UserSoftDeleted` (`:1330`)、`UserDeleted` (`:1359`)。
  - `authentication.yaml`: `SessionEnded` (`:1689`)。
  - `oauth2.yaml`: `RefreshTokenReuseDetected` (`:3262`)。
  - パスワード変更・MFA/authenticator reset 系のイベント (対象を実装時に棚卸しする)。
- Go 実装 (`backend/sharedsignals`): `UserRevocationEpoch` の domain/ports/db_memory/db_postgres
  (`AgentRevocationEpoch` と同じ構造で並行実装)、上記トリガーとの接続、OAuth2 `Introspect` への
  実配線。

## Out of Scope

- [[wi-58-continuous-access-evaluation-agent-revocation]] で Agent 側について既にカバーされている
  範囲の再設計。
- OIDC front/back-channel logout notification
  ([[wi-257-oidc-front-back-channel-logout-notifications]]) の再設計・重複実装。これは別の既存失効
  伝播経路であり、本 WI は統合しない (両者の役割分担は実装時に明確化する)。
- リスクスコアリングエンジンの構築 ([[wi-58-continuous-access-evaluation-agent-revocation]] と同じ)。
- 外部 IdP との相互運用認証取得。

## Design

- **`AgentRevocationEpoch` を `PrincipalRevocationEpoch` のような汎用テーブルへリネーム/統合する
  のではなく、対称な別テーブル `UserRevocationEpoch` を新設する方針を推奨する**。理由:
  - 既に wi-58 で `agent_revocation_epochs` テーブルと `AdvanceAgentRevocationEpoch` の DB 制約
    (条件付き `ON CONFLICT ... WHERE`) が本番相当で実装済みであり、破壊的リネームは avoidable な
    リスクを生む。
  - Agent と User では FK 先テーブル (`agents` vs `users`) が異なり、単一テーブルにすると
    `CHECK (subject_type IN (...) )` + 条件付き FK のような複雑さが増す。「3行の重複は早すぎる
    抽象化より良い」という方針に従い、対称な2テーブルのままにする。
  - `CheckRevocationEpoch`/`AdvanceRevocationEpoch` (Agent 用) と
    `CheckUserRevocationEpoch`/`AdvanceUserRevocationEpoch` (User 用) をそれぞれ独立した
    interface として持ち、OAuth2 側は token の subject 種別に応じてどちらを呼ぶか分岐する。
- transmitter/receiver パイプライン (`SsfStream` 以下) は改修不要、そのまま再利用できる見込み。
  `ReceiveSecurityEvent` の subject 解決ロジックだけ Agent/User 両対応にする。
- `SessionEnded` は `sid` 単位のイベントであり、1つの `sid` に複数 User が絡むことは無いため
  (OIDC session は 1 User に閉じる)、`UserRevocationEpoch` への前進は素直に `user_id` へマップ
  できる想定。実装時に `SessionEnded` の payload に `user_id` が含まれるか確認する。

## Plan

- 未定。最低限のステップ:
  1. `UserRevocationEpoch` の specification 追加、`ReceiveSecurityEvent`/`Introspect` の拡張。
  2. Go 実装 (domain/ports/db_memory/db_postgres) を `AgentRevocationEpoch` と並行する形で追加。
  3. Authentication/IdManagement 側の各トリガーを棚卸しし、`AdvanceUserRevocationEpoch` への
     接続を実装する。
  4. OAuth2 `Introspect` への実配線。
- [[wi-58-continuous-access-evaluation-agent-revocation]] の T003 (Agent 側 Enforcement) が
  先に完了し、実装パターンが固まってから着手するのが望ましい (depends_on)。

## Tasks

- [ ] T001 [Spec] `UserRevocationEpoch`・`CheckUserRevocationEpoch`/`AdvanceUserRevocationEpoch`
      を追加し、`Introspect` の `ensures` を拡張し、`ReceiveSecurityEvent` の subject 解決を
      User にも対応させる。
- [ ] T002 [Domain/Persistence] `UserRevocationEpoch` の domain/ports/db_memory/db_postgres を
      `AgentRevocationEpoch` と同じ構造で実装する。
- [ ] T003 [Enforcement] `UserDisabled`/`UserSoftDeleted`/`UserDeleted`/`SessionEnded`/
      `RefreshTokenReuseDetected`/パスワード変更/MFA・authenticator reset を
      `AdvanceUserRevocationEpoch` に接続し、OAuth2 `Introspect` で評価する。
- [ ] T004 [Verify] Agent 側と同様のシナリオ (kill 前後 token、duplicate/out-of-order SET、
      cross-tenant subject、既存の front/back-channel logout との役割分担) を検証する。

## Verification

- `just check-scl` / `just verify-spec`
- `just test-go`
  - reason: revocation epoch の単調増加、Introspect の fail-closed 判定、各トリガーからの
    epoch 前進を検証する。
- 手動: ユーザーを disable/delete、または管理者がセッションを強制終了 → 発行済み token が
  即時に無効化されることを確認する。

## Risk Notes

人間ユーザーは Agent よりトラフィック量・トークン発行頻度が大きく、`UserRevocationEpoch` への
書き込み頻度 (特にログアウトのたびに前進させるなら) が Agent 側より遥かに高くなりうる。
`SessionEnded` の粒度 (通常ログアウトも毎回 epoch を前進させるべきか、明示的な強制失効のみに
絞るか) を実装時に検討し、書き込み負荷とセキュリティ要求のバランスを取る。判断に迷う場合は
「失効させる」側に倒す (fail-closed) が、性能への影響が無視できない場合は本 WI 内で NFR を
明示的に検討する。
