# SignInPolicy の用語集

| 用語 | 定義 | 別名 |
| --- | --- | --- |
| AppSignInPolicy | Application ごとに順序付けたサインイン規則の集合。トークンか Assertion の発行前に評価する。 | アプリ別サインインポリシー |
| TenantDefaultSignInPolicy | 有効なアプリ別の規則がない場合と、IdMagic 自身の画面へのサインインに使う、テナントのデフォルトの規則。 | テナントデフォルトのポリシー |
| SignInRule | 認証強度、信頼済みデバイスでの充足、再認証の期限、ネットワークと MFA 登録の条件を持つ一つの規則。 | サインイン規則 |
| SignInAuthentication | 評価器が読む認証待ち、MFA と実際の第二要素の成立、認証時刻と追加認証時刻の事実。認証方式の保存型は含めない。 | 認証事実 |
