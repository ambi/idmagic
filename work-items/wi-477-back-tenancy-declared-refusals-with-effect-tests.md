---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-001 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-005 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-013 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-014 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-017 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-018 }
---

# Tenancy が宣言する未検証の拒否 6 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 6 件が Tenancy の拒否である。

対象はテナント境界、システムコンソールへの越境、Hard Quota、テンプレート上書きの検証、テスト送信の宛先である。

`REQ-TENANCY-014` は通常のテナント管理者がシステムコンソールのテナント一覧へ到達しないことを宣言しており、素通りすれば他テナントの存在そのものが漏れる。

`REQ-TENANCY-018` はテスト送信が操作者本人にしか届かないことを宣言しており、素通りすれば任意の宛先への送信手段になる。

台帳が言えるのは「id を引用したテストが無い」ことだけで、「拒否を確かめるテストが無い」こととは違う。

`mise run report-coverage-debt` はこの差を named / nearby / none に分けるが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

この分類が別の Context で `REQ-APPLICATION-005` に対して指したのは、SAML 署名証明書とは無関係な配信一覧のカーソル検証テストだった。

そして注記だけを足す解消は、この項目では認めない。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の Tenancy の 6 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- 各テストのソースに対応する `REQ-TENANCY-NNN` を書く。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 防護の追加に新しい設計判断が要る場合は、その id だけを台帳に残し、引き取る work item を切って `depends_on` でつなぐ。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲はここに挙げた id に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ `REQ` id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- システムコンソールの認証強度と再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- システムコンソールのテナント横断読出しの有界化。
  [[wi-469-bound-system-console-cross-tenant-reads]] が扱う。
- Quota の値と算入対象そのものの変更。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

制御面の拒否は、テナント管理者のトークンを付けたシステムコンソールの経路から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-TENANCY-001 | 別テナントの realm を指定した参照を認めず、秘密を返さない | 応答が解決済みテナントの情報だけであり、クライアントシークレット、API トークン、秘密鍵を含まない |
| REQ-TENANCY-005 | 不完全な branding 入力を `InvalidRequestError` で拒否 | branding が保存されておらず、以後の画面がシステムデフォルトを使う |
| REQ-TENANCY-013 | Hard Quota 超過のリソース作成を `QuotaExceededError` で拒否 | リソースが作成されておらず、使用量が増えていない |
| REQ-TENANCY-014 | 通常のテナント管理者のシステムコンソール一覧参照を拒否 | 応答に他テナントが 1 件も含まれない |
| REQ-TENANCY-017 | 許可されない差し込み変数、片方だけの本文、カタログ外の locale を拒否 | 上書きが保存されておらず、既存の上書きも変わっていない |
| REQ-TENANCY-018 | 検証済みメールアドレスを持たない操作者のテスト送信を拒否 | メールが送信されていない |

`REQ-TENANCY-013` は、拒否が「作成しない」だけでなく「使用量を増やさない」ことまで含む。

使用量だけ増える実装は、以後の正当な作成まで拒否する形で被害が持続するため、両方を読む。

`REQ-TENANCY-018` の「送信されていない」は、送信の抽象の呼び出し回数ではなく、テスト用のメール送信先に何も届いていないことで確かめる。

`REQ-TENANCY-001` は否定形の宣言であり、応答に秘密が含まれないことをフィールド単位で確かめる。

### 却下した進め方

**named の 4 件を注記だけで閉じる案。**

分類の根拠は同じパッケージへ到達する拒否テストにエラー型名が現れることだけである。

**`REQ-TENANCY-014` を認可関数の単体テストで確かめる案。**

システムコンソールの経路は制御面の別ルート上にあり、拒否はその配線に宿る。

関数が正しくても呼ばれていなければ素通りする。

**Quota の拒否を使用量カウンタの単体テストで確かめる案。**

拒否の効果は作成の不在であり、カウンタの値はその一部にすぎない。

## Plan

1. 6 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 制御面とテナント設定の既存テストを調べ、拡張で足りるか新規が要るかを決める。
3. 越境の 2 件 (001、014) から着手し、テストが RED になることを先に観測する。
4. 入力検証の 2 件 (005、017) を進める。
5. Quota とテスト送信の 2 件 (013、018) を進める。
6. 防護そのものが無い id が見つかった場合は実装するか、引き取る work item を切る。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 6 件の入口と「変わっていないこと」を確定し、既存テストの流用可否を判断する。
- [ ] T002 [Acceptance] 越境の拒否 (001、014) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] 入力検証の拒否 (005、017) を効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] Quota とテスト送信の拒否 (013、018) を効果まで確かめるテストを書く。
- [ ] T005 [App] 拒否の実装が欠けていた経路を修正する、または引き取る work item を切って `depends_on` でつなぐ。
- [ ] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

制御面の拒否は [[wi-468-system-console-privileged-session-assurance]] と [[wi-469-bound-system-console-cross-tenant-reads]] が触れる範囲と経路を共有する。

同時に進める場合はテストの置き場所が重なるため、先に入った側へ合わせる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
