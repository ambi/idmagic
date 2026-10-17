---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-35767-publish-claim-issuance-as-claimmapping-public-operations]
change_kind: refactor
spec_impact: { kind: none, reason: "ID トークンとアクセストークンを組み立てて署名する Go のパッケージの置き場所だけを変える。発行するトークンの内容と署名、HTTP の応答、ドメインイベントは変えない。" }
---

# JWT の署名器を共有ライブラリから OAuth2 へ移す

## 動機

`tools/check/boundary-debt.json` の `shared-dependency` のうち 5 件は、`backend/shared/security/tokens_jose` から ClaimMapping、IdManagement、OAuth2、SigningKeys、Tenancy への依存である。
`tokens_jose` は OAuth2 の ID トークンとアクセストークンを組み立てて署名する。
OAuth2 の型とポート、ClaimMapping の発行、ユーザーの型を使うので、共有ライブラリの条件（D5）を満たさない。

## 対象範囲

- `tokens_jose` のうち OAuth2 のトークンを組み立てる部分を OAuth2 のパッケージへ移す。
- モジュールの型を使わない JOSE の署名の処理が残るなら、それだけを共有ライブラリに残す。
- 解消した違反 ID を台帳から消す。

## 対象外

- トークンの内容、署名の方式、鍵の選び方の変更。
- JWKS 取得器とリプレイ検証器に残る OAuth2 への依存。[wi-18088](wi-18088-remove-oauth2-types-from-shared-jose-verifiers.md) で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D5：モジュールの型と業務上の意味を必要とする処理は共有ライブラリに置かない。
- D8：トークンの内容の変更（クレームの追加、`act` の扱い）は OAuth2 の要件から生じ、現在は共有ライブラリを変えている。OAuth2 へ移せば、その変更が OAuth2 で閉じる。

ClaimMapping への依存は、[ClaimMapping の発行規則の公開](../done/wi-35767-publish-claim-issuance-as-claimmapping-public-operations.md)で公開された操作へ置き換える。

### 境界の比較

入力は OAuth2 の責務、`jwt_signer.go`、SharedSignals の SET 署名器である。
JWTSigner は OAuth2 のポートを実装し、クレームの構築と署名鍵の選択を隠す。
永続状態の更新とイベント発行はなく、KeyStore の読み取りと署名だけを行う。
D1、D3、D5、D6、D8 を適用する。

| 案 | 制約と変更シナリオ | 判断 |
| --- | --- | --- |
| 現状の共有配置 | OAuth2 の型、ClaimMapping、User、SigningKeys、Tenancy を知り、D5 に反する。クレーム変更が共有ライブラリへ波及する | 不採用 |
| 全体を OAuth2 へ移す | SET の署名も OAuth2 を呼ぶことになり、技術的な署名をプロトコルへ結び付ける | 不採用 |
| OAuth2 の内部へ JWTSigner を移し、PS256 の署名関数だけ共有に残す | 共有関数は秘密鍵、kid、ヘッダー、クレームだけを受け取る。鍵の所有型への変換は各利用側が行う。OAuth2 のクレーム変更は内部で閉じ、SET の意味は SharedSignals が保つ | 採用 |

組み立て地点には OAuth2 ルートの `NewJWTSigner(issuer string, ks signingports.KeyStore) *JWTSigner` を公開する。
内部の署名器と既存テストは同じパッケージ名のまま移し、期待値とテスト本体を保つ。
`SignPS256` の内部ラッパーは SigningKey を技術的な値へ変換するために残す。
追加する業務規則、公開 HTTP 契約、原子性の変更はない。

## 計画

1. 署名器とそのテストを OAuth2 内部へ移し、純粋な PS256 署名を共有に残す。
2. 組み立て地点とテストの import を追従し、解消した共有依存だけを台帳から除く。
3. 現在の設計と生成する依存図を同期する。

## タスク

- [x] T001 [Design] OAuth2 へ移す部分と共有ライブラリに残す部分を分ける。
- [x] T002 [App] 移して利用側を書き換える。
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run test-go-package -- ./backend/oauth2/internal/tokens_jose`：移した既存の署名とクレームのテスト。
- `mise run lint-go`：import と型の追従。
- `mise run test-go-race`：共有コードと組み立て地点が影響する全操作の既存テスト。契約証拠の事前照合で未解析282操作に参照不足はない。
- `mise run check`、`mise run render-docs`：配置、台帳、仕様影響、現在の設計の整合。
- 差分の行を対象にする `mise run test-go-mutation`：移動に伴う鍵から技術的な値への変換を調べる。既存のトークン規則自体は変更しない。

## リスク

- 署名の鍵の選び方を移動で変えると、発行したトークンを検証できなくなる。
  移す前に、署名済みのトークンのヘッダーとクレームを特性化テストで固定する。

## 要約

規範仕様の差分はない。
JWTSigner と既存のテストを OAuth2 内部へ移し、起動時の構築をルートで公開した。
PS256 署名は秘密鍵と kid を受け取る共有関数に残し、SharedSignals と OAuth2 で鍵の型から変換する。
共有依存は10件から6件へ減った。
JWKS とリプレイ検証に残る OAuth2 依存は wi-18088 へ分けた。
共有関数の変更行の mutation は実行可能な3変異を検出し、文字列演算を不正にする4変異はコンパイル不能だった。
移動したトークン規則そのものの全変異は対象外とし、鍵 ID の変換漏れを既存の HTTP テストで検出する。
