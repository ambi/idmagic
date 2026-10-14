---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# SCIM の有効/無効フラグを廃止し用語を SCIM アクセストークンへ統一する

SCIM を常時有効とみなす方針に統一し、テナントごとの有効/無効設定
(`ScimConfig` エンティティ・`scim_configs` テーブル・`/api/admin/scim/config`
エンドポイント・UI トグル) を全廃した。SCIM API のゲートは「有効な
アクセストークンで認証成功すること」のみに簡素化した。UI 用語を
「認証トークン」「Bearer トークン」から「アクセストークン」へ統一し、
日本語 UI に混在していた英語「Inbound Provisioning」を日本語表現に置換した。
glossary には認証方式を仕様として決めつけない中立的な `ScimToken` 定義を追加し、
将来の OAuth 2.0 発行アクセストークン受け入れの余地を残した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
