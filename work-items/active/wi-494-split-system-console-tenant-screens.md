---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-06
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/frontend/features/admin-console.md, requirement: REQ-FRONTEND-001 }
  - { path: docs/modules/frontend/features/admin-console.md, requirement: REQ-FRONTEND-002 }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ListTenants }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.CreateTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
---

# システムコンソールのテナント画面を一覧、詳細、作成、変更に分ける

## 動機

システムコンソールのテナント画面は、一覧、選択中テナントの参照、設定変更を一つの画面へ詰め込んでいる。
詳細カードに表示名、パスワードポリシー、上限、正規ロケーションの入力欄が常時現れるため、参照中なのか変更中なのかを画面構造から判断できない。
これは[UI 設計のナビゲーション](../../docs/design/application/user-interface.md#ナビゲーション)の「詳細を確認してから編集する」と、[編集と確定の単位](../../docs/design/application/user-interface.md#編集と確定の単位)に反する。

ほかにも、現在の画面は UI 設計の規則から外れている。

- 一覧行の無効化と再開は、確認なしで即時に実行される。[破壊的な操作](../../docs/design/application/user-interface.md#破壊的な操作)は確認ダイアログを求める。
- 正規ロケーションの切替は `window.confirm` で確認し、同じカードに表示名などの保存と別の保存ボタンを並べている。
- 一つの保存で、テナントの更新（PATCH）と上限の更新（PUT）を順に送る。後者だけが失敗すると、一部だけが保存された状態で失敗を表示する。
- テナントの作成にモーダルを使っている。

## 対象範囲

- システムコンソールのテナント UI を、次のルートへ分ける。テナントを指すパスには、管理 API と同じく realm を使う。
  - 一覧 `/system/tenants`
  - 作成 `/system/tenants/new`
  - 読み取り専用の詳細 `/system/tenants/$realm`
  - 表示名とパスワードポリシー上書きの変更 `/system/tenants/$realm/edit`
  - 上限の変更 `/system/tenants/$realm/quota`
  - 正規ロケーションの切替 `/system/tenants/$realm/endpoint-style`
- 一覧の行に、詳細、変更、無効化または再開の操作を表示する。無効化と再開は、行からも詳細からも確認ダイアログを経て実行する。
- 詳細画面に、テナント属性、状態、正規ロケーション、上限と使用量、作成日時と無効化日時を表示し、各変更画面への名前の付いた入口と、無効化または再開の操作を置く。詳細画面にはフォーム部品を置かない。
- 各変更画面は一つの保存で一つの API を呼び、成功したら詳細へ戻って再取得し、成功を一度だけ通知する。失敗したら変更画面に残り、入力を保持して失敗を表示する。
- 正規ロケーションの切替は、保存の前に、発行者、WebAuthn の RP ID、Cookie の適用範囲が変わることを確認ダイアログで示す。
- 新しい画面の文言を日英で用意する。
- これらの画面の振る舞いを、Frontend の規範シナリオとして追加する。

## 対象外

- MFA ステップアップと操作理由の入力。wi-468 が無効化、再開、正規ロケーション切替に適用し、この記録が作る確認ダイアログへ足す。
- テナントの削除。wi-34219 が扱い、この記録が作る詳細画面へ入口を置く。
- 対象テナントのユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの読み取り専用の確認。wi-33063 が扱う。
- テナント一覧のページングと、上限と使用量の一括取得。wi-469 が扱う。
- 管理 API の変更。既存の操作をそのまま使う。

## 設計

### 分割の経緯

2026-09-06 に起票したこの記録は、UI の再構成、テナントの削除、テナント横断のリソース読出しの三つを含んでいた。
三つは互いに独立して受け入れられる結果であり、削除はおよそ 100 のテーブルと外部ストアにまたがる。
2026-11-14 に利用者の判断で、削除を wi-34219、リソース読出しを wi-33063 に分け、この記録は UI の再構成だけを扱う。

同じ判断で、MFA ステップアップと操作理由を wi-468 に委ね、依存から外した。
wi-468 は無効化、再開、正規ロケーション切替を対象にこれらを所有しており、この記録で同じものを定めると所有が重なるためである。
起票時の `depends_on` のうち wi-470 と wi-471 は完了済みであり、wi-469 はこの記録の画面を前提にしないため、依存から外した。

### 起票時の判断から変えたもの

| 起票時 | 変更後 | 理由 |
| --- | --- | --- |
| 一覧の行から変更と無効化を取り除き、行は詳細への遷移だけにする | 行に詳細、変更、無効化または再開を表示し、無効化と再開は確認ダイアログを経る | UI 設計は行の操作をその場に表示すると定める。起票時の問題は、行に操作があることではなく、確認なしで実行されることである |
| 正規ロケーションの切替を変更画面に置き、通常の保存とは別に保存する | 専用の画面にし、詳細から名前の付いた入口で移る | UI 設計は、同じ編集画面へ独立した保存を並べないと定める |
| 変更画面に表示名、パスワードポリシー、上限、正規ロケーションを置く | 表示名とパスワードポリシーを `/edit`、上限を `/quota` に分ける | 二つは別の API であり、一つの保存にすると部分失敗を定める必要がある。画面を分ければ部分失敗が起きない |
| 詳細と変更のパスに `tenant_id` を使う | realm を使う | 管理 API のパスは realm でテナントを解決する（REQ-TENANCY-045、REQ-TENANCY-046）。外部の識別子は realm に統一する |
| 作成のモーダルを維持する | 作成を専用画面にする | UI 設計は、主要なリソースの作成と編集にモーダルを使わないと定める |

### 仕様の差分

Frontend の[管理コンソールのリソース操作](../../docs/modules/frontend/features/admin-console.md)に、システムコンソールのテナント画面の要件を一件追加する。
業務状態の許可と拒否は Tenancy の既存の要件が定めるので、変えない。

追加する要件（REQ-FRONTEND-003）：

- System 管理者がテナントの詳細を開いたとき、System は、現在の値を読み取り専用で表示し、表示名とパスワードポリシー、上限、正規ロケーションのそれぞれの変更画面への入口を置く。
- System 管理者が変更画面で保存したとき、System は、その変更画面が扱う API だけを呼び、成功したら詳細へ戻って再取得する。
- 保存に失敗した場合、System は、変更画面と入力を維持し、失敗を表示する。
- System 管理者が一覧の行または詳細から無効化または再開を選んだとき、System は、対象の realm、操作の結果、取り消せるかを示す確認を経てから要求する。確認を取り消した場合は要求しない。
- System 管理者が正規ロケーションを切り替えるとき、System は、発行者、WebAuthn の RP ID、Cookie の適用範囲が変わることを確認させてから要求する。
- System 管理者がテナントの作成を選んだとき、System は、専用の作成画面へ遷移し、作成に成功したら作成したテナントの詳細へ遷移する。

REQ-FRONTEND-001 と REQ-FRONTEND-002 の主語は「管理者」であり、管理コンソールを指す。
文書の概要に、システムコンソールの画面もこの文書が扱うことを書き足す。
二つの要件の文は変えない。

### 画面の構成

管理コンソールのグループ（`frontend/src/routes/admin/groups_/`）と同じく、`$realm.tsx` を親ルートにし、`$realm.index.tsx`、`$realm.edit.tsx`、`$realm.quota.tsx`、`$realm.endpoint-style.tsx` を子にする。
各ルートのローダーは `requireSystemAccount` を通し、`GetTenant` で対象を取得する。
無効化と再開の確認には、管理コンソールが破壊的な操作に使う既存の確認ダイアログの部品を使う。

## 計画

1. `spec-change` で REQ-FRONTEND-003 と例を追加し、`mise run check-spec` を通す。
2. 詳細画面に入力欄がないこと、行の無効化が確認なしで要求されないことを UI の単体テストで RED にする。
3. ルートとページを分け、各画面を実装する。
4. 既存の E2E（`tenant-quota.spec.ts`、`ui-scenario-smoke.spec.ts`）を新しい画面の経路へ追従させ、一覧から詳細、変更、保存、詳細での再取得までを確認する。

## タスク

- [ ] T001 [Spec] REQ-FRONTEND-003 と例を追加し、`affected_spec` に加える。
- [ ] T002 [Acceptance] 詳細に入力欄があること、行の無効化が確認なしで要求されることを UI の単体テストで観測し、RED を確認する。
- [ ] T003 [UI] 一覧、作成、詳細、表示名とパスワードポリシーの変更、上限の変更、正規ロケーションの切替をルートとページに分ける。
- [ ] T004 [UI] 無効化と再開の確認ダイアログを一覧の行と詳細に置き、正規ロケーションの切替の確認を部品のダイアログへ置き換える。
- [ ] T005 [Acceptance] 既存の E2E を新しい経路へ追従させ、変更後に詳細で再取得した値が表示されることを確認する。
- [ ] T006 [Verify] 仕様、UI の型と lint、単体テスト、E2E の検査を通す。

## 検証

- `mise run check-spec`：要件の追加。
- `mise run test-ui-unit` と UI の型検査と lint：画面とルートの変更。
- `mise run test-ui-e2e-file` で `tenant-quota.spec.ts` と `ui-scenario-smoke.spec.ts`：ルートの変更が正式な入口の配線に影響するため。
- `mise run check-work-items`、`mise run check-links`：記録の分割と移動。

受け入れ検証では、少なくとも次の観測結果を固定する。

- 詳細画面は現在値だけを表示し、変更用の入力と保存操作を持たない。
- 変更画面で保存すると、その画面が扱う API だけが呼ばれ、詳細へ戻り、再取得した値が表示される。保存に失敗すると変更画面と入力が残る。
- 無効化、再開、正規ロケーションの切替は、確認を経なければ要求されない。

## リスク

リスクは medium とする。
API と業務状態は変えないが、制御面の破壊的な操作の確認を担う画面であり、確認を経ずに要求する経路が残るとテナント全体の認証を止める。
確認を取り消したときに要求しないことを単体テストで固定する。

画面の配置は戻せるが、規範 ID を割り当てるため `reversibility` は irreversible とする。
