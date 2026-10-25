---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/requirements/standards.md, requirement: GDPR-ERASURE }
---

# CSV インポートの成果物の保持期間を、消去期限と実行中ジョブに整合させる

## Motivation

`csv_artifacts` と `csv_artifact_chunks` は、利用者とグループの CSV インポートの入力と、結果のエラーページを保持する。
入力の CSV には、メールアドレス、氏名、利用者名などの個人識別情報が含まれる。

起票時は、この二つのテーブルの行を削除する処理がなかった。
その後、[wi-18703](../done/wi-18703-align-identity-management-with-its-state-tables.md) が、作成から 30 日を過ぎた成果物と分割片を Batch の `retention-sweep` で消す処理を追加した。
2026-10-25 の棚卸しで `PurgeExpiredCSVArtifacts` と本番の Batch への配線を確認したため、削除処理の新設は本項目から外す。

利用者を消去すると、`users` の行は匿名化され、従属する資格情報は物理的に削除される。
その利用者を取り込んだ CSV の成果物は、保持期間が過ぎるまで元の値を含み得る。
`GDPR-ERASURE` が求める消去期限と現在の 30 日の関係、および実行中のインポートの入力を消さない条件が残る確認対象である。
現在の削除は作成時刻だけで判定するため、起票時のリスクに書いた「終端状態に達した成果物に限る」という条件をそのまま満たすとは扱わない。

## Scope

- 現在の 30 日の保持期間を `GDPR-ERASURE` の消去期限と照合し、所有する Context に判断を記録する。
- 長期間実行待ちまたは実行中のインポートについて、入力を消さない条件と、消去期限を超えて保持しない条件を決める。
- 既存の保持期間の削除を再利用し、不足する条件だけを実装とテストへ反映する。

## Out of Scope

- 利用者を消去した時点で、その利用者を含む成果物だけを書き換えること。成果物は行単位の索引を持たないので、保持期間で消すほうを先に扱う。
- wi-18703 が追加した成果物と分割片の削除処理を重複して作ること。
- バックアップに残る成果物。[[wi-612-reapply-erasure-after-restore]] が扱う。

## Risk Notes

保持期間を短くしすぎると、実行待ちのインポートジョブが入力を読めなくなる。削除はジョブが終端状態に達した成果物に限る。

## Verification

- `mise run verify`
