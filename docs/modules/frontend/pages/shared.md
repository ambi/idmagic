# 共通表示と外部の到達先

この文書は、ページを数える際に区別する共有要素と外部システムを示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Frontend のページへ戻る](README.md)

## 共通表示と外部の到達先

| 名前 | 識別子 | 区分 | 役割 |
| --- | --- | --- | --- |
| 外部 IdP | external-idp | 外部 | フェデレーションの認証先。認証の結果を IdMagic のコールバックへ返す。 |
| 呼び出し元アプリケーション | client-application | 外部 | 認可要求のリダイレクト先。認可結果と state をプロトコルに従って返す。 |
| 通知メール | email | 外部 | パスワード再設定またはメール確認のリンクを届ける。リンクから該当するページへ戻る。 |
| ダウンロード | download | 外部 | エクスポートしたファイルまたは本人のデータをブラウザーへ渡す。 |
| 共通エラー表示 | shared-error | 共通表示 | ルーターがページを表示できないときに出し、再ログインやポータルへの戻り方を示す。ページマーカーは error。 |
| 再認証 | step-up | 共通表示 | 元の操作の確認後、ポップアップで再認証する。成立後は一度だけ再試行し、開けなければページ全体で再認可する。 |

## ページを描画しない経路

| 経路 ID | 区分 | 理由 |
| --- | --- | --- |
| `/admin/users_/$id` | layout | ユーザーの詳細と編集を束ねる Outlet。 |
| `/admin/workload-identity_/$trustBundleId` | layout | 信頼バンドルの詳細、編集と関連付けを束ねる Outlet。 |
| `/system/tenants_/$realm` | layout | テナントの詳細、基本情報、入口方式とクォータを束ねる Outlet。 |
| `/admin/settings_/saml-idp-profiles_/$profileId` | layout | SAML IdP プロファイルの詳細と編集を束ねる Outlet。 |
| `/admin/identity-providers_/$id` | layout | 外部 IdP の詳細と編集を束ねる Outlet。 |
| `/admin/agents_/$agentId` | layout | Agent の詳細、編集と個別の管理ページを束ねる Outlet。 |
| `/admin/groups_/$groupId` | layout | グループの詳細と編集を束ねる Outlet。 |
| `/admin/applications_/$applicationId` | layout | アプリケーションの詳細、編集と同期管理を束ねる Outlet。 |
| `/admin/access-reviews_/$planId` | layout | レビュー計画の詳細と編集を束ねる Outlet。 |
| `__root__` | layout | エラー境界と Outlet を提供するルートで、独立したページではない。 |
| `/account` | layout | 本人向けの子ページを収める Outlet。概要ページは /account/ が描画する。 |
| `/admin` | layout | 管理コンソールの子ページを収める Outlet。概要ページは /admin/ が描画する。 |
| `/system` | layout | システムコンソールの子ページを収める Outlet。 |
| `/system/` | redirect | /system/tenants へ転送する入口。専用ダッシュボードはまだない。 |
| `/account/profile` | layout | プロフィールの詳細と編集を束ねる Outlet。詳細は /account/profile_/ が描画する。 |
