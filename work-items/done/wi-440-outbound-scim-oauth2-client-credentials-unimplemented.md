---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 接続が受け付ける `oauth2_client_credentials` を、実際にトークンを取って使う

`mise run spec-diff` は `no normative specification change against main` を返す。`REQ-` シナリオは動いていない。変わったのは `RFC7644-OUT-AUTHENTICATION` の `Adoption` (`partial` → `required`) と `Statement`、TypeSpec の 2 モデルの記述、そして実装である。`oauth2_client_credentials` の接続が、クライアント資格情報フロー (RFC 6749 §4.4) で取得したアクセストークンを提示するようになった。それまでは保存した `client_secret` をそのままベアラートークンとして送っており、この方式の接続は 1 件も配信できていなかった。トークンは失効の 60 秒手前まで再利用し、期限切れと下流の 401 で取り直す。401 による取り直しは 1 度だけである。**永続化も併せて足した** —— トークン URL・`client_id`・`scope` は以前 API が受理して検証したあと捨てられており、保存先が集約にも表にも無かった。列 3 つと、`active` な oauth2 接続に token URL と client_id を要求する CHECK 制約 1 つを加えている。未リリースなので、制約に掛かる既存行は無く、移行は要らない。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
