# Application Internals

## Assignment as a desired state, not a command

割り当ての作成と解除は、呼び出し元の Bounded Context (IdManagement の LifecycleWorkflow など) が「こうあるべき」という状態を渡す形をとる。HTTP には公開せず、同じプロセス内の Go 呼び出しとして各 Context のユースケースから使う。

すでにその状態であれば何も変えず、変更しなかったことを返す。存在しない割り当ての解除も同じく正常終了する。ワークフローは同じ発火事象で何度も再実行されうるので、この冪等性が無いと、再実行のたびに割り当てが増えるか、解除が失敗して実行全体を止めることになる。

呼び出し元が渡せるのは同じテナント内の識別子だけである。内部インターフェースであっても、テナント境界を呼び出し元の作法に委ねない。

## Sign-in policy evaluation

ApplicationCatalog は、テナントとアプリケーションごとに順序付けた `SignInRule` の集合を `AppSignInPolicy` として持つ。OIDC の認可、SAML の SSO、WS-Fed のサインインなど、フェデレーションを開始するたびにトークンや Assertion の発行前に評価する。アプリケーションとプロトコル設定の関連付けを確認するのと同じ関門で評価するため、別のプロトコルを入口に選んでもポリシー評価を迂回できない。設定できるのは評価器が実際に確認できる値だけである。

必須の認証強度は自由入力ではなく、`Password` または `Mfa` に制約した `RequiredAuthnStrength` 列挙であり、内部の ACR URN と AMR 値へ 1 対 1 で対応付ける。実際に存在する ACR 値は 2 種類だけで、制約のない文字列は設定ミスを招くためである。`reauth_max_age_seconds` は Authentication のステップアップ認証の直近性に対して評価し、`network_allow_cidrs` は管理者が指定して保存時に検証した CIDR に対してリクエスト元のクライアント IP を検査する。評価器が確認できない端末条件は受け付けない。

`Mfa` を要求する規則は、Authentication の信頼済みデバイスで満たしてよいかどうかを `allow_trusted_device` で区別する。既定は `true` で、記憶済みの端末を表す `tdev` の AMR 値でも要件を満たす。`false` は「毎回 MFA」を意味し、本物の第二要素の提示だけを認める。認証強度の語彙そのものを増やさないのは、要求している強度は同じで、その満たし方だけが違うからである。要求強度を `Mfa` と `MfaEveryTime` に分けると、強度の順序関係が 1 本の線でなくなり、テナントのデフォルトとの強弱の比較が定義できなくなる。

評価はすべてフェイルクローズで行う。OIDC は認証強度不足の結果を既存のステップアップ認証フローへ送れる。一方、SAML と WS-Fed には遷移先となるステップアップ認証機構がまだないため、明示的な拒否理由を付けてプロトコルトランザクションを直ちに停止する。空でない CIDR 許可リストにクライアント IP が一致しない場合や、リクエスト元のクライアント IP を特定できない場合は、ステップアップ認証の機会とはせず、無条件に拒否する。

## Tenant default policy composition

`TenantDefaultSignInPolicy` により、テナントは独自のポリシーを定義していないすべてのアプリケーションに対して、基準となるサインインポリシーを 1 つ設定できる。アプリケーションごとのポリシーと同じ `SignInRule` の語彙と評価器を使用し、別のポリシー言語は設けない。これはテナント Aggregate ではなくアプリケーションへのサインイン方法に関する概念なので、`Tenancy` ではなく ApplicationCatalog が所有する。

デフォルトポリシーとアプリケーションごとのポリシーは合成せず、後者で上書きする。アプリケーションが有効な規則を 1 つでも定義していれば、その規則がテナントのデフォルトを完全に置き換える。定義がなければデフォルトをそのまま適用する。`EffectiveSignInRules(default, app)` がどちらか一方を選び、共通のフェイルクローズ評価器に渡すため、各アプリケーションに実際に適用されるポリシーを 1 つの形で確認できる。

上書きによってアプリケーションはテナントのデフォルトより弱いポリシーを設定できるため、`AppSignInPolicyResponse` は `weaker_than_default` フラグを持つ。要求する認証強度を下げる、再認証の時間制限を緩めるか外す、許可ネットワークを広げる、デフォルトが禁じている信頼済みデバイスによる充足を許す場合に、`AppPolicyWeakerThanDefault(default, app)` がこの値を算出する。保存を禁止するのではなく、UI に警告を表示する。新しいテナントは規則が空の、すべてを許可するデフォルトから始める。デフォルトは通常のテーブル行として保存するので、規則を空にするか行を削除すれば、スキーマを変更せずにすべてを許可する状態へ戻せる。

## Application/protocol relation

Application が持つプロトコル設定は最大 1 つとし、作成時に固定する。`weblink` アプリケーションはプロトコル設定を持たず、`federated` と `service` のアプリケーションは、OAuth2 クライアント、SAML SP、WS-Federation RP のいずれか 1 つだけを持つ。作成後の再接続、切り離し、プロトコル種別の変更には対応しない。

各プロトコルのテーブル（`oauth2_clients`、`saml_service_providers`、`wsfed_relying_parties`）は、`NULL` を許容する一意な `application_id` を持つ。`application_id` が `NULL` でない場合は、テナントと固定のプロトコル判別子も含む複合外部キーで参照する。これにより、2 つのプロトコル行が同じ Application を参照すること、テーブルをまたいで重複して参照すること、テナントや種別が食い違うことをデータベース自身が拒否する。`NULL` は、Dynamic Client Registration や信頼管理 API で作成され、Application カタログには表示しない正当なレコードを表す。そのため、すべてのプロトコル設定に Application を必須とはしない。

カタログへの作成では、Application 行の作成とプロトコル行への `application_id` の設定を 1 つのトランザクションで確定する。後半が失敗しても、カタログにだけ表示される孤立した Application は残らない。Application を削除すると、それに紐づくプロトコル設定も連鎖して削除する。一方、Application が所有するプロトコル設定を各プロトコルの管理 API から直接削除しようとした場合は、競合として拒否する。削除は必ず所有元の Application を経由する。

OAuth2 のプロトコルテーブルは `oauth2_clients` とする。SAML と WS-Fed のテーブル（`saml_service_providers`、`wsfed_relying_parties`）と同様に、プロトコル固有の標準用語を使う。

## Portal application ordering and category

ApplicationCatalog は、エンドユーザーポータルでの手動の並び順と、管理者が定義するカテゴリの両方を扱う。どちらも IdentityManagement の User Aggregate ではなく、`Application` の表示に関する概念だからである。手動の並び順は `ApplicationOrdering` として、`(tenant_id, user_sub)` ごとの `application_id` の一覧で表す。`ListMyApplications` は、割り当て済みで可視かつ有効なアプリケーションを解決してから保存済みの並び順を適用する。割り当てが外れた項目は除外し、保存済みの一覧にない割り当て済みアプリケーションは名前順で末尾に加える。並び順が保存されていない場合は、すべてを名前の昇順に並べる。このため、割り当てが並行して変わっても一覧は壊れない。`ReorderMyApplications` は並び順の一覧を作成または更新するだけであり、個人の表示設定なのでドメインイベントを発行しない。カテゴリはテナントごとに管理者が定義し、Application ごとに 0 個以上を割り当てる。
