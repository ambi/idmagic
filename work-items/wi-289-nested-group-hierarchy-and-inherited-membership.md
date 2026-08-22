---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
priority: p2
depends_on: []
change_kind: feature
initial_context:
  source:
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/group/db_postgres
    - backend/application/usecases
  tests:
    - backend/idmanagement/group/domain/groups_test.go
    - backend/idmanagement/group/usecases/admin_groups_test.go
  stop_before_reading:
    - backend/oauth2
    - backend/saml
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.Group }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.CreateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.ListUserGroups }
---

# グループ階層 (入れ子グループ) と継承メンバーシップを導入する

## Motivation

現在の Group はフラットである。`backend/idmanagement/group/domain/groups.go` に
「階層・deny ルール・属性自動所属は持たない (effective_roles は union のみ)」と明記されており、
`spec/contexts/identity-management/decisions.md` の Group 集約もフラットな union を前提としている。

実際の組織はフラットではない。部門 → 課 → チームのような階層を IdP 側で表現できないと:

1. **運用コストが爆発する**。「全社員」「営業本部」「営業一課」の各グループに、
   同じユーザーを個別に登録し続けることになる。人事異動のたびに N 箇所の付け替えが必要。
2. **アプリ割当が階層を反映できない**。「営業本部配下の全員に CRM を割当」を表現できず、
   末端グループごとに割当を複製する。割当漏れが権限事故になる。
3. **外部ディレクトリの構造が失われる**。LDAP/AD ([[wi-95-ldap-ad-user-federation]]) と
   SCIM ([[wi-246-scim-multivalued-core-attributes-and-nested-group-members]]) は入れ子
   グループを持つため、取り込み時に階層を平坦化するしかない。これは元データの意味を落とす
   不可逆な変換であり、上流に書き戻せない。

競合比較:

- **Keycloak**: subgroup を第一級で持ち、ロールは親から子へ継承される。
- **Entra ID**: nested group をサポート (ただしアプリ割当への継承には既知の制約がある)。
- **Okta**: グループ階層を持たない (Okta は代わりに group rule で解く) — つまり
  「階層が無い」設計もあり得るが、Okta は強力な rule エンジンと push group で補っている。
  IdMagic は動的グループ (`spec/contexts/identity-management/decisions.md` の CEL 動的グループ規則) を持つが、
  LDAP/AD 取り込みの構造保持には rule では代替できない。

本 WI は Group に単一親の階層を導入し、「継承メンバーシップ」と「継承ロール」を
明示的に定義された導出値として提供する。

## Scope

- **decision**:
  - `spec/contexts/identity-management/decisions.md` へ記録する決定 (グループ階層と継承の意味): 単一親 (森構造) に限定する理由、循環禁止と最大深さ、
    継承の方向 (子グループのメンバーは親グループの継承メンバーである、の向きを 1 つに固定する)、
    `effective_roles` の再定義 (直接ロール ∪ 所属グループとその祖先のロール)、
    dynamic group (`spec/contexts/identity-management/decisions.md` の CEL 動的グループ規則) と階層の併用可否、
    ApplicationAssignment の解決に継承メンバーシップを含めるか
    (`spec/contexts/application/decisions.md` の fail-closed 割当と整合)、
    親削除時の子の扱い (拒否 / 昇格 / カスケード) を記録する。
- **specification**:
  - `IdManagement.models.Group` に `parent_id` (optional) と `depth` を追加する。
  - `GroupHierarchyError` (循環 / 深さ超過 / dynamic 親禁止) を追加する。
  - `ListGroups` に階層取得 (ツリー / 子のみ) のパラメータを追加し、
    `ListUserGroups` の応答に「直接所属」と「継承所属」を区別して返す。
  - `MoveGroup` (親の変更) interface を追加する。
  - `states` に GroupParentChanged event を追加する。
  - `EffectiveRoles` の published language の定義を「直接ロール ∪ 所属グループとその祖先の
    ロール」に更新する (Authentication / OAuth2 / ClaimMapping が参照する published 値なので
    影響範囲を specification で明示する)。
  - `objectives` に「深さ N・グループ数 M における effective roles 解決レイテンシ」目標を追加する。
  - `scenarios`: 子グループ所属者が親グループのロールを得る / 循環する親設定が拒否される /
    最大深さ超過が拒否される / 親グループへのアプリ割当が子の所属者に効く /
    親削除が子を持つ間は拒否される / 動的グループを親にできない。
