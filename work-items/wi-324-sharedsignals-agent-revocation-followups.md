---
status: pending
authors: [claude]
risk: medium
created_at: 2026-08-08
depends_on: [wi-58-continuous-access-evaluation-agent-revocation]
---

# SharedSignals エージェント失効の実装中に保留した項目 (Hard Quota・新規 token 発行停止・認証経路統一) を解消する

## Motivation

[[wi-58-continuous-access-evaluation-agent-revocation]] (CAEP/SSF によるエージェント revocation,
[[ADR-057]]) の T001〜T007 完了時点で、実装中に発見しながら意図的にスコープ外へ出した項目が
複数残っている。個々は wi-58 の中核 (kill-switch を既発行トークンへ即時反映する) を損なう
ものではないため先送りしたが、いずれも「fail-closed / 迷ったら失効する側に倒す」という
ADR-057 の方針に対して緩みが残る箇所であり、まとめて解消しておく価値がある。

1. **Hard Quota 未実装**: `spec/contexts/sharedsignals/SPECIFICATION.md` の `RegisterSsfTransmitterStream`/
   `RegisterSsfReceiverStream` は T001 時点で `QuotaExceededError` を宣言しているが、
   Tenancy 側に新規 quota resource を追加する作業が stream CRUD 単体 (T005) の範囲を大きく
   超えるため実装を見送った。specification の宣言と実装が乖離した状態が残っている。
2. **所有者オフボード後も新規 token 発行が止まらない**: kill-switch ([[ADR-048]]) は
   `Agent.Status` を変更し `IsActive()` (`backend/idmanagement/agent/domain/agents.go:62`) を
   fail-closed にするため新規発行を防ぐが、所有者オフボード (`UserDisabled`/`UserSoftDeleted`/
   `UserDeleted`) は配下 Agent の `AgentRevocationEpoch` こそ前進させる (wi-58 T003)ものの
   `Agent.Status` 自体には触れない。結果、オフボードされた所有者配下の Agent は
   **既発行 token は epoch で失効するが、client_credentials による新規発行は引き続き成功しうる**。
3. **admin/account portal の Bearer 認証経路が epoch/denylist をバイパスする**:
   `backend/shared/http/support_http/auth.go` の `resolveAuthnContext` は
   `TokenIntrospector.IntrospectAccessToken` を直接呼び、`AgentRevocationEpoch` 判定と
   `AccessTokenDenylist` の両方を実装している `oauth2/token/usecases.IntrospectToken` を経由
   しない。wi-58 T006 の調査で、Agent の `client_id` と User の `sub` (`users.id`) が独立生成の
   UUIDv4 で交差しないため**現状は悪用不可能**と確認済みだが、revocation 判定を通らない
   コードパスが存在すること自体は defense-in-depth の観点で望ましくない。
4. **`ManagementApiClient` (Agent 主体の API scope) の実 enforcement が無い**: SharedSignals の
   admin API は specification 上 `ManagementApiClientReadSharedSignals`/`WriteSharedSignals` policy を
   宣言しているが (T005)、このリポジトリには `ManagementApiClient` principal を実装している
   context が一つも無い ([[wi-274-application-admin-api-restructure-and-scopes]] の Risk Notes
   に「管理 API の PAT 解決は監査 actor 帰属と CSRF 除外を含む横断認証カーネルを要する」と
   明記された既知の保留事項)。SharedSignals 固有の問題ではなく全 context 共通の欠落。
5. **RFC 9493 Subject Identifiers の完全相互運用が無い**: `ReceiveSecurityEvent` の subject 解決
   (`backend/sharedsignals/usecases/receive.go` の `extractCaepEventAndSubject`) は idmagic 自身の
   transmitter が生成する自前ワイヤーフォーマット (`events.<uri>.subject.{subject_type,
   tenant_id, principal_id}`) だけを解釈する。外部の SSF transmitter (別 IdP・ガバナンス基盤) が
   RFC 9493 標準の Subject Identifiers (`email`/`iss_sub`/`opaque` 等) で SET を送っても
   subject を解決できず拒否される。

## Scope

- `spec/contexts/tenancy/SPECIFICATION.md`: `TenantQuota`/`TenantUsage` に `SsfStream` を Hard Quota resource
  として追加する (既定値は ADR-134 の他リソースと同じ桁感で検討)。
- `spec/contexts/sharedsignals/SPECIFICATION.md`: `RegisterSsfTransmitterStream`/`RegisterSsfReceiverStream` の
  `QuotaExceededError` に対応する `requires` を実装可能な形に確認・調整する。
