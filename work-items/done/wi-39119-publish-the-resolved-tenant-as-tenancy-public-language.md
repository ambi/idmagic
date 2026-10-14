---
status: completed
archived_from: d0267d4be1e827899e42d71fd755e184fbede85c
---

# 解決済みのテナントを Tenancy の公開契約として公開する

`mise run spec-diff` は規範の差分なしを報告する。
解決済みのテナントを `context.Context` で受け渡す `WithTenant`、`Tenant`、`TenantID`、`Issuer`、`URLPrefix` を、Tenancy のルートパッケージから公開パッケージ `backend/tenancy/ports` へ移した。
クォータの加減算は、素通しだった `tenancy/usecases` の 2 関数を消し、呼び出し側が `tenantports.QuotaRepository` の操作を直接呼ぶ形にした。
CSV 取り込みの PostgreSQL の committer は、構築関数の必須引数として Tenancy の `QuotaRepositoryInTx` を受け取り、行の書き込みと使用量の加減算を同じトランザクションで確定する。
境界の負債の台帳から Tenancy への `private-import` の 54 件を消し、台帳は 211 件から 157 件になった。
HTTP の応答、テナントの解決規則、発行するイベント、永続状態は変えていない。

全文は `git show d0267d4be1e827899e42d71fd755e184fbede85c:<このファイルのパス>` で読む。
