---
status: completed
---

# 上流の SAML IdP のメタデータから写した署名の証明書を、保存の時点で使える形に揃える

REQ-AUTHENTICATION-025 に二つの箇条を足した。
上流の SAML の接続の作成と更新で、署名の証明書を PEM か、メタデータの `X509Certificate` と同じ見出しのない DER の base64 で受け付け、同じ DER の正規の PEM で保存して返す。
一つの証明書として解析できない値は、400 と `invalid_request` で拒否する。
TypeSpec の `IdentityProviderConnectionRequest.saml_signing_certificates` に、受け付ける形を書いた。

Keycloak（PEM と base64 の両方）、Microsoft Graph（base64）、Auth0（PEM か CER）を調べ、Keycloak と同じ二つの形を採った。
Keycloak と違って保存の時点で解析するのは、IdMagic が保存の後に誤りを知らせる経路を接続の検査しか持たないためである。
上流の応答の署名の検証と接続の検査も同じ関数（`federation/domain` の `ParseSAMLSigningCertificate`）で読むので、写したままの値で保存済みの接続も移行なしにログインできる。
`IdentityProviderConnection.Validate` には解析を足していない。永続化の読み出しでも呼ばれ、保存済みの壊れた値が接続の読み出しを止めるためである。
更新では正規化してから信頼の根拠を比べるので、同じ証明書を形を変えて送り直しても `Disabled` に戻らない。
管理画面の証明書の欄も、共通の欄の規則 `containsCertificate` で同じ二つの形を受け付ける。
SP の登録の AuthnRequest の署名の証明書は、保存の時点で拒否しているので対象外とした。
