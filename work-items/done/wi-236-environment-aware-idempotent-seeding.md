---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-17
depends_on: []
change_kind: operations
initial_context:
  scl:
    Seeding:
      - glossary.SeedProfile
      - models.SeedRequest
      - models.SeedPlan
      - interfaces.SeedData
      - objectives.SeedScalability
      - scenarios.環境別の明示profileが選択される
      - scenarios.同一seedの再適用はno-opになる
      - scenarios.productionでdemoまたはperformance profileは拒否される
      - scenarios.manual driftは上書きせずconflictになる
  decisions: [ADR-084, ADR-118]
  source: [backend/seeding, backend/cmd/internal/bootstrap, backend/cmd/idmagic]
  tests: [backend/seeding, backend/cmd/internal/bootstrap]
  stop_before_reading: [frontend]
affected_spec:
  - { context: Seeding, kind: interface, element: SeedData }
  - { context: Seeding, kind: scenario, element: "環境別の明示profileが選択される" }
---

# 環境別 seed を安全・冪等・スケーラブルに計画して投入できるようにする

## Motivation
現行の seed は `backend/cmd/internal/bootstrap/seed.go` と `federation.go` にデモデータが
ハードコードされ、`SKIP_DEMO_SEED` を設定しない限りサーバ起動時に全環境へ投入される。
固定 ID と repository の upsert により一部の重複は避けているが、再実行のたびに時刻と
Argon2id password hash を更新し、`password_history` に新しい行を追加するため、同じ入力の
2 回目が no-op になる意味での冪等性はない。seed 単独のテスト、差分確認、件数指定、
大規模データ生成、部分失敗後の安全な再開も存在しない。

また、管理コンソール用 first-party client のような稼働に必要な bootstrap データと、既知の
デモパスワードを持つ user、サンプル group / application / federation trust が同じ処理に混在する。
このままでは環境ごとの最小データを選べず、production 相当環境へデモ資格情報を誤投入する危険が
ある。運用者が投入前に変更内容を確認でき、同じ宣言を何度適用しても状態が変わらず、性能検証用の
大量データも bounded resource で生成できる seed 基盤が必要である。

## Scope
- **SCL**:
  - 新しい `Seeding` bounded context の `spec/contexts/seeding.yaml` に `SeedProfile` を追加する。
  - `models` に環境・profile・件数・生成 seed・実行 mode を持つ `SeedRequest` と、
    create / update / no-op / conflict の件数および redacted 差分を持つ `SeedPlan` を追加する。
  - `interfaces.SeedData` に plan と apply の契約を追加し、dry-run は永続状態を一切変更せず、
    apply は同一入力の再実行を no-op にすることを保証する。
  - `objectives.SeedScalability` に batch size、同時実行数、メモリ上限、profile ごとの件数上限、
    大規模 plan の streaming 出力と完了時間の計測方法を定義する。
  - `scenarios` に環境別 profile、dry-run、同一 apply の再実行、drift、部分失敗からの再実行、
    production への demo/performance profile 拒否を追加する。
- **decision**:
  - bootstrap 必須データとサンプルデータの境界、profile manifest、安定 ID、drift policy、
    同時実行排他、transaction / checkpoint、secret 解決、ライブラリ採否を ADR に記録する。
- **seed core**:
  - `backend/seeding` を operations bounded context とし、profile/policy/planner/apply orchestration
    だけを所有する。各 resource の意味・不変条件・永続化は既存 record context に残す。
  - `bootstrap`、`development`、`test`、`performance` 等の明示的 profile を定義し、環境から
    暗黙推測せず、選択した manifest と許可 policy を検証して deterministic な plan を作る。
  - first-party client 等の必須 bootstrap と、user / group / application / OAuth2 client /
    SAML SP / WS-Fed RP 等のサンプル・合成データを分離する。
  - profile と logical key から安定した ID と値を生成し、現在状態との semantic diff により
    create / no-op / conflict を判定する。既定では既存の手動変更を上書きせず fail-closed とし、
    seed 管理対象だけを明示的な reconcile policy で更新できるようにする。
  - 大量データは全件をメモリへ保持せず batch / stream で計画・適用し、固定 generator seed と
    index から再現できるようにする。件数には安全な上限と明示 override を設ける。
