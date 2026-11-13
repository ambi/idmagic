---
status: completed
---

# Bearer で届いた要求のステップアップを RFC 9470 で求める

REQ-AUTHENTICATION-039 と REQ-AUTHENTICATION-004 を変更した。
アクセストークンで届いた要求のステップアップ認証は、トークンの `auth_time` で判定し、発行時刻とセッションの `step_up_at` を使わない。
満たさない要求は RFC 9470 の 401 `insufficient_user_authentication` と `max_age="300"` で拒否し、Cookie の要求は従来どおり 403 `step_up_required` で拒否する。
管理発行の API アクセストークンは、保持をステップアップ認証として扱う規則を明文化した。
ステップアップを求めるほかの 9 要件（REQ-AUTHENTICATION-003、010、012、029、033、038、REQ-IDMANAGEMENT-053、REQ-OAUTH2-043、REQ-APITOKENS-005）は、拒否の形を REQ-AUTHENTICATION-039 へ委ね、変えない内容だけを残した。
TypeSpec は、ステップアップを求める 15 操作の 401 に `InsufficientUserAuthenticationError` を宣言した。

着手前は、認証から 1 時間経ったトークンでもパスワードの変更が 204 で通ることを HTTP 境界で観測した。
アカウントポータルの機微な操作の多くはセッションの Cookie で呼ばれていたので、利用者と決めて共通の呼び出し処理の Bearer へ揃えた。
ポータルは再認証の要求を受けると確認を出し、応答の条件を付けた再認可をポップアップで行い、元の操作を一回だけ再送する。入力のダイアログは廃止した。
セッションの一覧と失効、一括の失効、メールアドレスの確認は Cookie のまま残し、`portal-api-calls` 検査が印のない直接の呼び出しを拒否する。
フロントエンド設計と認可設計に、呼び出しの経路とステップアップ認証の根拠を書いた。

確認の範囲の限界として、ポップアップの再認可はブラウザーの E2E では観測していない。認証から 5 分の経過をブラウザーで作れないので、`window.open` と `postMessage` を差し替えた単体テストで確かめた。
再ログインでブラウザーのセッションの認証時刻が新しくなり、Cookie の経路の 403 も解消することは、ログインが毎回新しいセッションを作るコードを読んで確かめたにとどまる。
Bearer での一括の失効の意味は [[wi-43589-decide-bearer-revocation-of-other-sessions]]、使われなくなったステップアップの API の廃止は [[wi-75789-retire-session-step-up-endpoints]] で扱う。
