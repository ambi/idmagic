---
id: idp-wi-110-http-server-hardening-timeouts-and-body-limits
title: "HTTP サーバに read/write/idle タイムアウトとボディ上限を設定しリソース枯渇 DoS を緩和する"
created_at: 2026-07-04
authors: ["tn"]
status: pending
risk: low
---
# Motivation
internal/bootstrap/server.go の起動は echo.StartConfig{Address: addr} のみで、
基盤の http.Server に ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout が
一切設定されていない。これは gosec G112 (CWE-400) が警告する Slowloris 攻撃・
接続枯渇のリスクに直結し、悪意あるクライアントが低速にヘッダ/ボディを送り続ける
だけでコネクションを占有できる。加えてリクエストボディの上限が無く、巨大な
JSON/フォーム送信でメモリを消費させられる。

IdP はインターネットに露出する認証エンドポイントを持つため、Go / Kubernetes の
本番プラクティスに従い、境界サーバとしての基本的なタイムアウトとボディ上限を
設定し、単純なリソース枯渇 DoS に対する下限の耐性を確保する必要がある。

# Responsibility (App vs Edge Proxy)
「タイムアウト/ボディ上限はアプリと前段プロキシ (Envoy / Nginx / Caddy / HAProxy) の
どちらが担うか」は、必要な知識を誰が持つかで切り分ける。

- **エッジが主防御**: TLS ハンドシェイク slowloris、volumetric DDoS、グローバルな接続数/
  帯域制限、粗いボディ上限。トランザクションの中身を知らずにエッジで安く止められ、
  トラフィック総量もプロキシしか観測できない。適切に設定されたプロキシ背後では、
  クライアント側 slowloris の大半はプロキシが吸収する (プロキシ↔アプリは keepalive 常設)。
- **アプリが担う理由 (本 WI が in scope とする根拠)**:
  1. Go の `http.Server` はタイムアウト未設定＝無制限がデフォルトで、`gosec G112 (CWE-400)`
     が確実に警告する既知の地雷。約10行のコストで済む多層防御の保険。
  2. idmagic は OSS 配布物でトポロジを選べない。`dev.sh` は素で `:8080` を公開し、
     プロキシ無し / 低機能プロキシ背後でも動く以上、secure-by-default はアプリ単体で
     成立しなければならない (README の `X-Request-ID` 思想と同じ)。プロキシは前提でなく
     上に重ねる強化層。
  3. プロキシ↔アプリは別ホップで、サイドカー / port-forward / SSRF / 別 Pod からは
     プロキシを迂回できる。自分のホップはアプリしか守れない。
  4. ボディ上限の妥当値はエンドポイント意味論に依存し (token/authorize/admin で異なる)、
     アプリしか正しい値を知らない。ここは本質的にアプリ担当。
- **結論**: アプリ側は「安全な保守的デフォルト＋env 上書き」を持つ。volumetric/ハンドシェイク
  DoS の主防御はエッジという役割分担を README (と、必要なら ADR) に明記する。
  「アプリがプロキシの仕事を奪う」のではなく「自分のホップを守り、プロキシ非依存にする」。

# Scope
- **scl**: spec/contexts/system.yaml objectives に HTTPServerHardening を追加し、read_header_timeout / read_timeout / write_timeout / idle_timeout / max_body_bytes の目標値を宣言する。
- **go**: echo.StartConfig または基盤 http.Server に ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout を設定する。, env (例: HTTP_READ_TIMEOUT 等) で上書き可能にしつつ、本番安全なデフォルトを与える。, リクエストボディ上限ミドルウェア (echo BodyLimit) を導入し、認可・トークン・admin API に適用する。

# Out of Scope
- アプリケーション層のレート制限 / bot 緩和 (既存 idp-wi-27 の範囲)。
- L7 リバースプロキシ (Caddy) 側のタイムアウト調整。
- 接続ドレイン / graceful shutdown の詳細 (既存 idp-wi-98 の範囲)。

# Verification
- just verify
- 手動: slowloris 相当の低速接続や上限超過ボディがタイムアウト/413 で拒否されることを確認する。

# Risk Notes
タイムアウトを過度に短く設定すると device flow polling や大きな admin 応答を
誤って切る恐れがあるため、write/idle は余裕を持たせ env で調整可能にする。
