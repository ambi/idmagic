---
depends_on: []
status: completed
authors: [tn]
risk: high
created_at: 2026-08-27
priority: p1
change_kind: docs
spec_impact: { kind: none, reason: "脅威モデルの正本を新設して既存の制御へ対応づけるもので、既存の規範シナリオ、規範 ID、TypeSpec シンボルを変更しない。" }
evidence_policy: risk-based-v2
approval:
  by: tn
  at: 2026-08-27
  scope: "docs/threat-model.md を正本文書として新設し、ROOT_DOCUMENTS と SPECIFICATION_FORMAT.md の配置図と docs/README.md の索引へ登録する。あわせて SPECIFICATION_FORMAT.md へ、脅威モデルという文書種の形式を定める節を新設する。信頼境界の所有を docs/deployment.md から docs/threat-model.md へ移す。分類は STRIDE 単独とし LINDDUN は併用しない。脅威 ID は THREAT-NNN の不変な連番とする。信頼境界は 11 個すべてを対象とする。Controls 列は既存の規範 ID と正本文書および runbook の規則だけを参照し、新しい ID 体系も、まだ存在しない制御への前方参照も置かない。更新の契機を DEVELOPMENT.md と update-design skill へ接続する。制御の欠落は表の中で可視化するにとどめ、是正は実装しない。攻撃手法の詳細と再現手順は書かない。"
  baseline: "76d23ce85050315cd9f50f4a8a55b7a8f1da5bcf"
initial_context:
  specification:
    - docs/authorization.md
    - docs/deployment.md
    - docs/standards.md
    - docs/capacity.md
    - docs/database.md
    - docs/observability.md
    - docs/scenarios.md#REQ-PLATFORM-001
    - docs/contexts/authentication/scenarios.md#REQ-AUTHENTICATION-009
    - docs/contexts/authorization/scenarios.md#REQ-AUTHORIZATION-005
    - docs/contexts/tenancy/scenarios.md#REQ-TENANCY-009
    - docs/contexts/oauth2/scenarios.md#REQ-OAUTH2-015
    - docs/contexts/data-keys/scenarios.md#REQ-DATAKEYS-005
    - docs/contexts/workloadidentity/scenarios.md#REQ-WORKLOADIDENTITY-002
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-006
    - docs/contexts/seeding/scenarios.md#REQ-SEEDING-005
  typespec: []
  source:
    - tools/check/src/specification-doc.ts
    - tools/check/src/security-controls.ts
  tests:
    - tools/check/src/specification-doc.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec/generated
---

# 脅威モデルを仕様の一種として持ち、制御の欠落を検出できるようにする

## Motivation

この製品はアイデンティティプロバイダーであり、認証、認可、テナント境界、暗号鍵、プロトコル互換を扱う。それにもかかわらず、`docs/` に脅威モデルの正本が無い。`docs/authorization.md` は主体とスコープとテナント境界を定め、`docs/standards.md` は準拠する外部規範を宣言し、`docs/deployment.md` は実行単位と配備の構成を示すが、いずれも「何が攻撃されうるか」を列挙したものではない。`docs/README.md` の索引は `docs/deployment.md` が信頼境界を持つと述べていたが、実際には同文書に信頼境界の記述は無く、宣言だけが残っていた。

その結果、検証の効き方に構造的な偏りが生じている。`DEVELOPMENT.md` の「Testing a refusal」と `checkRefusalCoverage` は、実装された制御が正しく拒否することをきわめて強く保証する。「拒否と応答してから操作を実行してしまう実装」を捕まえるところまで踏み込んでおり、実際にそれが出荷されて全行被覆を保ったまま生き延びた事例まで記録されている。しかしこの仕組みが答えられるのは「宣言された制御が働いているか」だけである。**制御そのものが最初から無い**場合、宣言も無く、シナリオも無く、テストも無く、`security-refusal-debt.json` にも載らない。何も落ちない。

