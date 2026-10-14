---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# テナントをまたぐ branding アセット取得の拒否契約を実装と一致させる

`mise run spec-diff` は REQ-TENANCY-004 のシナリオ変更を報告した。
`EX-TENANCY-004-02` は、別テナントの id で同じ kind のアセットを取得した要求を `InvalidRequestError` で拒否するという記述から、存在しない id を指定したときと同じ 404 not_found で応答し、応答にアップロードした PNG の内容を含まないという記述へ変わった。
TypeSpec の `GetTenantBrandingAsset` と handler は変更前から 404 not_found を返しており、製品コードと公開契約は変わらない。
`tools/check/example-coverage-debt.json` から `EX-TENANCY-004-02` の項目を削除した。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
