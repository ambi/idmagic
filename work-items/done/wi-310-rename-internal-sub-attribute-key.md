---
status: completed
archived_from: 1f22798b023940c568aa4d67caf0a35d66d9be68
---

# ClaimMapping の内部属性キー "sub" を protocol-neutral な名前に改名する

`AttrSub`（値 `"sub"`）を `AttrUserID`（値 `"user_id"`）へ改名し、`ResolveUserAttributes` /
`coreAttributeKeys` / SCL (`claim-mapping.yaml`) / frontend (`DEFAULT_NAMEID_SOURCE`,
`CORE_ATTRIBUTE_KEYS`, `coreAttributeUserIdLabel`) / dev seed data
(`backend/cmd/internal/bootstrap/federation.go`) / 関連テストの直書きリテラルを一括で
更新した。OIDC の ID Token/UserInfo wire claim `"sub"` および `reservedClaimTypes["sub"]`
は Out of Scope のとおり変更していない。
本システムは未リリースのため、既存永続データ (JSONB) との後方互換は対象外と判断し、
当初実装した read-time 互換 shim (`IssueClaimsWithFloor` 手前での旧キー読み替え) は
ユーザー指示により削除し、単純なリネームとした (T001/T004、Design の互換方針からの逸脱)。
**対応していないこと**: `spec/contexts/oauth2.yaml` の `OAuth2Client.claim_policy` 説明文
("NameID 相当の source 属性を sub の source として使う") は、内部キーではなく OIDC の
ID Token wire claim `sub` を指す記述であると判断し、変更していない。
test-first からの逸脱 (self-attest): T003 は振る舞い変更ではなく定数値とテストリテラルの
張り替えであり、新規の RED-GREEN サイクルは適用しなかった。

全文は `git show 1f22798b023940c568aa4d67caf0a35d66d9be68:<このファイルのパス>` で読む。
