# wi-626-align-cross-tenant-application-read-refusals

Application の管理 API について、実際に返していた 404 を [API 契約](../../../spec/contexts/application/main.tsp)へ追加した。

次の API 操作は、存在しない Application に 404 `application_not_found` を返す。
別テナントの Application も同じ応答になる。

- `IdMagic.Application.Operations.GetAdminApplication`
- `IdMagic.Application.Operations.UpdateAdminApplication`
- `IdMagic.Application.Operations.DeleteAdminApplication`
- `IdMagic.Application.Operations.UploadApplicationIcon`
- `IdMagic.Application.Operations.DeleteApplicationIcon`
- `IdMagic.Application.Operations.UpdateApplicationOidcConfig`
- `IdMagic.Application.Operations.RotateApplicationClientSecret`
- `IdMagic.Application.Operations.IssueApplicationClientSecret`
- `IdMagic.Application.Operations.RevokeApplicationClientSecret`
- `IdMagic.Application.Operations.UpdateApplicationWsFedConfig`
- `IdMagic.Application.Operations.UpdateApplicationSamlConfig`
- `IdMagic.Application.Operations.ListApplicationAssignments`
- `IdMagic.Application.Operations.AssignApplication`
- `IdMagic.Application.Operations.GetAppSignInPolicy`
- `IdMagic.Application.Operations.UpdateAppSignInPolicy`
- `IdMagic.Application.Operations.SetApplicationCategories`

次の API 操作は、存在しないカテゴリーに 404 `category_not_found` を返す。
別テナントのカテゴリーも同じ応答になる。

- `IdMagic.Application.Operations.UpdateApplicationCategory`
- `IdMagic.Application.Operations.DeleteApplicationCategory`

応答そのものは変わらない。
