---
depends_on: []
status: completed  # pending | in_progress | completed | cancelled
authors: ["tn"]
risk: medium
created_at: 2026-07-04
change_kind: feature
initial_context:
  specification:
    - spec/contexts/system/SPECIFICATION.md#REQ-SYSTEM-016
    - spec/contexts/system/SPECIFICATION.md#REQ-SYSTEM-017
  source:
    - backend/cmd/internal/bootstrap
    - backend/cmd/idmagic/server.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/cmd/idmagic-batch
    - backend/cmd/idmagic-seed/main.go
    - tools/check/src/check-boundaries.ts
    - justfile
    - README.md
  tests:
    - backend/cmd/internal/bootstrap
    - backend/cmd/idmagic-worker
affected_spec:
  - { path: spec/contexts/system/scenarios.md, requirement: REQ-SYSTEM-016 }
  - { path: spec/contexts/system/scenarios.md, requirement: REQ-SYSTEM-017 }
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
- `SharedConfig` は全 backend プロセスで共有し、`APIConfig`、`WorkerConfig`、`SeedConfig`
  はプロセス固有設定を同じ `ConfigLoader` へ読み込む。batch の restore consistency check は
  DB 以外の設定へ依存させず、専用 loader で `DATABASE_URL` だけを必須 secret として読む。

## Plan
- idmagic (API) が起動時に読む env をすべて `SharedConfig` (Assemble が使う共有部分) と
  `APIConfig` (idmagic 固有部分) へ集約する。domain/usecase へ config package を持ち込まない。
- source は environment のみを対象とする (config file source は Out of Scope、compiled
  default を安全な値とする)。unknown key 検出は見送り (Out of Scope)。
- single-field parse 後に runtime cross-field validation (persistence=postgres には
  DATABASE_URL 必須、key_provider=vault/data_key_provider=openbao には資格情報必須、
  authzen=remote には URL 必須、issuer URL 等) を行い、`Assemble()` や listener 起動より
  前に全 error をまとめて返す。
- field metadata と各 `Load*Config` を生成元に ConfigurationReference を作り、同じキーを
  複数プロセスが異なる既定値で読む場合は各 process section に個別の行を残す。README は
  手書き一覧を持たず生成物へリンクし、リポジトリ検証で定義との乖離を拒否する。

## Tasks
- [x] T001 [Inventory] idmagic (API) が起動時に読む全 env read (bootstrap 配下 + `cmd/idmagic/server.go`)
      を一覧化し、source は environment のみ (file source は Out of Scope として見送り)、
      secret 判定、fail-fast 対象を決定した。T004/T006 で idmagic-worker/idmagic-batch/
      idmagic-seed 固有 env と deploy consumer (compose/CI/manifest) まで棚卸しを完了した。
- [x] T002 [Config Core] `backend/cmd/internal/bootstrap/config.go` に typed `ConfigLoader`
      (String/Secret/Bool/Int/Int32/Uint32/Float/Duration/URL/Enum/List + Require)、`ConfigErrors`
      aggregation、`Secret` (String/GoString/MarshalJSON/MarshalText を常に `[REDACTED]` にする
      redaction)、生成用 field metadata (type/default/required/secret/allowed/constraint) を
      実装した。description は T005 の検査付き表で補い、file source loader は Out of Scope とした。
- [x] T003 [Validation] `sharedconfig.go` (`LoadSharedConfig`: persistence/authzen/webauthn/
      postgres/key_provider/data_key_provider/email·smtp/default_locale/breached_password_checker)
      と `apiconfig.go` (`LoadAPIConfig`: issuer/addr/otel/log_level/request_id/tenant_base_domain/
      trusted_forwarded_hops/rate_limits/http hardening/security headers/drain_grace_period) を
      実装し、`cmd/idmagic/server.go` の `Run()` 冒頭で両方をロードして `loader.Err()` を
      listener 起動・`Assemble()` より前に確認する (REQ-SYSTEM-016)。TLS 組み合わせ検証は
      対象env変数が存在しない (TLS はゲートウェイが終端) ため対象外。RED/GREEN:
      `TestLoadSharedConfigRejectsUnknownAdapterSelectors` で adapter selector の typo も拒否する。
