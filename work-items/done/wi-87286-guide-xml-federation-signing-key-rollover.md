---
status: completed
---

# XML 署名鍵の準備から Entra への証明書登録と切り替えまでを管理画面で案内する

REQ-SIGNINGKEYS-006 に、共有 XML 鍵の管理画面の要件を加えた。画面は鍵ごとの状態、フィンガープリント、旧鍵の公開の期限と、同じ鍵を信頼する Entra のドメイン、WS-Federation の RP、デフォルトの IdP プロファイルの SAML の SP を示す。
切り替えは、すべての Entra のドメインの照合が準備済みの証明書を `NextSigningCertificate` として確かめ、ほかの信頼先と全体を管理者が確認するまで無効にし、確定のときに読み直して対象が変わっていれば要求しない。確認は保存せず、画面を開くたびにやり直す。
REQ-WSFEDERATION-001 に、Entra の一覧と詳細から共有 XML 鍵の画面への入口を加えた。
`AdminKeyResponse` に、REQ-SIGNINGKEYS-001 が既に定めていたのに返していなかった `certificate_fingerprint_sha256` と `expires_at` を追加した。
旧鍵での署名へ戻す API は作らず、切り替えの前の取り消しと、切り替えの後に信頼先へ新しい証明書を登録する復旧を画面と Entra の runbook に分けて記した。
確かめた範囲は、メモリの鍵の保管とローカルの E2E で、Graph の読み戻しは期待値から組み立てた観測文書で代えた。実テナントでの切り替えと既存の信頼先への影響は WI-79 に残る。
