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
