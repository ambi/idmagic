---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-001, impact: conforms }
---

# worker が発行した Agent と所有者のイベントでも失効エポックを進める

## 動機

Agent と所有者のイベント（`AgentKilled`、`AgentDisabled`、`AgentCredentialUnbound`、`UserDisabled`、`UserSoftDeleted`、`UserDeleted`）に反応して失効エポックを進める `AgentRevocationReactor` は、`api` のプロセスの配信点（`backend/shared/http/server_http/routes.go`）にだけ組み込まれている。
`worker` のプロセスの配信点（`NewEmitFunc`）は反応器を持たない。

`worker` で実行する操作、たとえばライフサイクルワークフローの `disable_user` の手順が User を無効化すると、IdManagement は配下の Agent を無効化してイベントを発行するが、失効エポックは進まず、外部の受信側への CAEP のイベントの伝播も起きない。
REQ-PLATFORM-001 は、所有者の無効化がログイン、既存のセッション、配下のエージェントのトークンを同時に閉じることを保証している。

wi-26063 で SharedSignals と IdGovernance の内部設計をコードと照合して見つけ、`docs/domain/sharedsignals/design/risks.md` と `docs/domain/identity-governance/design/risks.md` に載せた。

## 対象範囲

- 失効エポックの反応を、`worker` を含むすべてのプロセスの配信点で働かせる。
- ワークフローの `disable_user` で、配下の Agent のトークンがイントロスペクションで無効になることを確かめる。
- 二つの `design/risks.md` の該当の行を消す。

## 対象外

- 配信不能の配送のやり直し。wi-16152 が扱う。

## 設計

反応器の組み立てを、`api` の経路の組み立てから、すべてのプロセスが使う起動処理の配信点（`backend/cmd/internal/bootstrap`）へ移す案を第一の候補とする。
外部への伝播の投影（SET の署名と配送の作成）も同じ場所へ移す。
`worker` のプロセスは SigningKeys の `KeyStore` を持つので、SET の署名は組み立てられる。

## 計画

1. `worker` で User を無効化し、配下の Agent の失効エポックが進まないことを、プロセスの配信点を通すテストで RED として確かめる。
2. 反応器を共通の配信点へ移す。

## タスク

- [ ] T001 [Acceptance] `worker` の配信点で RED を確かめる。
- [ ] T002 [App] 反応器を共通の配信点へ移す。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

反応器が二重に組み込まれると、同じイベントでエポックを二度進めようとする。
エポックの前進は単調で同じ時刻を進めないので害はないが、組み込みを一か所にする。
