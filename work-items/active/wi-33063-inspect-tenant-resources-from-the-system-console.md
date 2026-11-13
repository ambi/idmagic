---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-14
priority: p2
depends_on:
  - wi-494-split-system-console-tenant-screens
  - wi-468-system-console-privileged-session-assurance
change_kind: feature
affected_spec:
  - { path: docs/modules/tenancy/features/lifecycle.md, requirement: REQ-TENANCY-014 }
  - { path: docs/modules/system/features/http-api.md, requirement: REQ-SYSTEM-024 }
  - { path: docs/modules/system/features/http-api.md, requirement: REQ-SYSTEM-025 }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListAdminUsers }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetAdminUser }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListGroups }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetGroup }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListAgents }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetAgent }
  - { path: spec/modules/application/main.tsp, symbol: IdMagic.Application.Operations.ListAdminApplications }
  - { path: spec/modules/application/main.tsp, symbol: IdMagic.Application.Operations.GetAdminApplication }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ListAdminOAuth2Clients }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.GetAdminOAuth2Client }
---

# システム管理者が対象テナントのリソースを読み取り専用で確認できる

## 動機

システム管理者が見られるのはテナント属性とクォータ使用量の集計だけであり、使用量の根拠となるユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントを対象テナントごとに確認できない。
障害調査や削除前確認で、集計値から実体へたどれない。

この記録は wi-494 から分割した。
wi-494 が作るテナント詳細画面から、この記録が追加する読み取り専用のリソース画面へ移る。

## 対象範囲

- システムコンソールのテナント詳細から、ユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの一覧と個別詳細を読み取り専用で確認できるようにする。
- テナント横断読出しには専用のシステム API を追加し、テナント管理 API に `tenant_id` 切替パラメーターを加えない。応答はページサイズ上限（REQ-SYSTEM-024）と、対象テナントに束縛したカーソル（REQ-SYSTEM-025）を持ち、秘密値、パスワードハッシュ、クライアントシークレット、秘密鍵素材を返さない。
- IdentityManagement、Application、OAuth2、System の規範シナリオ、TypeSpec、内部機構を更新する。

## 対象外

- システムコンソールから、対象テナントのリソースを作成、更新、削除すること。テナント横断の変更権限をリソース管理へ広げない。
- システム管理者が対象テナントの利用者としてログインすること、対象テナントの管理コンソールへ権限を借りて入ること、または利用者を偽装すること。
- セッション、同意、SSF ストリーム、監査イベント、ジョブ、署名鍵、データ鍵、エクスポート成果物に同じ汎用リソース詳細画面を追加すること。
- MFA ステップアップの仕組みそのもの。wi-468 が定める `privileged_read` をこの読出しにも適用する。

## 設計

### 対象のリソース

今回の「各リソース」は、テナントの永続的な主体と連携設定を代表し、既存のクォータにも現れるユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの5種類とする。
セッションや監査のような運用時系列データまで一つの汎用一覧へ押し込まず、それぞれの専用画面へ委ねる。

### 読出しの境界

各モジュールは既存のテナント管理 API と別に、制御面主体と明示的な対象テナントを要求する読み取り専用の System 操作を所有する。
経路は既存の `/api/admin/v1/system/...` の区分に従う。
既存の応答モデルと権限判断を持たない表示部品は再利用できるが、テナント管理ページへ `systemMode` や `targetTenantID` を渡して横断能力を切り替える形にはしない。

一覧は対象テナント、絞り込み条件、ページサイズに束縛したカーソルを使う。
別テナントまたは別リソース種別のカーソルは拒否し、一回の要求の永続化処理量をページサイズの定数倍に制限する。
詳細取得も対象テナントを明示的に照合し、別テナントの同じ識別子を返さない。

システム画面は調査用の読み取り専用表示とし、編集、削除、秘密表示、資格情報発行への導線を置かない。
テナント削除（wi-34219）が先に完了していれば、`Deleted` は常に未存在として返し、`Deleting` は削除進行中であることを明示して読出しを拒否する。

