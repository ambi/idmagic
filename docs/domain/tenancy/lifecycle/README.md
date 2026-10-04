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

テナントは `Active` で通常稼働し、無効化で正規ロケーションへのすべての要求を止め、再開で `Active` に戻る。
テナントを物理削除する操作はない。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。正規ロケーションへの要求を受け付ける |
| Disabled | — | 正規ロケーションへの要求を拒否している。再開で `Active` に戻る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | realm != 'default' | Disabled |  |
| Disabled | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |
| Active | TenantEnabled | — | Active |  |

| State | 無効化 | 再開 | 正規ロケーションへの要求 |
|---|---|---|---|
| Active | → Disabled（default 以外）<br>拒否：400 invalid_request（default） | → Active | 何もしない |
| Disabled | → Disabled | → Active | 拒否：400 invalid_request |

## 操作

### システムの起動によるデフォルトテナントの作成

#### REQ-TENANCY-003 default テナントは起動時に自動作成され削除も無効化もできない

- IdP が起動したとき、realm が `default` のテナントがなければ、固定の `id`、realm `default`、表示名 `Default` の `Active` のテナントを作る。
- IdP が起動したとき、realm が `default` のテナントがすでにあれば、そのテナントを変えない。
- 起動時のテナントの作成では、イベントを発行しない。
- テナントを削除する API は提供しない。
- default テナントの無効化を要求された場合は、400 と `invalid_request` で拒否し、テナントを変えず、イベントを発行しない。
- **判断**：default テナントは制御面のテナントであり、System 管理者の所属先である。無効化を許すと、System 管理者が制御面の操作をすべて失う。

### System 管理者によるテナントの作成

#### REQ-TENANCY-025 System 管理者が作成するテナントの realm は、予約されていない単一の DNS ラベルである

- System 管理者がテナントを作成したとき、`id` にサーバーが採番した UUID、`status` に `active`、`endpoint_style` に `path` を持つテナントを作り、201 と作ったテナントを返し、`TenantCreated` を発行する。
- realm と表示名は、前後の空白を除いて検証し、保存する。
- realm は 1 文字以上 63 文字以下で、英小文字、数字、ハイフンだけからなり、先頭と末尾がハイフンでない値とする。
- `xn--` で始まる realm を指定された場合は、400 と `invalid_request` で拒否する。
- 次の realm を指定された場合は、400 と `invalid_request` で拒否する。`admin`、`www`、`api`、`login`、`id`、`sso`、`mail`、`smtp`、`status`、`docs`、`app`、`static`、`cdn`、`auth`、`account`。
- 単一の DNS ラベルでない realm と、空の表示名を指定された場合は、400 と `invalid_request` で拒否する。
- 既存のテナントと同じ realm を指定された場合は、409 と `tenant_conflict` で拒否する。
- 作成を拒否した場合は、テナントを作らず、イベントを発行しない。
- **例**：EX-TENANCY-025-01、EX-TENANCY-025-02、EX-TENANCY-025-03、EX-TENANCY-025-04、EX-TENANCY-025-05

### System 管理者によるテナントの一覧

#### REQ-TENANCY-026 テナント一覧は id の昇順で、各テナントの上限と使用量を添えて返す

- System 管理者がテナントの一覧を取得したとき、すべてのテナントを `id` の昇順で返す。
- 一覧の各テナントには、上限の上書き（`quota`）と使用量（`usage`）を添える。
- 上限または使用量を読み取れなかったテナントは、その項目を省いて返し、一覧の取得を失敗させない。
- **例**：EX-TENANCY-026-01、EX-TENANCY-026-02

#### REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

- `system_admin` ロールを持たない利用者、または制御面テナントに所属しない利用者がテナントの一覧を要求した場合は、403 と `access_denied` で拒否し、どのテナントの情報も返さない。
- **例**：EX-TENANCY-014-01

### System 管理者によるテナントの取得

#### REQ-TENANCY-045 System 管理者によるテナントの取得は、realm で指定したテナントを上限と使用量とともに返す

- System 管理者が realm を指定してテナントを取得したとき、そのテナントを、上限の上書き（`quota`）と使用量（`usage`）を添えて 200 で返す。
- 上限または使用量を読み取れなかった場合は、その項目を省いて返し、取得を失敗させない。
- 存在しない realm を指定された場合は、404 と `tenant_not_found` で拒否する。

### System 管理者によるテナントの無効化と再開

#### REQ-TENANCY-027 無効化と再開は、すでにその状態にあるテナントにも成功する

- System 管理者がテナントを無効化したとき、テナントを `Disabled` にし、`disabled_at` を要求の時刻にし、204 を返し、`TenantDisabled` を発行する。
- System 管理者がテナントを再開したとき、テナントを `Active` にし、`disabled_at` を消し、204 を返し、`TenantEnabled` を発行する。
- `Disabled` のテナントの無効化を要求されたとき、`disabled_at` を要求の時刻で上書きし、204 を返し、`TenantDisabled` を発行する。
- `Active` のテナントの再開を要求されたとき、204 を返し、`TenantEnabled` を発行する。
- 存在しない realm の無効化または再開を要求された場合は、404 と `tenant_not_found` で拒否し、イベントを発行しない。
- **例**：EX-TENANCY-027-01、EX-TENANCY-027-02

## セキュリティ上の考慮

テナントの作成、一覧、無効化、再開は制御面の操作であり、`system_admin` ロールと制御面テナントへの所属の両方を要求する。
この区分の理由は、[管理の認可を所属テナント内の admin とテナントを越える system_admin の 2 段にする](../design/decisions.md#管理の認可を所属テナント内の-admin-とテナントを越える-system_admin-の-2-段にする)。
