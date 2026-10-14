---
status: completed
archived_from: de4fffe6360a3678331f63077aa8b1d364530d65
---

# 保護リソースが DPoP 証明の `htu` をパスとして照合するので、適合クライアントが到達できない

保護リソースが、DPoP 証明の `htu` をパスではなくリクエストの絶対 URL と照合するようになった。
RFC 9449 に従うクライアントが、`/token` で得た送信者制約付きのトークンをそのまま保護リソースで使える。
`mise run spec-diff` は規範差分なしを返す。変えたのは `docs/domain/oauth2/internals.md` の仕組みの記述だけであり、
`RFC9449-API-TOKEN-DPOP` と `DpopProofClaims.htu: url` が既に求めていた形を実装で満たしにいった記録である。
直したのは `RequestHTU` の定義である。テナントの正規 issuer の origin にリクエストのパスを継ぐので、
path style で prefix が二重にならず、subdomain style でテナントの host が落ちない。
`/token`、`/userinfo`、保護リソースの 3 つの呼び出し元が同じ関数を通るので、subdomain style の
`/token` と `/userinfo` が基底の host を期待していた同じ欠陥も一緒に直った。
期待値を決められない場合に `""` と `htu` の無い証明が一致する経路は、`verifyDPoP` が空の `htu` を拒否することで閉じた。
保護リソースの呼び出し元（`auth.go`）は既に `RequestHTU(c, "")` を呼んでいたので、変更は関数の定義だけで済んだ。

全文は `git show de4fffe6360a3678331f63077aa8b1d364530d65:<このファイルのパス>` で読む。
