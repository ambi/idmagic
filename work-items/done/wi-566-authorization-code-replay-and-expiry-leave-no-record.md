---
status: completed
archived_from: 1efba586cf37772972d9a9745f54b287dae3c8b9
---

# 認可コードの再交換と期限切れが、宣言どおりの記録を残さない

`mise run spec-diff` は「no normative specification change against main」を返す。規範は
動いていない。動いたのは実装で、`RefreshTokenStore.RevokeFamily` の返り値を
`error` から `([]string, error)` に変え、新たに `Revoked` へ遷移した `TokenID` を
報告するようにした (`db_memory`・`db_postgres` 双方、sqlc 再生成込み)。認可コード
再提示 (`exchange_code.go`) と refresh トークン再利用 (`refresh_tokens.go`) の両経路は
この返り値を使って `revokeFamilyAndNotify` (パッケージ内共有) から `TokenRevoked` を
token ごとに発行するようになった。`/revoke` (`revoke_token.go`) も同じ返り値を使うよう
揃えた。`AuthorizationCodeStore` に `MarkExpired` の CAS 遷移 (`issued -> expired`) を
追加し、`ExchangeCodeForToken` は期限切れのまま提示された認可コードだけをこれで
`Expired` に遷移させる。`tools/check/example-coverage-debt.json` から
`EX-OAUTH2-005-07`、`EX-OAUTH2-005-08`、`EX-OAUTH2-006-02` の 3 行を外した。
併せて、認可コード再提示と refresh トークン再利用の 2 経路が同じ状況で同じイベント
種別集合を出すことを固定するテストを追加した (Scope の 4 点目)。この検査は「片方
だけ直してもう片方が黙る」将来の退行を防ぐものであり、両経路が対称に `TokenRevoked`
を欠いていた変更前のコードでは (どちらの集合も同じだったため) 偶然 GREEN だった。
今回の欠落そのものの検出は、下記の Primary Use Case Evidence が個別に担った。

全文は `git show 1efba586cf37772972d9a9745f54b287dae3c8b9:<このファイルのパス>` で読む。
