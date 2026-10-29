---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sharedsignals/features/revocation.md, requirement: REQ-SHAREDSIGNALS-001 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-012 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-047 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-081 }
---

# 束縛解除と再委譲の後も発行時のエージェントに基づいて停止を効かせる

## 動機

静的調査では `AccessTokenIsRevoked` が token の `client_id` から現在の Agent 束縛を引き、その Agent の epoch だけを見ることを確認した。
`agent_id` と `act` は照合せず、現在の束縛が無ければ未失効として返す。
束縛解除、別 Agent への再束縛、削除、中間 Agent の停止後に、発行済み token の失効を見落とす懸念がある。
workload Agent と交換する client が異なる経路も同じ条件に当たる。
これは静的な懸念であり、再現テストはまだ実施していない。

加えて Token Exchange の入力は通常 JWT の署名と期限を調べる introspector を通るが、HTTP の `/introspect` が加える失効判定を通っていない。
明示的に revoke した subject/actor token が再交換されないことを、同じ保証として確認する必要がある。

## 対象範囲

- 発行時の主体と委譲参加者を、現在の束縛の有無だけに依存せず識別して失効判定する。
- 解除、再束縛、削除、A から B への再委譲、workload と requester の違いを含める。
- token exchange の subject/actor 入力、`/introspect`、IdMagic 自身の Bearer 境界で共通の active 判定を使う。
- 所有者の停止後に明示的 Disabled が残る現行規範と、仕様の旧い「所有者復帰で自動再開」の記述を整合させる。
- 外部での JWT オフライン検証、イントロスペクション、SSF の保証と遅延を区別する。

## 対象外

- [[wi-378-user-visible-agent-delegation-grants]] のユーザー別委譲撤回。
- [[wi-323-caep-ssf-for-human-user-sessions]] の人間用 epoch、[[wi-16152-redeliver-dead-lettered-security-events]] の再配送。
- Agent の task/session 実行基盤、既に確定した業務処理の取消。
- [[wi-80420-enforce-agent-access-at-external-resource-servers]] の外部 PEP API。修正にこの新機能の完成を要求しない。

## 設計

最初に、束縛を残した正常な停止を対照にして問題候補を再現する。
要件差分の候補は、発行時に権限を支えた Agent の停止、束縛解除、削除が、以後の online 検証と再交換で有効な token として扱われないことである。
無効化と署名検証を一つの意味の active 判定にまとめ、取得不能を許可へ倒さない。

境界の入力は OAuth2 の発行時クレーム、IdManagement の Agent と束縛のライフサイクル、SharedSignals の epoch と所有する保存表現とする。
D1/D2/D7 に従い、Agent 状態と失効の所有者は維持し、OAuth2 が読み取りを合成する案を第一候補とする。
現在の `client_id` lookup だけを維持する案は、解除と再束縛の前後で同じ token の意味が変わるため採らない。
発行時の不変な Agent 識別、binding 世代、廃止記録、JTI 系譜の候補を比較し、削除後の識別と保持期間を満たす最小の表現を選ぶ。
新しい公開契約が必要なら、状態所有者の読み取りポートと既存のイベント契約を比較し、任意 SQL の公開や shared への業務判断の移動は避ける。

主要な入力の候補は `IssuedAgentContext`、`RevocationFacts` とし、`EvaluateTokenActive(claims, facts, now)` の計算と、状態取得と通知の作用を分ける。
束縛解除、epoch の確定、並行した token 発行の順序を規範にし、削除の cascade で必要な失効記録が消えないようにする。
`iat` と epoch の時刻精度、同時刻の発行、chain 参加者の issuer/subject の解決、欠けた履歴の扱いは実装前に確定する。

移行時は旧 token が持たない識別を無条件に有効とせず、旧 token の最大 TTL までの拒否または明示された制限を選ぶ。
保存表現を拡張する場合は段階移行し、失効記録を保持する。
ロールバックで失効済み token を復活させず、安全を保てない交換経路は停止する。
外部 JWT の署名だけによる検証は有効期限まで失効を検知できないため、即時停止と呼ばない。
online 検証または PEP を使う資源は、許可の鮮度と障害時の拒否を契約にする。

## 計画

1. 解除、再束縛、削除、中間停止、明示 revoke の再現と仕様差分を確定する。
2. token と失効記録の識別、時刻、保持、移行の問いを解決する。
3. active 判定を各 online 境界へ接続し、競合と障害時を確認する。

## タスク

- [ ] T001 [Diagnosis] 静的懸念を公開入口で再現し、正常な停止と新しい正当な token を対照にする。
- [ ] T002 [Spec] 発行時の識別、停止と解除の保証、時刻境界を規範化し、所有者復帰について古い記述を直す。
- [ ] T003 [Acceptance] 解除、再束縛、削除、chain 停止、revoke 済み入力で発行しない RED を作る。
- [ ] T004 [App] 共通の active 判定を introspect、交換入力、Bearer 検証へ接続する。
- [ ] T005 [Persistence] 必要な識別と保持を所有モジュール内で実装し、失敗時の部分確定と並行発行を扱う。
- [ ] T006 [Verify] メモリと PostgreSQL、障害、重複イベント、順序逆転、時刻精度、旧 token の移行を検証する。

## 検証

- `mise run check-spec` と `mise run check-boundaries`：停止規範とモジュール所有を確認する。
- `mise run test-go-package -- ./oauth2/...`、`mise run test-go-package -- ./sharedsignals/...`、`mise run test-go-package -- ./idmanagement/...`：失効、束縛、chain の検証を行う。
- `mise run test-go-package -- ./oauth2/internal/handlers_http` と `mise run test-go-package -- ./shared/http/server_http`：正式な token endpoint と introspect/Bearer 境界から、revoke 後に token と業務作用が生じないことを確認する。
- `mise run test-go-package -- ./oauth2/internal/token/usecases --race`：発行と解除、停止と再束縛の競合を検証する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP と DB を使う検証はサンドボックス外で実行する。
- 独立した観測として、停止前に発行した実 token が online 境界で拒否されることを検査する。

## リスク

現在の束縛に依存する判定を直すと、旧 token の互換性と削除時の保持方針が変わる。
失効情報を失う移行と、障害時の許可への退避を避ける。
外部 JWT の停止遅延と内部の見落としを混同せず、修正した範囲と残る制約を明示する。
