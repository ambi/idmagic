# SharedSignals の標準仕様

## Security Event Token (SET)

RFC 8417 — https://www.rfc-editor.org/rfc/rfc8417.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8417-SET-SIGNED | required | MUST | SET は jti/iat/iss/aud/events を含む署名済み JWT として発行する。 |
| RFC8417-SET-VERIFY | required | MUST | 受信側は署名検証を通過した SET だけを反映する。検証失敗、未知の鍵、改ざんを検知した場合は反映せず、フェイルクローズで拒否して監査する。 |

## Subject Identifiers for Security Event Tokens

RFC 9493 — https://www.rfc-editor.org/rfc/rfc9493.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9493-SUBID-FORMAT | partial | MUST | `format` メンバーを持つ Subject Identifier を受け取った場合、`format` の値でのみ種別を判定する。IdMagic が解釈するのは `iss_sub` と `opaque` であり、それ以外の `format` は主体を解決できないものとしてフェイルクローズで拒否する。 |
| RFC9493-SUBID-ISS-SUB | partial | MUST | `format=iss_sub` の `iss` は、受信ストリームに登録された `trusted_issuer` と完全一致しなければならない。一致しない `iss` は、SET の署名が正しくても主体を解決できないものとして拒否する。 |

## Shared Signals Framework

OpenID Shared Signals Framework 1.0 — https://openid.net/specs/openid-sharedsignals-framework-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SSF10-SUB-ID | partial | MUST | 送信する SET は最上位に `sub_id` を持つ。形式は `opaque` を使う。Discovery と Stream Management API は提供せず、ストリームは管理者が登録する。 |
| SSF10-RECEIVE-SUB-ID | partial | MUST | 最上位の `sub_id` が `iss_sub` または `opaque` の SET は、イベント内の `subject` を省略しても主体を解決して受理する。形式と発行者の検査は RFC9493-SUBID-FORMAT と RFC9493-SUBID-ISS-SUB に従う。`complex` などの未対応の形式は拒否する。 |
| SSF10-STATUS-NO-TRANSMIT | required | MUST | `disabled` のストリームへイベントを送らない。自動無効化の Stream Updated Event は送らず、`SsfStreamAutoDisabled` で観測する。 |

### SET の理由表記

送信する SET の `reason` は、管理 API の `RevocationReason` と表記を分ける。
従来の受信者が解釈する値を保つため、次の対応を使う。
署名済み SET の `Compact` と過去の監査記録は移行で書き換えない。

| 管理 API の値 | SET の reason |
| --- | --- |
| `agent_killed` | `AgentKilled` |
| `agent_disabled` | `AgentDisabled` |
| `agent_credential_unbound` | `AgentCredentialUnbound` |
| `agent_deleted` | `AgentDeleted` |
| `owner_disabled` | `OwnerDisabled` |
| `owner_deleted` | `OwnerDeleted` |
| `manual_admin` | `ManualAdmin` |
| `user_disabled` | `UserDisabled` |
| `user_soft_deleted` | `UserSoftDeleted` |
| `user_deleted` | `UserDeleted` |
| `session_revoked` | `SessionRevoked` |
| `password_changed` | `PasswordChanged` |
| `authenticator_changed` | `AuthenticatorChanged` |
| `refresh_token_reuse` | `RefreshTokenReuse` |
| `inbound_security_event` | `InboundSecurityEvent` |
