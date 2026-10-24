---
status: completed
---

# AD と LDAP を読み、差分を IdMagic へ送る Directory Connector の実行環境を作る

新しいモジュール DirectoryConnector と実行単位 `idmagic-directory-connector` を加え、閉域の AD と OpenLDAP を読んで Sourcing の Connector API へ User の差分を送れるようにした（REQ-DIRECTORYCONNECTOR-001〜006）。
`enroll` は標準入力の登録トークンで P-256 の鍵対の公開鍵を登録し、秘密鍵と登録の情報を所有者だけが読める状態のディレクトリに置く。`run` は要求ごとに新しい jti の ES256 の JWT で署名し、`check` は IdMagic へ送らずに送るはずの件数と飛ばすエントリを示す。
LDAP は検証した TLS（LDAPS か StartTLS）でだけ bind し、平文の LDAP と https でない realm の URL、互いに含み合う `base_dns`、未知のキーの設定で起動を拒否する。
差分は AD が `uSNChanged`（検索の前の `highestCommittedUSN`、`dsServiceName` が変われば full）、OpenLDAP が `modifyTimestamp`（検索の 5 分前から重ねて読む）で、削除は完全な再同期の完了でサーバーが無効にする。DirSync は Replicating Directory Changes の権限を求めるので採らなかった。
不変 ID は AD が `objectGUID` の標準の base64（Entra の sourceAnchor と同じ表記）、OpenLDAP が `entryUUID` で、項目は読んだ順に 100 件ずつ `<実行 ID>-<連番>` のバッチで送る。接続の失敗、408、429、5xx を 1 秒から倍にして 5 回まで送り直し、完了が受理されてから位置を保存し、中断したら同じ位置から新しい実行を始める。
当初は Sourcing の機能スライスに置き、サーバーの内部型をワイヤの型に再利用したが、別に配布され版がずれる実行単位なので、ほかのモジュールを import しない独立のモジュールへ分けた。他製品との比較から、複数の OU（Entra Cloud Sync）、送らない確認（GCDS）、HTTP プロキシ（Okta AD Agent）を取り込んだ。
テストの OpenLDAP は `osixia/openldap:1.5.0` に毎回作った CA と証明書を入れて使う（同梱の CA は期限切れ）。Docker がなければスキップする。
変更耐性では、domain、usecases、state_file、ldap_goldap の変異をすべて検出し、cli と api_http の生存は `> 299` → `>= 299`（299 を返す API がなく等価）だけである。未被覆は、`LoadConfig` が先に検証するので届かない `run` の LDAP の構成の失敗、`run -once` の同期の失敗の終了コード、OpenLDAP では届かない AD の RootDSE の読み取り（実物の AD は wi-79）である。手の故障注入では、位置の早すぎる保存、接続の報告の省略、固定の jti、ワイヤのキーの誤り、CLI での TLS の検証の省略をすべて検出した。最後のものは最初は検出できず、誤った CA の `check` を拒否する観測を E2E に足した。