- [x] T004 [Migration] REQ-SYSTEM-016: idmagic-worker/idmagic-batch/idmagic-seed の固有 env を
      `WorkerConfig`/`SeedConfig`/専用 `ConfigLoader` へ移し、旧 `EnvDefault`/`EnvInt`/
      `EnvDuration` を削除した。RED: `TestLoadSeedConfigRejectsUnknownProfileAndEnvironment`、
      GREEN: 同テスト、`TestLoadWorkerConfigRejectsMalformedIntervals`、
      `TestLoadWorkerConfigRejectsInvalidLane`。bootstrap 外の `os.Getenv` 等を境界検査で拒否する。
- [x] T005 [Generation] REQ-SYSTEM-017: `ConfigLoader` が記録する field metadata と
      `RenderConfigReference` から `CONFIGURATION.md` を生成し、`just generate-config-reference` /
      `just check-config-reference` を追加した。RED/GREEN:
      `TestRenderConfigReferenceRecordsTypeDefaultAndRequirement`、
      `TestRenderConfigReferenceKeepsProcessSpecificDefaults`。secret 値、条件付き必須、
      process 固有 default を検証する。
- [x] T006 [Deploy/Docs] README の手書き設定表を ConfigurationReference への導線へ置換した。
      compose/CI/deployment manifest は既存の互換 env 名を継続利用でき、キー名変更は不要だった。
- [x] T007 [Verify] 全 runtime mode の狭い Go テスト、設定リファレンス freshness、仕様・境界・
      API 互換性検査、`just verify`、`just spec-diff` を実行して Completion に記録する。

## Verification
- `just test-go-package ./backend/cmd/internal/bootstrap`
- `just test-go-package ./backend/cmd/idmagic-worker`
- `just check-config-reference`
- `just check-spec`
- `just check-api-compat`
- `just check-boundaries`
- `just verify`
- `just spec-diff`

## Risk Notes
黙って fallback していた挙動を fail-fast に変えると、既存のゆるい設定で
動いていた環境が起動できなくなり得る。全 backend プロセスへ導入し、
`HSTS_INCLUDE_SUBDOMAINS`/`CSP_REPORT_ONLY`/`HSTS_ENABLED` は「`true`/`false` 以外は
静かに既定値」から「`true`/`false` 以外は起動失敗」へ挙動を厳格化した (security に
効く boolean のため意図的な変更)。worker の duration/concurrency/lane と seed の
profile/environment も不正値を既定値へ戻さず起動を拒否するため、deploy 前に生成済み
ConfigurationReference と既存 manifest の値を照合する。

## Completion
- **Completed At**: 2026-08-13
- **Summary**:
  REQ-SYSTEM-016 を idmagic だけから全 backend プロセスへ拡張し、環境由来設定を
  bootstrap の型付き ConfigLoader に集約した。未知・型不正・範囲外・条件付き必須の
  問題は listener、依存接続、seed 適用より前に一括報告し、secret は formatting/JSON/
  生成ドキュメントで redaction する。REQ-SYSTEM-017 を追加し、process 固有 default と
  条件付き必須を含む `CONFIGURATION.md` を Config metadata から生成して freshness を
  リポジトリ検証へ組み込んだ。`just spec-diff` の意味差分は REQ-SYSTEM-017 の追加と
  REQ-SYSTEM-016 の変更。
- **Verification Results**:
  - `just spec-render` - passed (24 documents, 313 operations, 17 API tags, 747 TypeSpec symbols)
  - `just check-spec` - passed
  - `just check-api-compat` - passed
  - `just check-boundaries` - passed
  - `just test-go-package ./backend/cmd/internal/bootstrap` - passed
  - `just test-go-package ./backend/cmd/idmagic` - passed
  - `just test-go-package ./backend/cmd/idmagic-worker` - passed
  - `just test-go-package ./backend/cmd/idmagic-batch` - passed
  - `just test-go-package ./backend/cmd/idmagic-seed` - passed
  - `just check-config-reference` - passed
  - `just check-work-items` - passed
  - `just lint-go` - passed (0 issues)
  - `just verify` - passed
  - `just spec-diff` - added REQ-SYSTEM-017; changed REQ-SYSTEM-016