- `spec/contexts/identity-management/SPECIFICATION.md` または `spec/contexts/oauth2/SPECIFICATION.md`: 所有者オフボード時に
  配下 Agent の新規 token 発行を止める `ensures`/`requires` (Agent 側で明示的な状態遷移を新設する
  か、client_credentials 発行時に owner の Active 状態を確認するガードを追加するかは `## Design`
  で判断する)。
- `backend/shared/http/support_http/auth.go`: `resolveAuthnContext` を
  `oauth2/token/usecases.IntrospectToken` 経由に統一する (直接 `TokenIntrospector` を呼ぶ経路を
  廃止し、epoch/denylist 判定を一本化する)。
- `backend/sharedsignals/usecases/receive.go`: RFC 9493 Subject Identifiers (少なくとも `email`
  ないし `iss_sub`) を解釈できるよう subject 解決を拡張する。

## Out of Scope

- **`ManagementApiClient` の横断認証カーネルそのものの設計・実装**: [[wi-274]] が既に「監査 actor
  帰属と CSRF 除外を含む横断認証カーネルを要する」と評価した通り、これは SharedSignals 固有では
  なく全 admin API に共通する、本 WI 単体の範囲を大きく超えるアーキテクチャ投資である。優先度が
  上がった時点で専用の WI を別途立てるべきで、本 WI では着手しない (SharedSignals の
  `ManagementApiClient` policy は specification 宣言済みのまま、他 context と足並みを揃えて未実装で残す)。
  同種の scope 未配線は [[wi-320-agent-management-api-scope-wiring]] (`agents:read`/`write`) でも
  個別に扱われているが、そちらも IdManagement 単体の scope 配線可否を決めるだけで、横断カーネル
  自体は解決していない。
- [[wi-323-caep-ssf-for-human-user-sessions]] が対象とする User 側の revocation epoch 拡張 (別 WI
  で既に計画済み、本 WI は Agent 側の残課題のみを扱う)。
- SSF/CAEP の相互運用認証取得そのもの (外部 transmitter/receiver との契約締結等)。

## Design

- **Hard Quota**: 他リソース (`ResourceUsers`/`ResourceAgents` 等、`backend/tenancy/domain/tenancy.go`)
  と同じパターンで `ResourceSsfStreams` を追加し、`AdminStreamDeps` (T005) の
  `RegisterSsfTransmitterStream`/`RegisterSsfReceiverStream` に quota check を挿入する。Transmitter
  と Receiver を同一 quota で数えるか別枠にするかは実装時に判断する (SsfStream は方向を問わず
  1 テーブルの行なので、素直には同一 quota が妥当と見込む)。
- **所有者オフボード後の新規発行停止**: 2つの選択肢がある。
  (a) 所有者オフボード時に配下 Agent を明示的に `Disabled` 状態へ遷移させる (`IsActive()` が
  fail-closed に効く、既存の kill-switch と同じ経路を再利用できる)。
  (b) client_credentials 発行時に毎回 owner の `User.IsActive()` を確認するガードを追加する
  (Agent 自体の状態は変えない、所有者が復帰すれば自動的に発行が再開する)。
  (a) は「オフボード = 配下 Agent も無効化」という意味が明確で kill-switch との対称性が高い一方、
  所有者が復帰しても Agent が自動復活しない (別途 re-enable が要る)。(b) は所有者の状態変化に
  自動追随するが、token 発行のたびに owner lookup が増える。ADR-057/ADR-048 の fail-closed 方針に
  照らすと (a) の方が明快で推奨だが、決定は本 WI の実装時に確定する。
- **auth.go の経路統一**: `resolveAuthnContext` は portal scope 判定・DPoP 検証など
  `IntrospectToken` には無いロジックも持つため、単純な置き換えではなく `IntrospectToken` の
  `IntrospectDeps` (Agent/RevocationEpoch/Denylist repo) を `Authenticator` に注入し、
  `resolveAuthnContext` 内部で `IntrospectToken` を呼んでから既存の portal scope / DPoP 判定を
  続ける形にする。影響範囲が admin/account 全 API に及ぶため、既存の認証 test suite
  (`support_http/auth_test.go` 等) の regression を重点的に確認する。
- **RFC 9493 対応**: 完全実装ではなく、外部相互運用でまず必要になりそうな `email`/`iss_sub`
  形式から着手する。idmagic 自身の transmitter が使う自前形式との共存 (フォーマット判別) が要る。

## Plan

