---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# テナントを解決していない文脈で default テナントへ落ちない

`mise run spec-diff` は main に対する規範仕様の差分を報告しない。
REQ-TENANCY-006 は変えず、実装をそれに合わせた。
`tenancy.TenantID` は、文脈にテナントがないか ID が空のとき default テナントの ID を返さずに panic する。
`password.usecases.ResolveTenantPolicy` から default テナントをリポジトリで引く分岐を除いた。
テストは `testing_tenant` で default テナントを明示する。
洗い出しの結果、本番の経路にテナントのない文脈から `TenantID` を呼ぶものはなく、HTTP の応答、ジョブの結果、永続状態は変わらない。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
