---
status: completed
---

# SAML と WS-Federation の発行のイベントの payload に、利用者を userId で載せる

規範仕様の差分はない。`SamlSignInIssued`、`WsFedSignInIssued`、`WsTrustTokenIssued` の payload の JSON 名を、一括の置き換えで壊れた `userIdNone` から TypeSpec どおりの `userId` に直し、利用者で絞り込んだ監査の検索に SAML と WS-Fed のサインインと WS-Trust の発行が現れるようにした。
payload の項目名を TypeSpec と照合するテストを三つのイベントに足し、直す前は `userIdNone` で失敗することを確かめた。`userIdNone` を読む箇所はリポジトリになかった。
全イベントの構造体と TypeSpec のモデルを機械で照合する検査は足さず、wi-98584 の観察に原因を記録した。
