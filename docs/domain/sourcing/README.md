# Sourcing

## 責務と境界

外部の権威ある取り込み元から IdMagic へアイデンティティを取り込む。
情報の正は外部にあり、IdMagic の内部のプリンシパルはその写しである。
取り込み元との関連付け、外部の不変の ID との相関、取り込みの処理とカーソル、外部の状態に従う削除と無効化の規則を定め、取り込み元ごとに機能を設ける。

この Context に入るかどうかは、通信の方向や実行時の形ではなく、永続的な関連付けを持つ外部の権威があるかどうかで決まる。

| 扱わないもの | 担当 |
| --- | --- |
| 管理者による CSV のインポート | `IdManagement` |
| ログインの時点のフェデレーション | `Authentication` |
| 下流のシステムとの台帳の照合 | `Application`、`Provisioning` |
| User と Group の記録の正 | `IdManagement` |
| SCIM の API アクセストークンの発行とスコープの語彙 | `ApiTokens` |

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ScimUserRef` | テナント、SCIM の ID、内部の User の ID | `User` を参照する |
| `ScimGroupRef` | テナント、SCIM の ID、内部の Group の ID | `Group` を参照する |

- **判断**：取り込み元との対応（`ScimUserRef`、`ScimGroupRef`）は `User` と `Group` の Aggregate に埋め込まず、この Context の独立した Aggregate とする。埋め込むと、IdManagement の主体のモデルが取り込み元の存在を知ることになり、Context Map にない向きの依存ができるからである。

用語集の `IdentitySource`、`IngestionRun`、`SourceCursor` は、取り込み元の種類が増えたときの概念であり、現在の `scim` の機能は実体を持たない。
Context のルートにはファサードと組み立てだけを置き、複数の取り込み元に実在する共通点が判明するまでは、共通の機構を作らない。

## 公開する契約

SCIM のエンドポイントとモデルの形は TypeSpec の `Sourcing` のタグが、採用する SCIM の規則は[Sourcing の標準仕様](standards.md)が定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `UserLifecycle` | `IdManagement` が実装する | この Context が定める | User の無効化と再有効化、削除の予約を、IdManagement の User の操作として行う |
| `User` と `Group` の Repository | `IdManagement` が提供する | この Context が使う | User と Group とメンバーシップを読み書きする |
| API アクセストークンの認証 | `ApiTokens` が提供する | この Context が使う | SCIM の要求を認証し、スコープを解決する |

## 機能

| 機能 | 内容 |
| --- | --- |
| [SCIM による取り込み](scim/README.md) | 外部の IdP からの SCIM 2.0 による User と Group の同期 |

| 文書 | 内容 |
| --- | --- |
| [Sourcing の用語集](glossary.md) | この Context での語義 |
| [Sourcing の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Sourcing の設計](design/README.md) | 話題ごとの設計と重要な判断 |
