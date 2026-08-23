# Application Decisions

- Application、プロトコル設定、カテゴリ、割り当て、サインインポリシーの管理はいずれも `admin` ロールを持つ、有効かつ認証済みのユーザーに限る。AuthZEN の action は対象ごとに分かれ、`admin:applications_manage`、`admin:application_assignments_manage`、`admin:application_policies_manage`、`admin:application_categories_manage`、`admin:tenant_default_sign_in_policy_manage` を要求する。いずれも操作対象が呼び出し元と同じテナントに属することを条件とする。テナントのデフォルトサインインポリシーだけは Application 単位ではなくテナント設定なので `settings:read` / `settings:write` に対応させる。
- エンドユーザーが触れるのは自分自身の範囲だけである。`ListMyApplications` と `ReorderMyApplications` は認証済み本人の割り当てと表示設定に閉じ (`account:applications_read`)、他のユーザーの一覧や並び順へは到達できない。
- 割り当てを持たない主体は、どのプロトコルからもフェデレーションを完了できない。この関門は Application とプロトコル設定の関連付けを確認するのと同じ場所にあるため、別のプロトコルを入口に選んで迂回することはできない。`AssignApplicationDesiredState` と `UnassignApplicationDesiredState` は HTTP に公開せず、同じテナント内の識別子しか受け取らない内部インターフェースである。
- サインインポリシーには、評価器が実際に確認できる値だけを設定できるようにする。自由入力の認証強度や端末条件を許すと、保存はできるのに評価で必ず落ちる設定が作れてしまうからである。
- テナントのデフォルトとアプリケーションごとのポリシーは合成せず、上書きとする。合成規則を持つと、あるアプリケーションに実際に適用されるポリシーを 1 つの形で示せなくなるからである。
- MFA を信頼済みデバイスで満たしてよいかは、認証強度の語彙を増やさず `Mfa` 規則の属性として表す。強度の列挙を増やすと強弱の順序が 1 本の線でなくなり、テナントのデフォルトとの比較が定義できなくなるからである。
- デフォルトより弱いポリシーは保存を禁じず、警告として示す。運用上、特定のアプリケーションだけ要件を下げる必要は実在するため、禁止すると設定そのものが迂回されるからである。
- Application とプロトコル設定の 1 対 1 の関係は、JSON 配列による関連付けではなく、プロトコルテーブルからの複合外部キーで保証する。テナントや種別の食い違いをデータベース自身に拒否させるためである。
