---
status: completed
---

# HTTP を経ない文脈には ID だけのテナントの代わりにテナントの ID を入れる

wi-73604 と同じブランチで、`WithTenantID` と共有の一つの格納先を導入した。
HTTP を経ない13箇所と `testing_tenant.Default` は ID だけを運び、`Tenant` は nil を返す。
解決済みの文脈を ID だけの文脈へ上書きすると古い設定と HTTP の位置情報も消える。
`spec-diff` は規範差分なしを示した。対象 ID、パスワードポリシーの結果、Cookie、永続状態とイベントは変えていない。
停止中の Job の扱いと、背景処理のために設定を追加解決する変更は対象外とした。

全文は `git log -1 -- work-items/active/wi-73872-carry-only-the-tenant-id-in-contexts-built-outside-http.md` が示すコミットの親、またはそのコミットで読む。
