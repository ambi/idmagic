# WsFederation の重要な設計判断

この文書は、WsFederation の設計判断のうち、代替案を比べたものと、見直す条件が要るものを扱う。
一つの要件やモデルだけを正当化する判断は、機能仕様のその要件またはモデルの判断の欄に置く。

## WS-Trust の能動的な対応を usernamemixed の Issue だけに絞る

### 背景

SOAP、WS-Security、WS-Addressing、SAML の署名は相互に関係する。
対応するバインディングを増やすほど、再送や XML 署名の包み替えへの攻撃面が広がる。
能動的な WS-Trust を使うのは、主に Microsoft 365 型のリッチクライアントである。
Microsoft Entra hybrid join の端末は、コンピューターアカウントの Kerberos で `windowstransport` を呼ぶ。
Microsoft は `windowstransport` を社内向けの入口にだけ置き、社外へ公開してはならないと求める（[Microsoft Entra hybrid join の計画](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-plan)）。

### 決定

能動的な WS-Trust の対応範囲は、`/trust/usernamemixed` の WS-Trust 1.3 の `Issue` だけに絞る。
`Validate`、`Renew`、`Cancel` と、Kerberos と IWA の `windowstransport` は実装しない。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| WS-Trust の操作とバインディングを広く実装する | 汎用的な WS-Trust のクライアントと相互運用できる | 束縛を広く覆うほど、再送と XML の包み替えへの攻撃面が実質的に広がる | 対象のクライアントが使わない経路のために攻撃面を広げる |
| `windowstransport` を実装し、テナントの社内の出口 IP だけから受ける | Hybrid Join の端末を、Entra Connect の同期を待たずに登録できる | インターネット上の入口になり、社内向けにだけ置けという Microsoft の要求に従えない。端末の登録のクライアントがインターネット上のホストへ Kerberos の資格情報を自動で送るかも確かめられない | 要求に従えず、sync join という文書化された代替がある |

### 結果と再検討の条件

リッチクライアントのサインインに要る経路だけを、フェイルクローズで検証すれば足りる。
その代わり、WS-Trust の Kerberos による統合認証と、フェデレーションによる Hybrid Join の即時の登録は使えない。
Hybrid Join の端末は、即時の登録に失敗した後、Entra Connect がコンピューターを同期すると登録を終える（Windows 10 1803 以降。[Configure Microsoft Entra hybrid join](https://learn.microsoft.com/en-us/entra/identity/devices/how-to-hybrid-join) の Federation caveats）。
社内に置く入口を IdMagic が持つようになるか、Microsoft が `windowstransport` を社外に置くことを認めた場合に見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-004 | [WS-Trust の能動的 STS](features/active-sts.md) |
| REQ-WSFEDERATION-005 | [WS-Trust の能動的 STS](features/active-sts.md) |

## パッシブの統合 Windows 認証を Authentication へ委ねる

### 背景

WS-Fed の RP は `wauth` で統合 Windows 認証を要求する。
同じ本人確認を idmagic の通常ログインからも使うため、チケットの検証と鍵の管理には一つの所有者が必要である。

### 決定

WS-Fed は RP と返信先を検証した後、Authentication の公開操作で Negotiate を試す。
Authentication が鍵、対応付け、再送拒否、サインインポリシーとセッションを所有し、WS-Fed は認証済みの AMR `wia` を Kerberos の Assertion へ変換する。
チケットがない未認証の端末は通常ログインへ送り、別方式の認証だけで `wauth` を満たすことはない。

### 検討した代替案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| WS-Fed が直接 keytab とチケットを検証する | WS-Fed の内側だけで完結する | 通常ログインと再送拒否、ポリシー、セッションの処理が重複する | 本人確認の所有者をプロトコルごとに増やす |
| プロキシが利用者名をヘッダーで通知する | API に Kerberos の鍵をデプロイせずに済む | プロキシとの認証とヘッダーの信頼契約を別に管理する | 今回の直接 SPNEGO と混在させず、検証したチケットを唯一の入力にする |

### 結果と再検討の条件

既存のパッシブの要求、クレームと署名の経路を維持し、本人確認だけを共通の所有者へ委ねる。
認証の提供方式を変える必要が生じた場合は、[Authentication の判断](../authentication/decisions.md)と合わせて見直す。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-002 | [パッシブサインイン](features/passive-sign-in.md) |
| REQ-AUTHENTICATION-042 | [Kerberos による無音サインイン](../authentication/features/negotiate.md) |

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
Entra はどのドメインでも同じ realm を送るので、定型設定はテナントに一つの RP に複数のドメインを持ち、利用者の UPN のドメインで発行者の IssuerUri を選ぶ。
sourceAnchor の属性を、重ならず本人が書き換えられない属性に限り、設定の時点でフェデレーションするドメインの利用者の欠けた値と使えない値を拒否できる。
その代わり、Entra が求めるクレームが変わったら、定型設定のコードを変える。
Entra 以外にも定型設定が要る RP が増えた場合に、定型設定の一般化を検討する。

### 関連する要件

| 要件 | 機能仕様 |
| --- | --- |
| REQ-WSFEDERATION-001 | [RP と Entra フェデレーションの管理](features/relying-party.md) |

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
| REQ-WSFEDERATION-002 | [パッシブサインイン](features/passive-sign-in.md) |
| REQ-WSFEDERATION-004 | [WS-Trust の能動的 STS](features/active-sts.md) |
