---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-24
priority: p3
depends_on: [wi-11530-delegate-password-verification-to-active-directory-through-the-connector]
change_kind: feature
affected_spec:
  - { path: docs/modules/provisioning/features/synchronization.md, requirement: REQ-PROVISIONING-003 }
---

# IdMagic を正とする User を、Directory Connector で AD へ作成、変更、無効化する

## 動機

人事システムや IdMagic を正とする組織でも、ファイルサーバーや Windows のログオンのために AD のアカウントが要る。
今の Provisioning は SCIM の下流だけを扱い、閉域の AD へは届かない。

| 製品 | 機能 |
| --- | --- |
| Okta | AD Agent による AD へのプロビジョニングとデプロビジョニング。作成した AD のアカウントには乱数のパスワードを設定し、初回のサインインで変更させる |
| Microsoft Entra | 人事システムを起点にした AD へのプロビジョニング。プロビジョニングの Agent が AD のアカウントの作成、変更、有効化、無効化を受けて実行する |

## 対象範囲

- Provisioning の下流の種類に、Connector を通る AD を加える。
- 割り当てた User の作成、属性の変更、無効化を、指定した OU への AD のアカウントの操作として Connector の経路で送る。
- 作成した AD のアカウントの `objectGUID` を対応として記録し、同じ取り込み元からの取り込みと二重にならないようにする。
- 作成時の初期パスワードの扱いを決める。

## 対象外

- AD の Group の作成と所属の書き込み。
- OpenLDAP への書き込み。

## 設計

### 未解決の問い

- 同じ取り込み元を、取り込み（AD が正）と送り出し（IdMagic が正）の両方に使うときの正の決め方。属性ごとに正を分けるか、取り込み元ごとにどちらかに限るか。
- Provisioning と DirectoryConnector の境界。下流の AD のアダプターを Provisioning に置き、Connector は AD の操作だけを行う案と比べる。

## 計画

1. 境界の判断と正の決め方を設計に書き、利用者に確認する。
2. Provisioning と DirectoryConnector の仕様を更新してから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いと境界を決める。
- [ ] T002 [Spec] 要件を書く。
- [ ] T003 [App] AD への作成、変更、無効化を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- OpenLDAP の fixture で操作の経路を、Samba AD か実物の AD で AD のアカウントの属性を確かめる。

## リスク

- 取り込みと送り出しが同じ AD のアカウントを互いに書き換えると、変更が往復し続ける。正の決め方を先に決める。
- Connector に AD のアカウントの作成の権限が要り、侵害時の被害が大きくなる。
