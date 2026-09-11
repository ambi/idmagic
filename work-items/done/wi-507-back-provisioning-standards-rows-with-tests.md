---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと台帳の 1 行の削除だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/contexts/provisioning/standards.md
  typespec:
    - IdMagic.Contract.GroupPushConfig
  source:
    - backend/provisioning/client_scim/client.go
    - backend/provisioning/usecases/deliver.go
    - backend/provisioning/source_idmanagement/group_attribute_source.go
    - backend/provisioning/domain/connection.go
    - backend/provisioning/usecases/admin.go
    - tools/check/src/normative-coverage.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/provisioning/client_scim/conformance_test.go
    - backend/provisioning/client_scim/client_test.go
    - backend/provisioning/e2e_capture_delivery_test.go
  stop_before_reading:
    - frontend
    - backend/provisioning/db_postgres
    - backend/provisioning/handlers_http
---

# Provisioning が宣言する標準の最後の 1 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/provisioning/standards.md` の 1 行を引き取る。

**この文書は 13 行のうち 12 行が既に名指しを持つ。** [[wi-238-scim-inbound-list-query-conformance]] の適合作業が id を名指すテストを書いたからであり、消化が可能であることの実例になっている。残る 1 行は `RFC7643-OUT-GROUP-RESOURCES` だけで、採用は `partial` である。**12 行を書いた作業がこの 1 行だけを残したという事実そのものが、この行の観測が他の 12 行と違う形を要求している徴候である。** 単に忘れられただけなのか、`partial` の境界が書きにくかったのかを、まずそこから読む。

## Scope

- `RFC7643-OUT-GROUP-RESOURCES`（`partial`）を消化する。
- その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// RFC7643-OUT-GROUP-RESOURCES: <この行の何を固定しているか>` の注記を足す。
- `partial` の観測は、採用した範囲の振る舞いと、採用していない範囲がどう扱われるか（拒否するのか、単に提供しないのか）の両方を持つ。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。
- 12 行が既に消化されているので、この行が消えた時点で `docs/contexts/provisioning/standards.md` は台帳から完全に外れる。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。Sourcing が持つ同じ RFC の 5 行は [[wi-505-back-sourcing-standards-rows-with-tests]] が持つ。
- 既に名指しを持つ 12 行の観測の見直し。名指しが実在の検証に付いているかを疑う作業は本項目の対象ではない。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。`partial` の境界が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`partial` の観測は 2 つ要る。片方だけでは、全部を採用している実装とも、何も採用していない実装とも区別できない。

`RFC7643-OUT-GROUP-RESOURCES` は Group リソースの送出を扱う行である。したがって観測すべき境界は、送出する範囲と送出しない範囲の間にある。行の `Statement` がその境界をどこに引いているかを読み、境界の内側で実際に送出されること、および外側が拒否されるのか単に含まれないのかを、その書きぶりに合わせて観測する。

**先に、なぜこの 1 行だけが残ったのかを読む。** 既に名指しを持つ 12 行のテストが `RFC7643-OUT-GROUP-RESOURCES` の範囲に触れていながら名指していないだけなら、必要なのは境界の観測を足すことである。触れてすらいないなら、Group の送出そのものにテストが無いということなので、そこから書く。この 2 つは作業量が違うので、着手前に区別する。

## Plan

1. 既に名指しを持つ 12 行のテストを読み、`RFC7643-OUT-GROUP-RESOURCES` の範囲に触れているものがあるかを確かめる。
2. 行の `Statement` が引いている採用の境界を読み、境界の内側と外側それぞれの観測を決める。
3. 境界の内側を消化する。
4. 境界の外側を、拒否なのか非提供なのかに合わせて消化する。
5. `standards-coverage-debt.json` から `RFC7643-OUT-GROUP-RESOURCES` を外し、`docs/contexts/provisioning/standards.md` が台帳から完全に外れたことを確かめる。

## Tasks

