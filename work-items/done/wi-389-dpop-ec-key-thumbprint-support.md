---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# DPoP proof の JWK サムプリント計算を EC (ES256) 鍵にも対応させる

`jwkThumbprint` の正規メンバー集合を `kty` で分岐させ、EC は RFC 7638 §3.2 が定める `{crv, kty, x, y}` で計算するようにした。宣言どおり受理される ES256 DPoP proof が、署名検証を通ったあと最終段の jkt 算出で必ず落ちていた食い違いが解けている。未対応の `kty` は空のサムプリントで通さず明示エラーで拒否する (fail-closed)。RSA の集合 `{e, kty, n}` は変えていないので、発行済みの RSA 由来 `cnf.jkt` は動かない。`mise run spec-diff` は `no normative specification change against main` を返す。標準行 `RFC9449-TOKEN-BINDING` は元から MUST を宣言しており、実装がそれに追いついた変更である。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
