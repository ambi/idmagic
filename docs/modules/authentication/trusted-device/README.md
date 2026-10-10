# 信頼済みデバイス

## 概要

この文書は、第二要素の成立の後に本人が記憶させたブラウザーの発行、有効期間、効果の範囲、失効の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 信頼済みデバイスの発行、ログインでの第二要素の省略、期限と盗難の拒否、資格情報の変化による失効、本人による一覧と失効 |
| 行為者 | EndUser、本人 |
| 扱わないもの | 第二要素を求めるかの判定そのものは[多要素認証](../mfa/README.md)が扱う |

## モデル

常用の端末で毎回第二要素を求めると摩擦が大きい。
`TrustedDevice` は、本人が明示的に同意した一つのブラウザーを一定期間だけ覚えておき、その端末からのログインで第二要素の提示を省けるようにする。
第二要素を条件付きで飛ばす仕組みなので、設計はもっぱら「いつ発行しないか」と「いつ失効させるか」で成り立つ。

| 観点 | 規則 |
| --- | --- |
| 発行 | ログインで本物の第二要素（TOTP、WebAuthn）が成立した直後に限る。パスワードだけ、復旧コード、登録の専用の流れからは発行しない |
| 有効期間 | テナントの `trusted_device_max_age_seconds` で明示した場合だけ有効になり、デフォルトは 0（無効）である。0 なら同意の導線を出さず、送られた `remember_device` も無視する |
| 期限 | 絶対期限は `created_at + trusted_device_max_age_seconds` で、上限は 90 日。idle 期限は `last_used_at + min(30 日, max_age)` |
| 効果 | ログインの時点の第二要素だけを肩代わりし、`amr` に `tdev` を加えて `acr` を `urn:idmagic:acr:mfa` へ上げる。ステップアップ認証の直近性（`step_up_at`）には寄与しない |
| 失効 | 本人によるパスワードの変更とリセット、認証の要素の登録と解除、管理者による認証器のリセット、アカウントの無効化、本人または管理者による全セッションの失効は、対象のユーザーのすべてのデバイスを失効させる |

`SignInRule.allow_trusted_device=false` の規則は、`tdev` を MFA の充足として認めない。
`tdev` は `rc` と同じくこの製品に固有の IANA 未登録の値であり、「要素を提示したのではなく端末が記憶されていた」ことを RP に対しても隠さない。

