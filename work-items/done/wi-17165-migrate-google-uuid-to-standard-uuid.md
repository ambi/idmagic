---
status: completed
---

# google/uuid の直接利用を標準 uuid へ移す

`spec-diff` は規範仕様の差分なし。
backend の google/uuid の直接利用を Go 1.27 の標準 `uuid` へ移し、depguard で再導入を拒否するようにした。google/uuid は go-webauthn の間接依存として残る。
生成は `uuid.NewV4()` で版 4 を明示した。アクショントークンの ID は、注入した乱数から 16 バイトを読み、版と変種のビットを立てて組み立てる。これで google/uuid の `NewRandomFromReader` と同じ乱数から同じ ID になり、読み取り失敗の契約も保たれる。
PostgreSQL の `uuid` 列を照会する前の検査 5 か所は `sharedpg.IsUUIDText` にまとめた。標準 `Parse` は受理範囲が狭いため、大文字の `URN:UUID:` 接頭辞は 22P02 のエラーにならず、存在しない行として扱われる。検査の意図に沿う変化として、利用者の確認を得て受け入れた。
小文字の `urn:uuid:` が検査を通る既存の欠陥は wi-66744 に起票した。
