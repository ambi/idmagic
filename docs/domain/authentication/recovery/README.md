# 復旧コード

## 概要

この文書は、認証の要素を失ったときの控えとなる復旧コードの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 復旧コードの生成、再生成、失効と、復旧コードによる第二要素の成立 |
| 行為者 | EndUser、本人 |
| 扱わないもの | 第二要素を求めるかの判定は[多要素認証](../mfa/README.md)が扱う |

## モデル

復旧コードは、紛らわしい文字を除いた文字の集合から 10 文字のコードを 10 個生成する。
単回限りで使え、再生成は一式をまとめて置き換える。
生成、再生成、失効には、ステップアップ認証を求める。
復旧コードで第二要素が成立すると、`acr` は `urn:idmagic:acr:mfa` へ上がり、`amr` に `rc`（この製品に固有の IANA 未登録の値）が加わる。

- **判断**：復旧コードは TOTP や WebAuthn の要素を失ったときの控えとしてだけあり、`User.mfa_enrolled` に意図的に数えない。復旧コードを単独の第二要素として扱うと、ユーザーがそれを唯一の MFA として使え、控えを持つ意味が失われるからである。

## 操作

### 利用者による復旧コードの利用

#### REQ-AUTHENTICATION-036 復旧コードで成立した第二要素は MFA の要求を満たす

- 第二要素の照合を待つログインセッションの間、未使用の正しい復旧コードを受けたとき、Authentication は、そのコードを使用済みにし、`amr` に `rc` を加えて `acr` を `urn:idmagic:acr:mfa` にし、`BackupCodeConsumed` を発行し、認可を続けさせる。
- 復旧コードで第二要素を成立させた後、同じセッションの次の認可の要求を受けたとき、Authentication は、第二要素を再び求めない。
- 本人がステップアップ認証を経て復旧コードを生成または再生成したとき、Authentication は、紛らわしい文字を除いた 10 文字のコードを 10 個返し、以前の一式を置き換え、`RecoveryCodesGenerated` を発行する。
- 本人がステップアップ認証を経て復旧コードを失効させたとき、Authentication は、一式を消し、`RecoveryCodesRevoked` を発行する。
- Authentication は、復旧コードを `mfa_enrolled` に数えない。
- 誤った復旧コードか使用済みの復旧コードを受けた場合、Authentication は、401 と `invalid_recovery_code` で拒否し、`AuthenticationFailed` を発行する。
- 復旧コードの保管先へ到達できない場合、Authentication は、503 と `recovery_unavailable` で拒否する。
- **例**：EX-AUTHENTICATION-036-01、EX-AUTHENTICATION-036-02
