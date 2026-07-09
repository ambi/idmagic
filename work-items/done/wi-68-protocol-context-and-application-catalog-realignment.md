---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-27
---

# Protocol bounded context と Application Catalog の境界を整理する

## Motivation
`Federation` bounded context という名前は広すぎる。OAuth2/OIDC も SAML も federation
であり、WS-Federation / WS-Trust だけを `Federation` と呼ぶと、bounded context 名が
実際の責務を表さない。また管理 UI の「アプリケーション」は現在 OAuth2/OIDC client だけを
指しており、WS-Fed RP や将来の SAML SP と整合しない。

本 WI は ADR-064 に従い、protocol context と運用者向け Application Catalog の語彙を分離する。
短期的には `Federation` を `WsFederation` に改名し、OAuth2/OIDC client 画面を正確に表示する。
中期的には OIDC client / SAML SP / WS-Fed RP を束ねる ApplicationCatalog を導入できるよう、
仕様・ADR・UI・実装ディレクトリの境界を揃える。

## Scope
- **decision**:
  - ADR-064 を追加し、`Federation` bounded context 名を廃止する方針を確定する。
  - ADR-059 の bounded context 名に関する決定を ADR-064 で置き換え、claim mapping の決定だけを維持する。
- **scl**:
  - bounded_contexts の `Federation` を `WsFederation` に改名する。
  - WS-Fed / WS-Trust の models / events / interfaces の annotations を `WsFederation` に揃える。
  - ApplicationCatalog の vocabulary / model / future interface を追加するか、次 WI の initial context として明示する。
- **go**:
  - `internal/federation` を `internal/wsfederation` に移し、imports / package comments / aliases を揃える。
  - protocol-neutral な claim mapping / SAML assertion signing の最終配置を記録し、SAML 実装時に重複させない。
- **ui**:
  - 現在の `/admin/clients` が OAuth2/OIDC client 管理であることを明示する。
  - 将来の統合「アプリケーション」画面では OIDC / SAML / WS-Fed を protocol binding として束ねる方針を SCL / ADR に反映する。
- **documentation**:
  - README と wi-29 / wi-61 / wi-62 / wi-63 の完了記録に残る `Federation` bounded context 表現を更新する。

## Out of Scope
- ApplicationCatalog の完全実装 (共通 application aggregate、割当、所有者、ライフサイクル、共通監査)。
- SAML 2.0 IdP 本体 ([[wi-29-saml2-idp]])。
- Entra 実テナント検証 ([[wi-64-entra-domain-federation-m365-sso]])。
- WS-Fed / WS-Trust の wire behavior 変更。

## Verification
- `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui build`
- `bun --cwd idmagic/ui lint`
- `bun run yaml-check:work-items` (in: tools)
- `bun run yaml-check:scl` (in: tools)

## Risk Notes
主なリスクは実装挙動ではなく、仕様・ADR・パッケージ名・UI ラベルの不整合である。
単純な rename でも import path や生成 HTML、ワークアイテム traceability が広範囲に変わる。
wire behavior を変えず、検証は既存 WS-Fed / WS-Trust / OAuth2 テストが green のままかで確認する。

## Completion
- **Completed At**: 2026-06-27
- **Summary**:
  SCL / ADR / 実装ディレクトリ / 管理 UI の protocol 境界を ADR-064 に合わせて整理した。
  `Federation` bounded context 名を廃止し、WS-Federation passive と WS-Trust active STS は
  `WsFederation` に集約した。claim mapping / NameID / attribute release は `ClaimRelease`、
  key material lifecycle / JWKS 管理は `SigningKeys`、運用者向け上位概念は将来の
  `ApplicationCatalog` として SCL に反映した。
- **Verification Results**:
  - `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
    - result: ok
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok: 0 issues (run outside sandbox after package loading failed inside sandbox)
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui build`
    - result: ok
  - `bun --cwd idmagic/ui lint`
    - result: ok
  - `bun run yaml-check:work-items` (in: tools)
    - result: ok
  - `bun run yaml-check:scl` (in: tools)
    - result: ok
- **Affected Guarantees State**:
  - traceability: SCL / ADR / implementation / UI の protocol 境界名を `WsFederation` / `ClaimRelease` / `SigningKeys` / `ApplicationCatalog` に整理した
  - protocol isolation: WS-Fed / WS-Trust は `WsFederation`、OAuth2/OIDC は `OAuth2`、将来 SAML は `Saml` に分ける方針を SCL に反映した
  - product language clarity: `/admin/clients` は OAuth2/OIDC client 管理として表示し、統合 ApplicationCatalog まで「アプリケーション」表記を避けた
  - shared assertion reuse: claim release と signing key lifecycle を SCL 上の共有 capability として明示した
