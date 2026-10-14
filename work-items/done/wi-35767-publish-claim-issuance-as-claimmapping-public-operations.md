---
status: completed
archived_from: f762a8621a7475d0f43932dced124c67b3f78f85
---

# クレームの発行と開示規則の検証を ClaimMapping の公開操作にする

`mise run spec-diff` は規範の差分なしを報告する。
ClaimMapping の入力だけで決まる計算（`IssueClaimsWithFloor`、`ValidateClaimReleaseRules`、`ResolveUserAttributes`、属性の定義の合成）を `usecases` から公開パッケージの `domain` へ移し、属性スキーマの保存先の型と読み取りを `ports` に置いた。
下限を通らない発行は `domain` の非公開の `issueClaims` にし、ほかのモジュールからは `IssueClaimsWithFloor` しか呼べない。
`usecases` のパッケージは消した。
OIDC、SAML、WS-Federation、Application の呼び出し側は、関数の中身を変えず修飾子だけを書き換えた。
境界の負債の台帳から ClaimMapping への `private-import` の 11 件を消し、台帳は 128 件から 117 件になった。
発行するクレーム、開示の下限、HTTP の応答は変えていない。

全文は `git show f762a8621a7475d0f43932dced124c67b3f78f85:<このファイルのパス>` で読む。
