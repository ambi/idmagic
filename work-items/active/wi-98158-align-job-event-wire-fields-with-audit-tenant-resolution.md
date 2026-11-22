---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-22
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobEnqueued }
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobStarted }
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobSucceeded }
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobFailed }
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobRetried }
  - { path: spec/modules/jobs/models.tsp, symbol: IdMagic.Contract.JobCanceled }
---

# Jobs のイベントの送信表現を、監査のテナント解決と一致させる

## 動機

2026-11-22 に wi-611 の Batch の配線を検証したところ、`JobCanceled` は発行されて監査へ保存されるが、対象テナントの検索から取得できなかった。
`backend/jobs/domain/events.go` の六種類のイベントは JSON のタグを持たず、`TenantID` と `JobID` をそのまま送信する。
一方、TypeSpec は `tenantId` と `jobId` を定め、`backend/audit/event_record.go` の `NewEventRecord` は `payload["tenantId"]` だけから監査のテナントを決める。
そのため、取り消しイベントの保存だけでは、テナントを束縛した監査検索が成立しない。
送信表現と監査の意味が既存の規範と異なるので p1 とする。

## 対象範囲

- 六種類の Jobs イベントを、TypeSpec の送信表現と一致させる。
- 正式な発行先を通った後に、対象テナントとジョブで監査を取得できることを確かめる。
- 別テナントの記録が検索へ混ざらないことを確かめる。
- 保存済みの旧表現を移行する必要性を判断する。

## 対象外

- [[wi-611-csv-import-artifacts-are-never-deleted]] が扱う CSV 入力の保持と削除。
- [[wi-47234-require-reasons-for-control-plane-changes]] が扱う操作者と理由の追加。

## 設計

イベントの所有者が送信表現を直す案と、Audit が Jobs 固有の旧表現を読む案を比較する。
後者は誤った表現を継続させ、所有者以外へ知識を広げるため、まず所有者の修正を候補にする。
互換性と保存済みの記録の扱いは、着手時に公開状態と利用箇所から判断する。

## 計画

1. `MarshalDomainEvent` と正式な発行先を通す再現を用意する。
2. 六種類の送信表現と監査のテナント束縛をそろえる。
3. 利用側と既存の記録への影響を検証する。

## タスク

- [ ] T001 送信後のフィールド名と、テナント別の監査検索の失敗を再現する。
- [ ] T002 所有者の送信表現を修正する。
- [ ] T003 六種類のイベントと、別テナントの無混入を検証する。

## 検証

- `mise run test-go-package -- ./backend/jobs/domain` で送信表現を検査する。
- `mise run test-go-package -- ./backend/cmd/internal/bootstrap` で正式な発行先と監査を検査する。
- `mise run verify-changed -- main` で逆依存と仕様への影響を検査する。

## リスク

既存のイベントの購読者が大文字のフィールドへ依存している可能性がある。
名前を直すだけでは、保存済みの監査のテナントは補正されない。