- **go**:
  - domain に階層不変条件 (循環禁止・深さ上限・単一親・dynamic 親禁止) を実装する。
  - `effective_roles` の解決を祖先方向に拡張する。現在の union 実装
    (`groups.go` の `effective_roles(user) = user.roles ∪ ⋃ g.roles`) を
    「所属グループの祖先閉包を含む union」に置き換える。
  - 継承メンバーシップの解決は **closure table** (`group_closure(ancestor_id, descendant_id, depth)`)
    で持つ。再帰 CTE を毎回走らせるとホットパス (ログイン時の claim 解決) に効くため、
    書き込み時に閉包を更新する。
  - ApplicationAssignment の解決に継承メンバーシップを含める (`spec/contexts/identity-management/decisions.md` の決定に従う)。
- **persistence**:
  - `groups.parent_id` (自己参照 FK) と `group_closure` テーブルを
    `infra/schema/postgres.sql` に追加し、sqlc クエリを再生成する。
  - 親変更時の閉包再構築をトランザクション内で行う。
- **http**:
  - グループ作成 / 更新で親を指定できるようにし、ツリー取得と `MoveGroup` を追加する。
- **ui**:
  - グループ一覧をツリー表示にし、親の選択・移動、メンバー一覧での「直接 / 継承」の区別、
    継承ロールの由来 (どの祖先から来たか) を表示する。
- **documentation**:
  - README に階層の意味 (継承の向き・深さ上限・アプリ割当への影響) を追記する。

## Out of Scope

- 複数親 (DAG) 構造。単一親の森に限定する。
- deny / 排他ルール (「このグループのメンバーは別グループに入れない」)。
  → [[wi-154-entitlement-catalog-and-separation-of-duties]] の SoD が扱う。
- グループ階層に基づく委任管理のスコープ。→ [[wi-94-delegated-administration]]
- SCIM の nested group member 表現。→
  [[wi-246-scim-multivalued-core-attributes-and-nested-group-members]] (本 WI が土台を提供する)
- LDAP/AD からの階層取り込み。→ [[wi-95-ldap-ad-user-federation]]
- 動的グループルールで祖先を参照する式。

## Plan

- **継承の向きを 1 つに固定するのが設計の核**。「子のメンバーは親のメンバーでもある」
  (= 親が上位集合) を採る。これは組織階層の直感 (営業一課の所属者は営業本部の所属者) と
  一致し、Keycloak の継承方向とも一致する。逆向きを許すと effective roles の意味が
  文脈依存になるため、`spec/contexts/identity-management/decisions.md` で一方向に固定する。
- **closure table を選ぶ**。effective roles はログイン・トークン発行・claim 解決という
  最もホットな経路で解決される。ここで再帰 CTE を走らせるとレイテンシ目標
  (`Authentication.LoginLatency` / `OAuth2.TokenLatency`) を壊す。書き込み時 (グループ作成・
  親変更・削除) に閉包を更新し、読み取りを単純 JOIN にする。書き込み頻度は読み取りに比べて
  桁違いに低いので、この非対称性を利用する。
- **深さ上限を仕様にする**。無制限にすると閉包テーブルが爆発し、UI も破綻する。
  既定 8 段程度を上限とし、specification の constraint として固定する。
- **既存データの移行は無変更で済む**。`parent_id` を optional にすれば既存グループは
  すべて根グループになり、`effective_roles` の結果も変わらない。これを移行テストで固定する。
- **ApplicationAssignment への継承適用は権限拡大なので慎重に扱う**。割当は fail-closed で
  ポータル可視性とフェデレーション可否を制御しているため、継承を有効にすると
  「親に割当てたら子の全員がアプリを使える」という**既存テナントの実効権限が広がる**変更に
  なりうる。既存テナントには階層が存在しない (全グループが根) ため実害は無いが、
  `spec/contexts/identity-management/decisions.md` にこの意味を明記し、scenario で固定する。
- **dynamic group は親にできない**とする。動的グループのメンバーは式で決まるため、
  それを親にすると継承の意味が二重評価になる。子にすることは許す (dynamic な子グループが
  静的な親に属する) 方針を `spec/contexts/identity-management/decisions.md` で決める。
