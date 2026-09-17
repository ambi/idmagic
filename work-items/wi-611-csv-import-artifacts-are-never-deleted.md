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
  - { path: docs/standards.md, requirement: GDPR-ERASURE }
---

# CSV インポートの成果物が消えず、消去した利用者の個人識別情報を持ち続ける

## Motivation

`csv_artifacts` と `csv_artifact_chunks` は、利用者とグループの CSV インポートの入力と、結果のエラーページを保持する。
入力の CSV には、メールアドレス、氏名、利用者名などの個人識別情報が含まれる。

この二つのテーブルの行を削除する処理は、`backend` のどこにもない。
インポートが完了しても成果物は残り、テナントの `export_artifacts_bytes` の Soft 上限が蓄積を警告するだけである。

利用者を消去すると、`users` の行は匿名化され、従属する資格情報は物理的に削除される。
しかし、その利用者を取り込んだ CSV の成果物は元の値のまま残る。
`GDPR-ERASURE` は「削除要求後は法的保存義務を除く PII を定義済み期間内に消去する」と定めており、成果物はこの消去の対象から漏れている。

## Scope

- CSV インポートの成果物に保持期間を定め、期間を過ぎた成果物と分割片を削除する。
- 保持期間の値を、所有する Context の判断として記録する。
- 期間内の成果物が消去済みの利用者の値を含み得ることを、`GDPR-ERASURE` の「定義済み期間」と矛盾しない長さに収める。

## Out of Scope

- 利用者を消去した時点で、その利用者を含む成果物だけを書き換えること。成果物は行単位の索引を持たないので、保持期間で消すほうを先に扱う。
- バックアップに残る成果物。[[wi-612-reapply-erasure-after-restore]] が扱う。

## Risk Notes

保持期間を短くしすぎると、実行待ちのインポートジョブが入力を読めなくなる。削除はジョブが終端状態に達した成果物に限る。

## Verification

- `mise run verify`
