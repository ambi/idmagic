# WsFederation Decisions

- RP の登録・参照・削除と Entra ドメインフェデレーションプロファイルの設定は `AdminFederationTrustsManage` 権限 (AuthZEN action `admin:federation_trusts_manage`) を要し、`admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行える。
- プロトコルのエンドポイントは管理者の認可を通らず、それぞれ別の境界を持つ。`federationmetadata.xml` と `/trust/mex` はテナントの公開 Discovery であり認証を要さないが、公開するのは発行者、エンドポイント、署名証明書に限る。パッシブサインインはブラウザーのログインセッションで主体を決め、`wtrealm`、`wreply`、対象ユーザーの Application 割り当てを検証してから発行する。能動的な STS は UsernameToken で認証し、`AppliesTo` が登録済みの RP に解決できることを求める。いずれの経路でも、未登録の宛先にトークンを発行することはない。
- フェデレーションメタデータの公開と、クレーム対応付けの担当を分ける。`WsFederation` が Discovery 情報（発行者、エンドポイント、署名証明書）を公開し、`ClaimMapping` が WS-Fed、WS-Trust、SAML に共通するクレーム公開ポリシーを担う。
- 能動的な WS-Trust の対応範囲は `/trust/usernamemixed` の `Issue` だけに絞る。束縛を広く覆うほど、再送と XML の包み替えに対する攻撃面が実質的に広がるからである。
- Entra のドメインフェデレーションは、汎用の RP 設定ではなく専用の定型設定として扱う。手書きのクレーム設定を誤ると、Entra 側では原因を特定しにくい障害として現れるからである。
- この Context を `Generic` に分類する。レガシー互換のための標準実装であり、準拠が価値のすべてである。独自の語彙やモデルを足す余地が無く、足せば相互運用を損なう。
