# SharedSignals Standards

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
