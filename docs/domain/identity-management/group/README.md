# グループ

## 概要

この文書は、User を束ねてロールをまとめて付与する `Group` を、管理者が作成し、変更し、メンバーを管理する機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `Group` の作成と更新、連絡先とカスタム属性、手動のメンバーシップ、Group 由来のロールを含む実効ロール |
| 行為者 | 管理者 |
| コードの機能スライス | `backend/idmanagement/group` |
| 扱わないもの | CEL の規則による所属は[動的グループ](../dynamic-group/README.md)が、CSV による一括の操作は[グループ CSV](../group-csv/README.md)が、ロールの正規化と予約ロールは[ロール](../roles/README.md)が扱う |

## モデル

`Group` は、テナントに属する Aggregate root である。
メンバーシップと動的グループの規則は `Group` の境界の内側にあり、User はその ID だけで参照する。

| 項目 | 内容 |
| --- | --- |
| `id` | 生成した後に変わらない `group_<uuid>` |
| `name` | 編集できる名前。正規化、比較、一意性は[名前](../README.md#値オブジェクト)の定義に従う |
| `description` | 説明 |
| `email` | 部署のメーリングリストのような連絡先。形式だけを検査し、確認済みの印も一意性も持たない |
| `roles` | メンバーへまとめて付与するロール |
| `membership_type` | `manual`（管理者が所属を決める）または `dynamic`（規則の評価だけが所属を決める）。作成のときにだけ選べる |
| `attributes` | テナントの `TenantGroupAttributeSchema` で検証する疎な属性 |

User の実効ロールは、User に直接付与したロールと、所属する Group のロールの和集合である。
どの Group にも属さない User の実効ロールは、直接付与したロールと同じである。

- **判断**：実効ロールに優先順位や減算の規則を設けない。平らな和集合で足りるところに拒否の規則を入れると、評価の順序の複雑さだけが増える。
- **判断**：`membership_type` を後から変えない。手動の所属と動的な所属では既存のメンバーシップの意味が変わり、反転させると、どの経路で付いた所属なのかを後から復元できない。
- **判断**：Group の属性を User の属性の定義へ統合せず、別の `GroupAttributeDef` で定義する。Group には、共有できる組み込みのカタログも、本人が編集する画面も、クレームとしての開示もないからである。
- **判断**：メンバーシップを `Group` の境界に入れ、User には入れない。規則が指す Group がない状態や、削除した Group にメンバーが残る状態を一瞬でも作らないためである。逆向きにすると、User 一人の更新が、その人の属するすべての Group を直列化させる。

## 操作

### 作成

#### REQ-IDMANAGEMENT-024 Group の作成は、連絡先と属性をテナントのスキーマで検証して保存し、`GroupCreated` を発行する

- 管理者が Group を作成したとき、IdManagement は、指定した名前、説明、連絡先、ロール、所属の種類、属性で Group を作り、`GroupCreated` を発行し、下流のプロビジョニングへ通知する。
- メールアドレスの形式を満たさない連絡先を指定された場合、IdManagement は、InvalidEmailError で拒否し、Group を作らない。
- 定義していないキーの属性、または定義した型と一致しない値の属性を指定された場合、IdManagement は、InvalidGroupAttributeError で拒否し、Group を作らない。
- **例**：EX-IDMANAGEMENT-024-01、EX-IDMANAGEMENT-024-03

#### REQ-IDMANAGEMENT-060 Group の作成と更新は、名前、説明、連絡先を正規化して保存する

- 管理者が Group を作成または更新したとき、IdManagement は、名前を前後の空白を除いて保存する。
- 管理者が Group を作成または更新したとき、IdManagement は、説明を前後の空白を除いて保存し、空になる説明を設定しない。
- 管理者が Group を作成または更新したとき、IdManagement は、連絡先のメールアドレスの前後の空白を除き、表示名付きの形式からアドレスだけを取り出し、小文字にして保存し、空になる値を設定しない。
- 前後の空白を除くと空になる名前を指定された場合、IdManagement は、422 と `group_name_required` で拒否し、Group を作成も更新もしない。
- 同じテナントのほかの Group と[名前](../README.md#値オブジェクト)が同じ名前を指定された場合、IdManagement は、409 と `group_name_conflict` で拒否し、Group を作成も更新もしない。

#### REQ-IDMANAGEMENT-061 テナントに Group 属性スキーマがなければ、Group の作成と更新は属性を拒否する

- テナントが Group 属性スキーマを定義していない間、属性を一つでも含む作成と更新を要求されたとき、IdManagement は、422 と `invalid_attribute` で拒否する。
- 必須の属性を指定しない作成を要求された場合、IdManagement は、必須の属性の欠落として拒否する。

### 更新

#### REQ-IDMANAGEMENT-062 Group の更新は、値が変わった項目だけを記録する

- 管理者が Group を更新したとき、IdManagement は、値が変わった項目だけを保存し、`name`、`description`、`email`、`attributes`、`roles` のうち値が変わった項目だけを `changed_fields` に載せた `GroupUpdated` を発行し、下流のプロビジョニングへ通知する。
- 管理者が `attributes` を指定して Group を更新したとき、IdManagement は、属性を検証し、指定した対応表で属性の全体を置き換える。
- 管理者が `attributes` を指定せずに Group を更新したとき、IdManagement は、属性を検証しない。
- 管理者がどの項目の値も変えない更新を要求した場合、IdManagement は、成功を返し、`updated_at` を進めず、`GroupUpdated` を発行せず、下流のプロビジョニングへ通知しない。
- **例**：EX-IDMANAGEMENT-062-01、EX-IDMANAGEMENT-062-02

### メンバーの追加と除外

#### REQ-IDMANAGEMENT-015 User のメンバーへの追加は `GroupMemberAdded` を発行し、User の実効ロールに Group のロールを加える

- 管理者が User を Group に所属させたとき、IdManagement は、メンバーシップを作り、`GroupMemberAdded` を発行し、下流のプロビジョニングへ通知し、User の実効ロールに Group のロールを加える。
- 管理者が User を Group から外したとき、IdManagement は、メンバーシップを消し、`GroupMemberRemoved` を発行し、下流のプロビジョニングへ通知する。
- 管理者が同じ User を同じ Group へもう一度所属させた場合、IdManagement は、成功を返し、`GroupMemberAdded` を再発行しない。
- **例**：EX-IDMANAGEMENT-015-01、EX-IDMANAGEMENT-015-02

#### REQ-IDMANAGEMENT-063 手動のメンバーの追加は、削除されていない同じテナントの User を受け付ける

- 管理者が同じテナントの `Deleted` でない User（`Disabled` と `PendingDeletion` を含む）を追加したとき、IdManagement は、メンバーシップを作る。
- 管理者がメンバーでない User を除外した場合、IdManagement は、成功を返し、イベントを発行しない。
- 存在しない User、`Deleted` の User、別のテナントの User の追加を要求された場合、IdManagement は、404 と `user_not_found` で拒否し、メンバーシップを作らない。

#### REQ-IDMANAGEMENT-085 動的グループへの手動のメンバーの追加と除外は拒否する

- `membership_type=dynamic` の Group へのメンバーの追加と除外を要求された場合、IdManagement は、409 と `dynamic_membership_managed_by_rule` で拒否し、メンバーシップを変えない。
- **判断**：動的グループの所属は規則の評価だけが決める。手動の操作を許すと、所属がどの経路で付いたのかを区別できなくなる。
- **例**：EX-IDMANAGEMENT-085-01

#### REQ-IDMANAGEMENT-064 下流への通知に失敗した Group の変更は、確定したままエラーを返す

- 管理者が Group の作成、更新、削除、手動のメンバーの追加と除外をしたとき、IdManagement は、変更を確定してから下流のプロビジョニングへ通知する。
- 下流のプロビジョニングへの通知に失敗した場合、IdManagement は、エラーを返し、確定した変更と発行したイベントを取り消さない。

### 所属グループの参照

#### REQ-IDMANAGEMENT-084 User の所属グループの参照は、実効ロールと、Group 由来のロールと直接付与したロールを分けて返す

- 管理者が User の所属グループを参照したとき、IdManagement は、所属する Group と、実効ロール、Group 由来のロール（`group_roles`）、直接付与したロール（`direct_roles`）を返し、状態を変えない。
- 管理者が User の所属グループを参照したとき、IdManagement は、Group 由来のロールと直接付与したロールの和集合を実効ロールとして返す。
- 存在しない User と別のテナントの User の参照を要求された場合、IdManagement は、404 と `user_not_found` で拒否する。
- **例**：EX-IDMANAGEMENT-084-01

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| テナント境界 | テナントをまたぐメンバーシップは作らない。追加では対象の User を読み、別のテナントの User を存在しない User と同じ応答で拒否する |
| ロールの付与 | Group のロールはメンバー全員の実効ロールに加わるので、Group のロールの変更は管理者の権限の変更と同じ重さを持つ。予約ロールの制限は[ロール](../roles/README.md)が定める |
