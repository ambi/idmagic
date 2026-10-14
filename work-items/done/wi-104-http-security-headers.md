---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ログイン・同意・ポータルへ CSP / HSTS / frame-ancestors 等のセキュリティヘッダを一元適用する

System context に SecurityResponseHeaders / FrameAncestorsPolicy objective を追加し、
派生 HTML/JSON/OpenAPI を再生成した。設計判断は ADR-076 に明文化した (ヘッダ集合と値、
CSP 方式、frame-ancestors 'none'、app vs edge のヘッダ分担、HSTS の TLS 終端委譲、
report-only 段階導入、SAML/WS-Fed の form-action 例外)。
Go 側は support パッケージに SecurityHeadersMiddleware を新設し、bootstrap で
request_id / recover の内側・body limit 付近に登録した。全 backend レスポンスへ
nosniff / no-referrer / X-Frame-Options: DENY と、strict CSP
(default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self') を
一元付与する。HSTS は TLS 終端層所有として既定無効 (開発 http で抑制)、HSTS_ENABLED で
opt-in。CSP は CSP_REPORT_ONLY / CSP_REPORT_URI で report-only 段階導入に対応。
**nonce は採用せず hash 方式**にした (レビュー指摘)。Go がインライン script を返すのは
SAML ACS / WS-Fed 自動 POST の固定 submit script (`document.forms[0].submit()`) のみで、
SPA は gateway 配信で `script-src 'self'` (インライン無し) のため nonce の渡し先が無い。
固定スクリプトを `support.AutoSubmitScript` に一元定義し、その sha256 を script-src へ
pin。自動 POST の `onload` 属性を nonce/hash 対象外のため固定 `<script>` へ移し、
各ハンドラは `support.SetAutoPostFormCSP` で当該レスポンスの CSP に送信先 origin
(form-action) と script hash を許可する。gateway (ui/Caddyfile) は SPA の CSP を所有し、
backend proxied レスポンスには再付与しない旨をコメントで明文化した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
