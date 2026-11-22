# WorkloadIdentity のページ

この文書は、WorkloadIdentity が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[WorkloadIdentity のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## ワークロード ID

| ページ | URL | 目的 |
| --- | --- | --- |
| [ワークロードの信頼バンドル](admin-workload-identity.md) | `/admin/workload-identity` | 外部ワークロードの認証元を管理する |
| [信頼バンドルを登録](admin-workload-trust-bundle-create.md) | `/admin/workload-identity/new` | 外部発行者と検証に使う信頼情報を登録する |
| [信頼バンドルの詳細](admin-workload-trust-bundle-detail.md) | `/admin/workload-identity/$trustBundleId` | 発行者の信頼状態と Agent への関連付けを管理する |
| [信頼バンドルを編集](admin-workload-trust-bundle-edit.md) | `/admin/workload-identity/$trustBundleId/edit` | 発行者の信頼情報を更新する |
| [ワークロードの関連付けを編集](admin-workload-binding-edit.md) | `/admin/workload-identity/$trustBundleId/bindings/$bindingId/edit` | 主体のパターンと関連付ける Agent を変更する |
