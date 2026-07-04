# idp-ADR-079: Application sign-on policy evaluation

## ステータス
採用。Application context の `models.AppSignOnPolicy`、`interfaces.GetAppSignOnPolicy`、`interfaces.UpdateAppSignOnPolicy`、`invariants.AppPolicyFailClosed`、`invariants.AppPolicyEvaluatedAcrossProtocols` に反映。

## コンテキスト
idmagic は Application への割当を protocol binding ごとに fail-closed で確認するが、アプリケーションごとの認証強度や再認証条件は持っていない。高リスクな業務アプリに MFA や短い再認証間隔を求めるには、OIDC、SAML、WS-Fed の各開始経路で同じ条件を評価する必要がある。

## 決定
1. ApplicationCatalog が `AppSignOnPolicy` を tenant/application 単位で所有する。
2. policy は `SignOnRule` の順序付き集合として保存し、初期サポートは required ACR、required factor、reauth max age とする。
3. network / device 条件は将来の入力点として保持するが、評価できない条件が有効なら fail-closed で拒否する。
4. 評価点は既存の Application 割当ゲートと同じ federation 開始経路に置き、token / assertion 発行前に必ず通す。
5. OIDC で step-up により満たせる不足は既存の step-up 導線へ誘導する。SAML / WS-Fed は初期実装では protocol transaction を停止し、明確な拒否理由を返す。

## 却下した代替案
- 各 protocol context が独自に policy を持つ: 条件や失敗挙動が分岐し、迂回や設定不整合が起きやすい。
- OAuth2 client / SAML SP / WS-Fed RP に直接 policy を埋め込む: 管理者が扱う単位である Application とズレ、複数 binding を持つアプリで一貫性を保ちにくい。
- network / device 条件を無視して許可する: 管理者が強い条件を設定したつもりでも実際には効かないため、fail-open になる。

## 影響
- `application_sign_on_policies` テーブルを追加し、tenant/application 境界で保存する。
- 管理 API と UI に Application 配下の sign-on policy 編集面を追加する。
- federation 開始時の Application gate は割当結果だけでなく policy 評価結果を返す。
