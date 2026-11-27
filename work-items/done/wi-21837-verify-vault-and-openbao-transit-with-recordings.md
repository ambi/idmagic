---
status: completed
---

# Vault と OpenBao の Transit への前提を、実物から取った記録の再生と模倣で確かめる

Vault と OpenBao の Transit への前提を、実物の開発用サーバー（Vault 2.2.2、OpenBao 2.7.3）から取った記録の再生と、OpenBao の Transit の模倣で確かめるようにした。
`mise run record-transit -- <vault|openbao>` が固定したバージョンのサーバーを起動して往復を記録し、保存の前に、使った root トークン、秘密鍵の PEM、固定の DEK 以外の平文、鍵を書き出す経路を拒否する。
記録は SigningKeys の `keys_vault/testdata` と `envelope_openbao/testdata` に置き、再生は JSON の本文の欄の過不足まで照合する（`testing_httprecord.CompareJSON` を足した）。
模倣は運用者の回転と `min_decryption_version` の後の状態だけに使い、記録の全往復との照合と、記録から外した模倣を報告する検査を置いた。
手書きの本文で形を固定していた既存の単体テストは再生へ移して消し、一つの応答で決まる失敗の分岐を足した。

記録から三つの誤りが見つかり、利用者に確かめて直した。仕様の文面は変えていない。

- REQ-DATAKEYS-001：OpenBao のマスター鍵の名前 `idmagic/datakeys/<tenant>` は、OpenBao も Vault も `404 unsupported path` で拒否するので、既定設定の OpenBao の提供元は実物で一度も動かなかった。鍵名を `{prefix}-{tenant_id}`、既定の接頭辞を `idmagic-datakeys` にし、Transit の鍵名の規則に合わない `OPENBAO_DATA_KEY_PREFIX` を起動時に拒否する。初回公開前なので移行はない。
- REQ-SIGNINGKEYS-001：Vault の回転と両アダプターの鍵の作成は状態コードを読まず、Vault が回転を拒否してもローテーションは回転前の鍵を返して成功していた。200 と 204 以外を失敗にする。
- RFC7518-SIGNATURE-ALGORITHMS：Vault は `salt_length` のない PSS の署名を最大長の salt（RSA-2048 で 222 バイト）で作るので、Vault で署名した PS256 は IdMagic 自身の検証（salt 長 32）にも通らなかった。呼び出し元の `PSSOptions.SaltLength` を `hash`、`auto`、数値で送る。

確認した範囲は開発用サーバーの構成であり、本番の構成（監査デバイス、名前空間、ポリシー、HA の待機系）での応答は保証しない。
Vault の `sys/health` が待機系で返す 429 を不健全と扱うこと、Vault の以前の版との差、SAML などのスコープ ID が Vault の鍵名の規則に合うことは確かめていない。
変異テストは行わず、記録の欄名と署名の形を手で書き換えて再生が落ちることだけを確かめた。