- **application / adapters**:
  - 既存 bounded context の port / use case を通す orchestration とし、memory と
    PostgreSQL/Valkey の両構成で同じ契約を満たす。直接 SQL fixture でドメイン制約を迂回しない。
  - cross-context の依存順、bounded transaction、checkpoint、同時実行 lock を定義し、途中で
    失敗しても同じ request を再実行して収束できるようにする。
  - password hash、password history、`created_at` / `updated_at`、group membership など、現行で
    再実行時に変化または追加される値を no-op 時には変更しない。
- **CLI / operations**:
  - server 起動から独立した seed command と `just` recipe を追加し、profile、件数、generator
    seed、`dry-run` / `apply`、出力形式を指定できるようにする。
  - server の暗黙 demo seed を廃止し、ローカル `just dev` は development profile を明示する。
    production は bootstrap 必須データ以外を既定拒否し、既知のデモ資格情報を作成しない。
  - plan/apply の結果を構造化して出力するが、password、client secret、TOTP secret、hash、PII の
    全量は出力しない。
  - production bootstrap の first-party client は `SEED_FIRST_PARTY_REDIRECT_URIS` に明示した HTTPS URI
    だけを使い、未指定または localhost URI は書き込み前に拒否する。
- **tests / documentation**:
  - memory と PostgreSQL の contract test、dry-run 前後 snapshot、連続 apply、並行 apply、drift、
    partial failure、異なる profile、上限超過、大規模 batch の回帰テストを追加する。
  - README に profile matrix、安全な実行手順、production guard、secret 注入、既存
    `SKIP_DEMO_SEED` からの移行を記載する。

## Out of Scope
- production の tenant 設定を環境間で昇格する GitOps import/export。これは
  [[wi-102-realm-declarative-config-export-import]] が扱い、本 WI の seed manifest をその代替にしない。
- 検索・集計 query の性能改善と性能 objective 本体。大規模 seed 基盤は
  [[wi-161-large-tenant-performance-foundation]] から再利用できる境界にするが、read path の改善は
  同 WI が扱う。
- test ごとに DB を truncate して fixture を再ロードする仕組み。既存 `pgfixtures` の置換は行わない。
- production user、秘密鍵、実 client secret、実 TOTP secret の生成・保管。
- seed 対象外の既存データを削除する prune。削除要件が生じた場合は別 WI で安全契約を定義する。

## Plan
- 最初に現行 seed を inventory し、各レコードを `required bootstrap`、`development demo`、
  `test fixture`、`performance synthetic` に分類する。first-party client を demo user と同時に
  skip できる現状を解消し、production の安全な初期管理者 provisioning は seed と分離する。
- profile manifest は少量の宣言データと generator 設定だけを保持する。環境名からの自動選択では
  なく、運用者または `just` recipe が profile を明示し、environment policy がその組み合わせを
  許可するか検査する。profile の合成順と override 規則を安定化し、未知 key は拒否する。
- dry-run と apply は同じ planner を使う。dry-run は redacted plan のみを返し、apply はその plan
  と同じ semantic comparison を再評価してから書き込む。大規模 plan は summary と JSON Lines を
  stream し、全 operation の保持を要求しない。
- 冪等性は「同一 manifest・generator seed・secret version を再適用したとき、2 回目は全件 no-op
  で永続状態・履歴・timestamp が変わらない」と定義する。stable ID、自然 key、canonical value
  comparison を使い、manual drift は既定 conflict、明示 reconcile だけを更新とする。
- cross-context の単一巨大 transaction は採らず、依存順に bounded batch を適用する。各 operation
  を冪等にし、実行 lock と checkpoint または同等の再開情報により並行実行と途中失敗を扱う。
