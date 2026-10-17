---
status: completed
---

# JWT の署名器を共有ライブラリから OAuth2 へ移す

規範仕様の差分はない。
JWTSigner と既存のテストを OAuth2 内部へ移し、起動時の構築をルートで公開した。
PS256 署名は秘密鍵と kid を受け取る共有関数に残し、SharedSignals と OAuth2 で鍵の型から変換する。
共有依存は10件から6件へ減った。
JWKS とリプレイ検証に残る OAuth2 依存は wi-18088 へ分けた。
共有関数の変更行の mutation は実行可能な3変異を検出し、文字列演算を不正にする4変異はコンパイル不能だった。
移動したトークン規則そのものの全変異は対象外とし、鍵 ID の変換漏れを既存の HTTP テストで検出する。

全文は `git log -1 -- work-items/active/wi-91505-move-the-jwt-token-signer-into-oauth2.md` が示すコミットの親、またはそのコミットで読む。
