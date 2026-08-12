---
depends_on: []
status: in_progress  # pending | in_progress | completed | cancelled
authors: ["tn"]
risk: medium
created_at: 2026-07-04
change_kind: feature
initial_context:
  source: [backend/cmd/internal/bootstrap, backend/cmd/idmagic, backend/cmd/idmagic-worker]
affected_spec:
  - { path: spec/contexts/system/SPECIFICATION.md, requirement: REQ-SYSTEM-016 }
---

# 起動時の設定を集約・検証し fail-fast させ、単一の設定リファレンスを生成する

## Motivation
現状の設定は bootstrap の各所で `envDefault` / `envInt` / `envDuration` を
直接呼んで読み、値の妥当性を集中検証しない。特に `envInt` / `envDuration` は
パース失敗や負値を「静かに fallback へ戻す」ため、`TRUSTED_FORWARDED_HOPS` や
リテンション間隔のような security/運用に効く値をタイポしても、警告なく既定値で
起動してしまう。本番でこれは、意図した閾値が実は効いていないという silent
misconfiguration を招く。設定項目の網羅一覧も存在せず、spec/SPECIFICATION.md も
「全環境変数一覧は置かない」としているため、運用者が正となる設定表を持てない。

12-factor と Kubernetes のコンポーネント設定検証（無効な設定は起動拒否）に倣い、
idmagic も設定を 1 つの型へ集約し、起動時に検証して不正なら fail-fast し、
型定義から機械生成した設定リファレンスを提供すべきである。ISSUER のような
必須値の欠落・不正 URL、相互矛盾する組み合わせ（postgres 指定なのに DSN 無し等）を
起動時に明確なエラーで止める。

## Scope
- **decision**:
  - ADR は廃止済み (wi-358)。設定を集約する Config 型の位置づけ、fail-fast の対象
    （必須欠落・型不正・範囲外・相互矛盾）は本 work item と owning
    `spec/contexts/system/SPECIFICATION.md` の Design section / REQ-SYSTEM-016 に記録する。
    secret は値をログに出さない方針も同様にそこへ明記する。
- **go**:
  - env 由来設定を単一の Config 構造体へ集約してパース・検証する層を bootstrap に追加する。 検証失敗は Run() の起動前に集約エラーで返し、部分起動させない。
  - `envInt` / `envDuration` の「不正値を黙って fallback」を、少なくとも security/運用に 効く項目では明示エラー化する。範囲・必須・相互依存（persistence=postgres なら DSN 必須等）を検証する。
  - 検証済み Config を各 assemble / handler へ渡し、散在した os.Getenv 直参照を減らす。
  - secret（DSN・SMTP 資格情報・API キー等）は検証エラーやログに値を出さない。
- **documentation**:
  - Config 型定義から設定リファレンス（キー名・型・既定値・必須・意味）を生成し、 README から参照できるようにする。手書き一覧の二重管理を避ける。

## Out of Scope
- 動的な設定ホットリロード。
- 外部設定サービス（Consul / etcd 等）連携。
- 既存の環境変数キー名の一斉改名（互換維持を優先）。

## Design
- ADR は廃止済み (wi-358) のため、判断根拠は本 work item とこの Design section、および
  `spec/contexts/system/SPECIFICATION.md` の Design section / REQ-SYSTEM-016 に記録する。
- 検証と組み立てを2段階に分離する: `LoadSharedConfig`/`LoadAPIConfig` は I/O を一切行わず
  env を型付き struct へパースしながら `ConfigLoader` へ全エラーを集約するだけで、
  `Assemble()`/`assemblePostgres()` 等の実際の adapter 組み立て (DB pool open 等) は
  呼び出し側が `loader.Err() == nil` を確認した後にのみ行う。こうしないと「必須値欠落」と
  「DB接続失敗」のような無関係なエラーが混ざり、REQ-SYSTEM-016 が要求する「1回の起動試行の
  全問題を集約する」を満たせない。
- `SharedConfig` (Assemble が使う persistence/notification/webauthn/authzen/keystore/datakeys)
  と `APIConfig` (idmagic 固有の issuer/addr/rate limit/hardening 等) を分け、同じ
  `ConfigLoader` インスタンスを両方の Load 関数に渡すことで、API プロセス固有の設定と
  API/worker 共有の設定の問題が同一起動試行で一括報告される。
- 外部 env-parsing library (`caarlos0/env` 等) は採用しない: この wi の価値の大半
  (条件付き必須のcross-field検証、secret redaction、既存adapter固有型への詰め替え) は
  どのライブラリを使っても手書きになり、型変換の定型部分 (100〜150 行程度) を削れる
  だけで新規依存のコストに見合わない。
- 第一段の対象は idmagic (API) プロセスに限定した。`Assemble()` は idmagic-worker/
  idmagic-batch/idmagic-seed からも呼ばれる共有関数のため、`SharedConfig` の検証は
  副次的にこれらのプロセスにも及ぶが、worker 固有 (`JOB_*` 等) の移行は T004 で扱う。

