# Sourcing

外部の権威ある取り込み元から IdMagic へアイデンティティを取り込む責務を所有する。情報の正は外部にあり、IdMagic 内部のプリンシパルはその写しである。取り込み元との関連付け、外部の不変 ID との相関、取り込み処理とカーソル、外部の状態に従う削除・無効化の規則を定め、取り込み元ごとに機能単位を設ける。

この Context に入るかどうかは、通信の方向や実行時の形ではなく、永続的な関連付けを持つ外部権威が存在するかどうかで決まる。したがって、管理者による CSV インポート（IdManagement）、ログイン時のフェデレーション（Authentication）、下流システムとの台帳照合（Application または Provisioning）はいずれも対象外である。

現在の機能単位は `scim` だけである。SCIM 2.0 サーバーとして `/scim/v2/Users`、`/scim/v2/Groups` などを提供し、Okta、Google Cloud Identity、Entra ID などの外部 IdP からユーザーとグループの同期を受ける。Context のルートにはファサードと組み立てだけを置き、複数の取り込み元に実在する共通点が判明するまでは共通機構を作らない。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