- [x] T001 [Inventory] 既存の 12 行のテストが `RFC7643-OUT-GROUP-RESOURCES` の範囲に触れているかを確かめ、必要な作業量を区別する。
  **12 行のテストは 1 つも Group の送出に触れていない。** 名指しを持つ 12 行はすべて User の送出経路に
  あり、`schemas` を観測する `TestClient_SendsSchemasOnResourceRepresentations` も
  `fullLifecycleRequests` という User 専用の経路を歩く。Group に触れるテストは別に 7 件あるが
  (`TestClient_CreateGroup_ReturnsRemoteID`、`TestClient_PatchGroupMembers_SendsAddOperation`、
  E2E の 5 件)、行の節を区別する観測は持たない。作成は `POST /Groups` と応答の id しか見ず本文を読まない。
  したがって注記を足すだけでは済まず、本文の観測と境界の外側の観測を書く作業である。
- [x] T002 [Acceptance] 採用の境界の内側を消化する。
  `TestClient_GroupResourceBody_IsSchemasPlusMappedAttributes`
  (`backend/provisioning/client_scim/conformance_test.go`)。`groupPushRequests` が Group の送出経路を
  1 度ずつ通し、作成 (POST) と置換 (PUT) の本文が Group の URN 1 要素の `schemas` と対応付けが解決した
  属性だけを持ち、`members` を持たないこと、メンバーシップが別の `PatchOp` の増分 `add` として届くことを
  観測する。recipe: `mise run test-go-test -- ./backend/provisioning/client_scim TestClient_GroupResourceBody_IsSchemasPlusMappedAttributes`。
- [x] T003 [Acceptance] 採用の境界の外側を、拒否か非提供かに合わせて消化する。
  `TestE2E_GroupMembership_NeverSendsRemovalWhenAMemberLeaves`
  (`backend/provisioning/e2e_capture_delivery_test.go`)。行の書きぶりは非提供であって拒否ではない
  ——「除去は送らない」なので、メンバーが抜けた配信は失敗せず成功し、`remove` の Operation が 1 件も
  現れず、残ったメンバーだけの `add` が届く。recipe:
  `mise run test-go-test -- ./backend/provisioning TestE2E_GroupMembership_NeverSendsRemovalWhenAMemberLeaves`。
- [x] T004 [Ledger] `RFC7643-OUT-GROUP-RESOURCES` を `standards-coverage-debt.json` から外す。
- [x] T005 [Verify] `mise run verify`。
- [x] T006 [Defect] 取得元の選択が実装されていないことを
  [[wi-536-group-display-name-source-selects-nothing]] へ切り出す。

## Verification

- `RFC7643-OUT-GROUP-RESOURCES` が `tools/check/standards-coverage-debt.json` から消えている。
- `docs/contexts/provisioning/standards.md` の行が 1 つも台帳に残っていない。
- 注記を足したテストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **1 行だから注記だけで済ませる。** 件数が少ないことは、既存テストが行を区別している根拠にはならない。むしろ 12 行を書いた作業がこの 1 行を残したという事実が、区別が付いていないことの徴候である。T001 を飛ばさない。
- **`partial` を片側だけで消化する。** 境界の両側を観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無く、
  **製品コードも 1 行も変わっていない**。差分は `docs/contexts/provisioning/standards.md` の
  `RFC7643-OUT-GROUP-RESOURCES` に対する被覆の状態である。この 1 行が
  `tools/check/standards-coverage-debt.json` から消え、**台帳の `untested` は空になった**。台帳に残る
  文書はもう無く、名指しを持つ id は 288 件から 289 件へ増えた。
  **12 行のテストはこの行に触れてすらいなかった。** 名指しを持つ 12 行はすべて User の送出経路にあり、
  `schemas` を観測する `TestClient_SendsSchemasOnResourceRepresentations` すら `fullLifecycleRequests`
  という User 専用の経路を歩く。Group に触れるテストは別に 7 件あったが、作成は `POST /Groups` と応答の
  id しか見ず本文を読まない。**この 1 行が残ったのは忘れられたからではなく、Group の送出の本文を読む
  観測がどこにも無かったからである。**
  **`partial` の境界の両側を、行の書きぶりどおりに観測した。** 内側は
  `TestClient_GroupResourceBody_IsSchemasPlusMappedAttributes` が、Group の URN 1 要素の `schemas`、
  対応付けが解決した属性だけで組み立てた本文、その本文に `members` が無いこと、メンバーシップが別の
  `PatchOp` の増分 `add` として届くことを固定する。外側は
  `TestE2E_GroupMembership_NeverSendsRemovalWhenAMemberLeaves` が持つ。**行が言う外側は拒否ではなく
  非提供である**ため、メンバーが抜けた配信は成功したままで、`remove` が 1 件も現れず、残ったメンバーだけの
  `add` が届く、という 3 つを同時に観測する必要があった。成功を主張しない観測では、除去を失敗として
  扱う実装（Group から 1 人外しただけで配信が dead_letter に落ちる）を落とせない。
  既に行の節を観測していた 2 件（`TestE2E_GroupMembership_PatchesOnlyProvisionedMembers`、
  `TestE2E_GroupChange_ReachesRealDownstream`）には注記を足し、名指しを実在の観測へ繋いだ。
  **欠陥を 1 件見つけ、切り出した。** 行が言う「`displayName` の取得元は `display_name_source` が選ぶ」を
  読む production の経路が無い。`GroupAttributeSource` は接続を参照せず、`display_name` を常に
  `Group.Name` から解決する。管理画面は自由入力を受け付け、設定は保存され、配信は成功し、観測できる差だけが
  無い。この節だけはテストが書けないので
  [[wi-536-group-display-name-source-selects-nothing]] へ切り出した。行の他の節はすべて宣言どおりに
  実装されているので、`Adoption` の変更は要らない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`RFC7643-OUT-GROUP-RESOURCES` を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/provisioning/standards.md:11: RFC7643-OUT-GROUP-RESOURCES
    is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.` 報告はこの 1 行だけだった。
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (157 standard(s), 311 rule(s), 747 example(s), 289 id(s) named by a test)`
    を返す（288 → 289）。