- **判断**：パスワードだけの成功から発行しないのは、パスワードを知る攻撃者が自分の端末を記憶させられるなら、MFA の要件は最初から無いからである。復旧コードから発行しないのは、要素を失った状況の端末を、長期の信頼に足るものとして扱えないからである。
- **判断**：資格情報が変わるとすべて失効させるのは、それ以前に成立した第二要素の証明も、同時に古くなったからである。
- **判断**：端末の識別の方式は、[信頼済みデバイスを端末の指紋ではなくサーバーが発行するシークレットで識別する](../design/decisions.md#信頼済みデバイスを端末の指紋ではなくサーバーが発行するシークレットで識別する)。

## 状態遷移

### TrustedDeviceLifecycle

信頼済みデバイスは、発行された `Active` と、二度と第二要素を肩代わりしない `Revoked` の 2 状態しか持たない。期限切れは状態ではなく `Active` のレコードに対する時刻の判定であり、絶対期限か idle 期限のどちらかを過ぎたレコードは評価の時点で失効として扱う。`Revoked` は終端で、失効はレコードを削除せず `revoked_at` と `revoke_reason` を設定する tombstone なので、同じ理由での再失効は安全な no-op になる。

利用そのものは状態を変えない。期限内の照合が成功するたびに verifier を回転させて `last_used_at` を進め、cookie を再発行するが、レコードは `Active` のままである。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 期限内であれば第二要素の提示を肩代わりする。期限切れは状態ではなく時刻の判定である |
| Revoked | terminal | 二度と第二要素を肩代わりしない。レコードは削除せず `revoked_at` と `revoke_reason` を持つ tombstone として残る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TrustedDeviceRevoked | — | Revoked | revoked_at と revoke_reason を設定する |

| State | 記憶した端末でのログイン | 信頼済みデバイスの失効 | 資格情報の変更 |
|---|---|---|---|
| Active | 何もしない（期限内）<br>拒否：第二要素を求める（期限切れ） | → Revoked | → Revoked |
| Revoked | 拒否：第二要素を求める | 何もしない | 何もしない |

## 操作

### 利用者による端末の記憶

#### REQ-AUTHENTICATION-026 第二要素の成立時に本人が同意した端末は次回以降の第二要素を省略できる

- テナントの `trusted_device_max_age_seconds` が正の値の間、TOTP か WebAuthn の第二要素でログインを完了した User が端末の記憶に同意したとき、Authentication は、realm のスコープの HttpOnly の Cookie として信頼済みデバイスの資格情報を発行し、`Active` の信頼済みデバイスを記録し、`TrustedDeviceRegistered` を発行する。
- テナントの `trusted_device_max_age_seconds` が 0 の間、第二要素でログインを完了したとき、Authentication は、記憶の同意の導線を出さず、送られた `remember_device` を無視する。
- 復旧コードで第二要素を満たしたか、パスワードだけか登録の専用の流れでログインを完了したとき、Authentication は、端末を記憶しない。
- **例**：EX-AUTHENTICATION-026-01、EX-AUTHENTICATION-026-02、EX-AUTHENTICATION-026-03、EX-AUTHENTICATION-026-04

### 利用者による記憶した端末でのログイン

#### REQ-AUTHENTICATION-027 期限切れ・盗難・別テナントの信頼済みデバイス cookie は第二要素を省略できない

- 実効のサインインポリシーが `Mfa` で `allow_trusted_device=true` の間、`Active` で期限内の信頼済みデバイスの Cookie とともに正しいパスワードを受けたとき、Authentication は、第二要素の画面へ進めずにログインを完了させ、`amr` に `tdev` を加えて `acr` を `urn:idmagic:acr:mfa` にし、verifier を回転させて `last_used_at` を進め、Cookie を再発行する。
- 実効のサインインの規則が `allow_trusted_device=false` の間、信頼済みデバイスの Cookie とともに正しいパスワードを受けたとき、Authentication は、`tdev` を MFA の充足と認めずに第二要素の画面へ進ませる。
- 絶対期限か idle 期限を過ぎた Cookie、回転の前の古い Cookie、別のテナントの realm で発行された Cookie、verifier の一致しない Cookie、`Revoked` のデバイスの Cookie を受けた場合、Authentication は、第二要素を省略せずにログインセッションを `authentication_pending=true` にし、`amr` に `tdev` を加えない。
- **例**：EX-AUTHENTICATION-027-01、EX-AUTHENTICATION-027-02、EX-AUTHENTICATION-027-03、EX-AUTHENTICATION-027-04、EX-AUTHENTICATION-027-05

### 本人による信頼済みデバイスの失効

#### REQ-AUTHENTICATION-029 信頼済みデバイスは機微操作の再認証を肩代わりしない

- 信頼済みデバイスで第二要素を省略したとき、Authentication は、`step_up_at` を進めない。
- ステップアップ認証を経ていないセッションでパスワードの変更、認証の要素の解除、他のセッションの一括の失効を受けた場合、Authentication は、403 と `step_up_required` で拒否し、パスワード、認証の要素、セッションを変えない。
- 本人が信頼済みデバイスを一覧したとき、Authentication は、selector と verifier を含めずに最終利用の時刻の降順で返し、現在の端末を current として示す。
- 本人がステップアップ認証を経て一つの信頼済みデバイスを失効させたとき、Authentication は、204 を返し、そのデバイスを `Revoked` にし、`TrustedDeviceRevoked` を発行する。
- 本人がステップアップ認証を経て信頼済みデバイスを一括で失効させたとき、Authentication は、204 を返し、本人のすべての `Active` のデバイスを `Revoked` にし、デバイスごとに `TrustedDeviceRevoked` を発行する。
- 本人が `Revoked` のデバイスの失効を要求したとき、Authentication は、204 を返し、最初の失効の時刻を保持し、イベントを発行しない。
- ステップアップ認証を経ていないセッションで信頼済みデバイスの失効を受けた場合、Authentication は、403 と `step_up_required` で拒否する。
- 本人のものでないか存在しないデバイスの失効を受けた場合、Authentication は、404 と `trusted_device_not_found` で拒否する。
- **例**：EX-AUTHENTICATION-029-01、EX-AUTHENTICATION-029-02、EX-AUTHENTICATION-029-03

### 本人と管理者による資格情報の変更

#### REQ-AUTHENTICATION-028 資格情報が変わると信頼済みデバイスはすべて失効する

- 本人がパスワードを変更かリセットしたか、認証の要素を登録か解除したか、管理者が認証器をリセットしたか、User を無効化したか、本人か管理者が全セッションを失効させたとき、Authentication は、その User のすべての `Active` の信頼済みデバイスを `Revoked` にし、デバイスごとに `TrustedDeviceRevoked` を発行する。
- **例**：EX-AUTHENTICATION-028-01、EX-AUTHENTICATION-028-02、EX-AUTHENTICATION-028-03、EX-AUTHENTICATION-028-04、EX-AUTHENTICATION-028-05、EX-AUTHENTICATION-028-06

## セキュリティ上の考慮

本人はアカウントのポータルから個別に、あるいは一括で失効でき、その操作自体がステップアップ認証の対象である。
保持するのは、デバイスの識別子と時刻、User-Agent から導いたブラウザーと OS の系統だけのラベル（例：`Chrome / macOS`）である。
生の User-Agent も IP も cookie の平文も保存しない。
一覧で自分の端末を見分けるのに要る粒度はそこまでで、それ以上は失効の判断に寄与せず、漏れたときの被害だけが増えるからである。
監査のイベント（`TrustedDeviceRegistered`、`TrustedDeviceRevoked`）も同じ粒度に留める。