- seed framework の採用は見送る。
  - [`go-testfixtures/testfixtures`](https://github.com/go-testfixtures/testfixtures) はテスト DB の
    cleanup / fixture load が主目的で、非破壊な環境 seed と domain port 経由の適用に合わない。
  - [`romanyx/polluter`](https://github.com/romanyx/polluter) は YAML から DB へ直接投入する
    test-oriented library で、dry-run、semantic diff、memory adapter、domain invariant を扱わない。
  - [`brianvoe/gofakeit`](https://github.com/brianvoe/gofakeit) は固定乱数 seed による再現可能な値生成を
    提供するが、投入・冪等性・drift policy は提供しない。標準ライブラリによる index-based generator
    で性能・可読性が不足すると実測された場合だけ、performance profile の非機密表示値生成に限定して
    採用を再評価する。
- 大規模 profile の具体的な users / groups / applications / memberships 件数は
  [[wi-161-large-tenant-performance-foundation]] の scale profile と同期し、本 WI 側で別の規模定義を
  増やさない。通常 verify は小規模 contract、opt-in recipe は大規模 throughput を検証する。

## Tasks
- [x] T000 [Boundary] Seed の profile・policy・plan/apply orchestration を新しい `Seeding` bounded context とし、record context の resource ownership と分離する判断を ADR-118 と Architecture に記録する。
- [x] T001 [Inventory/ADR] 現行 seed 全件と repository semantics を棚卸しし、bootstrap/demo 境界、profile manifest、stable ID、drift、排他、checkpoint、secret、ライブラリ不採用方針を ADR に具体化する。
- [x] T002 [SCL] `Seeding` bounded context を追加し、`SeedProfile`、`SeedRequest`、`SeedPlan`、`SeedData`、`SeedScalability` と正常・境界・失敗・拒否 scenario を `spec/contexts/seeding.yaml` に追加する。
- [x] T003 [Render] `just scl-render` で SCL 派生物を同期する。
- [x] T004 [Core] profile parser / validator、deterministic generator、semantic planner、redacted streaming result、件数上限を実装する。
- [x] T005 [Apply] bounded batch、依存順、同時実行 lock、部分失敗から再実行可能な apply orchestration を port / use case 経由で実装する。
- [x] T006 [Idempotency] password/history/timestamp/membership を含む全 seed 対象を、同一入力の再実行では永続状態を変更しないよう修正する。
- [x] T007 [CLI/Startup] 独立 seed command と `just` recipe を追加し、server の暗黙 demo seed、`just dev` の明示 profile、production guard、`SKIP_DEMO_SEED` 移行を整備する。
- [x] T008 [Verify] memory/PostgreSQL adapter 共通の port 契約に対し、dry-run、連続・並行 apply、drift、partial failure、profile 分離、上限、batch の contract と race test を追加する。
- [x] T009 [Scale/Docs] wi-161 と共有する大規模 profile の opt-in 計測 recipe、README の profile matrix・安全手順・secret/PII 非出力説明を追加する。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just test-go-race`
- 新設する軽量 seed contract 用 `just` recipe
- 新設する opt-in 大規模 seed throughput 用 `just` recipe
- 手動: development profile を dry-run し、表示された create / no-op / conflict 件数と apply 結果が一致し、secret と PII 全量が出力されないことを確認する。
- 手動: 同じ manifest、generator seed、secret version で apply を 2 回実行し、2 回目が全件 no-op で DB snapshot、password history、`created_at` / `updated_at` が変化しないことを確認する。
- 手動: 途中で 1 batch を失敗させて同じ request を再実行し、重複や欠落なく目的状態へ収束することを確認する。
- 手動: production environment に demo / performance profile を指定すると、書き込み前に fail-closed で拒否され、既知のデモ資格情報が作成されないことを確認する。

## Risk Notes
seed は資格情報と複数 bounded context の永続状態を書き換えるため、誤った環境判定や drift の
自動上書きは security incident とデータ破壊に直結する。profile は明示選択、production は拒否既定、
dry-run は副作用なし、secret は参照注入・非表示、manual drift は fail-closed とする。

大規模 seed を単一 transaction や全件 in-memory plan にすると DB lock、OOM、長時間 rollback を招く。
bounded batch と streaming summary を用い、途中失敗は各 operation の冪等性で再実行可能にする。
並行実行では安定 ID だけに依存せず排他と conflict handling を検証する。外部 faker の出力変更を
永続 ID や比較値へ混ぜると依存更新だけで全件 drift になるため、採用時も表示用の非識別値へ限定する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  Seeding bounded context、明示 profile、冪等な plan/apply、独立 CLI と運用手順を実装した。
- **Affected Guarantees State**:
  - production は bootstrap 以外の profile を書き込み前に拒否する。
  - 同一 request の再適用は seed 管理対象の timestamp と password history を変更しない。
  - manual drift は上書きせず conflict として失敗する。
- **Verification Results**:
  - `just yaml-check` — passed（SCL / Architecture / Work Item validation）
  - `just verify-go` — passed（Go lint and race-enabled tests）
  - `just seed development development dry_run` — passed（create plan contained only redacted logical keys）