対象テナントの管理コンソールをそのまま開く案は採らない。
制御面 User は `default` テナントに所属しており、別テナントのセッションやロールを借りなければテナント内 API を正しく呼べない。
暗黙の偽装を導入せず、横断読出しであることが経路、TypeSpec 操作、シェルから分かる専用境界を使う。

### 仕様の所有

IdentityManagement、Application、OAuth2 は自身のリソースを制御面から読み出す操作と返してよい項目を所有する。
System はシステムコンソールだけが横断 UI の入口となる不変条件を所有する。
既存シナリオへ異なる主体を詰め込まず、SystemAdministrator を主体とする新しい規範シナリオと TypeSpec 操作を割り当て、割り当て後に `affected_spec` を更新する。

### 未解決の問い

- 着手時に、モジュール間の依存を増やさずに各モジュールの System 操作を組み立てられるかを、境界を選ぶ判断手順で確かめる。

## 計画

1. 5種類のリソースについて、主体、経路、応答、拒否を規範シナリオと TypeSpec へ先に定義する。
2. 制御面から対象テナントのリソースを読めないことを HTTP の境界で RED として固定する。
3. 各モジュールに、制御面主体限定、対象テナント固定、カーソル有界、秘密を含まない System 読出し API を実装する。
4. テナント詳細から辿る読み取り専用の一覧と詳細を日英で実装する。

## タスク

- [ ] T001 [Spec] IdentityManagement、Application、OAuth2、System に規範シナリオと TypeSpec 操作を追加し、`affected_spec` を割り当てた ID と記号へ更新する。
- [ ] T002 [Acceptance] 制御面から対象テナントのリソースを読めないことを HTTP の境界で観測し、RED を確認する。
- [ ] T003 [App] 5種類の System 一覧・詳細 API を、制御面主体、対象テナント、ページサイズへ固定して実装する。
- [ ] T004 [Acceptance] 別テナントの識別子とカーソル、権限不足の主体を拒否し、他テナントの情報を返さないことを確認する。
- [ ] T005 [UI] 5種類のリソース一覧・詳細を読み取り専用で実装し、既存の表示部品だけを権限判断なしで共有する。
- [ ] T006 [Docs] リリースノートに System API と画面を記載する。
- [ ] T007 [Verify] 仕様、生成物、API 互換性、セキュリティ制御、モジュール境界、UI の検査を通す。

## 検証

- 着手時に[最終検査の選択](../../docs/development/specification-first-workflow.md#最終検査の選択)に従って検査を選ぶ。少なくとも `check-spec`、`check-api-compat`、`check-security-controls`、`check-boundaries`、HTTP 統合テスト、`test-ui-unit` が対象になる。

受け入れ検証では、少なくとも次の観測結果を固定する。

- 5種類の一覧と詳細は選択したテナントの情報だけを返し、ページサイズを超えて読み出さず、別テナントへ束縛されたカーソルを拒否し、秘密値を含まない。
- システムコンソールのリソース画面には作成、更新、削除、資格情報発行の操作がなく、テナント管理 API を横断読出しへ流用しない。

## リスク

リスクは high とする。
制御面のリソース読出しは新しいテナント横断能力であり、対象照合を誤ると別テナントの個人情報を返す。
専用 System 操作と認可をサーバー側へ置き、画面の非表示だけを認可にしない。
応答モデルを既存管理 API から機械的に流用せず、秘密と変更用メタデータを除いた項目を TypeSpec で明示する。

一つの画面コンポーネントをモード切替で共有すると、テナント内画面に横断対象指定が漏れる。
共有するのは表、状態バッジ、読み取り専用の詳細表示など権限判断を持たない部品だけとし、ページ、ローダー、API クライアント、ルートガードは分離する。

公開 API 操作と規範 ID は撤回できないため、`reversibility` は irreversible とする。
