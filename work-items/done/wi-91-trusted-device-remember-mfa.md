---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# 信頼済みデバイス記憶 (remember this device) で MFA を一定期間スキップする

ログインで本物の第二要素 (TOTP / WebAuthn) が成立した直後に本人が同意したブラウザーを、テナントが明示的に有効にした期間だけ記憶し、次回以降のログインで第二要素の提示を省略できるようにした。端末は指紋ではなく `selector.verifier` のサーバー発行資格情報で識別し、cookie は realm scope の HttpOnly / SameSite=Lax、保存するのは `selector` と `SHA-256(verifier)` だけで、利用のたびに verifier を回転させる。省略が成立したセッションは `amr` に非 IANA 拡張値 `tdev` を得て `acr` が `urn:idmagic:acr:mfa` へ上がるので、記憶による充足と本物の第二要素は下流からも区別できる。`SignInRule.allow_trusted_device=false` はこの充足を認めず「毎回 MFA」を表す。ステップアップ再認証は `tdev` を候補に持たず `step_up_at` も進めないため、機微操作の再認証は肩代わりされない。有効期限は絶対期限 (テナント設定、上限 90 日) と idle 期限 (最終利用から 30 日、絶対期限が短ければそちら) の両方で切る。失効はパスワードの変更とリセット、認証要素の登録と解除、管理者による認証器リセット、全セッション失効、アカウント無効化、匿名化 cascade から配線した。テナント設定 `trusted_device_max_age_seconds` の既定は 0 (機能無効) である。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
