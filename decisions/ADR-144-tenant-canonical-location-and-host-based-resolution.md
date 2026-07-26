---
status: accepted
authors: [tn]
created_at: 2026-07-26
supersedes: [ADR-033]
---

# ADR-144: テナントは正規ロケーションを 1 つだけ持ち、path prefix と subdomain から選ぶ

## コンテキスト

ADR-033 はテナント解決を path prefix (`/realms/{realm}`) に決め、subdomain 戦略を
「将来差し替え可能な slot」として残した。その slot を埋める段になり、テナント固有 origin を
必要とする機能 (`__Host-` cookie、テナント別 WebAuthn RP ID、テナント別ブランド origin) が
path prefix 単独では成立しないことが明確になった。

一方で path prefix を捨てる選択も採れない。ワイルドカード DNS と証明書を必須にすると、
単一ホスト運用とローカル開発のハードルが上がる。Keycloak が path style を採る理由がそのまま残る。

両方式を持たせる設計を検討した際、最初の案は「両経路から到達でき、issuer は片方に固定」だった。
これは OpenID Connect Discovery 1.0 §4.3 と RFC 8414 §3.3 に違反する。両仕様は discovery 文書の
`issuer` が文書の取得元 URL と一致することを要求するため、片方の origin では必ず不一致になる。
加えて issuer 固定側でしかブラウザフローが起きないため、もう一方の origin の cookie 分離も
RP ID も一度も使われない。**問題は両方式の併存ではなく、1 テナントが 2 つの origin を持つこと**
だと判明した。

## 決定

**1 テナント = 1 正規ロケーション = 1 issuer** を不変条件とする。テナントは
`Tenant.endpoint_style` (`Path` | `Subdomain`) が指す正規ロケーションからのみ到達でき、
他方の経路では不在として扱う。issuer / cookie scope / WebAuthn RP ID はすべて正規ロケーション
から導出する。解決順序と fail-closed 規則は `Tenancy.interfaces.ResolveTenant` を正とする。

この不変条件により、両方式は互いに干渉せず併存する。`{base}/realms/acme/.well-known/...` が
`issuer: {base}/realms/acme` を返す構成も、`https://acme.{base}/.well-known/...` が
`issuer: https://acme.{base}` を返す構成も、それぞれ単独で discovery 仕様に適合する。

`Subdomain` は `tenant_base_domain` が設定された配備でのみ選択できる。未設定の配備では全テナントが
`Path` に留まり、ワイルドカード DNS も証明書も要求しない。

`endpoint_style` の変更は通常の属性更新から分離し `SetTenantEndpointStyle` に閉じる。issuer と
WebAuthn RP ID を同時に変えるため、既発行 token の `iss` 検証・全 RP の設定・既存 passkey が
すべて壊れる。Okta も custom domain の issuer 切り替えについて同じ影響 (metadata URL 更新、
アプリ設定更新、passkey 再登録) を明記している。

realm は不変とする。issuer とホスト名の両方に現れる以上、rename は同じ破壊を伴い、
`endpoint_style` 変更と別経路で同じ事故を起こす余地を残す理由がない。

**ADR-033 の §2 (未 prefix ルートを `default` へ解決) と §3 の `LEGACY_BARE_ISSUER` を撤回する。**
prefix 無し経路は `default` テナントの第 2 ロケーションであり不変条件に反する。どちらも
既存 RP を壊さないための互換措置であり、未リリースの現時点では守るべき互換が存在しない。
ADR-033 の §1 (protocol route を `/realms/{realm}` 配下に置く)、§4、§5、§6 は有効なまま残る。

## 却下した代替案

- **path style の全廃 (subdomain 専用)**: 規則は最小になるが、ワイルドカード DNS と証明書を
  全配備に強制し、単一ホスト運用とローカル開発を壊す。Keycloak が path style を採る利点を失う。
- **両経路から到達可能にし issuer は片方に固定**: 上記のとおり discovery 仕様違反になり、
  かつ非 issuer 側の origin が実際には使われないため、cookie 分離も RP ID も機能しない。
- **Okta 型の dynamic issuer (到達した origin をそのまま issuer にする)**: 同一テナントが
  複数 issuer を持ち、片方で発行した token がもう片方で検証できない。Okta 自身も multibrand
  構成に限定して推奨している。
- **ベースドメインを Public Suffix List に登録して兄弟サブドメイン間の cookie 汚染を防ぐ**:
  外部申請と反映待ちに依存し、自前ホスト運用者には適用できない。`__Host-` prefix が同じ脅威を
  ブラウザ側の仕組みだけで塞ぐため、外部依存を持ち込む理由がない。
- **realm の rename を許し alias 期間を設ける**: alias 期間中はテナントが 2 つのロケーションを
  持ち、不変条件が崩れる。rename の運用価値より不変条件の単純さを採る。

## 影響

- SCL: `Tenancy` に `models.TenantEndpointStyle` と `Tenant.endpoint_style` を追加。
  `interfaces.ResolveTenant` の入力に Host を追加し解決順序を差し替え。
  `interfaces.SetTenantEndpointStyle` を追加。`Tenant.realm` を不変とし DNS ラベル準拠に厳格化。
- データ: `tenants` に `endpoint_style` 列 (既定 `path`) を追加。
- 運用: `TENANT_BASE_DOMAIN` を追加。設定した配備は `*.{base}` のワイルドカード DNS と
  証明書を ingress 側で用意する (本 ADR は証明書の発行・更新を扱わない)。
  `LEGACY_BARE_ISSUER` を削除。`WEBAUTHN_RP_ID` は `Path` テナントの既定値に降格。
- 既存 realm の再検証は行わない。厳格化した realm 制約は新規作成にのみ適用する。
- 顧客所有のカスタムドメインは同じ不変条件の上に `endpoint_style` の第 3 の値として載る
  ([[wi-299-tenant-custom-domain]])。