`SPECIFICATION_FORMAT.md` は「拒否が書かれていない制御は、照合すべき記述を持たない。実装が拒否をやめても何の記述とも矛盾せず、仕様は製品が後退したと言えない」と述べている。この洞察は正しく、そして一段上にも同じ形で当てはまる。**脅威が書かれていなければ、制御の欠落は何の記述とも矛盾しない。**

`security-refusal-debt.json` という負債の明示管理が既にあることは良い兆候である。足りないのはその上流、すなわち「どの脅威に対してどの制御が応えるか」の対応であり、それがあって初めて「応える制御が無い脅威」を検出できる。

## Scope

- **脅威モデルの正本**：`docs/` に脅威モデルを置く。信頼境界、資産、主体、脅威、対応する制御、受容した残余リスクを持つ。
- **分類の枠組み**：STRIDE を基本とし、個人データの扱いについては LINDDUN の privacy に関する分類を併用するかを判断する。
- **既存資産との接続**：脅威の各項目を、対応する `REQ-<CONTEXT>-NNN`、`standards.md` の規範 ID、`docs/authorization.md` の境界のいずれかへ結びつける。結びつく先が無い脅威が、制御の欠落である。
- **欠落の扱い**：対応する制御が無い脅威を、負債として明示管理する。受容する場合は受容した理由と再検討の条件を書く。
- **更新の契機**：脅威モデルをいつ見直すかを `DEVELOPMENT.md` のループへ接続する。新しい信頼境界、新しい主体の種類、新しい外部連携が加わったときが契機になる。
- **粒度の決定**：全体の脅威モデルを 1 つ持つか、Context ごとに持つかを決める。

## Out of Scope

- 見つかった制御の欠落の実装。本 work item は脅威と制御の対応を作り、欠落を可視化するところまでを担う。個々の是正は脅威ごとに別の work item が扱う。
- 攻撃手法の詳細や再現手順の記録。脅威の記述は「何が起こりうるか」であって「どうやるか」ではない。
- 侵入試験や外部の監査の実施。
- 暗号プロトコルの形式検証。wi-421 が範囲外と判断している。

## Design

置き場所は `docs/` 直下とする。`SPECIFICATION_FORMAT.md` が `docs/authorization.md` について述べる理由がそのまま当てはまる。「認可を確認しに来た人が欲しいのは、製品の認可であって、1 つの Context の取り分ではない」。脅威も同じで、Context ごとに分割すると、複数の Context にまたがる攻撃経路——たとえばテナント境界を越える経路や、認証から認可へ主体が伝播する経路——を書く場所が無くなる。ファイル集合が閉じているため、`SPECIFICATION_FORMAT.md` の配置図と検査器の許可リストを同時に更新する。wi-415 が同じ操作を行うため、実装が前後する場合は結論を共有する。

分類の枠組みは STRIDE を採る。理由は、脅威の見落としを防ぐための網としては十分に粗く、かつこの製品の主要な関心（なりすまし、改竄、否認、情報漏洩、サービス拒否、権限昇格）を素直に覆うからである。攻撃木や攻撃ライブラリを採らないのは、網羅性の主張が弱く、維持の費用が高いためである。LINDDUN については、`docs/standards.md` が GDPR の消去と処理記録を既に規範として持ち、`docs/contexts/audit/` が個人識別情報の変換を持っているため、privacy の分類を全面的に導入する価値があるかを T002 で判断する。

脅威と制御の対応の形式は、`docs/capacity.md` の `SLO-*` に倣って安定 ID を持つ表とする。他の文書とテストがこの ID を参照でき、ID から `spec-where` で引ける。制御の側は既存の ID（`REQ-*`、規範 ID）を参照し、新しい ID 体系を制御の側に作らない。二重の命名を避けるためである。

粒度は全体で 1 つとする。ただし表の各行は影響する Context を列として持ち、Context ごとの読み方ができるようにする。

対応する制御が無い脅威の扱いは、`security-refusal-debt.json` の形式に倣う。ただし負債ファイルではなく脅威モデルの表の中に「対応する制御なし」の状態として書く。負債を別ファイルへ出すと、脅威の一覧を読んだ人が欠落に気づかない。欠落こそがこの文書の最も重要な出力である。

