# サインイン履歴

## 概要

この文書は、本人が自分のサインイン履歴を参照する API の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 本人のサインインの成功と失敗の履歴の参照 |
| 行為者 | 本人 |
| 扱わないもの | 履歴の元になる認証のイベントの記録と保持は、[認証のイベントの記録](../design/authentication-events.md)が扱う |

## 操作

### 本人によるサインイン履歴の参照

#### REQ-AUTHENTICATION-014 ユーザーは自分のサインイン履歴を確認できる

- 本人がサインインの履歴を取得したとき、Authentication は、本人のサインインのイベントだけを返す。
- 第二要素を使ったサインインを返すとき、Authentication は、`pwd` と第二要素の `amr` を持つ完了の後の `UserAuthenticated` として返す。
- 認証の手段に WebAuthn を含むサインインを表示するとき、Authentication は、`webauthn` という技術の名前ではなく「パスキー」と表示する。
- **例**：EX-AUTHENTICATION-014-01、EX-AUTHENTICATION-014-02
