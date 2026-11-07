---
status: cancelled
---

# WebAuthn で受け付ける attestation の形式を決め、評価の範囲を限る

中止した。
登録では、返された attestation をこれまでどおり go-webauthn の形式ごとの検証器へ渡し、受け付ける形式を絞らない。
WebAuthn Level 3 の §5.1.3 では、`none` を要求されたクライアントは `none` か、AAGUID がゼロで x5c のない自己 attestation の `packed` だけを返す。
このため、置き換えがクライアントの MAY だという着手時の前提は誤りで、仕様どおりのクライアントの入力が TPM や X.509 の処理に到達することはない。
attStmt を評価せずに `none` として扱う案は、§7.1 の step 22 の検証から外れ、自己 attestation による所持の証明を失い、前例もないため採らない。
go-webauthn を使う Gitea、Pocket ID、Authelia、Teleport も、届いた形式をすべてライブラリの検証器へ渡している。
`none` と自己 attestation の `packed` に絞る案は手順に沿うが、得られるのは不具合のあるクライアントや攻撃者の入力の到達を防ぐことだけで、変更の負担に見合わないと判断した。
調査の中で、登録が既存の credential ID を拒否しない欠陥を見つけ、別の作業項目に分けた。
