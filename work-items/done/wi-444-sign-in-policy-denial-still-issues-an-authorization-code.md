---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# サインインポリシーが拒否した認可要求に、認可コードが発行されている

`enforceDefaultSignInPolicy` の 3 つの拒否分岐が、応答を書いたうえで `support.ErrResponseWritten` を包む `ErrSignInPolicyRefused` を返すようになった。それまでは `support.WriteProblem` の戻り値 (書き終えたとき `nil`) をそのまま返しており、呼び出し元 3 か所の `if err != nil { return err }` が素通りしていた。結果として、サインインポリシーが拒否した認可要求に対して 403 / 401 を書いたあとも `completeAfterAuthn` が走り、`AuthorizationCodeIssued` が発行され認可コードが保存されていた。利用者には拒否に見えて、認可コードは存在している状態だった。`mise run spec-diff` は `no normative specification change against main` を返す。規範は動かしておらず、`REQ-APPLICATION-009` が既に求めている「トークン発行前にポリシーを評価し、拒否する」に実装が追いついた変更である。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
