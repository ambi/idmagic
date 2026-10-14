---
status: completed
archived_from: 485010ca1fe8e7ff6f2c75320d721170407cdaa5
---

# SigningKeys が宣言する具体例 14 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して、TypeSpec の `AdminRotateKeyRequest` の追加を返した。
SigningKeys が台帳に残していた 14 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
管理 API と JWKS の 6 件（`001-01`、`002-01`、`004-01`、`010-01`、`010-02`、`011-02`）は `backend/signingkeys/handlers_http/scenario_examples_test.go` を新設し、`testing_stack` の上で観測した。ローテーションと無効化の効果は `/jwks` と鍵ストアの現在の鍵から読み直す。
健全性の 2 件（`008-01`、`009-01`）は、到達不能にできる Vault Transit の代役と、全テナントの読み出しを数えるテナント保存先を配線した `health_examples_test.go` で観測した。`009-01` は Group 由来の `system_admin` を持つ他テナントの利用者を含む。
XML フェデレーションの 2 件（`005-01`、`006-01`）は、SSO が発行した Assertion をどの証明書で検証できるかと、SAML と WS-Federation のメタデータで観測した。`007-01` は Vault の鍵ストアを作り直す前後で、メタデータが公開する証明書のフィンガープリントを比べる。`011-01` は既存テストへ `access_denied` の観測を足した。
`003-01` と `012-01` は、バッチの `run` が起動設定の ConfigLoader と依存の組み立てを引数で受ける構造変更のうえで、設定の拒否が依存の組み立てより前に起きることを観測した。
照合で分かった食い違いは 2 件で、どちらも本項目で直した。現在の署名鍵の無効化が、契約の `InvalidRequestError` ではなく `active_key_cannot_be_disabled` を返していた。また `XmlFederationSigning` 鍵をローテートする入口が管理 API にもバッチにも無く、`006-01` の前提に届かなかった。利用者の判断により、`RotateTenantSigningKey` へ任意の本文 `{"usage": ...}` と 400 `InvalidRequestError` を宣言し、ハンドラーが指定の用途の鍵をローテートするようにした。
ライフサイクルバッチによる XML フェデレーション鍵の周期ローテーションとアーカイブは [[wi-55565-rotate-xml-federation-signing-keys]] へ切り出した。

全文は `git show 485010ca1fe8e7ff6f2c75320d721170407cdaa5:<このファイルのパス>` で読む。
