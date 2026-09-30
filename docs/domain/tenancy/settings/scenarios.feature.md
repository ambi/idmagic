# Feature: テナント設定のシナリオ

## 入力

### Rule: REQ-TENANCY-019 管理者はパスワードポリシー設定を参照・更新できる

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-019-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- Then 更新後の設定に新しい最小長が反映される
- Then 上書きは永続化され、プロセス再起動後の設定取得でも同じ値が返る
- When 管理者 "operator" が max_age_days=90 を保存する
- Then 以後のパスワード検証と有効期限判定にテナントの上書き値が使われる

#### Example: EX-TENANCY-019-02 標準値より弱い上書き (最小長を下回る / 最大長を上回る / 履歴件数を下回る) を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- But 標準値より弱い上書き (最小長を下回る / 最大長を上回る / 履歴件数を下回る) を保存する
- Then エラー "PolicyOverrideWeakerError"

#### Example: EX-TENANCY-019-03 max_age_days に system ceiling の範囲外 (30 未満、または 3650 超) を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" がパスワードの最小長を更新する
- But max_age_days に system ceiling の範囲外 (30 未満、または 3650 超) を保存する
- Then エラー "PolicyOverrideWeakerError"

### Rule: REQ-TENANCY-021 委譲深さの上書きは厳しい方向にのみ働く

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-021-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- Then 設定取得のレスポンスは現在の上書き値と、上書きが無いときに適用されるシステムデフォルトの双方を返す

#### Example: EX-TENANCY-021-02 システムデフォルトより小さい値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But システムデフォルトより小さい値を保存する
- Then 上書きが永続化され、以後のトークン交換の判定に使われる

#### Example: EX-TENANCY-021-03 システムデフォルトを超える値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But システムデフォルトを超える値を保存する
- Then エラー "PolicyOverrideWeakerError"

#### Example: EX-TENANCY-021-04 1 未満の値を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But 1 未満の値を保存する
- Then エラー "PolicyOverrideWeakerError"

#### Example: EX-TENANCY-021-05 0 を保存する

- Given ロール=["admin"] のユーザー "operator" が管理画面の設定を開いている
- When 管理者 "operator" が委譲深さの上限を保存する
- But 0 を保存する
- Then 上書きを解除し、システムデフォルトを継承する状態へ戻す

### Rule: REQ-TENANCY-028 信頼済みデバイスの有効期間は 0 で無効にし、90 日を超える値を拒否する

- `trusted_device_max_age_seconds` を省略した更新は、保存済みの値を変えない。
- `0` は信頼済みデバイスを無効にし、設定の取得ではこの項目を省いて返す。
- 1 以上 7,776,000（90 日）以下の値を保存する。
- 負の値と 7,776,000 を超える値は `policy_override_weaker` の 422 で拒否し、保存済みの値を変えない。
- 設定の取得は、上限の 7,776,000 を `trusted_device_max_age_seconds_ceiling` として常に返す。
- 保存済みの値が範囲の外にあるテナントでは、信頼済みデバイスを無効として扱う。
- **理由**：[セキュリティポリシーの上書き](decisions.md)
- **担保手段**：`usecases.Update`、`Tenant.EffectiveTrustedDeviceMaxAge`

#### Example: EX-TENANCY-028-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- When "operator" が `trusted_device_max_age_seconds` に 2,592,000 を保存する
- Then 設定の取得は 2,592,000 と上限 7,776,000 を返す

#### Example: EX-TENANCY-028-02 0 を保存する

- Given テナントの `trusted_device_max_age_seconds` は 2,592,000 である
- When "operator" が `trusted_device_max_age_seconds` に 0 を保存する
- Then 設定の取得は `trusted_device_max_age_seconds` を返さない

#### Example: EX-TENANCY-028-03 上限を超える値を保存する

- Given テナントの `trusted_device_max_age_seconds` は 2,592,000 である
- When "operator" が `trusted_device_max_age_seconds` に 7,776,001 を保存する
- Then `policy_override_weaker` の 422 で拒否され、保存済みの値は 2,592,000 のままである

