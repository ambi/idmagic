---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-19
---

# 署名鍵 rotation を運用で自動化する (scheduler + grace 期間中の旧鍵保持)

## Motivation
`RotateSigningKey` use case と admin endpoint
`POST /api/admin/keys/rotate` は実装済 (ADR-009 = 90 日 cadence /
旧鍵を JWKS に最低 7 日間残す) だが、運用面が抜けている:

  - 定期実行する scheduler / Kubernetes CronJob が無い。admin が
    手動で叩く前提だと、運用者の作業忘れが期限切れリスクに直結する。
  - grace 期間中の旧鍵が `KeyStore` で「retired だが保持」状態として
    明示されていない。`/jwks` は active 鍵だけを返す実装になっている
    可能性が高く、grace 期間内の既発行 JWT の検証が壊れる潜在リスク
    がある (RP 側の `kid` lookup が落ちる)。
  - rotation 後の旧鍵を grace 期間経過後に物理的に dispose する
    経路が無く、archive table が単調増加する。

ADR-009 で決めた cadence / grace の運用面を、追加 ADR なしで実装に
落とし切るのが本 WI のゴール。実 KMS / HSM への差し替えは Phase 8 で
別 WI として扱う。

## Scope
- **decision**:
  - 新規 ADR は不要。ADR-009 (key rotation strategy) の運用実装。
- **scl**:
  - `objectives` の `SigningKeyLifecycle` に `rotation_cadence_days` (既定 90) と `signing_key_grace_days` (既定 7) を value として 宣言する (ADR-009 と整合)。`signing_key_archive_days = 2555` は別物 (compliance archive) として残す。
- **go**:
  - domain:
    - `internal/oauth2/ports/key_store.go` の `SigningKey` 値型に `RetiredAt *time.Time` と `ExpiresAt *time.Time` を追加。 `RetiredAt != nil` の鍵は「signing には使わないが、verify (JWKS) には公開する」状態。`ExpiresAt < now` の鍵は dispose 対象。
    - `KeyStore` interface に以下を追加: `ListPublicKeys(ctx)` (active + retired-not-expired を返す)、`PruneExpired(ctx, before time.Time)`。 既存 `Rotate(ctx)` は既存鍵を `RetiredAt = now` / `ExpiresAt = now + grace_days` で印を付けてから新鍵を生成する ように改修する。
  - usecases:
    - 新規 `PruneExpiredSigningKeys(ctx, deps, now)` を追加。 `KeyStore.PruneExpired` を呼び、削除した kid を `SigningKeyArchived` event として発火する。
    - 既存 `RotateSigningKey` は変更なし (内部で `Rotate` が retired 印を付ける改修に乗る)。
  - http:
    - `/jwks` (`internal/adapters/http/discovery_handler.go` 近辺) を `ListPublicKeys` を呼ぶ実装に変更する。active と retired の両方 を返すことで、grace 期間内の既発行 JWT 検証が落ちないようにする。
  - cmd:
    - 共通 one-shot `cmd/idmagic-batch` に `signing-key-lifecycle` を追加。引数:
        --cadence-days   (default 90)
        --grace-days     (default 7)
      起動時に全 tenant を一度評価し、`cadence-days` を超えていれば rotate。
      grace 経過鍵を archive して終了する。周期・再試行は外部 scheduler が所有する。
  - persistence:
    - `infra/schema/postgres.sql` を更新し、`signing_keys` テーブルに `retired_at TIMESTAMPTZ NULL` と `expires_at TIMESTAMPTZ NULL` を追加。`psqldef --dry-run` で差分をレビューしてからデプロイ前に適用する。 既存行は両カラム NULL で active 扱い。memory adapter にも対応フィールドを反映する。
  - audit:
    - `SigningKeyRotated` (既存) に加え、新規 `SigningKeyArchived` (kid / retiredAt / expiresAt / disposedAt) を SigningKeys SCL と domain event に追加する。outbox 経由で publish。
- **infra**:
  - `idmagic-batch signing-key-lifecycle` を1日1回起動する CronJob skeleton を追加する。
  - 既存 retention sweep も通常 worker から `idmagic-batch retention-sweep` へ移す。
- **documentation**:
  - `idmagic/README.md` の Phase 1 「署名鍵 rotation の自動化」行を completion で除去を記録する。`/jwks` が retired 鍵も返す挙動を README §マルチテナンシー の鍵節と合わせて言及する。

## Out of Scope
- HSM / KMS への鍵移管。`KeyStore` は in-memory + PostgreSQL のまま。 実 KMS は Phase 8。
- Per-tenant 鍵化 (README §マルチテナンシー が Phase 8 と明示)。
- rotation 失敗時の自動 retry / alert (本 WI は失敗を log + event で 可視化するに留め、運用通知は Phase 8 の observability で扱う)。
- 旧鍵で署名された access token の forced revocation (rotation は backward verify を許す設計。Token Denylist 系は別 WI)。

