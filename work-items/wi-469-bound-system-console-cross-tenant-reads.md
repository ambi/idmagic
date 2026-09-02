---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p2
depends_on: [wi-462-control-plane-console-single-entry]
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-014 }
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-008 }
  - { path: docs/contexts/data-keys/scenarios.md, requirement: REQ-DATAKEYS-006 }
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-004 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ListTenants }
  - { path: spec/contexts/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.ListTenantKeyHealth }
  - { path: spec/contexts/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.ListTenantDataKeyHealth }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
---

# システムコンソールのテナント横断読出しを有界化する

## Motivation

システムコンソールの既存三画面は、HTTP 要求ごとに全テナントを読み出す。

`ListTenants` は全テナント取得後に各テナントのクォータと使用量を別々に読むため、1回の表示に `1 + 2N` 回の永続化呼出しが発生する。

署名鍵ヘルスは全テナントを直列に走査し、テナントごとに全鍵取得とプロバイダー健全性確認を行う。

DEK ヘルスも全テナント取得後にテナントごとの有効鍵を読む。

さらに監査一覧は結果をキーワードカーソルで有界化している一方、各ページで絞り込み対象全体の正確な `count(*)` を実行する。

これらの処理時間、DB 接続占有、外部 KeyProvider 呼出し数は、要求されたページサイズではなく全テナント数または全一致イベント数に比例する。

システムコンソールへの一回のアクセスが大規模環境ほど認証系と共有する資源を長時間占有するため、水平スケールだけでは解消しない。

## Scope

- システムコンソールのコレクション API に署名付きカーソルとページサイズ上限を導入し、応答を一ページへ制限する。
- `ListTenants` のクォータと使用量をページ内テナントに対する一括読出しへ変更し、テナントごとの N+1 呼出しをなくす。
- 署名鍵と DEK の健全性を、要求中の全テナント直列走査ではなく、期限と取得時刻を持つスナップショットからページングして返す。
- 健全性の更新はワーカーが固定件数のバッチで行い、外部プロバイダー呼出しに期限と同時実行上限を設ける。
- 一部テナントの鍵またはプロバイダー確認が失敗しても、他テナントの結果を返し、対象行に失敗状態と最終成功時刻を示す。
- 横断監査一覧では各ページの正確な総件数計算を廃止し、前後カーソルの生成に必要な有界クエリだけを同期実行する。
- UI に次ページ、再読込、スナップショット取得時刻、期限切れ、部分失敗を表示する。
- HTTP 要求一件当たりの DB 行数、外部呼出し数、同時実行数が設定された上限を超えないことを検証する。
- Tenancy、SigningKeys、DataKeys、Audit、System の仕様と容量計画を更新する。

## Out of Scope

- API 全体の負荷連動入場制御、PostgreSQL 接続予算、HPA。
  [[wi-396-prioritize-login-under-saturation]] が扱う。
- ジョブレーン内のテナント公平性。
  [[wi-427-job-lane-tenant-fairness]] が扱う。
- PostgreSQL の読み取りレプリカまたはパーティション構成。
  [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] が扱う。
- 監査エクスポートの実行方式。
  [[wi-464-bound-the-synchronous-audit-event-export]] が扱う。
- KeyProvider 自体の高可用性構成。
- 正確な総件数を別の集計読み出しモデルとして提供すること。

## Design

HTTP 読出しの不変条件を、**一回の要求が行う処理量は全体件数ではなくページサイズの定数倍で決まる**こととする。

テナント一覧、署名鍵ヘルス、DEK ヘルスは共通のページング形式を使うが、各コンテキストが自身の読み出しポートとスナップショットを所有する。

単一の System データベーステーブルへ各コンテキストの内部モデルを集約する案は採らない。

鍵状態の意味と更新規則が System へ漏れ、SigningKeys と DataKeys の正が二つになるためである。

健全性画面の再読込は同期全件走査を開始せず、更新ジョブを冪等に要求して現在のスナップショットを返す。

ワーカーは固定件数のテナントを処理して継続カーソルを保存し、一つの実行が全テナントをメモリへ保持しない。

外部 KeyProvider の障害はテナント行の状態として記録し、最初の失敗で一覧全体を 500 にしない。

監査一覧から正確な総件数を外すと、総ページ数と最終ページへの直接リンクは返せなくなる。

キーセットページングの前後移動は維持し、正確な件数より要求時間と DB 負荷の上限を優先する。

## Plan

1. 現在の各 API について、テナント数に対する DB 呼出し数、外部呼出し数、メモリ保持件数を計測する受け入れ RED を作る。
2. ページング、スナップショット鮮度、部分失敗、総件数を返さない監査ページの規範を先に更新する。
3. テナント、クォータ、使用量のページ内一括読出しポートを実装する。
4. SigningKeys と DataKeys にスナップショットとバッチ更新ジョブを実装する。
5. 監査ページから正確な `Count` を外し、カーソル契約と UI を更新する。
6. システムコンソールの三画面をページングと鮮度表示へ対応させる。
7. 小規模と大規模のテナント集合で、要求ごとの処理量が上限内で一定であることを測定する。

## Tasks

- [ ] T001 [Measure] 既存 API の DB 呼出し数、外部呼出し数、保持件数をテナント数別に測定し、RED を確認する。
- [ ] T002 [Spec] ページング、スナップショット鮮度、部分失敗、監査総件数廃止を規範化する。
- [ ] T003 [Tenancy] ページ内テナントのクォータと使用量を一括取得する読み出しポートを実装する。
- [ ] T004 [SigningKeys] 有界バッチで更新する署名鍵ヘルススナップショットを実装する。
- [ ] T005 [DataKeys] 有界バッチで更新する DEK ヘルススナップショットを実装する。
- [ ] T006 [Audit] 横断一覧から正確な総件数クエリを外し、キーセットカーソルだけで移動できるようにする。
- [ ] T007 [UI] ページング、鮮度、期限切れ、部分失敗を日英で表示する。
- [ ] T008 [Acceptance] テナント数を増やしても HTTP 要求一件の処理量が上限を超えないことを確認する。
- [ ] T009 [Observability] バッチ遅延、スナップショット年齢、部分失敗、外部呼出し時間をメトリクスへ追加する。
- [ ] T010 [Verify] 仕様生成物を再生成し、容量と標準の検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run check-monitoring`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

スナップショット化によって表示は現在値ではなく取得時点の値になる。

取得時刻と期限切れを応答と画面へ必ず出し、古い成功値を現在の正常状態として表示しない。

バッチの同時実行上限を高くすると、同期走査をワーカーへ移しただけで KeyProvider と DB の集中は残る。

HTTP 要求とワーカーの両方について上限を測定する。

監査の正確な総件数を削除すると UI 契約が変わるため、前後移動と絞り込み変更の回帰を確認する。

新しいページング契約、スナップショットの意味、規範 ID を公開するため、`reversibility` は irreversible とする。