未解決の論点として、脅威モデルの被覆を機械検査するかどうかは決めない。wi-418 が規範とシナリオの被覆を検査する仕組みを作るため、その仕組みが脅威 ID にも使えるかは wi-418 の結論を見てから判断する。

## Plan

1. `docs/authorization.md` の主体とテナント境界、`docs/deployment.md` の実行単位と配備の構成を出発点に、資産と境界を洗い出す。
2. STRIDE の各分類を境界ごとに当て、脅威を列挙する。列挙は網羅を目指すが、完全性は主張しない。
3. 各脅威を既存の `REQ-*`、規範 ID、`docs/authorization.md` の記述へ結びつける。結びつかないものを欠落として記録する。
4. 欠落のそれぞれについて、是正するか受容するかを判断する。受容する場合は理由と再検討の条件を書く。
5. LINDDUN の併用の要否を判断する。
6. 更新の契機を `DEVELOPMENT.md` のループへ接続する。
7. 是正すると判断した欠落を、別の work item として切り出す。

## Tasks

- [x] T001 [Baseline] 資産、信頼境界、主体を既存文書から洗い出す。11 の信頼境界と 11 の資産を `docs/threat-model.md` に記録した。
- [x] T002 [Spec] STRIDE で脅威を列挙し、LINDDUN の併用の要否を判断する。83 件を列挙し、LINDDUN は併用しないと判断して理由と再検討の条件を書いた。
- [x] T003 [Tooling] `SPECIFICATION_FORMAT.md` の配置図と検査器の許可リストへ新しいファイル名を加え、あわせて §8 として文書種の形式を定めた。§8 の追加は承認範囲を超えるので `Post-Approval Changes` に記録した。wi-415 とは、正本文書のファイル集合を広げるという結論を共有する。
- [x] T004 [Spec] 脅威と制御の対応表を書き、結びつかない脅威を欠落として記録する。`covered` 70、`planned` 8、`accepted` 5。
- [x] T005 [Spec] 欠落ごとに是正か受容かを判断し、受容には理由と再検討の条件を添える。
- [x] T006 [Spec] 更新の契機を `DEVELOPMENT.md` の現在状態の同期と `update-design` skill へ接続する。
- [x] T007 [Verify] 是正すると判断した欠落を別の work item として切り出す。wi-426、wi-427、wi-428、wi-429 を起票した。THREAT-074 は wi-417、THREAT-082 は wi-100 と wi-291 が既に担う。表から前方参照を除いたため、対応は work item の側が `THREAT-NNN` を名指しする向きで持つ。THREAT-082 だけはその逆参照を持たない（理由は `Post-Approval Changes` にある）。

## Independent Verification Findings

新しい文脈のエージェントによる独立検証で 14 件の指摘を受け、うち実質的な欠陥 13 件を是正した。検証者は識別子の実在性 139 件をすべて再確認したうえで、**引用の妥当性**に踏み込んだ。以下は是正した内容である。

