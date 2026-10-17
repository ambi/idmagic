---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-35767-publish-claim-issuance-as-claimmapping-public-operations]
change_kind: refactor
spec_impact: { kind: none, reason: "SAML のアサーションを組み立てて署名する Go のパッケージの置き場所と、それを使う側の依存だけを変える。発行するアサーション、署名、HTTP の応答、ドメインイベントは変えない。" }
---

# SAML のアサーションの生成を WsFederation の非公開パッケージから出す

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 4 件は、Saml と Tenancy から `backend/wsfederation/tokens_saml` への依存である。
`tokens_saml` は SAML 1.1 と SAML 2.0 のアサーションを組み立てて署名し、WS-Federation と SAML の両方のプロトコルが使う。
Tenancy の handler は、依存の束に署名器の型（`SignerProvider`）を持っているだけである。

`tokens_saml` は WsFederation の型、ClaimMapping の発行結果、SigningKeys の型を使うので、そのままでは共有ライブラリの条件（D5）を満たさない。

## 対象範囲

- アサーションの生成の置き場所を決めて移し、Saml と WsFederation の呼び出しを書き換える。
- Tenancy の handler の依存の束から署名器の型を外すか、公開された型に置き換える。
- 解消した違反 ID を台帳から消す。

## 対象外

- アサーションの内容と署名の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

| 案 | 判断の材料 |
| --- | --- |
| WsFederation の公開パッケージにする | 移動が最小。Saml から WsFederation への辺は残る |
| プロトコルに依存しない組み立てと署名を共有ライブラリへ分け、WS-Federation と SAML の固有の部分を各モジュールへ残す | D5 を満たす形に分けられれば、Saml と Tenancy から WsFederation への辺が消える |
| SAML のアサーションを担う新しいモジュールを作る | D8。独立して変えたい変更シナリオがあるかを確かめてから比べる |

着手時に `tokens_saml` が WsFederation の型を使う箇所を数え、共有ライブラリへ分けられるかを判断する。
ClaimMapping の発行結果の型は、[ClaimMapping の発行規則の公開](../done/wi-35767-publish-claim-issuance-as-claimmapping-public-operations.md)の後の公開された型を使う。

### 境界の判断の記録

着手時の `tokens_saml` は、WsFederation の `AuthnMethodClass` を 2 か所、ClaimMapping の `ClaimIssuanceResult` を入力の 1 フィールド、SigningKeys の `KeyStore` を署名器の解決で使う。
WsFederation の型はパスワード認証かどうかの分岐にだけ使い、発行結果は NameID と属性の名前と値にだけ使う。
`KeyStoreSignerProvider` を作るのは組み立て地点（`cmd/idmagic` と `testing_stack`）だけである。

| 部分 | 案 | 採否と理由 |
| --- | --- | --- |
| アサーションの組み立てと XML 署名 | 共有ライブラリ `backend/shared/security/tokens_saml` へ移し、入力を SAML の語彙（認証コンテキスト、NameID、属性）にする | 採用。SAML のアサーションの符号化と署名は技術的な規則で、モジュールの型も業務上の意味も要らない（D5）。認証方式と発行結果から SAML の語彙への変換は利用側が持つ |
| 同 | WsFederation の公開パッケージにする | 不採用。Saml と Tenancy から WsFederation への辺が残る |
| 同 | SAML のアサーションを担う新しいモジュール | 不採用。所有する状態も独立して変えたい変更シナリオもない（D8） |
| 鍵の保存先から署名器を作る | SigningKeys のルートの操作 `FederationSigner` にし、実装を SigningKeys の `internal/` に置く | 採用。XML 連携署名の鍵の用途を所有するのは SigningKeys で、作るのは組み立て地点だけである（D6） |
| 同 | 共有ライブラリへ入れる | 不採用。SigningKeys の型を使うので D5 を満たさない |
| 発行結果から SAML の属性への変換 | Saml と WsFederation にそれぞれ置く | 採用。変換の意味は利用側が持つ（D6）。数行の重複は受け入れる |

Tenancy の handler の依存の束は、共有ライブラリの `SignerProvider` を持つ。
署名のスコープを付ける `WithSignerScope` は SigningKeys の公開ポートの `WithKeyScope` を包むだけなので、Saml は公開ポートを直接使う。

出力が変わらないことは、移す前に未署名のアサーションの XML を SAML 2.0 と SAML 1.1 で文字列として固定するテストを足し、移した後も同じ期待値で通すことで確かめる。
公開状態は `unpublished` なので、`TestCharacterize*` の特性化テストは求められない。

## タスク

- [x] T001 [Design] 案を比較して置き場所を決める。
  比較は設計の境界の判断の記録にある。
- [x] T002 [App] 生成を移し、利用側を書き換える。
  移す前に未署名のアサーションの XML を `testdata` へ固定し、移した後も同じファイルで通した。
  利用側へ移した写像（認証方式から認証コンテキスト、発行した claim から属性）は、それぞれのモジュールでテストを足した。
  写像を誤らせる注入を、足したテストが検出することを確かめた。
  鍵のスコープを SigningKeys の公開ポートで付けると Saml から SigningKeys への循環が増えたため、スコープは共有ライブラリの `WithSignerScope` で文脈に載せ、SigningKeys の解決器が鍵のスコープへ写す。
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
  `private-import` の 2 件と、WsFederation を経由していた `module-cycle` の 3 件を消した。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 署名の対象の正規化を移動で変えると、既存の SP がアサーションを拒否する。
  移す前に、署名済みのアサーションの出力を特性化テストで固定する。

## 要約

`mise run spec-diff` は、作業範囲の起点に対して規範の変更なしと報告した。
SAML のアサーションの組み立てと XML 署名を、WsFederation の非公開パッケージから共有ライブラリ `backend/shared/security/tokens_saml` へ移した。
入力は SAML の語彙（認証コンテキスト、NameID、属性）にし、発行結果と認証方式からの写像は Saml と WsFederation がそれぞれ持つ。
XML 連携署名の鍵から署名器を作る処理は SigningKeys のルートの `FederationSigner` にし、組み立て地点が各モジュールへ渡す。
Saml と Tenancy から WsFederation への `private-import` 2 件と、それに伴う `module-cycle` 3 件が台帳から消えた。
ハンドラーが写像の結果をアサーションの入力へ渡すことを確かめる段のテストは、移す前からなく、今回も足していない。
