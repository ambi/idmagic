# WI-599: JSON リクエストの未知プロパティとサイズ上限を統一する

作業項目は `wi-599-decode-admin-request-bodies-strictly` である。

管理 API、アカウント API、ブラウザー API は、JSON リクエストに含まれる未知のプロパティを無視する。
既知のプロパティの型または値が不正な場合と、JSON の構文が不正な場合は、引き続きエラーを返す。
標準仕様が未知のフィールドの拒否を要求するプロトコルオブジェクトには、その標準固有の検証を適用する。

SAML サービスプロバイダーの登録、WS-Federation の証明書利用者の登録、Entra フェデレーションの設定、テナントクォータの更新では、JSON リクエストボディを 64 KiB までに制限する。
上限を超えた要求は 400 の `invalid_request` となり、設定は変更されない。

規範上の振る舞いは [REQ-PLATFORM-005](../../domain/scenarios.feature.md) と [REQ-TENANCY-017](../../domain/tenancy/scenarios.feature.md) が定める。
