# テナントのライフサイクル

## 概要

この文書は、テナントの作成、デフォルトテナントの自動作成、無効化と再開、制御面のテナント一覧の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | テナントの作成、デフォルトテナントの自動作成、テナントの無効化と再開、制御面のテナント一覧 |
| 行為者 | System（起動時のデフォルトテナントの作成）、System 管理者（作成、一覧、無効化、再開）、テナント管理者（一覧の拒否） |
| 扱わないもの | テナントの設定値の変更は[テナント設定](../settings/README.md)が、上限の変更は[リソース上限](../quota/README.md)が、正規ロケーションの切り替えは[テナントの解決](../resolution/README.md)が扱う。テナントの物理削除は提供していない |

## モデル

テナントは、不変の UUID の `id` と、変更できて一意な `realm` を持つ。
外部に示す識別子には `realm` を、他の Aggregate からの参照には `id` を使う。

- **判断**：鍵を二つに分ける理由は、[テナントのキーを不変な UUID と可変な realm に分ける](../design/decisions.md#テナントのキーを不変な-uuid-と可変な-realm-に分ける)。

## 状態遷移

### TenantLifecycle

テナントは `Active` で通常稼働し、`Disable` で全プロトコルルートを停止する。`Enable` で復帰できる。物理削除は対象外とする。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。全プロトコルルートがレスポンスする |
| Disabled | — | 全プロトコルルートを停止している。`Enable` で復帰できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |

## 操作

### システムの起動によるデフォルトテナントの作成

#### REQ-TENANCY-003 default テナントは起動時に自動作成され削除も無効化もできない

### System 管理者によるテナントの作成

#### REQ-TENANCY-025 System 管理者が作成するテナントの realm は、予約されていない単一の DNS ラベルである

- realm と表示名は、前後の空白を除いてから検証する。
- realm は 1 文字以上 63 文字以下で、英小文字、数字、ハイフンだけからなり、先頭と末尾はハイフンではない。
- `xn--` で始まる realm を拒否する。
- 次の realm を拒否する。`admin`、`www`、`api`、`login`、`id`、`sso`、`mail`、`smtp`、`status`、`docs`、`app`、`static`、`cdn`、`auth`、`account`。
- 空の表示名を拒否する。
- 検証に反する要求は `invalid_request` の 400 で拒否し、テナントを作成しない。
- 既存のテナントと同じ realm の要求は `tenant_conflict` の 409 で拒否する。
- 作成したテナントは、状態が `active`、`endpoint_style` が `path` で、`id` はサーバーが採番した UUID である。
- 作成に成功した場合だけ `TenantCreated` を発行する。
- **担保手段**：`ValidateNewRealm`、`usecases.Create`

### System 管理者によるテナントの一覧

#### REQ-TENANCY-026 テナント一覧は id の昇順で、各テナントの上限と使用量を添えて返す

- 一覧は `id` の昇順で返す。
- 各テナントに、上限の上書き（`quota`）と使用量（`usage`）を添える。
- 上限または使用量を読み取れなかったテナントは、その項目を省いて返し、一覧の取得は失敗させない。
- **担保手段**：`TenantRepository.FindAll`
- **要判断**：読み取りの失敗を、上書きがないテナントと区別できない。項目を省くことを続けるか、取得を失敗させるかを決める。

#### REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

### System 管理者によるテナントの無効化と再開

#### REQ-TENANCY-027 無効化と再開は、すでにその状態にあるテナントにも成功する

- 無効化は `disabled_at` を要求の時刻に設定し、`TenantDisabled` を発行する。すでに無効なテナントでも同じで、`disabled_at` を上書きする。
- 再開は `disabled_at` を消し、`TenantEnabled` を発行する。すでに有効なテナントでも同じである。
- どちらも成功時に 204 を返す。
- 存在しない realm には `tenant_not_found` の 404 を返し、イベントを発行しない。
- **担保手段**：`SetDisabled`
- **要判断**：状態が変わらない要求でも `disabled_at` が動き、イベントを重ねて発行する。状態遷移表は `Active` と `Disabled` の間の遷移だけを定める。再実行を何もしない操作にするかを決める。

## セキュリティ上の考慮

テナントの作成、一覧、無効化、再開は制御面の操作であり、`system_admin` ロールと制御面テナントへの所属の両方を要求する。
この区分の理由は、[管理の認可を所属テナント内の admin とテナントを越える system_admin の 2 段にする](../design/decisions.md#管理の認可を所属テナント内の-admin-とテナントを越える-system_admin-の-2-段にする)。
