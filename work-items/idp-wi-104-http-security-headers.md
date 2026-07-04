---
id: idp-wi-104-http-security-headers
title: "ログイン・同意・ポータルへ CSP / HSTS / frame-ancestors 等のセキュリティヘッダを一元適用する"
created_at: 2026-07-04
authors: ["tn"]
status: pending
risk: medium
---
# Motivation
現状 Go 側にセキュリティレスポンスヘッダ（Content-Security-Policy、
Strict-Transport-Security、X-Frame-Options / frame-ancestors、
Referrer-Policy、X-Content-Type-Options 等）を付与する middleware が見当たらない。
IdP のログイン・同意（consent）・アカウントポータルは資格情報と認可判断を
扱う最も攻撃価値の高い画面で、CSP 不在は XSS 経由の資格情報窃取、
frame-ancestors 不在は consent/login のクリックジャッキング（同意の乗っ取り）に
直結する。これは本番 IdP では譲れない基本防御である。

Keycloak はログインテーマに既定で frame-options / CSP / HSTS 等を出し、
OWASP ASVS もこれらを要求する。idmagic も UI が別プロセスでも gateway で
同一オリジンに統合される構成（ARCHITECTURE.md）なので、認証系レスポンスに
一元的にヘッダを適用し、CSP は nonce ベースで UI ビルドと整合させるべきである。

# Responsibility (App vs Edge Proxy)
「セキュリティヘッダを前段プロキシで付与する」選択肢もあるが、ヘッダごとに担当が分かれ、
特に CSP はアプリ専有が原理的必然となる。必要な知識を誰が持つかで切り分ける。

- **CSP (nonce ベース) → 実質アプリ専有**: per-request nonce をレスポンスヘッダと HTML 内の
  `<script nonce=…>` の両方に一致させて注入する必要があり、ページをレンダリングするアプリ
  (＋UI ビルド) にしかできない。プロキシで一致させるには HTML 書き換えが要り非現実的。
  よって「前段に委譲」が原理的に選べない決定打。
- **frame-ancestors / X-Frame-Options → アプリ寄り**: 「login/consent/portal は `'none'`」という
  per-route の機微判断はアプリの知識。プロキシで一律付与すると埋め込みが要る画面と競合する。
- **HSTS → エッジ委譲を許容**: TLS を終端する層が所有すべきヘッダ。アプリが出すなら
  「TLS 終端が前段にある」前提が要る。HSTS だけは前段委譲を明示的に許してよい
  (本 WI は「TLS 終端前提を明示、開発 http では抑制」を維持)。
- **X-Content-Type-Options / Referrer-Policy → どちらでも可 (安価)**: 多層で出して困らない。
- **OSS 前提の補強**: idmagic は不特定運用者が動かす配布物で、プロキシ無し / 低機能プロキシ
  背後でも consent 画面はクリックジャッキング保護されねばならない。secure-by-default は
  アプリ単体で成立させる。
- **結論**: CSP と frame-ancestors はアプリ所有必須 (プロキシ委譲不可)、HSTS はエッジ委譲を
  許容、粗いヘッダは多層。この分担を新規 ADR に明文化する。

# Scope
- **decision**: 新規 ADR: 適用するヘッダ集合と各値、CSP の方式（nonce か hash か）、 IdP 画面は frame-ancestors 'none'（クリックジャッキング防止）とする方針、 OAuth/OIDC のリダイレクト・POST バインディング（SAML ACS 等）と矛盾しない範囲を定義する。
- **scl**: System context に SecurityResponseHeaders / FrameAncestorsPolicy の objective を追加する。
- **go**: セキュリティヘッダ middleware を追加し、認証系・ポータル・consent レスポンスへ一元適用する。 HSTS は TLS 終端前提を明示し、開発（http）では抑制できるようにする。, CSP を nonce ベースにし、per-request nonce を UI へ受け渡す。unsafe-inline に依存しない。 SAML/WS-Fed の自動 POST フォーム等インライン script が要る箇所は nonce/hash で許可する。, report-only モードと report 収集の切替を用意し、段階導入できるようにする。
- **ui**: Bun ビルドの生成物が nonce ベース CSP と両立するよう、インライン script/style の扱いを整える。, CSP 違反で画面が壊れないことを e2e で担保する。
- **documentation**: README に TLS 終端・HSTS・CSP report エンドポイントの設定を書く。

# Out of Scope
- WAF / CDN 側のヘッダ注入。
- Subresource Integrity（SRI）による外部 CDN 資産の固定（外部資産を持たない前提）。
- CORS ポリシーの再設計（必要なら別 WI）。

# Verification
- [object Object]
- [object Object]
- [object Object]
- 手動: login / consent / account portal のレスポンスに CSP・HSTS・frame-ancestors 'none' が 付き、iframe 埋め込みが拒否されることを確認する。
- 手動: authorization_code フローと SAML POST バインディングが CSP enforce 下で通ることを確認する。

# Risk Notes
厳格な CSP はインライン script/style を壊しやすく、特に SAML/WS-Fed の自動 POST や
UI ビルド生成物で事故りやすい。report-only で違反を洗い出してから enforce に切り替え、
正規プロトコルフローの回帰を e2e で先に固定する。
