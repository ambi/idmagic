---
status: accepted
authors: ["tn"]
created_at: 2026-07-20
---

# ADR-134: テナントリソースクォータの分類・既定値・移行方針

## コンテキスト
idmagic はマルチテナント IdP として稼働する。一部のテナントによる過剰なリソース（User, Group, Client, Sessionなど）の生成が、システム全体の可用性・パフォーマンス・コストに悪影響を及ぼすのを防ぐ必要がある（Noisy Neighbor問題の防止）。現状はテナントごとのリソース作成数に上限がなく、無制限にリソースを作成できる。これを制限し、テナントごとの Budget を導入する必要がある。

## 決定
テナントリソースクォータ（Tenant Resource Quotas）を以下のように導入・分類し、管理する。

1. **Quotaの分類**
   - **Hard Quota**: トランザクション内で同期的に評価され、超過すると作成（Create/Register等）がエラーとなり拒否される厳格な制限。対象：`users`, `groups`, `agents`, `applications`, `oauth2_clients`, `active_sessions`, `consents`, `active_jobs`
   - **Soft Quota**: 超過しても操作自体は成功するが、非同期的に警告（Warning/Audit Event）が通知される遅延評価の制限。対象：`audit_events_retained`, `export_artifacts_bytes`

2. **既定値（Default Quotas）**
   新規テナント作成時に自動付与される初期値。
   - `users`: 10,000
   - `groups`: 1,000
   - `agents`: 100
   - `applications`: 50
   - `oauth2_clients`: 100
   - `active_sessions`: 50,000
   - `consents`: 10,000
   - `active_jobs`: 10

3. **System Admin Override (調整)**
   System Admin は個別のテナントに対してクォータ上限値を個別に上書き（Override）できる。テナント管理者（Tenant Admin）は自テナントの利用量と上限を参照できるが、変更はできない。

4. **既存テナントへの移行方針 (Migration)**
   導入直後に既存テナントが突然のロックアウトや作成不能に陥ることを避けるため、移行時には「十分に大きい安全な上限値（例：既存の利用実績の2倍、または現在の既定値の10倍など）」を一律で割り当てる。その後、バックグラウンドの Backfill / Reconciliation ジョブで実際の利用量（Usage）を集計し、必要に応じて System Admin が手動で調整や警告を行う。

## 却下した代替案
- 案 A: APIレート制限のみで防ぐ
  - なぜ採らないか: 短期間のスパイクは防げるが、長期間にわたる継続的なリソース生成によるDB容量の肥大化・コスト増を防げないため。
- 案 B: Soft Quotaのみとする
  - なぜ採らないか: 悪意のあるAPI呼び出しやバグ（無限ループなど）によって短時間でDBリソースが枯渇するリスクを、事後対応では防げないため。

## 影響
- **SCL**: `Tenancy` context に `TenantQuota`, `TenantUsage` モデル、及び作成系APIに関連するクォータ検証の仕様（precondition、scenarios）を追加する。
- **データ**: DBにテナントごとのクォータ設定テーブルと利用量（Usage counter）テーブルが追加され、リソース作成時に更新される。
- **運用**: Reconciliation ジョブが追加され、利用量カウンタと実データ件数の不整合を定期的に補正する。
