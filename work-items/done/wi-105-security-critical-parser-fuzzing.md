---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# SAML XML・JWT/JWE・redirect_uri・PKCE 等のセキュリティクリティカルなパーサに Go native fuzzing を導入する

攻撃者制御の入力を受け取る 39 個のパース・照合境界に fuzz target を追加した（既存 7 個と合わせて 46 個）。対象は元の記述をなぞらず、`etree` / `encoding/xml`、`base64`、`url.Parse`、`Parse*` / `Decode*` / `Verify*` / `Normalize*` の全出現を洗い出し、到達可能性と帰結の 2 軸で絞って決めた。`redirect_uri` の照合は `RedirectURIAllowed` として切り出し、`authorize.go` と `end_session.go` の 2 つの手書き実装を置き換えた。上流 IdP の ID Token 検証は `verifyUpstreamIDToken` として HTTP から切り離した。CI は変更していない。探索は `mise run test-go-fuzz-all` で手元から回す。
作業中に本番の欠陥を 4 件見つけて直した。サインアウトがサーバー側セッションを失効させない（REQ-AUTHENTICATION-035 を追加）、コスト未検査でパスワード検証がパニックまたは OOM する、salt と digest が空だと鍵長 0 で segfault する、整合トークンが非正規な符号化を受理して可鍛になる、の 4 件である。後ろ 3 件は fuzz が自力で見つけた。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
