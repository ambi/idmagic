---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `Fapi2SecurityProfile` を選んでも追加の制約が 1 つも掛からない

`fapi_profile` に `fapi_2_security_profile` を選んだクライアントへ、FAPI 2.0 Security Profile の
追加制約 3 つが実際に掛かるようになった。認可リクエストは PAR 経由に限られ、クライアント認証は
`private_key_jwt` と `tls_client_auth` に限られ、アクセストークンは DPoP か mTLS の送信者制約を
必要とする。判断は保存された `FapiProfile` から毎回導き、派生フラグは保存しない。プロファイルを
選んでいないクライアントの振る舞いは 1 つも変わらない。
あわせて `/register` が `private_key_jwt` のクライアントを 1 件も登録できなかった欠陥を直した。
登録前の検証に使う候補が必須項目 `UpdatedAt` を欠いており、鍵の有無によらず全件が
`private_key_jwt requires non-empty inline jwks` で落ちていた。この欠陥は本項目の Scope の直上に
あった。FAPI クライアントの認証方式を非対称に限る以上、その方式で登録できなければ制約が
空虚になるためである。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
