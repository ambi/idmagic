---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-08
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/standards.md, requirement: GDPR-PROCESSING-RECORDS }
---

# 監査記録の保持について、決定と実装が食い違っている

## Motivation

`docs/contexts/audit/decisions.md` は「監査レコードは GDPR 第 30 条 (処理活動の記録) を根拠に、追記のみで 7 年間保持する。削除やアーカイブのインターフェースは提供しない」と決めている。

実装はそうなっていない。

- `backend/audit/ports/audit_event_repository.go` は `RetentionCutoff` を定義し、`db_postgres` と `db_memory` の両方が `DeleteOlderThan` を実装している。これは決定が「提供しない」と書いた削除のインターフェースそのものである。
- `backend/authentication/usecases/retention.go` の `DefaultRetentionPolicy` は、成功・一般監査を 365 日、失敗詳細を 30 日、集約と MFA とセッションを 90 日で切る。7 年ではない。
- `backend/cmd/internal/bootstrap/retention.go` の `RunRetentionSweepOnce` がそれを適用し、`backend/cmd/idmagic-batch/main.go` が外部スケジューラー向けの一括処理として公開している。`mise run batch` から到達できる。

同じ port のコメントは「本実装は監査保管 (SCL objectives.AuditLogRetention 7y) ではなく、直近のオペレーション可視化を目的としたショートウィンドウ用途を想定する」と書いており、食い違いを認識してはいる。ただし `AuditLogRetention` という名前はリポジトリのどこにも宣言されておらず、この注記が指す先は既に存在しない。

**どちらが正しいかは決まっていない。** 7 年保持が正しいなら sweep の既定と `DeleteOlderThan` の位置づけを見直すことになり、種類別の短い保持が正しいなら `decisions.md` の決定を書き換えることになる。どちらであれ規範の変更なので、判断を含めて 1 つの work item が持つ。

[[wi-502-back-cross-cutting-standards-rows-with-tests]] が `GDPR-PROCESSING-RECORDS` にテストを対応付ける過程で見つけた。同行の `Statement` は「セキュリティおよび認可イベントの監査記録を定義済みの期間保持する。保持期間は Audit Context が定める」であり、**「定義済みの期間」があること自体は成り立っている**（同項目の `TestRetentionKeepsSecurityAndAuthorizationRecordsWithinTheDefinedPeriod` が期間の内側と外側の対で観測している）。食い違っているのは、その期間が Audit Context の決定と一致していない点である。

## Scope

- 7 年保持と種類別の短い保持のどちらが製品の意図かを決め、根拠を残す。
- 決めた側に合わせて、`docs/contexts/audit/decisions.md` か `RetentionPolicy` と `DeleteOlderThan` の位置づけのどちらかを変える。
- `backend/audit/ports/audit_event_repository.go` の、存在しない `SCL objectives.AuditLogRetention` を指す注記を落とす。
- 行の担い手の位置も合わせて見直す。行は「保持期間は Audit Context が定める」と言うのに、期間を計算する `RetentionPolicy` は `backend/authentication/usecases` にある。

## Out of Scope

- 監査イベントの検索属性と PII の変換方針。`docs/contexts/audit/internals.md` が持つ。
- `GDPR-PROCESSING-RECORDS` へのテストの対応付け。[[wi-502-back-cross-cutting-standards-rows-with-tests]] が済ませた。
- Purge の cascade の欠け。[[wi-513-purge-leaves-webauthn-credentials-and-recovery-codes]] が持つ。
- 監査記録の外部転送 (SIEM、ログストリーミング)。

## Risk Notes

- **決めずに実装だけ揃えてしまう。** 7 年と 365 日は運用費用も法的な意味も違う。どちらが正しいかを先に決め、根拠を `docs/contexts/audit/decisions.md` に残してから片側を動かす。
- **sweep を止めると単一テーブルが肥大する。** `retention.go` は「partition 化は行わない判断のため、retention が単一テーブルの肥大と PII 滞留を抑える唯一の機構になる」と書いている。7 年保持を採るなら、この機構が抜けた後の肥大への答えが要る。
- **監査記録の消去が GDPR-ERASURE と衝突する。** 監査記録は消去の例外側にあるので、保持期間を延ばす方向は `GDPR-ERASURE` の「法的保存義務を除く」と整合するが、失敗イベントに残る平文のユーザー名 (`TestRetentionSweepKeepsFailureUsernamePlaintext`) の扱いは別に判断が要る。

## Verification

- `docs/contexts/audit/decisions.md` の保持の決定と、`RetentionPolicy` の既定および `DeleteOlderThan` の位置づけが一致している。
- `backend/audit/ports/audit_event_repository.go` に、存在しない宣言を指す注記が残っていない。
- [[wi-502-back-cross-cutting-standards-rows-with-tests]] が足した `TestRetentionKeepsSecurityAndAuthorizationRecordsWithinTheDefinedPeriod` が、決めた期間で通る。
- `mise run verify`