#### Example: EX-TENANCY-028-04 範囲の外の値が保存されている

- Given テナントに、上限を超える `trusted_device_max_age_seconds` が保存されている
- When 信頼済みデバイスの有効期間を読み出す
- Then 有効期間は 0 で、信頼済みデバイスは無効として扱われる

### Rule: REQ-TENANCY-029 通知のデフォルト言語は、同梱翻訳のある言語だけを受け付ける

- `default_locale` は、前後の空白を除いてから検証する。
- 空文字列は設定を消し、システムのデフォルト言語を使う状態に戻す。
- 同梱翻訳のない言語は `invalid_request` の 400 で拒否し、保存済みの値を変えない。
- 設定の取得は、同梱翻訳のある言語の一覧を `supported_locales` として返す。
- **担保手段**：`usecases.Update`、`LocaleSupported`

#### Example: EX-TENANCY-029-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- When "operator" が `default_locale` に "ja" を保存する
- Then 設定の取得は `default_locale` が "ja" で、`supported_locales` に "ja" と "en" を含む

#### Example: EX-TENANCY-029-02 空文字列を保存する

- Given テナントの `default_locale` は "ja" である
- When "operator" が `default_locale` に空文字列を保存する
- Then 設定の取得は `default_locale` を返さない

#### Example: EX-TENANCY-029-03 同梱翻訳のない言語を保存する

- Given テナントの `default_locale` は "ja" である
- When "operator" が `default_locale` に "fr" を保存する
- Then `invalid_request` の 400 で拒否され、`default_locale` は "ja" のままである

### Rule: REQ-TENANCY-030 パスワードポリシーの上書きは 0 以下の項目を継承として読み、上書きを含む更新だけが基準時刻を進める

- 上書きの各項目は、省略または 0 以下の値を、プロダクトのデフォルト値の継承として読む。
- すべての項目が継承である上書きは、上書きそのものを消す。
- 要求が `password_policy_override` を含む場合は、値が保存済みと同じでも `password_policy_updated_at` を要求の時刻に進める。
- 要求が `password_policy_override` を含まない場合は、`password_policy_updated_at` を変えない。
- **担保手段**：`usecases.Update`

#### Example: EX-TENANCY-030-01 すべての項目を 0 にして保存する

- Given テナントはパスワードの最小長を 16 に上書きしている
- When "operator" がすべての項目が 0 の `password_policy_override` を保存する
- Then テナントの上書きは消え、設定の取得は `password_policy_override` を返さない

#### Example: EX-TENANCY-030-02 表示名だけを更新する

- Given テナントはパスワードの最小長を 16 に上書きしている
- When "operator" が表示名だけを更新する
- Then `password_policy_updated_at` は変わらない

#### Example: EX-TENANCY-030-03 同じ上書きをもう一度保存する

- Given テナントはパスワードの最小長を 16 に上書きしている
- When "operator" が最小長 16 の `password_policy_override` をもう一度保存する
- Then `password_policy_updated_at` は二度目の要求の時刻に進む

## 作用

### Rule: REQ-TENANCY-031 設定の更新は、要求に含まれた項目を TenantUpdated に記録する

- 更新に成功した場合だけ `TenantUpdated` を発行する。
- `changed_fields` には、値が変わったかにかかわらず、要求に含まれた項目の名前を載せる。
- **担保手段**：`Deps.handleUpdateAdminSettings`
- **要判断**：値が変わらない項目も `changed_fields` に載る。実際に変わった項目だけにするかを決める。

#### Example: EX-TENANCY-031-01 保存済みと同じ表示名を送る

- Given テナントの表示名は "Acme" である
- When "operator" が表示名 "Acme" で設定を更新する
- Then "TenantUpdated" が発行され、`changed_fields` は "display_name" を含む

#### Example: EX-TENANCY-031-02 拒否される更新

- Given テナントの表示名は "Acme" である
- When "operator" が表示名 "Acme Corp" と、上限を超える `max_delegation_depth` を一つの要求で送る
- Then 要求は拒否され、表示名は "Acme" のままで、イベントは発行されない
