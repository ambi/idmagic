---
status: completed
archived_from: da24914877ec006021d0b68143f428ec5b909fb4
---

# IdMagic workspace を SCL 3.0 へ切り替え旧形式を撤去する

- root context mapを含む全18 SCL文書を3.0へ統一し、SCL 2.0 schema、version dispatcher、旧fixtureを
撤去した。Go runtime loaderとpolicy projectionも3.0のmodels、interface contracts、authorization、
scenarios、flowsへ切り替え、旧section依存を除去した。
- RA/SCL/Architecture文書、SCL skills、未完了work itemの仕様語彙を3.0へ同期し、全IdMagic/tool派生物を
sourceから再生成した。OpenAPIのpath/query重複parameterを除去し、path/method/operation集合179件が
cutover前後で不変であることを確認した。
- TOTP登録確認をSCLのprotected accessに合わせてstep-up対象へ追加した。E2Eで発見したテスト基盤の
locale固定、ユーザー詳細link誤選択、suite間process停止競合も修正し、全19 browser scenarioを通した。

全文は `git show da24914877ec006021d0b68143f428ec5b909fb4:<このファイルのパス>` で読む。
