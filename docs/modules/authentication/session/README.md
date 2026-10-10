# ログインセッション

## 概要

この文書は、`LoginSession` の保存、有効期限、本人と管理者による一覧と失効、サインアウトの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | セッションの保存と有効期限、本人による一覧と失効、管理者による一覧と個別の失効と全失効、サインアウト |
| 行為者 | 本人、テナント管理者、EndUser |
| 扱わないもの | ログインそのものは[ログイン](../sign-in/README.md)が扱う |

## モデル

セッションの有効期限は、作成時に設定する固定の 1 時間（`SessionTTLSeconds`）であり、利用によって延ばさない。

`amr` の語彙は `LoginSession` の検証が閉じた集合として持ち、作成時と第二要素の成立時の両方で、保存の前にこれを通す。
`acr` は同じ場所で `amr` から導き、`otp`、`webauthn`、`hwk`、`swk`、`rc`、`tdev` のいずれかがあれば `urn:idmagic:acr:mfa` へ上がる。
`federated` はここに含めない。上流の IdP が何を検証したかはブローカーには分からないので、`urn:idmagic:acr:mfa` を名乗ると強度を偽ることになるからである。

- **判断**：`amr` の語彙を書き込みの側で閉じるのは、`amr` が ID トークンとアクセストークンの `amr` のクレームとしてリライングパーティーまで届く値だからである。呼び出し元がそれぞれ正しい値を渡していることに頼ると、要素を一つ足すたびに語彙の外の値が混ざる余地が残る。

## 操作

### 本人によるセッションの一覧と失効

#### REQ-AUTHENTICATION-013 ユーザーは自分の有効なセッションを一覧して失効できる

- 本人がセッションを一覧したとき、Authentication は、本人の有効なセッションを、プロセスの再起動を挟んでも返す。
- 本人が現在以外のセッションを失効させたとき、Authentication は、そのセッションを失効させ、204 を返し、`SessionEnded` を発行し、以後の一覧に返さない。
- 本人が現在以外のすべてのセッションを失効させたとき、Authentication は、現在のセッションだけを残す。
- 失効済みのセッションの失効を再び要求されたとき、Authentication は、204 を返し、失効の時刻を変えない。
- 本人のものでないか存在しないセッションを指定された場合、Authentication は、404 と `session_not_found` で拒否する。
- Authentication は、セッションの有効期限を作成の時刻から 1 時間に固定し、利用によって延ばさない。
- **例**：EX-AUTHENTICATION-013-01、EX-AUTHENTICATION-013-02、EX-AUTHENTICATION-013-03

### 管理者によるセッションの一覧と失効

#### REQ-AUTHENTICATION-021 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる

- 管理者が User のセッションを一覧したとき、Authentication は、有効なセッションを開始の時刻の降順で返す。
- 管理者がセッションを 1 件失効させたとき、Authentication は、`revoke_reason=admin_revoke` で失効させ、204 を返し、`SessionEnded` を発行する。
- 管理者が User のすべてのセッションを失効させたとき、Authentication は、残りのすべてのセッションを失効させ、204 を返す。
- 失効済みのセッションの失効を再び要求されたとき、Authentication は、204 を返し、`revoked_at` を最初の値のまま残す。
- 別のテナントの管理者か `admin` のロールを持たない利用者が要求した場合、Authentication は、403 と `access_denied` で拒否する。
- **例**：EX-AUTHENTICATION-021-01、EX-AUTHENTICATION-021-02、EX-AUTHENTICATION-021-03

### 利用者によるサインアウト

#### REQ-AUTHENTICATION-035 サインアウトはテナントのエンドポイント形式によらずサーバー側のセッションを失効させる

- 利用者が OIDC、SAML のシングルログアウト、WS-Federation のサインアウトでサインアウトしたとき、Authentication は、サーバーの側のセッションを失効させ、以後の認証の解決で未認証として扱う。
- 失効させたセッションの ID を Cookie で再び提示されたとき、Authentication は、認証しない。
- パスの形式のテナントで接頭辞のない Cookie を受けたとき、Authentication は、同じセッションとして読んで失効させる。
- **例**：EX-AUTHENTICATION-035-01、EX-AUTHENTICATION-035-02、EX-AUTHENTICATION-035-03
