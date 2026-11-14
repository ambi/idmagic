---
depends_on: [wi-24943-route-seed-writes-through-context-use-cases]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-04
priority: p3
group: tenant-configuration
change_kind: feature
affected_spec:
  - { path: docs/modules/tenancy/features/integration-endpoints.md, requirement: REQ-TENANCY-001 }
  - { path: docs/modules/application/features/catalog.md, requirement: REQ-APPLICATION-007 }
---

# テナントの構成を Seeding のマニフェストへ書き出し、別の環境へ同じ構成を適用できるようにする

## 動機

テナントとその配下の構成（Application、クライアント、割り当て、クレームの開示、連携の信頼、署名鍵のポリシー、属性のスキーマなど）は、管理 API と管理コンソールからしか作れない。
本番の運用では、ステージングで作った構成を本番へ同じ形で移し、構成をコードとしてレビューし、障害の後に再現する手段が要る。
Keycloak はレルムを JSON で書き出して取り込め、これが環境の昇格と GitOps の土台になっている。

起票時はこの項目が書き出しと冪等な取り込みの両方を担っていた。
その後 [Seeding](../../docs/modules/seeding/README.md) が、宣言したマニフェストの計画（`dry_run`）、冪等な適用、手動の変更を競合として扱う検出、シークレットの提供元による解決を持った（[seed の計画と適用](../../docs/modules/seeding/features/seed-run.md)）。
取り込みは Seeding の適用で足りるので、この項目は、既存のテナントの構成を Seeding のマニフェストとして書き出す側だけを扱う。

Seeding は今、所有者のユースケースではなく Repository の保存を直接呼ぶ。
書き出した構成を別の環境へ適用すると、検証と監査を通らない書き込みが本番に入るので、wi-24943 を先に完了させる。

新しい機能領域なので、優先度は p3 とする。

## 対象範囲

- テナントの構成を、Seeding のマニフェストの形式で書き出す操作を加える。列挙の順を安定させ、二つの書き出しの差分を読めるようにする。
- シークレット（秘密鍵、パスワード、クライアントシークレット）は書き出さず、Seeding のシークレットの提供元の参照に置き換える。
- 書き出す対象の構成の種類と、マニフェストがまだ表せない種類を一覧にし、表せない種類はマニフェストの文法へ足すか、対象外として記録する。
- 書き出したマニフェストを別のテナントへ `dry_run` と適用で移し、同じ構成になることを往復のテストで固定する。

## 対象外

- User、認証イベント、トークンなど運用のデータの移送。バックアップと復元が扱う。
- シークレットの平文の書き出し。
- 環境の間の双方向の常時の同期。

## 設計

書き出しの入口を管理 API に置くか、Seeding と同じくプロセスを起動できる運用者の CLI に置くかを着手時に決める。
Seeding は HTTP の入口を設けない判断を持つので、書き出しも同じ境界に置く案から比べる。

## 検証

- 書き出したマニフェストを空のテナントへ適用し、もう一度書き出すと同じマニフェストになる往復のテスト。
- 書き出したマニフェストにシークレットの平文が含まれないことを確かめるテスト。
- `mise run verify`

## リスク

マニフェストの形式は、書き出しを公開すると利用者が保存して使うので、後から変えにくくなる。
形式のバージョンを持たせ、読み取りの互換性の扱いを Seeding の判断に書く。
