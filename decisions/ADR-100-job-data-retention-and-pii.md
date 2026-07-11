---
status: accepted
authors: [tn]
created_at: 2026-07-12
---

# ADR-100: ジョブ params/result は平文 JSONB で保持し、終端ジョブは TTL で purge する

## コンテキスト
[[ADR-098-durable-job-queue-skip-locked-lease]] で定義した `jobs` テーブルの
`params`（enqueue 時の入力）と `result`（完了時の出力）には、将来の consumer
（CSV 一括インポートのユーザー属性、SCIM 同期のペイロード等）次第で PII が
含まれ得る。`jobs` は tenant-owned aggregate として `tenant_id` を必須にし
（`ARCHITECTURE.md` の tenant_id 4 分類ルール、
[[ADR-082]]/[[ADR-083-tenant-id-key-policy]] の方針に整合）、無期限に
蓄積させると監査対象データが際限なく増える。at-rest 暗号化の要否と保持期間
を、実装前に確定させておく必要がある。

## 決定
1. **at-rest 暗号化は本 WI では追加しない**: `params`/`result` は素の
   `JSONB` カラムに平文で保持する。個別ジョブ種別が機密性の高い PII を
   扱う場合は、呼び出し側（各 consumer の usecase）が enqueue 前に
   最小化・マスキングする責務を負う。より強い保護が要る場合は、将来
   [[wi-97-envelope-encryption-at-rest]] のフィールド暗号化パターンを
   `params`/`result` カラムへ適用することを妨げないが、本 WI のスコープ
   （core runtime のみ）には含めない。
2. **保持期間**: 終端状態 (`Succeeded` / `Failed` / `Canceled`) に到達した
   ジョブは、既定 30 日（環境変数 `JOB_RETENTION_DAYS` で上書き可能）で
   物理削除する。`Queued` / `Running` のジョブは purge 対象にしない。
3. **purge の実行主体**: [[ADR-099-job-worker-execution-model-and-fault-tolerance]]
   で `idmagic-worker` に移設する periodic goroutine（旧
   `startRetentionSweep`）が、既存の監査/認証イベント保持期間 sweep と
   同じ周期で `jobs` テーブルの終端レコード TTL purge も行う。専用の
   `JobKind` は設けない（テナント横断の一括 DELETE のため、tenant-scoped
   Job としては表現しない、[[ADR-099-job-worker-execution-model-and-fault-tolerance]]
   の却下案を参照）。

## 却下した代替案
- **暗号化を本 WI で必須にする**: 現時点で `jobs` の実consumer は疎通確認
  用の no-op/echo job のみで、実際にどの粒度の PII が乗るかは各機能側 WI
  （CSV import、SCIM 再送等）でないと確定できない。先回りして汎用の
  フィールド暗号化を core runtime に組み込むと、鍵管理・パフォーマンスの
  複雑さを、必要性が確定する前に負うことになる。
- **保持期間を無期限にする**: 監査要件が明示的にない限り、実行ログとしての
  ジョブレコードを無期限保持する理由がなく、テーブル肥大化とテナント横断の
  PII 残留リスクを増やすだけである。
- **`Queued`/`Running` も TTL 対象にする**: 実行前・実行中のジョブを消すと
  ジョブの取りこぼしに直結するため、終端状態のみを purge 対象にする。

## 影響
- `deploy/schema/postgres.sql` の `jobs` テーブルは `params`/`result` を
  `JSONB` 型（暗号化なし）で定義する。
- `backend/bootstrap/retention.go`（`idmagic-worker` へ移設後）に、
  既存の監査/認証イベント purge に加えて `jobs` 終端レコードの
  `DELETE ... WHERE status IN ('succeeded','failed','canceled') AND
  updated_at < now() - $JOB_RETENTION_DAYS` 相当の purge を追加する。
- `spec/contexts/jobs.yaml` の `objectives` に `JobRecordRetention`
  （`kind: retention`、既定 30d）を追加する。
