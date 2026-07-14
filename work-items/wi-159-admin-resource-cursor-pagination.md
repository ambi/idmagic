---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 管理対象リソース一覧をカーソルページネーション対応にする

## Motivation
大規模テナントでは User、Group、Agent、Application、Consent、AuditEvent などの管理対象リソースが大量になる。
現状の一覧画面や一部 API は全件または大きめの limit 取得を前提にしており、テナント規模が増えると
レスポンス遅延、DB 負荷、ブラウザ描画負荷、メモリ使用量が急増する。

管理者が日常的に使う一覧は、最大規模でも安定して先頭ページを開け、検索・フィルタ・次ページ遷移を
予測可能なコストで実行できる必要がある。この WI は一覧 API と画面の標準ページング契約を定義し、
主要 admin / account 一覧を cursor-based pagination へ揃える。

## Scope
- **scl**:
  - `IdentityManagement` context の `ListAdminUsers` / group / agent 一覧系 interface に page size、cursor、sort、filter、返却 metadata を追加する。
  - `OAuth2` context の `ListAdminAuditEvents` / consent / client 系 interface に同じ pagination contract を適用する。
  - `Application` context の application 一覧 / assignment 一覧へ pagination contract を適用する。
  - `Authentication` context の account activity / admin sign-in activity / auth event bucket 一覧の limit-only 契約を見直し、必要なものを cursor 対応にする。
  - `flows` と `scenarios` の AdminUsers / AdminGroups / AdminAgents / AdminApplications / AdminConsents / AdminAuditEvents / AccountActivity などにページング UI 要件を追加する。
  - `scenarios` に先頭ページ、次ページ、フィルタ変更、削除を挟むページ遷移、権限拒否、tenant 境界の代表例を追加する。
  - `objectives` に一覧 API の p95 / p99、最大 page size、安定 sort key、深いページでも劣化しないことを追加する。
- **go/usecase/http**:
  - 主要 list usecase の入力・出力を `page_size` / `cursor` / `next_cursor` / `has_more` に揃える。
  - cursor は tenant、sort key、filter 条件を含めて署名または検証し、他 tenant / 他 query へ流用できないようにする。
  - offset pagination や全件取得前提の repository method を、安定 index を使う keyset pagination に置き換える。
- **persistence**:
  - PostgreSQL repository に tenant_id + sort key + id の複合 index を追加し、検索条件ごとの query plan を確認する。
  - memory repository も同じ contract を満たし、cursor の境界・削除済み行・同一 timestamp の tie-break をテストする。
- **ui**:
  - 一覧画面に次/前または「さらに読み込む」操作、page size、filter 変更時の cursor reset、読み込み中/空/エラー状態を実装する。
  - Dashboard や Role detail など既存 list endpoint を集計目的に流用している箇所は、全件取得せず summary endpoint または capped query に切り替える。
- **tests**:
  - contract test、repository test、handler test、主要画面の component/e2e test を追加する。

## Out of Scope
- CSV export や bulk import の非同期化。CSV export は [[wi-148-admin-resource-csv-export]]、job runtime は [[wi-126-async-job-runner]] / [[wi-157-job-admin-operations-surface]] で扱う。
- テナント別の総量制限・作成拒否。これは [[wi-160-tenant-resource-quotas]] で扱う。
- 横断検索や集計 read model の導入。これは [[wi-161-large-tenant-performance-foundation]] で扱う。
- SCIM の RFC 準拠ページング全面対応。必要なら別 WI として切り出す。

## Plan
- 最初に SCL で共通 `PageRequest` / `PageResult` 相当の語彙と interface 契約を定める。
- API は keyset pagination を標準にする。既存 URL との互換のため `limit` は `page_size` へ読み替えてもよいが、offset は新規 contract に入れない。
- Cursor は opaque token とし、UI は中身を解釈しない。tenant_id、filter hash、sort、last key、expiry を含めて改ざんを検出する。
- 実装は監査イベント、ユーザー、グループ、エージェント、アプリケーションの順に、データ量・運用重要度が高い一覧から進める。
- UI は仮想スクロールを先に入れず、ページ単位の DOM サイズを制限する。必要になった画面だけ後続で virtualization を入れる。

## Tasks
- [ ] T001 [SCL] 共通ページング語彙、主要 list interface、UX、scenario、objective を更新する。
- [ ] T002 [Render] `just scl-render` で派生物を更新する。
- [ ] T003 [Go] cursor encode/decode、validation、handler input/output contract を実装する。
- [ ] T004 [Persistence] PostgreSQL / memory repository を keyset pagination 化し、必要な index / migration を追加する。
- [ ] T005 [UI] 主要 admin / account 一覧を cursor contract とページング UI に移行する。
- [ ] T006 [Test] cursor 改ざん、tenant 境界、filter 変更、同一 sort key、削除を挟む遷移の test を追加する。
- [ ] T007 [Verify] `just yaml-check`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: 一覧画面のページ遷移、filter reset、戻る操作は browser behavior を含むため。
- 手動: 1 万件以上の users / groups / audit events を持つテナントで、初期表示、次ページ、filter 変更、詳細遷移からの復帰が軽く動くことを確認する。
- 手動: 別 tenant の cursor、改ざん cursor、古い filter の cursor が拒否または安全に無効化されることを確認する。

## Risk Notes
ページネーションは単なる UI 変更ではなく、外部契約、tenant isolation、DB index、削除や同時追加時の整合性に影響する。
offset pagination は深いページで遅く、同時更新で重複・欠落が起きやすいため、keyset cursor を標準にする。
Cursor に tenant や filter を含めないと情報漏えいまたは境界越えの探索に使われるため、opaque かつ検証可能な token として扱う。