- 未決定: 親削除時の挙動は「子を持つ間は削除拒否」を第一候補とする (最も驚きが少ない)。

## Tasks

- [ ] T001 [Spec] `Group` に parent_id / depth、GroupHierarchyError、MoveGroup、
      GroupParentChanged、ListGroups/ListUserGroups の拡張、EffectiveRoles の定義更新、
      objective、scenario 6 件を追加し `mise run check-spec` を通す。
- [ ] T002 [Spec] グループ階層と継承の意味を `spec/contexts/identity-management/decisions.md` に記録し、
      同じ文書のフラットな union を前提とした Group 集約の項目を書き換える。
- [ ] T003 [Domain] 階層不変条件 (単一親 / 循環禁止 / 深さ上限 / dynamic 親禁止) と
      祖先閉包を含む effective_roles を実装する。RED: 循環設定・深さ超過・dynamic 親が
      落ちるテストと、子所属者が親ロールを得るテストを先に書く
      (scenario `IdManagement.nested_group_role_inheritance`) → GREEN。
- [ ] T004 [Persistence] `groups.parent_id` と `group_closure` を `infra/schema/postgres.sql` に
      追加し、`mise run sqlc-generate` でクエリを再生成する。RED: 既存グループが全て根として
      読めるマイグレーション互換テスト → GREEN。
- [ ] T005 [Closure] 作成 / 親変更 / 削除時の閉包更新をトランザクション内で実装する。
      RED: 深い階層の親を移動しても閉包が正しく再構築されるテスト、並行更新で閉包が
      壊れないテスト (`mise run test-go-race`) → GREEN。
- [ ] T006 [Usecase] `MoveGroup`、ツリー取得、`ListUserGroups` の直接 / 継承区別、
      親削除の拒否を実装する。RED → GREEN。
- [ ] T007 [Assignment] ApplicationAssignment の解決に継承メンバーシップを含める。
      RED: 親グループへの割当が子の所属者に効くテスト、無関係な兄弟グループには効かない
      テスト → GREEN。
- [ ] T008 [HTTP] 親指定・ツリー取得・移動の API を追加する。RED: 循環要求が 400 になる
      handler テスト → GREEN。
- [ ] T009 [UI] グループ一覧のツリー表示、親選択・移動、メンバーの直接 / 継承区別、
      継承ロールの由来表示を追加する。RED: presentation logic の unit test → GREEN。
- [ ] T010 [Perf] 深さ上限・グループ数の想定上限で effective roles 解決の計測を行い、
      objective を満たすことを確認する (満たさない場合は closure のインデックスを見直す)。
- [ ] T011 [Docs] README に階層の意味と運用上の注意を追記する。
- [ ] T012 [Verify] 下記 Verification を緑にする。`mise run spec-render` を実行する。

## Verification

- `mise run check` / `mise run check-spec` / `mise run check-work-items` / `mise run check-ids`
- `mise run test-go` / `mise run test-go-race` / `mise run verify-go`
- `mise run verify-ui` / `mise run test-ui-unit`
- 手動: `mise run dev` で 3 段の階層を作り、(1) 末端グループのユーザーが最上位グループの
  ロールを得ること、(2) 最上位グループへのアプリ割当がポータルに反映されること、
  (3) 循環する親設定が UI でエラーになること、(4) 子を持つグループが削除できないこと、
  を確認する。

## Risk Notes

`effective_roles` は Authentication / OAuth2 / ClaimMapping が参照する **published language** で
あり、その定義変更は認可判断の意味を変える。既存テナントは全グループが根になるため実効値は
変わらないが、これを移行テストで明示的に固定する。
ApplicationAssignment に継承を効かせるのは**権限が広がる方向**の変更である。fail-closed の
割当ゲートを緩める形になるため、`spec/contexts/identity-management/decisions.md` に意味を明記し、「親に割当てると子の全員に効く」ことを
UI でも明示する。
closure table は書き込み時の整合性が壊れると継承が静かに間違う (権限事故になる)。
親変更をトランザクション内の再構築に限定し、race テストで固定する。
深さ無制限は閉包爆発を招くため、specification の constraint として上限を持ち、超過を fail-closed で拒否する。
