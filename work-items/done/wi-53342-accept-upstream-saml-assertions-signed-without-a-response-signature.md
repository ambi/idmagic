---
status: completed
---

# 上流 SAML IdP が Assertion だけに署名した応答を受け付け、実物の IdP の記録で検証する

上流 SAML の接続が、Response と Assertion のどちらか、または両方に署名した応答を受け付けるようにした。抽出に使うのは署名で保護された唯一の Assertion だけである。
`REQ-AUTHENTICATION-001` には、署名の置き場所と、拒否する場合（どちらにも署名がない、署名を検証できない、Assertion がちょうど一つでない）の二文を足した（spec-diff：REQ-AUTHENTICATION-001 の変更）。
証拠は、`mise run record-saml` で Keycloak 26.8.2 から記録した四通りの実際の応答の再生である。この記録で直す前の実装が Assertion だけに署名した応答を拒否することと、親の `saml:` 接頭辞のため Assertion 単体の検証に名前空間の移し替えが要ることが分かった。
署名の包み替え、二つ目の Assertion、改ざん、Response の署名だけが壊れた応答の拒否は、記録を変えた入力で確かめた。確かめた範囲は Keycloak の開発モードの応答に限る。Entra ID の実物での確認は wi-79 に残る。
署名しない Response の `Destination`、`InResponseTo`、`Status` は今と同じく照合するが、署名による保護はない。