- **T002 と T003 をセキュリティ修正として先行させる** (2026-08 に優先度を整理)。
  4 項目は当初すべて同列に並べていたが、実態として性質が異なる:
  - **T002 (所有者オフボード後の新規 token 発行停止)** — 現状、オーナーが無効化・削除されると
    revocation epoch は進むが `Agent.Status` は変わらないため、**配下の Agent は
    `client_credentials` で新しい token を取り続けられる**。退職者のエージェントが生き残るという
    実害のある穴であり、他の項目と同列に置くのは実態に合わない。
  - **T003 (`resolveAuthnContext` の経路統一)** — epoch と denylist を迂回する経路が残っている。
    後述の Risk Notes のとおり、現状非悪用であることは**設計ではなく偶然に依存している**。
  - T001 (Hard Quota) は specification と実装の宣言不一致であり、機能的な穴ではない。
  - T004 (RFC 9493) は外部相互運用の具体的なニーズが出てから着手でよい。
- したがって着手順は **T002 → T003 → T001 → T004** とする。T003 は影響範囲が広いため
  単独 PR で慎重に進める (Risk Notes 参照)。
- 各項目は独立に完了・レビューできるため、`## Tasks` は項目ごとに RED→GREEN で進める。
- この優先度整理は [[wi-369-agent-capability-survey-2026-08]] の棚卸しによる。

## Tasks

- [ ] T001 [specification/Quota] `SsfStream` を Hard Quota resource として追加し、
      `RegisterSsfTransmitterStream`/`RegisterSsfReceiverStream` の `QuotaExceededError` を実装する。
- [ ] T002 [specification/Enforcement] 所有者オフボード後に配下 Agent の新規 token 発行を止める
      (`## Design` の (a)/(b) を確定し実装する)。
- [ ] T003 [Auth] `support_http/auth.go` の `resolveAuthnContext` を `IntrospectToken` 経由に統一し、
      admin/account portal の Bearer 認証でも epoch/denylist が一貫して効くようにする。
- [ ] T004 [Receiver] `ReceiveSecurityEvent` の subject 解決に RFC 9493 Subject Identifiers
      (`email`/`iss_sub` 等) の解釈を追加する。
- [ ] T005 [Verify] 各項目の受け入れシナリオ (quota 超過での 429/403、オフボード後の新規発行
      拒否、admin/account portal での revoke 済み token 拒否、外部形式 SET の受理) を検証する。

## Verification

- `just check` (specification/architecture/work-items)
- `just build-go` / `just verify-go`
  - reason: quota 判定・token 発行拒否・認証経路統一・RFC 9493 subject 解決のいずれも
    fail-closed 境界を持つため、race-enabled test と lint を通す。
- 手動: 所有者オフボード → 配下 Agent の新規 token 発行が拒否されることを確認する。
  admin/account portal を revoke 済み token で叩き 401 になることを確認する。

## Risk Notes

- **T003 (auth.go 経路統一) が最もブラスト半径が大きい**: admin/account portal の全 API が通る
  共通認証経路の変更であり、regression が起きると管理機能全体の認証が壊れうる。既存の
  `support_http/auth_test.go` の regression を重点的に見た上で、単独 PR として小さく進める。
- **T001 (Hard Quota) は Tenancy 側のスキーマ変更を伴う**: 既存テナントへの安全な既定値付与は
  ADR-134 の移行方針 (généreux な初期上限 → reconciliation で追随) に従う。
- 4項目とも個別には低〜中リスクだが、まとめて1セッションで済ませようとすると regression の
  切り分けが難しくなるため、`## Plan` の通り T003 は他と独立した PR に分離することを推奨する。
- **T003 の「現状非悪用」は設計ではなく偶然に依存している**: `resolveAuthnContext` が epoch と
  denylist を迂回しても現時点で悪用できないのは、Agent の `client_id` と User の `sub` の
  ID 空間が交差しないためである。これは**そう設計したから安全なのではなく、たまたま衝突して
  いないから安全**という状態であり、principal の識別子体系に触れる将来の変更 (ID 形式の統一、
  新しい principal 種別の追加、外部 IdP からの subject 写像など) で無言のうちに壊れる。
  「現状非悪用だから急がない」という判断の根拠にしてはならない。
- **`ManagementApiClient` の横断的欠落は本 WI では解決しない**: Out of Scope に記載のとおり
  本 WI の範囲外だが、同じ穴が [[wi-320-agent-management-api-scope-wiring]]、本 WI、
  [[wi-274-application-admin-api-restructure-and-scopes]] の Risk Notes の **3 箇所で追跡され
  続けている**。各 WI が個別に見送り続ける限り解決しないため、次にこの穴を踏む work item が
  出た時点で専用の work item に切り出す (この判断は
  [[wi-369-agent-capability-survey-2026-08]] に記録した)。
