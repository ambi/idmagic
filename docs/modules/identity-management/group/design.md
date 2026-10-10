# グループの設計

この文書は、[グループ](README.md)の規則を保証する仕組みを扱う。

この機能の設計のうち、ほかの文書が扱う内容は次のとおりである。

- 信頼性設計：通知の失敗の扱いは規則 REQ-IDMANAGEMENT-064 が定め、モジュールの設計に従う

## アーキテクチャ

| 構成要素 | 責務 | 依存先 |
| --- | --- | --- |
| 管理者のユースケース | 作成、更新、削除、メンバーの追加と除外、所属グループの参照。変更を確定してから下流へ通知する | `GroupRepository`、`UserRepository`（メンバーの検証）、`EffectiveGroupAttributeSchemaReader`、`ProvisioningNotifier` |
| 実効ロールの合成 | 直接付与したロールと所属する Group のロールを、重複を除いて昇順に並べる | なし（純粋な関数） |
| 管理 API のハンドラー | 管理者の認可、入力の変換、応答の型への変換 | 管理者のユースケース |

実効ロールは、管理コンソールの認可とアカウントのセルフサービスの両方が、User の直接のロールではなく合成した結果で判定する。

### 実行時の流れ：削除

Group の削除は、メンバーシップを連鎖して消す。
メンバーごとの `GroupMemberRemoved` を発行してから、最後に `GroupDeleted` を発行する。

## データ

| データ | 内容 |
| --- | --- |
| 名前の一意性 | テナントと名前の比較キーの組の一意性の制約が同じ名前を防ぐ。仕組みは[排他と一意性](../design/data.md#排他と一意性)が定める |
| 属性のスキーマ | `TenantGroupAttributeSchema` は `Tenancy` が持つテナント単位の Aggregate である。どのプリンシパルのスキーマでも、テナント単位のスキーマの管理は `Tenancy` の関心事なので、`TenantUserAttributeSchema` と同じ場所に置く |
| 属性の定義の項目 | `GroupAttributeDef` は `key`、`label`、`type`、`multi_valued`、`required` だけを持つ。User の属性の定義にある本人による編集、クレームの名前、公開範囲は持たない |
