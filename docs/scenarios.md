# Cross-Context Scenarios

複数の Bounded Context が協調して初めて成り立つ振る舞いを置く。1 つの Context が単独で満たし検証できるものは、その Context の `scenarios.md` にある。

**ここに置く基準は「その Context だけでは `WHEN` を起こせないこと」である。** 引き金を持つ Context と結果を観測する Context が違う振る舞いは、どちらの `scenarios.md` に書いても片側の話にしかならず、保証の全体がどこにも書かれない状態になる。各シナリオは参加する Context を名指す。

### REQ-PLATFORM-001: 主体の無効化は、その主体へ到達するすべての経路を閉じる
参加する Context: IdManagement、Authentication、OAuth2、SharedSignals

- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は Active であり、認証済みのブラウザーセッションを持つ
- GIVEN "alice" は Agent "A1" と "A2" を所有し、両者に有効なアクセストークンが発行されている
- WHEN 管理者が "alice" を無効化する
- THEN "alice" のステータスは無効になる
- THEN "alice" の既存セッションによる認証必須 API の呼び出しは拒否される
- THEN "alice" の新規ログインは、正しいパスワードでも拒否される
- THEN "A1" と "A2" の AgentRevocationEpoch が同一エポックへ前進し、発行済みトークンはイントロスペクションで無効になる
- THEN "A1" と "A2" のそれぞれについて "AgentAccessRevoked" が発行される

**3 つの経路は同時に閉じる。** 1 つでも開いたままなら、この保証に違反する。どれか 1 つだけを述べた記述は、無効化がセキュリティ機能として成立していることの根拠にならない。

外部への伝播はこの保証に含まれない。SharedSignals の送信先が到達不能でも、ここまでは成立する（`REQ-SHAREDSIGNALS-007`）。**外部が知るより先に、内部で閉じ切ることがこのシナリオの主張である。**

### REQ-PLATFORM-002: 削除の予約は到達経路を閉じ、猶予期間内の復元は開き直す
参加する Context: IdManagement、Authentication

- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は Active である
- WHEN 管理者が "alice" の削除を予約する
- THEN "alice" のステータスは PendingDeletion になる
- THEN "alice" のログインは、正しいパスワードでも拒否される
- WHEN 管理者が猶予期間内に "alice" を復元する
- THEN "alice" のステータスは Active に戻る
- THEN "alice" はログインできる

削除の予約は無効化と別の状態遷移だが、**到達経路を閉じるという観測結果は同じでなければならない。** 片方だけが閉じる実装は、どちらの Context の記述にも違反しないまま成立してしまう。

### REQ-PLATFORM-003: 記録の正の変更は、有効な接続を持つ下流へ配信される
参加する Context: IdManagement、Application、Provisioning、Jobs

- ACTOR TenantAdministrator
- GIVEN Application "app-1" に有効な ProvisioningConnection が存在する
- GIVEN User "ユーザー-1" は "app-1" に割り当て済みである
- WHEN 管理者が "ユーザー-1" を作成、無効化、削除、または "app-1" への割り当てを解除する
- THEN 変更と同じトランザクションで `ProvisioningDelivery` が作成される
  - ALT 変更のトランザクションがロールバックする → `ProvisioningDelivery` も作成されない
- THEN `worker` が下流へ反映し、配信のステータスが `succeeded` になる

**配信行は発火元の変更と同時にコミットまたはロールバックする。** 記録の正が変わったのに配信が作られない状態、およびその逆は、いずれもこの保証に違反する。個々の変更がどの下流操作へ対応するかは Provisioning の `scenarios.md` が持つ。