## Plan
- idmagic (API) が起動時に読む env をすべて `SharedConfig` (Assemble が使う共有部分) と
  `APIConfig` (idmagic 固有部分) へ集約する。domain/usecase へ config package を持ち込まない。
- source は environment のみを対象とする (config file source は Out of Scope、compiled
  default を安全な値とする)。unknown key 検出は見送り (Out of Scope)。
- single-field parse 後に runtime cross-field validation (persistence=postgres には
  DATABASE_URL 必須、key_provider=vault/data_key_provider=openbao には資格情報必須、
  authzen=remote には URL 必須、issuer URL 等) を行い、`Assemble()` や listener 起動より
  前に全 error をまとめて返す。
- 残タスク (T004-T006): worker/batch/seed 固有 env の移行、field metadata からの
  reference 生成 just recipe、compose/CI/deployment manifest と README の移行は
  後続の work item に切り出す。

## Tasks
- [x] T001 [Inventory] idmagic (API) が起動時に読む全 env read (bootstrap 配下 + `cmd/idmagic/server.go`)
      を一覧化し、source は environment のみ (file source は Out of Scope として見送り)、
      secret 判定、fail-fast 対象を決定した。idmagic-worker/idmagic-batch/idmagic-seed 固有の
      env (JOB_*, WORKER_ID, SEED_* 等) と deploy consumer (compose/CI/manifest) の棚卸しは
      未実施 (T004/T006 で扱う)。
- [x] T002 [Config Core] `backend/cmd/internal/bootstrap/config.go` に typed `ConfigLoader`
      (String/Secret/Bool/Int/Int32/Uint32/Float/Duration/URL/Enum + Require)、`ConfigErrors`
      aggregation、`Secret` (String/GoString/MarshalJSON/MarshalText を常に `[REDACTED]` にする
      redaction) を実装した。field metadata レジストリ (name/type/default/required/secret/
      process/description/example/deprecation を一箇所に持つ表) は T005 (未着手) 向けに見送り、
      今回は各 `Load*Config` 内の個別呼び出しに留めている。file source loader は未実装。
- [x] T003 [Validation] `sharedconfig.go` (`LoadSharedConfig`: persistence/authzen/webauthn/
      postgres/key_provider/data_key_provider/email·smtp/default_locale/breached_password_checker)
      と `apiconfig.go` (`LoadAPIConfig`: issuer/addr/otel/log_level/request_id/tenant_base_domain/
      trusted_forwarded_hops/rate_limits/http hardening/security headers/drain_grace_period) を
      実装し、`cmd/idmagic/server.go` の `Run()` 冒頭で両方をロードして `loader.Err()` を
      listener 起動・`Assemble()` より前に確認する (REQ-SYSTEM-016)。TLS 組み合わせ検証は
      対象env変数が存在しない (TLS はゲートウェイが終端) ため対象外。
- [ ] T004 [Migration] idmagic は完了。idmagic-worker/idmagic-batch/idmagic-seed 固有の
      `os.Getenv`/`EnvDefault`/`EnvInt`/`EnvDuration` 直参照 (JOB_*, WORKER_ID,
      EPHEMERAL_SWEEP_INTERVAL, SHARED_SIGNALS_DELIVERY_INTERVAL, SEED_* 等) は未移行。
      直接 `os.Getenv` を禁止する architecture test/lint も未追加。
- [ ] T005 [Generation] 未着手。
- [ ] T006 [Deploy/Docs] 未着手。compose/CI/deployment examples/README は現行の env 名のまま。
- [x] T007 [Verify] (idmagic scope) missing/malformed/unknown/conflicting values と secret
      redaction を単体テストで、fail-fast (集約エラー・部分起動なし) と正常起動を手動起動で
      検証した。all runtime modes (worker/batch/seed 固有設定) と generated reference
      freshness は T004/T005 が未着手のため対象外。

## Verification
- `just test-go-race` - 合格
- `just lint-go` - 合格
- `just verify` - 合格
- 手動: `TRUSTED_FORWARDED_HOPS=not-a-number PERSISTENCE=postgres` で起動し、
  `DATABASE_URL: is required` と `TRUSTED_FORWARDED_HOPS: must be an integer` の
  2 件が 1 回の集約エラーで返り、listener が起動しないことを確認した。
- 手動: 既定値のみ (memory persistence) で起動し `/livez` が 200 を返すことを確認した。
- 手動 (未実施, T005 未着手のため): 生成された設定リファレンスが実 Config 型と一致することを確認する。

## Risk Notes
黙って fallback していた挙動を fail-fast に変えると、既存のゆるい設定で
動いていた環境が起動できなくなり得る。今回は idmagic (API) プロセスに限定して導入し、
`HSTS_INCLUDE_SUBDOMAINS`/`CSP_REPORT_ONLY`/`HSTS_ENABLED` は「`true`/`false` 以外は
静かに既定値」から「`true`/`false` 以外は起動失敗」へ挙動を厳格化した (security に
効く boolean のため意図的な変更)。worker/batch/seed 固有設定への展開 (T004) は
本番デプロイでの実地確認を経てから段階的に進める。