- **THREAT-068 の制御 3 つがいずれも脅威に応えていなかった。** `REQ-PROVISIONING-001` は管理 API のスコープ、`REQ-PLATFORM-003` は配信のトランザクション、`GDPR-PROCESSING-RECORDS` は事後の記録であり、「誤った接続先へ送る」ことを防ぐものは 1 つも無かった。同じファイルに実在する `REQ-PROVISIONING-002`（`base_url` の https と内部 IP の拒否）、`REQ-PROVISIONING-015`（テナント境界）、`REQ-PROVISIONING-018`（フェイルクローズ）へ差し替えた。
- **THREAT-072 の `covered` が過大主張だった。** `docs/contexts/jobs/internals.md` は「`JobKind` ごとの品質の制御も、利用側ごとの順序や流量の制限も提供しない」と明言しており、同型の THREAT-022 を `planned` としながらこちらを `covered` とする根拠が無かった。`planned` へ移し、wi-427 の Scope を投入元別の偏りまで広げた。
- **`planned` が「何も無い」と「あるが規範化されていない」を区別できなかった。** THREAT-012、THREAT-062、THREAT-082 は制御が皆無ではなく、それぞれゲートウェイへの要求、runbook の指示、固定した依存バージョンが実在する。`Controls` にその部分的な保護を書き、応える規範が 1 つも無い行だけを `—` とする規則を定めた。欠落の可視化がこの文書の唯一の存在理由である以上、この混同は中心的な弱点だった。
- **THREAT-001 の `OIDC-CORE-CSRF` は別種の CSRF だった。** フェデレーションの callback における `state` の照合であり、ファーストパーティーのセッション CSRF には応えない。削除した。
- **THREAT-061 が `REQ-SYSTEM-016` を引き忘れていた。** 脅威はログ側にも及ぶが、引かれていたのは設定リファレンス側だけだった。
- **THREAT-032 が `optional` / `MAY` の規範を根拠に含めていた。** 提供しない構成では制御が存在しないことが表から読めないので削り、実質の制御である `REQ-AUTHORIZATION-007` だけを残した。
- **THREAT-034 の受容理由が不正確だった。** フェイルオープンは HIBP を使う追加の照合に限られ、同梱辞書の照合は確実に働く。限定を書き足した。
- **退役の手順が実行不可能だった。** 「見出しに後継を書く」は `scenarios.md` の規約をそのまま持ち込んだもので、行として書かれる脅威には見出しが無い。`Status` に `retired` を足し、`Threat` 列に後継を書く形へ改めた。
- **`Controls` の文法が実際の用法を覆っていなかった。** 節を伴わない文書参照と `wi-NNN` が未定義だった。前者を文法に加え、応える規範が無い行の `—` を定め、runbook を指す行は必ず `planned` になるという規則を明示した。後者は再承認で却下されたため、表から除いた。
- **表ヘッダー 3 つが日本語だった。** `CLAUDE.md` の言語表は table headers を英語と定める。英語へ直した。
- **`Category` 列の語彙が不揃いだった。** 原語と短縮形が混在していたので STRIDE の 6 語へ統一し、日本語の意味との対応表を置いた。
- **`docs/README.md` で信頼境界の所有者が二重になった。** しかも `docs/deployment.md` には信頼境界の記述が実在しなかった。索引と `SPECIFICATION_FORMAT.md` の配置図から `docs/deployment.md` の信頼境界を外し、`docs/threat-model.md` が単独で所有する形にした。
- **より直接的な経路が抜けていた。** `REQ-SOURCING-005` は「`GroupMembership` が同期され User の有効ロールが更新される」と定めており、上流から実効ロールへは動的規則を経ない直接の経路がある。THREAT-083 として追加し、wi-428 の Scope へ加えた。

是正しなかった指摘が 1 件ある。`DOCUMENTATION_GUIDE.md` が新しい文書種に追随していないという指摘は、承認範囲外であり、同文書が既にこのリポジトリに存在しない `reliability.md` と `recovery.md` を含むなど完全一致を意図していないため、本 work item では扱わない。

## Post-Approval Changes

独立検証が、承認範囲を超えた変更を 3 件指摘した。再承認を求め、次の結論を得た。

1. **`SPECIFICATION_FORMAT.md` に §8「Threat model」を新設した（旧 §8 は §9 へ繰り下げ）。** `approval.scope` は「配置図への登録」までしか認めていなかった。新しい文書種を足す以上その形式を定める節が要るという判断であり、**再承認された**。
2. **`docs/deployment.md` から信頼境界の所有権を外した。** 索引と配置図が、実在しない記述の所有を宣言していたための是正である。**再承認された**。
3. **`Controls` に `planned` 行の `wi-NNN` を書いていた。** 正本文書に、まだ存在しない制御への前方参照を持ち込むものであり、`docs/` 配下でこれを行う文書はほかに 1 つも無い。**却下された。** 表から `wi-NNN` を除き、応える規範が 1 つも無い行は `—` とした。

