# WorkloadIdentity の重要な設計判断

この文書は、WorkloadIdentity の設計判断のうち、代替案を比べたものと、見直す条件が要るものを扱う。
一つの要件やモデルだけを正当化する判断は、機能仕様のその要件またはモデルの判断の欄に置く。

## 外部のアテステーションを検証する RP に徹し長期のシークレットを配らない

### 背景

Agent の実行環境は、Kubernetes、クラウドのインスタンス、SPIFFE の基盤など、すでに自身の資格情報を持っていることが多い。
長期のシークレットを Agent へ配ると、その漏えいが Agent の成りすましに直結する。

### 決定

IdMagic は SPIRE のサーバーやエージェントを同梱、運用せず、外部のアテステーションを検証する RP として動作する。
検証したアテステーションは Token Exchange の subject として扱い、長期のシークレットを配布する専用の資格情報の経路は設けない。
信頼するのは事前に登録した発行者だけであり、Trust On First Use は許可しない。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| SPIRE のサーバーを同梱し、IdMagic が SVID を発行する | 実行環境に基盤がなくても、ワークロードを認証できる | 製品が実行環境のアテステーションの基盤を運用することになる | 既存の実行環境の資格情報を使えば足り、運用の範囲が広がりすぎる |
| Agent に長期のクライアントシークレットを配る | 実行環境を問わず使える | シークレットの配布、保管、ローテーションを利用者に負わせる | 漏えいが成りすましに直結する |

### 結果と再検討の条件

Agent は、実行環境が発行する短命のトークンだけで、IdMagic のアクセストークンを得る。
その代わり、アテステーションを発行できない実行環境では、この経路を使えない。
X.509-SVID による mTLS など、JWT 以外のアテステーションが要る場合に見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WORKLOADIDENTITY-001 | [アテステーションの交換](features/attestation-exchange.md) |
| REQ-WORKLOADIDENTITY-002 | [アテステーションの交換](features/attestation-exchange.md) |

## 複数の関連付けに一致した主体は優先順位で選ばず拒否する

### 背景

関連付けは主体の glob パターンで書くので、一つの主体が複数の関連付けに一致しうる。

### 決定

同じ主体に複数の有効な `AgentWorkloadBinding` が一致したら、どれかを選ばずに `ambiguous_match` で拒否する。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| より具体的なパターンを優先する規則を置く | 広いパターンと狭いパターンを併用できる | 関連付けを 1 本足しただけで、既存のワークロードの対応先が変わりうる | 管理者の意図しない成りすまし先の変化を、交換の時点まで検出できない |

### 結果と再検討の条件

関連付けを追加しても、既存のワークロードの対応先は変わらず、重なりは交換の拒否として現れる。
その代わり、管理者はパターンが重ならないように関連付けを書く必要がある。
重なりを登録の時点で検出して拒否できるようになった場合に見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WORKLOADIDENTITY-005 | [アテステーションの交換](features/attestation-exchange.md) |

## SPIFFE の規則は信頼設定の種別で選び、トークンの内容で選ばない

### 背景

