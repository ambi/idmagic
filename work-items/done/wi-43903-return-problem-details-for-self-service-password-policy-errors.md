---
status: completed
---

# パスワードの変更と再設定のパスワードポリシー違反を Problem Details で返す

`mise run spec-diff` は、REQ-AUTHENTICATION-010 と REQ-AUTHENTICATION-016、`ChangePassword` と `ResetPasswordWithToken` の変更と、`PasswordPolicyError` の IdManagement から Authentication への移動を示す。
パスワードの変更と再設定は、ポリシーに違反する新しいパスワードを、独自形式の 400 ではなく 422 の Problem Details（`urn:idmagic:error:password_policy`）で拒否し、違反した規則のコードを `violations` で返す。
これで `password_policy` を返す三つの操作がすべて同じ `PasswordPolicyError` を返す。
違反コードを所有する Authentication に宣言を置き、名前空間は `IdMagic.Contract` のままなので IdManagement の参照は変わらない。
ブラウザーの読み取りは既に Problem Details の `type` を読めたので振る舞いを変えず、テストの本文を新しい形にした。
再設定のポリシー違反を HTTP の境界で固定するテストがなかったので足した。
