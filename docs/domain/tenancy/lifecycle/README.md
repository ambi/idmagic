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

- realm が `default` のテナントがない間、IdP が起動したとき、Tenancy は、固定の `id`、realm `default`、表示名 `Default` の `Active` のテナントを作る。
- realm が `default` のテナントがある間、IdP が起動したとき、Tenancy は、そのテナントを変えない。
- IdP が起動したとき、Tenancy は、イベントを発行しない。
- Tenancy は、テナントを削除する API を提供しない。
- default テナントの無効化を要求された場合、Tenancy は、400 と `invalid_request` で拒否し、テナントを変えず、イベントを発行しない。
- **判断**：default テナントは制御面のテナントであり、System 管理者の所属先である。無効化を許すと、System 管理者が制御面の操作をすべて失う。

### System 管理者によるテナントの作成

#### REQ-TENANCY-025 System 管理者が作成するテナントの realm は、予約されていない単一の DNS ラベルである

- System 管理者がテナントを作成したとき、Tenancy は、`id` にサーバーが採番した UUID、`status` に `active`、`endpoint_style` に `path` を持つテナントを作り、201 と作ったテナントを返し、`TenantCreated` を発行する。
- System 管理者がテナントを作成したとき、Tenancy は、realm と表示名を前後の空白を除いて検証し、保存する。
- 空、64 文字以上、英小文字と数字とハイフン以外の文字を含む、または先頭か末尾がハイフンである realm を指定された場合、Tenancy は、400 と `invalid_request` で拒否する。
- `xn--` で始まる realm を指定された場合、Tenancy は、400 と `invalid_request` で拒否する。
- realm に `admin`、`www`、`api`、`login`、`id`、`sso`、`mail`、`smtp`、`status`、`docs`、`app`、`static`、`cdn`、`auth`、`account` のどれかを指定された場合、Tenancy は、400 と `invalid_request` で拒否する。
- 空の表示名を指定された場合、Tenancy は、400 と `invalid_request` で拒否する。
- 既存のテナントと同じ realm を指定された場合、Tenancy は、409 と `tenant_conflict` で拒否する。
- 作成を拒否した場合、Tenancy は、テナントを作らず、イベントを発行しない。
- **例**：EX-TENANCY-025-01、EX-TENANCY-025-02、EX-TENANCY-025-03、EX-TENANCY-025-04、EX-TENANCY-025-05

### System 管理者によるテナントの一覧

#### REQ-TENANCY-026 テナント一覧は id の昇順で、各テナントの上限と使用量を添えて返す

- System 管理者がテナントの一覧を取得したとき、Tenancy は、すべてのテナントを `id` の昇順で返す。
- System 管理者がテナントの一覧を取得したとき、Tenancy は、各テナントに上限の上書き（`quota`）と使用量（`usage`）を添える。
- 一覧の取得でテナントの上限または使用量を読み取れなかった場合、Tenancy は、そのテナントの項目を省いて返し、一覧の取得を失敗させない。
- **例**：EX-TENANCY-026-01、EX-TENANCY-026-02

#### REQ-TENANCY-014 通常のテナント管理者はシステムコンソールのテナント一覧にアクセスできない

- `system_admin` ロールを持たない利用者、または制御面テナントに所属しない利用者がテナントの一覧を要求した場合、Tenancy は、403 と `access_denied` で拒否し、どのテナントの情報も返さない。
- **例**：EX-TENANCY-014-01

### System 管理者によるテナントの取得

#### REQ-TENANCY-045 System 管理者によるテナントの取得は、realm で指定したテナントを上限と使用量とともに返す

- System 管理者が realm を指定してテナントを取得したとき、Tenancy は、そのテナントを、上限の上書き（`quota`）と使用量（`usage`）を添えて 200 で返す。
- 上限または使用量を読み取れなかった場合、Tenancy は、その項目を省いて返し、取得を失敗させない。
- 存在しない realm を指定された場合、Tenancy は、404 と `tenant_not_found` で拒否する。

### System 管理者によるテナントの無効化と再開

#### REQ-TENANCY-027 無効化と再開は、すでにその状態にあるテナントにも成功する

- System 管理者がテナントを無効化したとき、Tenancy は、テナントを `Disabled` にし、`disabled_at` を要求の時刻にし、204 を返し、`TenantDisabled` を発行する。
- System 管理者がテナントを再開したとき、Tenancy は、テナントを `Active` にし、`disabled_at` を消し、204 を返し、`TenantEnabled` を発行する。
- テナントが `Disabled` の間、System 管理者が無効化を要求したとき、Tenancy は、`disabled_at` を要求の時刻で上書きし、204 を返し、`TenantDisabled` を発行する。
- テナントが `Active` の間、System 管理者が再開を要求したとき、Tenancy は、204 を返し、`TenantEnabled` を発行する。
- 存在しない realm の無効化または再開を要求された場合、Tenancy は、404 と `tenant_not_found` で拒否し、イベントを発行しない。
- **例**：EX-TENANCY-027-01、EX-TENANCY-027-02

## セキュリティ上の考慮

テナントの作成、一覧、無効化、再開は制御面の操作であり、`system_admin` ロールと制御面テナントへの所属の両方を要求する。
この区分の理由は、[管理の認可を所属テナント内の admin とテナントを越える system_admin の 2 段にする](../design/decisions.md#管理の認可を所属テナント内の-admin-とテナントを越える-system_admin-の-2-段にする)。