3 の帰結として、脅威と是正の対応は work item の側が `THREAT-NNN` を名指しする一方向だけで持つ。この向きのほうが腐りにくい。work item が完了して `work-items/done/` へ移っても、正本文書の側には壊れる参照が残らないからである。`mise run spec-where THREAT-NNN` は両方向を返すので、引ける状態は保たれる。

**ただし THREAT-082 だけは逆参照を持たない。** 担い手は `wi-100`（SBOM と署名）と `wi-291`（依存の脆弱性管理）だが、どちらも本 work item より前に起票されており、脅威 ID を書き加えることは承認範囲の外である。この 1 行だけは、表からも work item からもたどれない。是正するなら、その 2 件を触る別の変更として行う。

`mise run spec-diff 76d23ce85050315cd9f50f4a8a55b7a8f1da5bcf` は `no normative specification change` を返す。規範シナリオ、規範 ID、TypeSpec シンボルはいずれも変えていないので `spec_impact: none` は維持される。上の 3 件はいずれも `spec-diff` が見ない平面にあり、この検査では捕まらなかった。承認境界の逸脱を検出したのは検査ではなく独立検証である。

## Post-Approval Findings

実装中に、承認範囲の外にある欠陥を 1 件見つけた。是正は本 work item では行わず、wi-430 として切り出した。

`SPECIFICATION_FORMAT.md` は「The file set and the file names are *(checked)* for `docs/` and `docs/contexts/<context>/`; anything else at those two levels is not a canonical document and is rejected.」と述べるが、**拒否されない**。`tools/workspace/src/workspace.ts` の `scanCanonicalDocuments` は許可リストに一致する名前だけを拾う絞り込みであり、一致しないファイルは黙って無視される。したがって `docs/` 直下に置いた未登録の Markdown は、検証もされず、生成される仕様サイトにも現れず、何も落とさないまま存在できる。

これは本 work item の Acceptance RED の設計を直接覆した。当初は「新しいファイルを置くと `check-spec` が拒否する」ことを RED として予定していたが、実際には exit 0 で通る。代わりに、正本文書として不正な本文（二重の H1）を置いても検証されないことを RED とし、許可リストへの登録によって同じ本文が落ちるようになることを GREEN とした。この差し替えは規範の変更ではなく、証拠の境界の選び直しなので、再承認は要していない。

wi-418 が扱う「宣言されているが検査されていない」という同じ型の欠陥であり、`docs/standards.md` の被覆の主張と並ぶ 2 件目である。

## Verification

- `mise run check-spec` が新しい正本文書を受け入れる。
- 脅威モデルの全行が、対応する制御の ID を持つか、欠落として理由付きで記録されている。
- `docs/authorization.md` が述べる境界と、脅威モデルの信頼境界が矛盾しない。
- `mise run spec-where` で脅威 ID から関連箇所を引ける。
- `mise run verify`

## Risk Notes

脅威の列挙は網羅性を主張できない。主張してしまうと、書かれていない脅威が「検討済みで問題なし」と読まれる。文書の冒頭で「これは現時点で識別した脅威であり、完全な一覧ではない」と明示し、更新の契機を定めることで、静的な保証と誤読されることを防ぐ。

欠落が多数見つかった場合、文書自体が「対応していないことの一覧」になり、公開の是非が問題になる。この文書はリポジトリの内側にあり公開物ではないが、欠落の記述は攻撃の手掛かりになりうる。脅威の記述を「何が起こりうるか」に限り、再現手順を書かないという原則がこの危険を抑える。

脅威モデルは書いた直後から古くなる。更新の契機を `DEVELOPMENT.md` のループへ接続することが、飾りにしないための唯一の仕掛けである。接続先を決められないなら、この文書は作らないほうがよい。

## Completion

