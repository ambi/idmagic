# WsFederation の重要な設計判断

この文書は、WsFederation の設計判断のうち、代替案を比べたものと、見直す条件が要るものを扱う。
一つの要件やモデルだけを正当化する判断は、機能仕様のその要件またはモデルの判断の欄に置く。

## WS-Trust の能動的な対応を usernamemixed の Issue だけに絞る

### 背景

SOAP、WS-Security、WS-Addressing、SAML の署名は相互に関係する。
対応するバインディングを増やすほど、再送や XML 署名の包み替えへの攻撃面が広がる。
能動的な WS-Trust を使うのは、主に Microsoft 365 型のリッチクライアントである。

### 決定

能動的な WS-Trust の対応範囲は、`/trust/usernamemixed` の WS-Trust 1.3 の `Issue` だけに絞る。
`Validate`、`Renew`、`Cancel` と、Kerberos と IWA の `windowstransport` は実装しない。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| WS-Trust の操作とバインディングを広く実装する | 汎用的な WS-Trust のクライアントと相互運用できる | 束縛を広く覆うほど、再送と XML の包み替えへの攻撃面が実質的に広がる | 対象のクライアントが使わない経路のために攻撃面を広げる |

### 結果と再検討の条件

リッチクライアントのサインインに要る経路だけを、フェイルクローズで検証すれば足りる。
その代わり、Kerberos による統合認証や Hybrid Azure AD Join の端末の登録は使えない。
対象のクライアントが `windowstransport` などの別の経路を必須とするようになった場合に見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-004 | [WS-Trust の能動的 STS](../active-sts/README.md) |
| REQ-WSFEDERATION-005 | [WS-Trust の能動的 STS](../active-sts/README.md) |

## Entra のドメインフェデレーションを専用の定型設定として扱う

### 背景

Microsoft Entra のドメインフェデレーションは、UPN と ImmutableID のクレームを特定の形で求める。
手書きのクレームの設定を誤ると、Entra 側では原因を特定しにくい障害として現れる。

### 決定

Entra のドメインフェデレーションは、汎用の RP の設定ではなく、専用の定型設定（`EntraFederationProfile`）として扱う。
必須のクレームは定型設定で固定する。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| 汎用の RP の設定とクレームのポリシーで書いてもらう | 専用の設定と画面が要らない | クレームの設定の JSON を手書きすることになる。設定の時点で sourceAnchor の安定性や一意性を保証できない | 誤りが Entra 側の原因の分かりにくい障害として現れる |

### 結果と再検討の条件

管理者は、ドメイン、IssuerUri、sourceAnchor の属性を指定するだけで、Entra との信頼関係を作れる。
設定の時点で、既存のユーザーの sourceAnchor の欠落、重複、変換できない値を拒否できる。
その代わり、Entra が求めるクレームが変わったら、定型設定のコードを変える。
Entra 以外にも定型設定が要る RP が増えた場合に、定型設定の一般化を検討する。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-001 | [RP と Entra フェデレーションの管理](../relying-party/README.md) |

## フェデレーションメタデータの公開とクレームの対応付けの担当を分ける

### 背景

WS-Fed の RP は、メタデータから発行者、エンドポイント、署名証明書を得て、トークンのクレームで利用者を識別する。
同じクレームの公開ポリシーを、WS-Fed、WS-Trust、SAML で使う。

### 決定

`WsFederation` が Discovery の情報（発行者、エンドポイント、署名証明書）を公開し、`ClaimMapping` が WS-Fed、WS-Trust、SAML に共通するクレームの公開ポリシーを担う。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| このモジュールが WS-Fed のクレームの規則を持つ | WS-Fed に特有の規則を書きやすい | SAML と同じ公開ポリシーを二重に持つ | プロトコルごとに非公開の属性が漏れる形を作れる |

### 結果と再検討の条件

クレームの公開の下限は、どのプロトコルでも同じ経路で検査される。
その代わり、WS-Fed に特有のクレームの整形（Entra の ImmutableID など）は、このモジュールが `ClaimMapping` の前段で行う。
WS-Fed にだけ必要なクレームの規則が増え、前段の整形で表せなくなった場合に見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-002 | [パッシブサインイン](../passive-sign-in/README.md) |
| REQ-WSFEDERATION-004 | [WS-Trust の能動的 STS](../active-sts/README.md) |