## Plan
- [[ADR-009-key-rotation-strategy]] と `spec/contexts/signing-keys.yaml` の `SigningKeyLifecycle`（active/verifying/retired/archive）を正本にする。現行 KeyStore は tenant-aware で `GetAllKeys` を JWKS/管理 API が共有しているため、公開用 list port を増やさず state/時刻で filtering する責務を KeyStore に置く。
- 現行 `RotateSigningKey` と `KeyStore.Rotate` が旧鍵を verifying として残す経路を維持し、`retired_at`/`archive_after` の設定、期限到来鍵を archive する use case、`SigningKeyArchived` event を追加する。物理削除は archive retention 後の別段階とし、grace 終了と compliance archive を混同しない。
- 外部 scheduler が共通 one-shot `backend/cmd/idmagic-batch signing-key-lifecycle` を起動する。複数 CronJob/手動起動の重複に対して PostgreSQL advisory lock 内で cadence を再判定し、API/worker process 内 ticker は採らない。retention sweep も同じ batch executable の別サブコマンドへ移す（ADR-124）。
- `postgres_valkey` では signing_keys schema/query/sqlc を拡張し、memory adapter も同じ clock-driven lifecycle を実装する。既存行は active と解釈できる nullable migration にする。
- cadence/grace は typed batch config に寄せる。[[wi-103-startup-config-validation-and-reference]] 未完了でも batch 自身で `grace < cadence` と正値を fail-fast 検証できるようにする。

## Tasks
- [x] T001 [SCL] lifecycle/objective、archive event、成功・拒否 scenario を更新し `just scl-render` を実行した。
- [x] T002 [Domain] `SigningKey` に lifecycle 時刻を追加。RED: `TestInMemoryKeyStoreArchivesExpiredVerifyingKey` を先に fail 確認（scenario `grace期間終了後の署名鍵はJWKSから除去されarchiveされる`）→ GREEN。
- [x] T003 [Port/Usecase] 公開用列挙・archive port と `ArchiveExpiredSigningKeys(now)` を実装し、clock/grace を `RotateSigningKey` から注入する。
- [x] T004 [Memory] tenant 分離、JWKS overlap、期限後非公開、archive を実装・テストした。
- [x] T005 [Postgres] nullable lifecycle columns と repository query を追加。tenant advisory lock により concurrent rotation を直列化する。
- [x] T006 [Process] `TestParseSigningKeyLifecycleConfigRejectsGraceAtCadence`（scenario `lifecycle設定が不正なbatchは起動しない`）で fail-fast config を検証。`idmagic-batch` の one-shot retention/signing-key サブコマンドを追加し、通常 worker から periodic retention を除去した。
- [x] T007 [Deploy] retention は毎時、署名鍵 lifecycle は日次で one-shot 起動する個別 CronJob、`Forbid`、deadline、runtime ConfigMap/Secret 注入を追加した。
- [x] T008 [Verify] memory/PostgreSQL adapter の lifecycle test を通し、全 Go test・build・lint、Kubernetes manifest、SCL/YAML を検証した。

## Verification
- `just test-go`
  - reason: KeyStore.Rotate が retired/expires を埋めること、ArchiveExpired が grace 経過鍵だけを archive すること、ListPublicKeys が active+verifying を返すこと、batch config と concurrent due rotation を検証すること。
- `just lint-go`
- `just build-go`
- `just check-k8s dev`
- `just check-k8s prod`
- `just verify`
- PostgreSQL adapter test で2つの同時 `RotateIfDue` のうち1つだけが回転することを確認する。

## Risk Notes
鍵 lifecycle は安全側 fail-close で倒すべき領域だが、本 WI の主リスクは
「rotate のタイミングが揃わない」「旧鍵を消し過ぎる」の 2 点:

  (1) cadence + grace の関係: grace_days < cadence_days を不変条件と
      して持ち、`cmd` 起動時に違反したら exit。grace > cadence は
      archive 表が単調増加する。

  (2) /jwks の retired 公開: retired 鍵を返さないと grace 内の RP 検証が
      落ちる。逆に retired を返し続けると JWKS が肥大化する。
      `ExpiresAt` 経過の鍵は `/jwks` から外す処理を ListPublicKeys に
      埋め、PruneExpired で物理 dispose を別経路にする (公開と dispose
      を分ける)。

  (3) Multi-tenant 共有鍵: 現状の共有鍵を rotate する影響は全 tenant に
      及ぶ。本 WI ではこれを既存制約として明記し、per-tenant 化は
      Phase 8 の前提条件として残す。

## Completion
- **Completed At**: 2026-07-18
- **Summary**: 署名鍵 lifecycle を `retired_at`、`expires_at`、`archived_at` の
  3 時刻に正規化し、旧鍵を grace 期間だけ JWKS へ残して archive する経路を
  memory/PostgreSQL/Vault adapter、HTTP JWKS、監査イベント、one-shot batch、CronJob に通した。
  retention sweep は通常 worker から同じ batch executable の別サブコマンドへ移した。
  `rotated_at` は `retired_at` と重複するため削除した。
- **Verification Results**:
  - `just test-go` - passed
  - `just build-go` - passed
  - `just lint-go` - passed
  - `just yaml-check` - passed
  - `just check-k8s dev` - passed
  - `just check-k8s prod` - passed
  - `just verify` - passed
- **Affected Guarantees State**:
  - JWKS overlap: active と grace 未経過の旧鍵だけを返し、期限後は archive して除外する。
  - tenant isolation: KeyStore と batch は tenant context ごとに鍵を操作する。
  - concurrent rotation: PostgreSQL adapter は tenant advisory lock 内で cadence を再判定し、重複起動でも一度だけ回転する。
  - backwards compatibility: 既存行の lifecycle カラムは NULL を active と解釈する。
  - out of scope: Vault/KMS の永続的な過去鍵ミラー、forced revocation、通知/retry、per-tenant KMS 鍵化は対象外のまま。
