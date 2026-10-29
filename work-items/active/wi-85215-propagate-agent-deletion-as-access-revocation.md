---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-10-30
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sharedsignals/features/revocation.md, requirement: REQ-SHAREDSIGNALS-001 }
  - { path: spec/modules/sharedsignals/models.tsp, symbol: RevocationReason }
---

# Agent の削除でも失効エポックを進め、AgentAccessRevoked を外部へ伝播する

## 動機

[[wi-41012-keep-agent-revocation-effective-across-delegation-and-rebinding]] で、IdMagic 自身の online 検証は削除した Agent のトークンを無効と判定するようになった。
判定は Agent が存在しないことを読むだけで、SharedSignals の失効エポックは進まず、`RevocationEpochAdvanced` と `AgentAccessRevoked` も発行しない。
強制終了、無効化、束縛の解除はいずれも `AgentAccessRevoked` を SET として送信側のストリームへ投影するので、より強い停止である削除だけが外部の受信側へ届かない。
受信側は削除した Agent のセッションとトークンを止める契機を得られない。

さらに `agent_revocation_epochs` の Agent への外部キーは `ON DELETE CASCADE` であり、削除の確定より後に動く反応器はエポックを記録できない。
削除前に進めたエポックも行ごと消える。

## 対象範囲

- `AgentDeleted` を `AgentRevocationReactor` の契機に加え、削除した Agent のエポックを削除の時刻へ進め、`RevocationEpochAdvanced` と `AgentAccessRevoked` を発行する。
- 失効の理由に `AgentDeleted` を加える（ドメインの列挙、TypeSpec の `RevocationReason`、スキーマの CHECK、SET の `reason` と `initiating_entity` の対応）。
- `agent_revocation_epochs` から Agent への外部キーを外し、削除した Agent のエポックを保持する。
- REQ-SHAREDSIGNALS-001 の契機に削除を加え、revocation.md のモデルの表を更新する。

## 対象外

- 削除以外の契機の変更、SET の再配送（[[wi-16152-redeliver-dead-lettered-security-events]]）。
- 削除した Agent のエポックの保持期間の短縮。行は Agent ごとに 1 行で、Agent の識別子は再利用しないため、無期限に保持する。
- IdMagic 内の online 検証。[[wi-41012-keep-agent-revocation-effective-across-delegation-and-rebinding]] が Agent の不在で判定している。

## 設計

`AgentDeleted` は `DeleteAgent` が削除の確定後に発行し、管理 API の `ReactiveEmit` が反応器へ渡す。
反応器は既存の `AdvanceRevocationEpoch` を呼ぶだけで、投影（`ProjectAgentAccessRevoked`）は Agent を引かずにイベントの `AgentID` から SET の主体を組むので、削除後でも成り立つ。

候補：

| 候補 | 判断 |
| --- | --- |
| 外部キーを外し、削除後にエポックを記録する（採用） | 既存の反応器とイベントの経路をそのまま使える。テーブル所有者は SharedSignals のままで、IdManagement の表への参照が減る |
| 削除の前にエポックを進める（IdManagement が SharedSignals を同期で呼ぶ） | IdManagement から SharedSignals への逆向きの依存が要り、モジュール間の依存規則に反する |
| エポックを記録せず SET だけを送る | `AgentAccessRevoked` を発行する経路が 2 つになり、ローカルの失効を外部への伝播より先に確定させる判断（sharedsignals の decisions.md）と食い違う |

`tenant_id` の `tenants` への外部キー（`RESTRICT`）は残す。
テナントの物理削除の経路はないため、Agent の cascade に頼っていた後始末は失われない。

## 計画

1. revocation.md の契機と REQ-SHAREDSIGNALS-001、TypeSpec の `RevocationReason` を更新し、`mise run check-spec` と `mise run build-site` を通す。
2. 削除した Agent のエポックが記録され `AgentAccessRevoked` が投影されることを、管理 API の削除から観測する RED を作る。
3. 列挙、反応器、SET の理由の対応、スキーマを更新する。スキーマは `infra/schema/README.md` の手順で DROP CONSTRAINT の計画をレビューする。
4. メモリと PostgreSQL の両アダプターで、削除後のエポックの記録と保持を確かめる。

未解決の問い：SET の `initiating_entity` を削除で `admin` とするか。強制終了と同じ管理者の操作なので `admin` を第一候補とし、実装前に確定する。

## タスク

- [ ] T001 [Spec] REQ-SHAREDSIGNALS-001 の契機、モデルの表、`RevocationReason` に削除を加える。
- [ ] T002 [Acceptance] 管理 API の削除で、エポックの記録と送信側ストリームへの `AgentAccessRevoked` の配送が起きない RED を観測する。
- [ ] T003 [App] 反応器、列挙、SET の理由の対応を更新し、GREEN にする。
- [ ] T004 [Persistence] `agent_revocation_epochs` の Agent への外部キーを外し、PostgreSQL アダプターで削除後の記録を確かめる。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-published-api-compat`：列挙の追加が公開契約の互換性を保つことを確かめる。
- `mise run test-go-package -- ./backend/sharedsignals/internal/usecases` と `./backend/sharedsignals/internal/handlers_http`：反応器と、管理 API からの投影を確かめる。
- `mise run check-schema` と PostgreSQL の epoch リポジトリのテスト：外部キーを外した後の記録と保持を確かめる。HTTP と DB を使う検査はサンドボックスの外で実行する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

`RevocationReason` への値の追加は、SET の受信側と監査の読み手に見える。既知でない値を拒否する受信側があれば、削除の SET だけが受理されない。
外部キーを外すと、存在しない Agent のエポックを書けるようになる。書き込みは反応器が IdManagement のイベントから行うものに限られるので、任意の値が入る経路は増えない。
