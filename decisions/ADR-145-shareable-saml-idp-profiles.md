---
status: accepted
authors: [tn]
created_at: 2026-07-26
---

# ADR-145: SAML IdP profileは共有可能なモデルとし専用利用も同じ制約で表す

## コンテキスト

多くのSAML連携ではテナント共通のentityID・metadata・証明書を共有する方が設定と鍵運用を
単純にできる。一方、取引先ごとのtrust境界、証明書更新のblast radius、entityIDの分離を
求める連携では、アプリ専用のIdP設定が必要になる。

常時共有に固定すると後者を安全に扱えず、常時SP専用にすると一般的な連携で鍵とmetadataが
不必要に増える。作成時に別種の永続モデルへ分岐させると、route・署名・管理操作の検証規則が
二重化する。

## 決定

SAML IdP profileはテナント内で複数SPから参照できる単一モデルとし、共有可否を
`shared` / `dedicated` の制約で表す。各SPは必ず1 profileを参照し、`dedicated` profileは
最大1 SPからだけ参照できる。

テナントの`default` profileは共有とし、テナント共通SAML URLを使用する。追加profileは
profile IDを含むentityID・endpoint・署名鍵scopeを持つ。現在の設計は
[SAML architecture](../backend/saml/ARCHITECTURE.md)に記録する。

## 却下した代替案

- 常にテナント共有: 証明書・entityID・変更影響範囲を分離する要求を満たせない。
- 常にSP専用: 一般的な連携でもSP数に比例して鍵、metadata、rotation運用が増える。
- 共有profileと専用profileを別モデルにする: protocol routeとcross-profile検証が二系統になり、
  同じ安全条件を別実装で維持する必要がある。

## 影響

- `Saml.models.SamlIdentityProviderProfile` と、profile bindingを持つ
  `Saml.models.SamlServiceProvider` を追加・変更する。
- `Saml.interfaces.SamlSingleSignOn`、`PublishSamlMetadata`、profile管理interfaceは、
  routeのprofileとSP bindingの一致を必須にする。
- `SigningKeys.models.SigningKey` はprofileごとのXML federation鍵を分離するscopeを持つ。
- `Application.models.ApplicationSamlConfig` は割当済みprofileを管理画面へ公開する。
