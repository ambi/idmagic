# WsFederation Standards

## Web Services Federation Language (WS-Federation) Version 1.2

1.2 — https://docs.oasis-open.org/wsfed/federation/v1.2/ws-federation.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSFed-PassiveSignIn | required | MUST | wsignin1.0 で登録済み wtrealm と許可済み wreply にだけトークンを返す |
| WSFed-SilentSignIn | excluded | MAY | silent sign-in / prompt=none 相当の無音認証 |

## WS-Trust 1.3

1.3 — https://docs.oasis-open.org/ws-sx/ws-trust/v1.3/ws-trust.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSTrust13-IssueBearer | required | MUST | Issue 要求に対して Bearer SAML assertion を RSTR で返す |
| WSTrust13-WindowsTransport | excluded | MAY | WindowsTransport / Kerberos based active profile |

## Web Services Security UsernameToken Profile 1.1.1

1.1.1 — https://docs.oasis-open.org/wss-m/wss/v1.1.1/os/wss-UsernameTokenProfile-v1.1.1-os.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSS-UsernameTokenPassword | required | MUST | WS-Trust active STS は UsernameToken username/password を認証する |

## Web Services Addressing 1.0 - Core

1.0 — https://www.w3.org/TR/ws-addr-core/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WSAddressing-MessageIDToAction | required | MUST | MessageID はリプレイ防止のために検証し、To は能動的 STS のエンドポイント、Action は Issue として検証する |