[JWT-SVID 仕様](https://github.com/spiffe/spiffe/blob/main/standards/JWT-SVID.md)は、`sub` が SPIFFE ID であることを求め、`iss` を定めない。
参照実装の go-spiffe は、`sub` のトラストドメインで検証に使うバンドルを選び、`kid` のないトークンを拒否し、`iss` を確かめない。
`kid` は仕様の JOSE ヘッダーの節では任意だが、バンドルの各鍵には `kid` が必須であり、`kid` で鍵を一つに決められる。
一方、Kubernetes の投影 ServiceAccount トークンなど OIDC 互換の JWT の `sub` は SPIFFE ID ではない。
SPIRE は `jwt_issuer` を設定しない限り `iss` を入れないため、`iss` だけで信頼設定を探すと、デフォルトの SPIRE の JWT-SVID を検証できない。

### 決定

信頼設定に検証の種別（`oidc` または `spiffe`）を持たせ、`spiffe` の信頼設定にだけ SPIFFE の規則を掛ける。
種別は登録時に決め、更新では変えない。
`spiffe` の信頼設定の発行者は任意とし、信頼設定は `iss` で探し、なければ `sub` のトラストドメインで探す。
トラストドメインは、テナントの `spiffe` の信頼設定の間で一意にする。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| `sub` が `spiffe://` で始まるときだけ SPIFFE の規則を掛ける | 信頼設定に項目を足さずに済む | トークンの内容で検証の厳しさが変わる | `sub` の形を選べる発行者が、規則を外せる |
| `iss` を必須のままにし、SPIRE に `jwt_issuer` の設定を求める（Entra ID と PingFederate の形） | 信頼設定の探し方と一意性の制約を変えずに済む | デフォルトの SPIRE で使えず、SPIRE の OIDC Discovery Provider を外部から到達できる発行者として置く運用が増える | SPIFFE の信頼の単位はトラストドメインであり、`iss` はその代わりにならない。この形の使い方は `oidc` の信頼設定で引き続き使えるため、`spiffe` を足しても失われない |

### 結果と再検討の条件

デフォルトの SPIRE が発行した JWT-SVID を、発行者を登録せずに検証できる。
別のトラストドメインの主体を持つトークンは、`iss` と鍵が一致しても拒否する。
その代わり、管理者は信頼設定の登録時に種別を選び、種別を変えるには登録し直す。
署名の前に読む `sub` を信頼設定の選択に使うため、選択と主体の特定を混同しない実装の規則（[実現方式](features/attestation-exchange.md#実現方式)）を保つ必要がある。
X.509-SVID を受け付ける場合に、種別とトラストドメインの扱いを見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WORKLOADIDENTITY-008 | [信頼設定と関連付けの管理](features/trust-configuration.md) |
| REQ-WORKLOADIDENTITY-011 | [アテステーションの交換](features/attestation-exchange.md) |

## SPIFFE バンドルの鍵は取得した文書の形式で選び、Bundle Endpoint は `https_web` だけに対応する

### 背景

[SPIFFE Trust Domain and Bundle](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Trust_Domain_and_Bundle.md) のバンドルは JWKS であり、鍵ごとに `use` が `x509-svid`、`jwt-svid`、`wit-svid` のいずれかである。
`use` がないか未知の値の鍵は無視しなければならない（4.2.2 節）。
`spiffe_refresh_hint` は秒単位の推奨の再取得の間隔で、ないときは 5 分程度を使う（4.1.2、6.1 節）。
[SPIFFE Federation](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Federation.md) は、Bundle Endpoint のクライアントに `https_web` と `https_spiffe` の両方への対応を求める（5.2 節）。

一方、Entra ID は SPIRE の OIDC Discovery Provider の JWKS で JWT-SVID を検証し、鍵に `use: sig` を求める（SPIRE の `set_key_use`）。
この JWKS は SPIFFE バンドルではなく、`jwt-svid` の鍵を含まない。

他製品の扱いは次のとおりである。

| 製品 | 鍵の選別 | 再取得の間隔 | Bundle Endpoint の認証方式 |
| --- | --- | --- | --- |
| Keycloak（SPIFFE IdP） | `jwt-svid` と `use` のない鍵を使い、なければ `sig` の鍵に戻る | ヒントをキャッシュの寿命にし、`kid` の不一致で取得し直す | `https_web` だけ |
| Vault Enterprise（SPIFFE auth） | SPIFFE バンドルとして読む | ヒントを表示する | `https_web` と `https_spiffe` |
| Entra ID、Google Cloud IAM、Okta、PingFederate | SPIFFE バンドルを読まず、汎用の OIDC の JWKS を使う | 汎用の OIDC | — |

### 決定

`spiffe` の信頼設定では、取得した JWKS に `use` が `x509-svid`、`jwt-svid`、`wit-svid` のいずれかの鍵があれば SPIFFE バンドルとして読み、`jwt-svid` の鍵だけを使う。
なければ通常の JWKS として読み、`use` が `sig` か省略の鍵だけを使う。
`jwks_uri` の取得結果は 5 分を上限に再利用し、SPIFFE バンドルの `spiffe_refresh_hint` が短ければ、30 秒を下限としてその秒数まで縮める。
Bundle Endpoint は Web PKI で検証する `https_web` だけに対応する。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| `jwt-svid` の鍵だけを使う | SPIFFE の仕様どおりで、規則が一つで済む | OIDC Discovery Provider の JWKS を登録した `spiffe` の信頼設定は鍵を一つも得られない | Entra ID と同じ構成の SPIRE を、トラストドメインの検証付きで使えなくなる |
| Keycloak と同じく、`use` のない鍵も SPIFFE バンドルから使う | Keycloak と同じ結果になる | 仕様が無視を求める鍵で署名を検証する | 取得元の誤りで混ざった鍵を受け付ける |
| `spiffe_refresh_hint` をそのまま再利用の期間にする（Keycloak の形） | 取得元の意図どおりに取得の回数を減らせる | 28 日のようなヒントでは、鍵の入れ替えに遅れる | `kid` の不一致で取得し直す仕組みを持たない |
| `https_spiffe` にも対応する（Vault Enterprise の形） | 仕様の MUST を満たす | 信頼設定にエンドポイントの SPIFFE ID と最初のバンドルを足し、X.509-SVID の検証器を要する | 利用者の要求がなく、Keycloak も対応しない |

### 結果と再検討の条件

SPIFFE バンドルの `x509-svid` の鍵で署名したトークンを受け付けない。
OIDC Discovery Provider の JWKS も、`spiffe` の信頼設定でそのまま使える。
SPIFFE バンドルに `use` のない鍵を混ぜた取得元では、その鍵を使えない。
取得元の文書の形式で読み方を選ぶが、取得元は管理者が登録したものであり、ワークロードが提示するトークンでは選べない。
`https_spiffe` の Bundle Endpoint だけを公開する利用者の要求が生じたら、対応を見直す。
`kid` の不一致で取得し直す仕組みを足すなら、ヒントを 5 分より長く使うことを見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WORKLOADIDENTITY-003 | [アテステーションの交換](features/attestation-exchange.md) |
| REQ-WORKLOADIDENTITY-012 | [アテステーションの交換](features/attestation-exchange.md) |
| REQ-WORKLOADIDENTITY-008 | [信頼設定と関連付けの管理](features/trust-configuration.md) |