- **Unit RED Evidence**:
  - **Test**: 行に対応付けた 4 件のテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 10 件の故障すべてを、対応するテストが検出した。生存はゼロ。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
- **Change-Resistance Results**:
  行が言っていることを production 側で崩し、対応するテストが落ちることを 10 件観測した。注入のたびに
  `git diff --stat` で着弾を確かめ、観測後に元へ戻している。1 件（メンバーシップをリソース本文に載せる）は
  最初の書き方が宣言前の `doc` を参照してビルドできず、観測にならないので `BuildResource` の後へ
  組み直した。

  | 注入した故障 | 落ちたテストと観測 |
  |---|---|
  | Group の URN を User の URN に取り違える | `TestClient_GroupResourceBody_IsSchemasPlusMappedAttributes`: `POST /Groups の schemas = [...core:2.0:User], want [...core:2.0:Group]` |
  | 作成の本文から `schemas` を落とす | 同テスト: `POST /Groups の schemas が配列でない: map[displayName:engineering]` |
  | 置換 (PUT) の本文から `schemas` を落とす | 同テスト: `PUT /Groups/remote-1 の schemas が配列でない` |
  | メンバーシップをリソース本文に載せる | 同テスト: `POST /Groups の本文に members がある` |
  | 対応付けを経ずに解決済みの属性をそのまま載せる | 同テスト: `members` と対応付けの無い `description`、さらに `display_name` がそのまま本文に現れる |
  | 増分 `add` ではなく `remove` を送る | `TestE2E_GroupMembership_NeverSendsRemovalWhenAMemberLeaves`: `メンバーが抜けた後の PATCH op = "remove", want add` |
  | 増分ではなく全置換 (`replace`) を送る | 同テスト: `PATCH op = "replace", want add` |
  | 除去できないことを拒否として扱う（配信を失敗させる） | 同テスト: `ExecuteDelivery() error = provisioning: member removal is not supported`。**非提供と拒否を分けている観測がこれで落ちる** |
  | 抜けたメンバーが一覧に残り続ける (`RemoveMember` が消さない) | 同テスト: `メンバーが抜けた後の PATCH members = [remote-user-1 remote-user-2], want [remote-user-1]` |
  | 相関を持たないメンバーの識別子をこちら側で作る | `TestE2E_GroupMembership_PatchesOnlyProvisionedMembers`: `PATCH members = [remote-user-1 user-never-provisioned]`。除去の検査は通るので、2 つが別の節を観測していることが読める |
  | `displayName` の既定の取得元を Group の名前から外す | `TestE2E_GroupChange_ReachesRealDownstream`: `POST /Groups body = map[displayName:group-eng ...], want displayName=engineering` |
- **Verification Results**:
  - `mise run verify` - passed
