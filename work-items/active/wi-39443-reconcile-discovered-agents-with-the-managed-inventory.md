---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-074 }
---

# 外部で発見したエージェントを管理台帳と突き合わせる

## 動機

[[wi-59-agent-governance-guardrails-audit-inventory]] は登録済み Agent の所有者不在、休眠、未束縛を扱う。
外部の SaaS、MCP 接続、開発基盤で発見された未登録 Agent は、登録済み台帳だけでは把握できない。
組織全体の台帳として使うには、外部の発見器から来た観測を管理済みの主体と突き合わせ、責任者が扱いを決める必要がある。
本項目は、全サービスの発見器を自作する代わりに、発見結果の取り込みと確認に範囲を絞る。
未登録主体の利用状況を確認する機能であり、既存 Agent の停止修正や本人の委譲付与より後に扱う p2 とする。

## 対象範囲

- 認証した観測元から、外部 issuer と subject、外部登録 ID、観測時刻、利用資源、責任者候補を上限付きで取り込む。
- 外部観測を未確認の事実として保存し、管理済み Agent、Client、Workload binding と突き合わせる。
- 未登録、一致候補、一致確認、除外、古い観測の状態を区別し、曖昧な一致を自動確定しない。
- 管理者が一致を確認するか、既存の登録手順を使って有効な所有者を持つ Agent にする。
- [[wi-59-agent-governance-guardrails-audit-inventory]] の台帳へ未確認観測を区別して表示し、[[wi-213-access-certification-campaigns]] は確認済みの管理対象の定期レビューを担う。

## 対象外

- SaaS ごとのクローラー、端末のスキャン、ネットワーク傍受、モデルの実行内容の収集。
- 観測だけを理由に Agent を有効化したり、資格情報を束縛したり、権限を付与すること。
- 外部サービスのアカウントを自動停止する操作、グローバル Agent Registry。
- 全 Agent の発見を保証する宣言。

## 設計

要件差分の候補は、認証済み観測元の未確認情報を権限付与と分離し、確認した関連付けだけを管理台帳へ反映できることである。
`ObservedAgent` は管理用 `Agent` の第二の正本ではなく、発見元と鮮度を持つ観測である。
名前や表示名の一致だけでは主体を結ばず、外部 issuer/subject と登録済みの対応を材料にする。
一つの外部 subject が複数の Agent に一致する場合は未確認のままにする。

境界の入力は IdManagement の登録と所有者、WorkloadIdentity の対応付け、Sourcing の取り込み、wi-59 の読み取り台帳である。
D1/D2/D8 に従い、外部観測を Sourcing に置く案と、Agent の管理確認を IdManagement の機能に置く案を比較する。
観測元の信頼と再送を隠す範囲、確認後の登録の所有者、既存の SCIM と同じか異なる更新意味を比較し、保存先を名前から決めない。
Agent に未確認データを直接上書きする案は、外部の観測元へ所有者と権限の書き込みを委ねるため採らない。
初期の利用側は一つの認証済み取り込み元と管理画面に限り、新モジュールや公開範囲が要るかは実装前に記録する。

主要型の候補は `AgentObservation`、`ObservationSource`、`AgentMatchDecision` とする。
`MatchObservation(observation, candidates)` は候補と理由を計算し、`ImportObservations(ctx, source, batch, now)` と `ConfirmAgentMatch(ctx, decision)` が永続化と監査を編成する。
取り込み元の機械認証、独立した import 権限、観測の一意キー、再送、順序逆転、保持期限、既存 Agent 削除後の扱いを実装前に決める。

移行は観測専用の追加で始め、既存 Agent を一括変換しない。
ロールバックでは取り込みを停止し、確認履歴を保持する。
確認済み対応を戻しても既存の権限を増減しない。

## 計画

1. 一つの外部発見結果の具体例と、取り込み元の信頼境界を確定する。
2. 観測と管理主体を分けたモデル、確認操作、状態遷移を仕様化する。
3. 取り込み、冪等な突合、管理者の確認、台帳表示を接続する。

## タスク

- [ ] T001 [Design] 責務の所有者、取り込み権限、一意キー、保持と一致の根拠を比較して決める。
- [ ] T002 [Spec] 観測の登録、確認、除外、再観測の規則と、観測から権限を付与しない保証を追加する。
- [ ] T003 [Acceptance] 取り込み後の未確認表示、管理者による一致確認、無権限の確認拒否の RED を作る。
- [ ] T004 [App] 観測の保存、上限、冪等性、曖昧一致の保留、確認の監査を実装する。
- [ ] T005 [UI] 管理済み主体と未確認観測を区別し、発見元と鮮度、一致根拠を表示する。
- [ ] T006 [Verify] 偽の観測元、テナント越え、重複、順序逆転、期限切れ、同名衝突、並行確認と Agent 削除を検証する。

## 検証

- `mise run check-spec` と `mise run check-boundaries`：観測と権限の分離、保存先と依存を確認する。
- `mise run test-go-package -- <設計で選んだ所有モジュール>`：正常取り込み、重複、古い観測、曖昧な一致を確認する。具体的なパッケージは着手時に確定する。
- `mise run test-go-one -- <所有パッケージ> <正式HTTP入口の受け入れテスト>`：取り込み後も Agent、binding、role に無許可の変更がないことを確認する。
- `mise run verify-ui` と対象の `test-ui-e2e-file`：画面を変更した層の検査と、未確認情報から権限が付かない操作を確認する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP と DB を使う検証はサンドボックス外で実行する。

## リスク

未確認の観測を本人性や所有権の証明として扱うと、偽の発見結果から権限が生じる。
取り込みと確認の権限を分け、名前では結ばず、曖昧な候補は管理者の確認待ちにする。
観測は外部システムの稼働状況を示すため、最小の属性、保持期限、テナント分離を定める。
