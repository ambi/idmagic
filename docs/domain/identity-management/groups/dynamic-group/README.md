# 動的グループ

## 概要

この文書は、`membership_type=dynamic` の Group の所属を、CEL の規則の評価で決める機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 規則の保存、有効化と無効化、プレビュー、全件の再評価、User の変更に伴う再評価 |
| 行為者 | 管理者、再評価のジョブ |
| コードの機能スライス | `backend/idmanagement/group` |
| 扱わないもの | Group そのものと手動の所属は[グループ](../group/README.md)が、評価に使う User の属性の定義は[ユーザー](../../principals/user/README.md#属性の定義)が扱う |

## モデル

`DynamicGroupRule` は `Group` の境界の内側にあり、一つの動的グループに一つだけある。

| 項目 | 内容 |
| --- | --- |
| `expression` | User の属性を参照し、真偽値を返す CEL の式 |
| `version` | 式の保存、有効化、無効化のたびに一つ進む版 |
| `enabled` | 規則が有効か |

動的な所属は、由来を規則とし、所属を作ったときの規則の版を記録する。
有効なのは、記録した版が現在の規則の版と一致する所属だけである。

## 状態遷移

### DynamicMembershipEvaluationLifecycle

全件の再評価は、`queued` から `running` を経て、`succeeded` または `failed` で終わる。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 全件再評価を受理した。実行を待つ |
| running | — | 全件再評価を実行している |
| succeeded | terminal | 全件再評価が完了した |
| failed | terminal | 全件再評価が失敗した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | — | running |  |
| running | DynamicMembershipEvaluated | — | succeeded |  |
| running | DynamicMembershipEvaluationFailed | — | failed |  |

## 操作

### 規則の保存と有効化

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 式、または有効化の指示 |
| 成功時の作用 | 規則を保存して版を進め、`DynamicGroupRuleUpdated` または `DynamicGroupRuleEnabled` を発行する。有効な規則の保存と有効化は、全件の再評価を予約する |
| 拒否 | 式の制約の違反（422 `invalid_dynamic_group_rule`）、手動の Group への保存（409 `dynamic_membership_managed_by_rule`）、規則のない Group の有効化。動的グループへの手動のメンバー操作。どの拒否も規則と所属を変えない |
| 冪等性 | すでに有効な規則の有効化は、版を進めずイベントを発行しない |

#### REQ-IDMANAGEMENT-020 管理者は CEL の規則で動的グループの所属を管理できる

- 管理者は、動的グループに CEL の規則を保存して有効化できる。
- 有効化した規則の全件の再評価の後、規則に一致する有効な User だけが、規則を由来として所属する。
- 実効ロールと Application の割り当ては、その所属を参照する。
- **担保手段**：`usecases.UpdateDynamicGroupRule`、`usecases.SetDynamicGroupRuleEnabled`、`usecases.ReconcileDynamicGroup`
- **例**：EX-IDMANAGEMENT-020-01

#### REQ-IDMANAGEMENT-066 動的グループの規則の版と有効化

- 初めて保存した規則は無効であり、版は 1 である。
- 式の保存のたびに版を一つ進め、有効か無効かは保存前の状態を引き継ぐ。
- 有効化と無効化は、それぞれ版を一つ進める。すでにその状態にある規則の有効化と無効化は、版を進めずイベントを発行しない。
- 有効な規則の式の保存と、規則の有効化は、全件の再評価を予約する。
- `membership_type=manual` の Group への規則の保存は、409 と `dynamic_membership_managed_by_rule` で拒否する。
- 規則を持たない Group の有効化と無効化は、422 と `invalid_dynamic_group_rule` で拒否する。
- **担保手段**：`usecases.UpdateDynamicGroupRule`、`usecases.SetDynamicGroupRuleEnabled`

#### REQ-IDMANAGEMENT-065 動的グループの規則の式の制約

- 式は 1 バイト以上 4,096 バイト以下とする。
- 式の開き括弧は 20 個以下、空白で区切った語は 200 個以下とする。
- 呼び出せる関数は `startsWith`、`endsWith`、`contains`、`matches`、`lowerAscii`、`size`、`exists`、`all`、`timestamp` だけとする。
- `matches` の正規表現は文字列の定数で、256 文字以下とする。
- 式は `user.<属性>` で User の属性を一つ以上、100 個以下参照する。参照できるのは `id`、`preferred_username`、`name`、`given_name`、`family_name`、`email`、`email_verified` と、組み込みとテナント定義の属性だけである。`roles` は参照できない。
- 式の結果は真偽値とする。
- 評価の計算量の上限は 10,000 とする。
- 制約に違反する規則の保存とプレビューは、422 と `invalid_dynamic_group_rule` で拒否し、規則を変えない。
- **担保手段**：`domain.CompileDynamicGroupRule`

#### REQ-IDMANAGEMENT-022 不正な CEL の規則と動的グループの手動操作は拒否される

- 定義していない属性、または許可していない関数を参照する式の保存は拒否する。
- 動的グループへの `AddGroupMember` と `RemoveGroupMember` の手動の呼び出しは拒否し、メンバーシップを変えない。
- **判断**：動的グループの所属は規則の評価だけが決める。手動の操作を許すと、所属がどの経路で付いたのかを区別できなくなる。
- **担保手段**：`domain.CompileDynamicGroupRule`、`usecases.AddMember`、`usecases.RemoveMember`
- **例**：EX-IDMANAGEMENT-022-01

### 規則の無効化

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 成功時の作用 | 規則を無効にして版を進め、`DynamicGroupRuleDisabled` を発行し、再評価を待たずに動的な所属をすべて外す |
| 冪等性 | すでに無効な規則の無効化は、版を進めずイベントを発行しない |

#### REQ-IDMANAGEMENT-068 規則の無効化は、動的グループの所属をすべて直ちに外す

- 規則の無効化は、再評価を予約せず、その場で動的グループのメンバーシップをすべて外す。
- **担保手段**：`usecases.SetDynamicGroupRuleEnabled`、`usecases.ReconcileDynamicGroup`

### 全件の再評価

| 項目 | 内容 |
| --- | --- |
| 行為者 | 再評価のジョブ。User の変更に伴う再評価は、User を変えた操作 |
| 入力 | 予約した時点の規則の版 |
| 成功時の作用 | 規則に一致する User を所属させ、一致しない User を外し、件数を載せた `DynamicMembershipEvaluated` を一つ発行する |
| 冪等性 | 規則がない、無効である、または版が異なるジョブは、所属を変えずに成功する |

#### REQ-IDMANAGEMENT-067 動的グループの規則に一致するのは `Active` の User だけである

- `Active` でない User は、式の値によらず規則に一致しない。
- 評価に失敗した User は一致しないものとして扱い、再評価の結果の誤りの件数に数える。
- **担保手段**：`CompiledDynamicGroupRule.Evaluate`、`usecases.ReconcileDynamicGroup`

#### REQ-IDMANAGEMENT-023 評価できない規則は権限を付与しない

- 規則の版が進むと、古い版の所属は、再評価を待たずに実効ロールに数えない。
- 再評価で評価に失敗した User は、新しい版の所属を得ない。
- **担保手段**：`usecases.ReconcileDynamicGroup`、`usecases.SyncDynamicGroupsForUser`
- **例**：EX-IDMANAGEMENT-023-01

#### REQ-IDMANAGEMENT-069 古い版の再評価のジョブは、所属を変えずに成功する

- 再評価のジョブは、予約した時点の規則の版を持つ。
- 実行時に規則がない、無効である、または版が異なるジョブは、メンバーシップを変えずに成功として終わる。
- **担保手段**：`usecases.DynamicGroupReconcileHandler`

#### REQ-IDMANAGEMENT-070 動的な所属の変化は、メンバーごとのイベントを発行しない

- 規則の再評価と、User の変更に伴う再評価で所属が増減しても、`GroupMemberAdded` と `GroupMemberRemoved` を発行しない。
- 全件の再評価は、追加、除外、変化なし、誤りの件数を載せた `DynamicMembershipEvaluated` を一つ発行する。
- **担保手段**：`usecases.ReconcileDynamicGroup`、`usecases.SyncDynamicGroupsForUser`
- **要判断**：動的グループのロールは所属した User の実効ロールに加わるが、誰がいつ加わったかは監査の記録に残らない。メンバーごとのイベントを発行するかを決める。

### プレビュー

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 未保存の式と、100 件以下の User |
| 成功時の作用 | User ごとに一致の有無と、追加、除外、変化なしの判定を返す。所属を変えない |
| 拒否 | 101 件以上の User、存在しない User と別のテナントの User を含む指定、式の制約の違反 |

#### REQ-IDMANAGEMENT-021 CEL の規則は保存前に選んだユーザーでプレビューできる

- 管理者は、選んだ 100 件以下の User で、保存していない式を評価できる。
- プレビューの応答は、User ごとの一致の有無と、追加、除外、変化なしの判定を返し、属性の値そのものは返さない。
- **担保手段**：`usecases.PreviewDynamicGroupRule`
- **例**：EX-IDMANAGEMENT-021-01

#### REQ-IDMANAGEMENT-071 規則のプレビューは 100 件以下の、同じテナントの User だけを受け付ける

- 101 件以上の User を指定したプレビューは、422 と `invalid_dynamic_group_rule` で拒否する。
- 存在しない User または別のテナントの User を一つでも含むプレビューは、結果を返さず 404 と `user_not_found` で拒否する。
- **担保手段**：`usecases.PreviewDynamicGroupRule`

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 権限の付与 | 評価に失敗した User と、`Active` でない User は所属させない。判定できないときは権限を付与しない側へ倒す |
| 参照できる属性 | 式は `roles` を参照できない |
| 情報の開示 | プレビューは属性の値を返さない |
| 計算量 | 式の長さ、括弧の数、参照の数、評価の計算量に上限を置き、規則による評価の負荷を限る |