- **Completed At**: 2026-08-27
- **Summary**:
  製品に脅威モデルの正本が生まれた。83 件の脅威を 11 の信頼境界にわたって STRIDE で分類し、それぞれに応える制御を既存の規範 ID で名指しした。`covered` 70、`planned` 8、`accepted` 5。これまで、制御が実装されていることは `checkRefusalCoverage` が確かめられたが、制御がそもそも無いことは何とも矛盾せず検出できなかった。その 13 件が初めて記述として存在する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`docs/threat-model.md` に H1 を 2 つ持つ本文を置いた状態）
  - **Requirement**: N/A: 正本文書の登録は製品の振る舞いではないため、対応する規範シナリオを持たない。
  - **Observed Failure**: exit 0 で通過し、`docs/threat-model.md` は出力に一度も現れなかった。139 件の文書が検証されたが、この 1 件は対象外だった。当初は「未登録のファイルは拒否される」ことを RED として予定していたが、実際には拒否されないと判明したため、証拠の境界を「不正な本文が検証されない」ことへ選び直した。
  - **Detection Reason**: 二重の H1 は正本文書として明確に不正であり、検証されていれば必ず落ちる。落ちないことが、その文書が検証の対象外であることを一意に示す。登録後は同じ本文が `document must contain exactly one H1` で exit 1 になる。ファイル名が出力に現れるかどうかだけを見る弱い確認では、「一覧に載っているが本文は見られていない」実装と区別できない。
- **Unit RED Evidence**:
  - **Test**: `bun test check/src/specification-doc.test.ts` の `documentKind > names the grammar of each canonical document`
  - **Requirement**: N/A: 検査器の内部関数であり、製品の規範シナリオに対応しない。
  - **Observed Failure**: `expect(documentKind('docs/threat-model.md')).toBe('prose')` が `Expected: "prose" / Received: undefined` で失敗した（28 pass, 1 fail）。
  - **Detection Reason**: `documentKind` が `undefined` を返すことは、そのパスが正本文書として扱われないことと同義であり、`validateDocument` が本文の文法検査へ進まない直接の原因である。`ROOT_DOCUMENTS` へ 1 行足すと 29 pass, 0 fail になる。ファイルの存在や `check-spec` の総合結果ではなく、判定関数そのものを境界に置いたので、失敗の原因が許可リスト以外にありえない。
- **Post-Approval Changes**:
  独立検証が承認境界の逸脱を 3 件指摘し、再承認で 2 件が承認、1 件が却下された。詳細は `Post-Approval Changes` 節に記録した。`mise run spec-diff 76d23ce85050315cd9f50f4a8a55b7a8f1da5bcf` は `no normative specification change` を返す。
- **Independent Verification**:
  実装していない新しい文脈のエージェントが、仕様と標準の両面をレビューした。識別子の実在性 139 件を再確認したうえで**引用の妥当性**に踏み込み、14 件を指摘。うち 13 件を是正した。最も重いのは、THREAT-068 の制御 3 つがいずれもその脅威に応えていなかったこと（正しい `REQ-PROVISIONING-015` は同じファイルに実在した）、THREAT-072 の `covered` が `contexts/jobs/internals.md` の明示的な否定と矛盾していたこと、`planned` が「制御が皆無」と「制御はあるが規範化されていない」を区別できていなかったことである。是正しなかった 1 件は `DOCUMENTATION_GUIDE.md` の追随で、承認範囲外かつ同文書が既にこのリポジトリと一致していないため見送った。詳細は `Independent Verification Findings` 節にある。
- **Change-Resistance Results**:
  代表的な誤実装として「`ROOT_DOCUMENTS` への登録を欠いた実装」を明示的に注入した。`docs/threat-model.md` に二重の H1 を残したまま登録行を外すと `check-spec` は exit 0 で通り、ファイル名は出力に 0 回しか現れない。登録行を戻すと同じ本文が exit 1 で落ちる。したがってこの 1 行が、文書を検証下に置いている唯一の要因であることが確かめられた。あわせて、文書が自ら定めた規則を機械的に検査した。`covered` の行に work item が現れない（0 件）、`runbooks/` を引く行が `covered` にならない（0 件）、`THREAT-NNN` に重複が無い（0 件）、引用した規範 ID 114 件と節参照 15 件がすべて解決する。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
