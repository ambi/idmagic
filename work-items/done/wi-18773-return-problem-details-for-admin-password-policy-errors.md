---
status: completed
---

# 管理者による User の作成のパスワードポリシー違反を Problem Details で返す

`mise run spec-diff` は、REQ-IDMANAGEMENT-042 と `CreateAdminUser` の変更、`PasswordPolicyError` の追加を示す。
管理者による User の作成は、パスワードポリシーに違反する初期パスワードを、独自形式の 400 ではなく 422 の Problem Details（`urn:idmagic:error:password_policy`）で拒否し、違反した規則のコードを拡張メンバー `violations` で返す。
ステータスは、ガイドラインの「400 と 422 の区別」がポリシー違反を 422 に分類することに合わせ、利用者に確認して 400 から変えた。
違反コードの値と順序は Authentication の PasswordPolicy が所有し、IdManagement の契約は列挙しない。
管理コンソールの User の作成画面は、`password_policy` を表示言語の文言で示す。
Authentication の `ChangePassword` と `ResetPasswordWithToken` は同じ独自形式を 400 で返したままで、別の作業項目で扱う。
