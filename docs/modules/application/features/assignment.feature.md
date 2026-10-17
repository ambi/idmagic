# Feature: 割り当ての例

## Rule: REQ-APPLICATION-011 割り当てのない主体はプロトコル経由でアプリケーションへフェデレーションできない

### Example: EX-APPLICATION-011-01 通常経路

- Given アプリケーション "portal" にユーザー "alice" は割り当てられていない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- Then 割り当てがないため、フェデレーションを拒否する

### Example: EX-APPLICATION-011-02 管理者が事前に "portal" へ "alice" を `visibility=visible` で割り当てる

- Given アプリケーション "portal" にユーザー "alice" は割り当てられていない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- But 管理者が事前に "portal" へ "alice" を `visibility=visible` で割り当てる
- Then "alice" はフェデレーションを完了できる

## Rule: REQ-APPLICATION-012 hidden の割り当てはポータル一覧から除外するがプロトコルの利用は許可する

### Example: EX-APPLICATION-012-01 通常経路

- Given 管理者が "portal" にユーザー "alice" を `visibility=hidden` で割り当てている
- When "alice" が自分のポータルアプリケーション一覧（ListMyApplications）を取得する
- Then 一覧に "portal" は含まれない
- When "alice" が "portal" へのフェデレーションを試みる（強制点は OAuth2.Authorize）
- Then `hidden` の割り当てがあるため、フェデレーションを完了できる

## Rule: REQ-APPLICATION-014 あるべき状態を指定した割り当てはグループ割り当てを変更しない

### Example: EX-APPLICATION-014-01 通常経路

- Given "alice" は動的グループを介して "portal" へのグループ割り当て（`subject_type=group`）をすでに持つ
- When IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
- Then "alice" 個人への直接ユーザー割り当て（`subject_type=user`）が作成される
- Then グループ割り当て（`subject_type=group`）のレコードは変更されない
- When LifecycleWorkflow が後から UnassignApplicationDesiredState を呼び出す
- Then 直接ユーザー割り当てだけが削除され、グループ割り当ては残る
- Then フェデレーションは引き続き許可される

### Example: EX-APPLICATION-014-02 個人への直接割り当てが指定どおりの `visibility` ですでに存在する

- Given "alice" は動的グループを介して "portal" へのグループ割り当て（`subject_type=group`）をすでに持つ
- When IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
- But 個人への直接割り当てが指定どおりの `visibility` ですでに存在する
- Then 変更せずに `changed=false` を返す
