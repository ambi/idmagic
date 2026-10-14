---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Purge が WebAuthn 資格情報とリカバリコードを消さない

Purge の cascade が破棄する資格情報に、WebAuthn 資格情報とリカバリコードの 2 種が加わった。
これで `docs/domain/standards.md` の `GDPR-ERASURE` が言う「Authentication の資格情報破棄」に、
Authentication が持つ資格情報の取りこぼしが無くなる。
`mise run spec-diff` は規範差分なしであり、本項目は既存の `GDPR-ERASURE` を実装で満たす。
変更は 3 層に分かれる。`AdminUserDeps` へ port を 2 つ足して `cascadeDeleteForSub` から
`DeleteAllForSub` を呼び、`deps_http.Deps` と `adminUserDeps` と `routes.go` でそこへ配線し、
両 port の doc コメントを実際の呼び出し元に合わせた。`backend/cmd/internal/bootstrap` は
既に両 repository を `Authentication` モジュールへ入れていたので、足したのは中間の 2 層である。
**観測を 2 つに分けたことが効いた。** use case を直接呼ぶ観測は、配線の 2 層を 1 つも通らない。
実際、`adminUserDeps` から port を落とす故障も `routes.go` から落とす故障も、use case 側の
テストは緑のまま通した。管理 API を入口にした観測だけが両方を捕まえている。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
