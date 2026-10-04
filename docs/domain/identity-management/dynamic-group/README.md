# 動的グループ

## 概要

この文書は、`membership_type=dynamic` の Group の所属を、CEL の規則の評価で決める機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 規則の保存、有効化と無効化、プレビュー、全件の再評価、User の変更に伴う再評価 |
| 行為者 | 管理者、再評価のジョブ |
| コードの機能スライス | `backend/idmanagement/group` |
| 扱わないもの | Group そのものと手動の所属は[グループ](../group/README.md)が、評価に使う User の属性の定義は[ユーザー](../user/README.md#属性の定義)が、ジョブの実行と再試行は `Jobs` が扱う |

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
失敗した再評価は、`Jobs` の試行の上限まで `queued` に戻してやり直す。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 全件再評価を受理した。実行を待つ |
| running | — | 全件再評価を実行している |
| succeeded | terminal | 全件再評価が完了した |
| failed | terminal | 全件再評価が失敗した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | — | running |  |
| running | DynamicMembershipEvaluated | — | succeeded | DynamicMembershipEvaluated |
| running | JobRetried | job.attempts < job.max_attempts | queued |  |
| running | DynamicMembershipEvaluationFailed | job.attempts >= job.max_attempts | failed |  |

| State | 実行の開始 | 全件の再評価の終了 |
|---|---|---|
| queued | → running | 何もしない |
| running | 何もしない | → succeeded（評価を終えた、または規則がない、無効である、版が異なる）<br>→ queued（失敗し、試行が上限に達していない）<br>→ failed（失敗し、試行が上限に達した） |
| succeeded | 何もしない | 何もしない |
| failed | 何もしない | 何もしない |

## 操作

### 規則の保存と有効化

#### REQ-IDMANAGEMENT-066 規則の保存と有効化は版を一つ進め、有効な規則の保存と有効化は全件の再評価を予約する

- 管理者が初めて規則を保存したとき、IdManagement は、版を 1 とする無効な規則を作り、`DynamicGroupRuleUpdated` を発行する。
- 管理者が式を保存したとき、IdManagement は、版を一つ進め、保存前の有効か無効かを引き継ぎ、`DynamicGroupRuleUpdated` を発行する。
- 管理者が無効な規則を有効化したとき、IdManagement は、版を一つ進め、`DynamicGroupRuleEnabled` を発行する。
- 規則が有効な間、管理者が式を保存したとき、IdManagement は、全件の再評価を予約する。
- 管理者が無効な規則を有効化したとき、IdManagement は、全件の再評価を予約する。
- 管理者がすでに有効な規則を有効化した場合、IdManagement は、成功を返し、版を進めず、イベントを発行しない。
- `membership_type=manual` の Group への規則の保存を要求された場合、IdManagement は、409 と `dynamic_membership_managed_by_rule` で拒否する。
- 規則を持たない Group の有効化を要求された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 保存または有効化を拒否した場合、IdManagement は、規則と所属を変えない。

#### REQ-IDMANAGEMENT-065 規則の保存とプレビューは、式の制約に違反する規則を拒否する

- 空、または 4,097 バイト以上の式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 開き括弧が 21 個以上、または空白で区切った語が 201 個以上の式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- `startsWith`、`endsWith`、`contains`、`matches`、`lowerAscii`、`size`、`exists`、`all`、`timestamp` 以外の関数を呼ぶ式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- `matches` の正規表現が文字列の定数でないか、257 文字以上の式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- `user.<属性>` で User の属性を一つも参照しないか、101 個以上参照する式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- `id`、`preferred_username`、`name`、`given_name`、`family_name`、`email`、`email_verified`、組み込みの属性、テナント定義の属性のどれでもない属性（`roles` を含む）を参照する式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 結果が真偽値でない式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 評価の計算量が 10,000 を超える式を指定された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 制約に違反する規則の保存とプレビューを要求された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否し、規則を変えない。

#### REQ-IDMANAGEMENT-022 未定義の属性または許可していない関数を参照する規則の保存は拒否する

- 定義していない属性、または許可していない関数を参照する式の保存を要求された場合、IdManagement は、拒否し、規則を変えない。
- **例**：EX-IDMANAGEMENT-022-01

### 規則の無効化

#### REQ-IDMANAGEMENT-068 規則の無効化は版を一つ進め、動的グループの所属をすべて直ちに外す

- 管理者が有効な規則を無効化したとき、IdManagement は、版を一つ進め、`DynamicGroupRuleDisabled` を発行する。
- 管理者が有効な規則を無効化したとき、IdManagement は、全件の再評価を予約せず、その場で動的グループのメンバーシップをすべて外す。
- 管理者がすでに無効な規則を無効化した場合、IdManagement は、成功を返し、版を進めず、イベントを発行しない。
- 規則を持たない Group の無効化を要求された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。

### 全件の再評価

#### REQ-IDMANAGEMENT-020 有効な規則の全件の再評価は、規則に一致する User だけを規則由来のメンバーにする

- 有効な規則の全件の再評価を終えたとき、IdManagement は、規則に一致する有効な User だけを、規則を由来として所属させる。
- 全件の再評価を終えたとき、IdManagement は、実効ロールと Application の割り当ての判定に、その所属を使う。
- **例**：EX-IDMANAGEMENT-020-01

#### REQ-IDMANAGEMENT-067 動的グループの規則に一致するのは `Active` の User だけである

- User が `Active` でない間、規則を評価するとき、IdManagement は、式の値によらずその User を規則に一致させない。
- User の評価に失敗した場合、IdManagement は、その User を規則に一致しないものとして扱い、再評価の結果の誤りの件数に数える。

#### REQ-IDMANAGEMENT-023 評価できない規則は権限を付与しない

- 規則の版が進んだとき、IdManagement は、古い版の所属を、再評価を待たずに実効ロールから外す。
- 再評価で User の評価に失敗した場合、IdManagement は、その User に新しい版の所属を与えない。
- **例**：EX-IDMANAGEMENT-023-01

#### REQ-IDMANAGEMENT-069 古い版の再評価のジョブは、所属を変えずに成功する

- 全件の再評価を予約したとき、IdManagement は、予約した時点の規則の版をジョブに記録する。
- 再評価のジョブの実行時に規則がないか、無効であるか、版が異なる場合、IdManagement は、メンバーシップを変えず、イベントを発行せずに成功として終わる。

#### REQ-IDMANAGEMENT-070 動的な所属の変化は、メンバーごとのイベントを発行しない

- 全件の再評価を終えたとき、IdManagement は、追加、除外、変化なし、誤りの件数を載せた `DynamicMembershipEvaluated` を一つ発行する。
- 規則の再評価と、User の変更に伴う再評価で所属が増減したとき、IdManagement は、`GroupMemberAdded` と `GroupMemberRemoved` を発行しない。

### プレビュー

#### REQ-IDMANAGEMENT-021 規則のプレビューは、選んだ User ごとに一致の有無を返し、属性の値を返さない

- 管理者が 100 件以下の User を選んで保存していない式をプレビューしたとき、IdManagement は、User ごとの一致の有無と、追加、除外、変化なしの判定を返し、所属を変えない。
- 管理者が式をプレビューしたとき、IdManagement は、属性の値そのものを返さない。
- **例**：EX-IDMANAGEMENT-021-01

#### REQ-IDMANAGEMENT-071 規則のプレビューは 100 件以下の、同じテナントの User だけを受け付ける

- 101 件以上の User を指定したプレビューを要求された場合、IdManagement は、422 と `invalid_dynamic_group_rule` で拒否する。
- 存在しない User または別のテナントの User を一つでも含むプレビューを要求された場合、IdManagement は、結果を返さず、404 と `user_not_found` で拒否する。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 権限の付与 | 評価に失敗した User と、`Active` でない User は所属させない。判定できないときは権限を付与しない側へ倒す |
| 参照できる属性 | 式は `roles` を参照できない |
| 情報の開示 | プレビューは属性の値を返さない |
| 計算量 | 式の長さ、括弧の数、参照の数、評価の計算量に上限を置き、規則による評価の負荷を限る |
