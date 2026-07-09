---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-076: Apply HTTP security response headers at the app boundary

## コンテキスト
idmagic の Go 境界サーバには、これまでセキュリティレスポンスヘッダ（Content-Security-Policy、Strict-Transport-Security、X-Frame-Options / frame-ancestors、Referrer-Policy、X-Content-Type-Options）を付与する middleware が無かった。SPA は gateway（Caddy が参照実装）が同一オリジンへ統合して配信し、その静的 HTML には既に強い CSP（`script-src 'self'`）や frame-ancestors 'none' が付与されている。一方で Go backend が返すレスポンス（`/authorize`、`/token` などの API、および SAML ACS / WS-Fed への自動 POST フォーム HTML）は `reverse_proxy` を素通りし、CSP も frame-ancestors も nosniff も付いていなかった。

特に SAML / WS-Fed の自動 POST フォームは、`<body onload="document.forms[0].submit()">` というインライン event handler を持つ Go レンダリングの HTML であり、consent 直後にブラウザで自動送信される。これがフレーム埋め込み可能かつ CSP 不在のままだと、consent/login のクリックジャッキング（同意の乗っ取り）や XSS 経由の資格情報窃取の面が残る。本番 IdP では譲れない基本防御であり、OWASP ASVS も Keycloak のログインテーマ既定もこれらを要求する。

## 決定

`spec/contexts/system.yaml` の `objectives.SecurityResponseHeaders` / `objectives.FrameAncestorsPolicy`、`internal/shared/adapters/http/support`（securityheaders middleware）、SAML / WS-Fed の自動 POST フォームアダプタ、`ui/Caddyfile`（gateway 側の SPA ヘッダ）に反映。

1. Go 境界に securityheaders middleware を追加し、backend が返す**全レスポンス**に secure-by-default でヘッダを一元付与する。既定値は本番安全側に倒し、env で上書き・切替できる。
2. 付与するヘッダと既定値:
   - `X-Content-Type-Options: nosniff`（常時）
   - `Referrer-Policy: no-referrer`（常時）
   - `X-Frame-Options: DENY` と CSP `frame-ancestors 'none'`（常時、FrameAncestorsPolicy）
   - `Content-Security-Policy: default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'nonce-<per-request>'`
   - `Strict-Transport-Security`（条件付き、後述）
3. **CSP は `'unsafe-inline'` に依存せず、インライン script は CSP hash で許可する**。base CSP は script-src を持たず（default-src 'none' が全 script を拒否）、Go がインライン script を返す唯一の箇所は SAML / WS-Fed の自動 POST フォームである。その submit script は `document.forms[0].submit()` という**固定の一行**なので、`onload` 属性ではなく固定内容の `<script>` へ移し、その sha256 を CSP `script-src 'sha256-…'` に pin して当該レスポンスにのみ許可する。属性の inline event handler は hash / nonce いずれの対象でもないため、`<script>` 要素へ寄せる。**per-request nonce は採用しない**（後述の却下案）。
4. **frame-ancestors / CSP はアプリ所有必須**。frame-ancestors の per-route 判断（IdP 画面は 'none'）と、レスポンスごとに form-action へ送信先を許可する判断は、ページをレンダリングするアプリにしかできずプロキシへ委譲できない。よってこれらは前段委譲を原理的に選べず、アプリ単体で secure-by-default を成立させる。
5. **自動 POST フォームの form-action 例外**。SAML ACS / WS-Fed の自動 POST は cross-origin へ form 送信するため、default-src 'none' のままでは送信がブロックされる。当該レスポンスに限り CSP `form-action` に送信先 URL（ACS / ReplyURL）の origin のみを明示的に許可し、併せて固定 submit script の hash を script-src に許可する。frame-ancestors は 'none' を維持する。
6. **HSTS はエッジ委譲を許容**。TLS を終端する層が所有すべきヘッダであり、アプリが出すのは「TLS 終端が前段にある」前提が要る。既定（開発 http）では抑制し、`HSTS_ENABLED=true` で明示的にオプトインしたときだけ `max-age=31536000; includeSubDomains` を付与する。TLS 終端層（gateway / LB）が HSTS を所有する構成では、アプリ側は無効のままでよい。
7. **段階導入**。`CSP_REPORT_ONLY=true` で `Content-Security-Policy-Report-Only` に切替え、`CSP_REPORT_URI` で違反レポート収集先を指定できる。厳格な CSP がインライン script/style を壊すリスクに対し、report-only で違反を洗い出してから enforce へ切り替える運用を可能にする。
8. **SPA のヘッダは gateway が所有**。gateway が配信する SPA 静的 HTML は Vite ビルドが外部 module script（`script-src 'self'`）のみでインライン script を持たないため nonce も hash も不要で、既存の Caddyfile が CSP / frame-ancestors を付与する。Go の securityheaders は backend レスポンス（gateway を素通りする経路）を担う。この分担を Caddyfile と README に明文化する。

## 却下した代替案
- **全ヘッダを前段プロキシで付与する**: CSP は per-request nonce をヘッダと HTML の両方に一致注入する必要があり、プロキシで HTML を書き換えるのは非現実的。frame-ancestors の per-route 判断もアプリの知識。OSS 配布物として、プロキシ無し / 低機能プロキシ背後でも consent 画面はクリックジャッキング保護されねばならない。よって CSP / frame-ancestors のアプリ所有は原理的必然。
- **CSP に `'unsafe-inline'` を許可して自動 POST の onload をそのまま通す**: XSS の主要防御である CSP を無力化する。hash で許可すれば `'unsafe-inline'` 無しで同じフローが通る。
- **per-request nonce ベースの CSP を採用する**: nonce は per-request の乱数生成、context 伝播、ヘッダと HTML の `<script nonce>` の一致注入という plumbing を要する。本サービスで Go がインライン script を返すのは固定内容の自動 POST submit script のみで、SPA は gateway 配信で `script-src 'self'`（インライン無し）のため nonce を渡す相手がいない。固定スクリプトには静的 hash が nonce と同等に厳格（`'unsafe-inline'` 不要）で、per-request 状態を持たない分だけ単純。よって nonce ではなく hash を採る。
- **HSTS を常時付与する**: TLS 終端前提を暗黙化し、開発 http や TLS 未終端構成で誤ったヘッダを出す。TLS を終端する層が所有すべきという原則に反する。

## 影響
- SCL の System context に `SecurityResponseHeaders` / `FrameAncestorsPolicy` の security objective を追加する。
- HTTP support layer に securityheaders middleware（CSP builder・固定 submit script の hash pin・env 設定）を追加し、bootstrap の最外付近（request_id / recover の内側）で登録する。
- SAML `EncodePostForm` / WS-Fed `RenderPassiveForm` は `onload` を固定内容の `<script>` へ移す（内容は `support.AutoSubmitScript`）。各ハンドラは当該レスポンスの CSP に送信先 form-action と submit script hash を加える（`support.SetAutoPostFormCSP`）。
- README に TLS 終端・HSTS・CSP report エンドポイントの設定と、gateway / app のヘッダ分担を追記する。
