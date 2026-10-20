---
status: completed
---

# WS-Trust のトークン発行を Application の割り当てとサインインポリシーで判定する

REQ-WSFEDERATION-004 に、WS-Trust の発行がパッシブサインインと同じアプリケーションの関門で判定され、関門の拒否とサインインポリシーの追加認証の要求を 403 で拒否する二文を加えた。
TypeSpec はすでに 403 を宣言していたので変えていない。

能動的な STS は資格情報を照合した後、関門を呼ばずにトークンを発行していた。
受け入れテストで、割り当てのない利用者にも、MFA を求めるポリシーのアプリケーションにも 200 でトークンが出ることを確かめてから直した。
`WsTrustService` が、パスワードだけで認証した事実と発信元の IP で関門を呼び、拒否を 403 と `WsTrustTokenRejected` に写す。
拒否時の `AppStepUpRequired` と `AppAccessDeniedByPolicy` の発行はパッシブサインインと同じ関数に寄せた。

変更行のミューテーションは、関門を検査する受け入れテストが別のパッケージにあるため到達を数えられなかった。
代わりに、認証の事実を MFA にする、拒否を 400 にする、アプリケーションのイベントを出さない、拒否の理由を捨てて発行する、の四つを手で注入し、すべて受け入れテストが検出することを確かめた。
WS-Trust のパスワード認証の委譲は wi-88033 に残る。

全文は `git log -1 -- work-items/active/wi-52958-gate-ws-trust-issuance-by-application-assignment-and-policy.md` が示すコミットの親、またはそのコミットで読む。
