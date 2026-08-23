# Application Glossary

| Term | Definition | Aliases |
|---|---|---|
| Application | 運用者が接続、割り当て、監査する業務アプリケーション。`federated` または `service` の Application は、プロトコル設定を最大 1 つ持つ。 | アプリケーション, Application |
| ApplicationProtocol | Application が利用する 1 つのプロトコル設定への型付き参照。OAuth2Client、SamlServiceProvider、WsFedRelyingParty のいずれか 1 つを指す。 | application_protocol |
| AppSignInPolicy | Application ごとに順序付けた `SignInRule` の集合。フェデレーションの開始ごとに、トークンや Assertion の発行前に評価する。 | サインインポリシー |
| ApplicationAssignment | Application を利用できる主体を表す割り当て。`subject_type` が `user` の直接割り当てと `group` のグループ割り当てがあり、`visibility` が `hidden` の割り当てはポータルに現れないままフェデレーションだけを許可する。 | 割り当て |
