---
status: completed
archived_from: c44b8afa453624adbd91d700df2426dec606dd63
---

# 拒否の応答を書くヘルパーを 1 段挟むと R1 が見えなくなる

R1 と R2 が「応答を書いてその結果を返す」関数を認識する方法を、`WriteProblem` という名前の直接照合から、echo の応答書き込みメソッドを出発点にした一覧の構築へ替えた。ガードが writer をどれだけ深く包んでも同じ形として落ちる。製品の振る舞いは変えていない (`mise run spec-diff`: `no normative specification change against main`)。指摘の本文は `requireAuthorizationAdmin -> WriteAdminAccessError -> WriteProblem -> NoStoreJSON -> c.JSON` のように連鎖を示すので、どこを直せばよいかが指摘だけで分かる。R2 の要求ガード判定も同じ一覧に載せ替えた。
検査の強化のみで、`backend/` への変更は無い。現在のツリーに違反は無かったため許可リストは置いていない。

全文は `git show c44b8afa453624adbd91d700df2426dec606dd63:<このファイルのパス>` で読む。
