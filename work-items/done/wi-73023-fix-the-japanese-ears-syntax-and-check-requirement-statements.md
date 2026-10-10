---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 要件文の構文を改め、仕様の検査を加えるだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/development/format-rationale.md
  source:
    - tools/check/src/specification-rules.ts
    - tools/check/src/check-specification-rules.ts
    - tools/check/src/gherkin-scenarios.ts
  tests:
    - tools/check/src/specification-rules.test.ts
  stop_before_reading: [frontend, backend]
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-001 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-002 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-003 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-006 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-007 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-008 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-010 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-011 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-013 }
  - { path: docs/domain/identity-management/admin-access/README.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-016 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-017 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-018 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-019 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-020 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-021 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-022 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-023 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-024 }
  - { path: docs/domain/identity-management/admin-access/README.md, requirement: REQ-IDMANAGEMENT-025 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-027 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-028 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-030 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-031 }
  - { path: docs/domain/identity-management/roles/README.md, requirement: REQ-IDMANAGEMENT-032 }
  - { path: docs/domain/identity-management/roles/README.md, requirement: REQ-IDMANAGEMENT-033 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-034 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-035 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-036 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-037 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-038 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-040 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-044 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-045 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-047 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-048 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-050 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-051 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-052 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-054 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-055 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-056 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-057 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-058 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-059 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-061 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-062 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-063 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-064 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-065 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-066 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-067 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-068 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-069 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-070 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-071 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-072 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-074 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-075 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-076 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-077 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-078 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-079 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-080 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-081 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-082 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-084 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-085 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-086 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-087 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-088 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-001 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-002 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-003 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-004 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-005 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-006 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-007 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-008 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-009 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-010 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-011 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-012 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-013 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-014 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-015 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-016 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-017 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-018 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-019 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-020 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-021 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-022 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-023 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-024 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-025 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-028 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-029 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-030 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-031 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-033 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-034 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-035 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-036 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-038 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-039 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-040 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-041 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-042 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-043 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-044 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-045 }
---

# 要件文を、応答の主体と前置きの標識を固定した日本語の EARS で書き、`check-spec` で検査する

## 動機

`SPECIFICATION_FORMAT.md` は要件文を EARS 形式で書くと定めるが、日本語の型が EARS の制約をほとんど含まず、実際にはどの文でも書ける。

| EARS の規則（Mavin） | 現在の日本語の型 | 起きていること |
| --- | --- | --- |
| 応答する主体（system name）をちょうど一つ書く | 主体を書かない | 常時の型「〈対象〉は、〈結果〉。」に、主語と述語のあるどの平叙文も当てはまる |
| 前置きは閉じた集合のキーワード（While、When、Where、If…then）で始め、時間の順に並べる | 文末の「とき」「場合は」「の間は」「では」で見分けるだけで、順序の決まりがない | 「〜とき、〜場合だけ、実行する」のように、契機と条件が応答の中に混ざる |
| 契機は高々一つ、前提は零個以上 | 制限がない | 一行に条件がいくつも入り、どれが契機かが読めない |
| `shall` で義務を示す | 文末が「〜する」「〜できる」「〜とする」「〜である」と混ざる | 能力や定義の文が要件文として残る |

Tenancy と IdManagement の書き直し（wi-71372、wi-83002）も、この型で書いた。
wi-35451 で残る 19 Context の約 260 件を同じ型で書き直す前に、型と検査を直す。

## 対象範囲

- `SPECIFICATION_FORMAT.md` の EARS 形式の節を、下の設計の構文に改める。
  `docs/development/format-rationale.md` に、各規則の理由と参考にした資料を書く。
- `check-spec` に要件文の構文の検査を加える。
  検査は Context の一覧に載せた Context にだけ適用し、一覧は増える方向にだけ変える。
- IdManagement と Tenancy の要件文を新しい構文で書き直し、検査の一覧に載せる。

## 対象外

- 残る 19 Context の書き直し。
  wi-35451 が、この記録の構文で行い、Context を検査の一覧に加える。
- 要件の義務の変更。
  書き直しで見つけた未記載の振る舞いや食い違いは、仕様にない振る舞いの分類に従って起票し、この記録では扱わない。
- 例の付録（`acceptance.feature.md`）の文体。
  Gherkin の Given、When、Then が構造を与えている。

## 設計

### 参考にした資料

- [EARS（Alistair Mavin）](https://alistairmavin.com/ears/)：要件は、零個以上の前提、零個か一個の契機、ちょうど一つの主体、一つ以上の応答からなり、節は常に時間の順に並ぶ。
- [EARS 記法の日本語テンプレート（ISSOH）](https://www.issoh.co.jp/tech/details/11489/)：日本語は述語が文末に来るので「条件 → 主体 → 応答」の順と相性がよい。一方、「〜場合」一語で状態、契機、異常、構成のすべてを書けてしまうことが日本語版の最大の曖昧さだと指摘し、型ごとに標識を書き分けることを勧める。主体には一般名詞ではなく応答に責任を持つ具体名を書き、義務の文末は一つに統一する。
- 南山大学の日本語版 EARS の卒業論文（2021 年、佐伯研究室）も、日本語の特性に合わせて EARS を拡張する必要を述べている。本文は公開先の PDF が読めなかったので、この記録の判断には使っていない。

### 構文

要件文は、前置き、主体、応答をこの順に並べた一文とする。

```text
[〈構成〉では、][〈状態〉の間、…][〈契機〉とき、 | 〈条件〉場合、]〈主体〉は、〈応答〉。
```

| 型 | 書き方 | 例 |
| --- | --- | --- |
| 常時 | 〈主体〉は、〈応答〉。 | ApiTokens は、発行した JWT を保存しない。 |
| 状態 | 〈状態〉の間、〈主体〉は、〈応答〉。 | User が `Active` 以外の間、IdManagement は、その User の新規のサインインを拒否する。 |
| 契機 | 〈契機〉とき、〈主体〉は、〈応答〉。 | 管理者が User を無効化したとき、IdManagement は、`UserDisabled` を発行する。 |
| 望まない入力 | 〈条件〉場合、〈主体〉は、〈応答〉。 | `expiry_days` が 0 以下の場合、ApiTokens は、400 と `invalid_request` で拒否し、トークンを発行しない。 |
| 構成 | 〈構成〉では、〈主体〉は、〈応答〉。 | パスワードの再利用を禁止するテナントでは、Authentication は、履歴にあるパスワードへの変更を拒否する。 |
| 複合 | 〈構成〉では、〈状態〉の間、〈契機〉とき、〈主体〉は、〈応答〉。 | `Disabled` のテナントの間、管理者が再開を要求したとき、Tenancy は、テナントを `Active` にする。 |

| 規則 | 内容 | 検査 |
| --- | --- | --- |
| 主体 | 主体は、その機能仕様が属する Context の `README.md` の H1（`ApiTokens`、`IdManagement` など）とし、「〈主体〉は、」の形で一度だけ書く | 機械 |
| 前置きの標識 | 構成は「では、」、状態は「の間、」、契機は「とき、」、望まない入力は「場合、」で終える。「場合」は望まない入力の標識にだけ使い、状態、契機、構成には使わない | 機械 |
| 前置きの順序と数 | 構成、状態、契機または望まない入力の順に並べる。構成と状態は零個以上、契機と望まない入力は合わせて零個か一個とする | 機械 |
| 応答 | 応答は一つ以上の動作を「、」でつないで書いてよい。応答の中に条件（「とき」「場合」「の間」「なら」「限り」「〜ば」）を書かない | 機械 |
| 義務 | 応答は動作の現在形（「〜する」「〜しない」「〜を返す」など）で終える。`REQ-*` の見出しが規範の効力を与えるので、「しなければならない」は重ねない。「できる」「てもよい」「望ましい」「である」「とする」で終えない | 機械 |
| 一文一要件 | 一行に一文だけを書く。同じ前置きの応答が長くなる場合は、前置きを繰り返して行を分ける | 機械（行の中の「。」） |
| 定義を書かない | 値の定義や正規化（「〜の値は、〜とする」）は要件文にせず、モデルの節、値オブジェクト、TypeSpec に書く | 機械（「とする」の禁止） |
| 表の行 | 条件の組み合わせの表の行は、検査の対象にしない。表の直前に、表を参照する要件文を置く | レビュー |

### 判断

- **主体に Context 名を使う。**
  EARS の主体は応答に責任を持つシステムの名前であり、仕様の木で応答に責任を持つ最小の単位は Context である。
  名前はすでに H1 にあるので、構文の検査は文書の外の設定を読まずに済み、`SPECIFICATION_FORMAT.md` を別のリポジトリへ切り出しても同じ規則が働く。
  主体を必須にすると、常時の型がどんな平叙文も受け入れる抜け穴がふさがり、「User の基本項目は〜」のような定義の文が主体の欠落として検出される。
  画面の表示も、その画面を持つ機能の Context を主体にする。
- **望まない入力の標識を「場合、」にする。**
  ISSOH の記事は「もし〜ならば」を勧めるが、既存の要件文は拒否を「〜場合は、」で書いており、「もし」を付けても情報は増えない。
  曖昧さの原因は「場合」が型を問わず使われることなので、「場合」を望まない入力の標識に限ることで同じ効果を得る。
  「場合は、」の「は」は落とし、ほかの標識と同じく読点の直前を標識にする。
- **応答を一つに限らない。**
  Mavin の規則は一つ以上の応答を認める。
  拒否の要件文には、呼び出し元が観測する内容と変わらない内容の両方を書く既存の規則があり、二つを別の行に分けると前置きを繰り返すだけになる。
  代わりに、条件が応答の中へ入り込むことを禁じる。現在の書き方を自由な自然言語にしていた主因は、応答の数ではなくこちらである。
- **義務の文末に「しなければならない」を使わない。**
  要件文は `REQ-*` の見出しの下の箇条書きにだけ現れ、そこに置かれた文はすべて規範である。
  文末の統一は、現在形に限り、能力、許可、推奨、定義の文末を禁じることで得る。

### 採らなかった案

| 案 | 利点 | 欠点 | 採らない理由 |
| --- | --- | --- | --- |
| 英語のキーワードを前置きに残す（「When 管理者が〜とき、」） | 型が一目で分かり、検査も容易 | 日本語の文に英語の語が混ざり、文末の標識と二重になる | 文末の標識だけで型を機械的に判定できる |
| 主体を製品名（`IdMagic`）に固定する | 主体の選択に迷わない | すべての文で同じ語になり、どの Context の責任かを示さない | 主体を書く意味が、型の検査のための記号だけになる |
| 要件ごとに一つの応答に限る | 要件文が短くなる | 拒否の行が倍になり、前置きを繰り返す | Mavin の規則にない制約であり、自由さの原因を除かない |
| 構文の検査を置かずレビューに任せる | 実装が要らない | 現在の書き方がそうして崩れた | 約 400 件の要件を人が同じ基準で読み続けられない |

### 検査

`tools/check/src/specification-rules.ts` の既存の要件の検査（曖昧な語の検出）と同じ場所に、要件文の構文の検査を加える。

- 対象は、検査の一覧に載せた Context の機能仕様で、`REQ-*` の見出しの下の、欄で始まらない箇条書きの行である。
  廃止した要件、表の行、バッククォートの中は対象にしない。
- 行を「、〈主体〉は、」で前置きと応答に分け、前置きを末尾から標識で節に分けて、型、順序、数を判定する。
- 違反は要件の ID と、どの規則に反したか（主体の欠落、未知の前置き、前置きの順序、応答の中の条件、禁じた文末）を報告する。
- テストの判定の基準は「失敗しない」ではなく、規則ごとの違反の例を一つずつ拒否し、上の表の例をすべて受理することとする。
- 一覧は `check-unspecified-vocabulary` と同じく、書き直しを終えた Context を加える形で広げる。

### 実装中に決めたこと

- 書き直しの途中で、検査が見逃す形が二つ見つかった。どちらも規則に加え、先にテストで違反の例を置いてから検査を直した。
  - 「〜がなければ、〜を作る」のように、仮定の「〜ば」で応答の中に条件を入れる形。応答の中の条件の語に「〜ば」（「れば」「ければ」）を加えた。
  - 一行に「。」で区切った二つの文を書き、二文目を主体のない自由な文にする形。行の中の「。」を拒否し、一文一要件を機械の検査にした。括弧の中の「。」は数えない。
- 書き直しは、まず「場合は、」を「場合、」に置き換え、最後の前置きの標識の後に主体を差し込む機械的な置換を行い、残りを一行ずつ書き直した。機械的に主体を差し込んだ行も読み、主語が二重になった行（「IdManagement は、更新後のプロフィールは〜」など）を直した。
- 値の定義と制約の列挙（「式は 4,096 バイト以下とする」など）は、違反を拒否する望まない入力の要件文に書き換えた。拒否の応答はもとの要件文の拒否の行が述べていたものを使い、新しい応答を加えていない。
- 表の直前には、表を参照する要件文を加えた（REQ-TENANCY-036、REQ-IDMANAGEMENT-037、038）。
- REQ-TENANCY-013 のデータエクスポートへの参照は、要件文ではなく判断の欄に移した。応答はデータエクスポートの要件が定めており、この要件の義務ではないためである。

## 計画

1. `SPECIFICATION_FORMAT.md` と理由の文書を改める。
2. 検査を、規則ごとの違反の例と受理の例のテストから作る。
3. Tenancy、続いて IdManagement の要件文を書き直し、それぞれ検査の一覧に加える。
   書き直しは、型を当てはめる中で義務を変えないことを、`mise run spec-diff` で要件ごとに確かめる。
4. wi-35451 は、この記録の完了後に再開する。
   wi-35451 ですでに書き直した ApiTokens と ClaimMapping の要件文は、その再開時に新しい構文で書き直す。

未解決の問い：なし。
主体、標識、応答の数、文末は上の判断で決めた。

## タスク

- [x] T001 [Spec] `SPECIFICATION_FORMAT.md` の EARS 形式の節と、理由の文書を改める。検査：`mise run check-spec`。
- [x] T002 [Tooling] 要件文の構文の検査を加える。RED：`mise run test-tools-file -- check/src/specification-rules.test.ts` が、`verifyEarsStatements` のない状態で 9 件失敗した。
- [x] T003 [Spec] Tenancy の要件文を書き直し、検査の一覧に加える。RED：一覧に加えた時点で `mise run check-spec` が 170 行を拒否した。
- [x] T004 [Spec] IdManagement の要件文を書き直し、検査の一覧に加える。RED：一覧に加えた時点で `mise run check-spec` が 363 行を拒否した。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run spec-diff` で、要件の差分が要件文の書き直しだけであり、ID とタイトルの変更、要件の追加と削除がないこと。
- `mise run verify`

## リスク

- 型に当てはめる書き直しで、要件の義務が落ちる、または増える。
  要件ごとに書き直しの前後を突き合わせ、条件を応答から前置きへ移すときに条件を落とさない。
- 文末と標識の機械的な判定が、正しい文を誤って拒否する。
  Tenancy と IdManagement の書き直し後の全文を受理することを検査のテストの条件にし、誤検出があれば規則を直す。
- 前置きを繰り返す行が増え、仕様が長くなる。
  書き直しの前後の行数を完了の節に残し、wi-35451 の見積もりに使う。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果、規範の差分は REQ-IDMANAGEMENT と REQ-TENANCY の 131 要件の本文の書き直しだけであり、要件の追加、削除、ID とタイトルの変更はなく、`affected_spec` と一致した。
  `SPECIFICATION_FORMAT.md` の EARS 形式の節を、前置き、主体、応答の順の日本語の構文に改めた。主体は Context の `README.md` の H1、前置きの標識は「では、」「の間、」「とき、」「場合、」に固定し、「場合」を望まない入力に限った。応答の中の条件と、能力、許可、推奨、定義の文末を禁じ、一行に一文だけを書く。理由と参考にした資料は `docs/development/format-rationale.md` に書いた。
  `check-spec` に要件文の構文の検査を加え、Tenancy と IdManagement を検査の一覧に載せた。残りの Context は wi-35451 が書き直して加える。
  対象の 20 文書は 2,017 行から 2,060 行になった。前置きを繰り返して行を分けた分だけ増えた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 要件文の構文の変更であり、製品の要件の義務を変えない。
  - **Observed Failure**: Tenancy と IdManagement を検査の一覧に加えた時点で、要件文 533 行（Tenancy 170 行、IdManagement 363 行）を「statement names no responder」として拒否した。
  - **Detection Reason**: 主体のない文を常時の型として受け入れていた抜け穴を、実際の仕様の全文に対して検出する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/specification-rules.test.ts`
  - **Requirement**: N/A: 仕様の検査の追加であり、製品の要件はない。
  - **Observed Failure**: `verifyEarsStatements` のない状態で、構文の規則ごとの 9 件のテストが失敗した。書き直しの途中で加えた「〜ば」の条件のテストは、検査を直す前に `Expected - 1` で失敗した。
  - **Detection Reason**: 規則ごとに違反の例を一つずつ拒否し、すべての型と複合の前置きを受理することを表明するので、規則の欠落と誤検出の両方を区別する。
- **Change-Resistance Results**:
  検査に手で故障を入れ、テストが検出することを確かめた。前置きの順序の判定の除去、二つ目の契機の許容、「場合」の予約の除去、主体の二重の許容、応答の条件の語「限り」の除去の 5 件は、それぞれ対応するテストが失敗した。
  配線の故障として、Tenancy の要件文から主体を消すと、`mise run check-spec` が `REQ-TENANCY-003 statement names no responder` で失敗した。
  TypeScript の検査には変異のツールがないので、体系的な変異は行っていない。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run spec-diff -- main` - 成功（131 要件の本文の変更だけ）
  - `mise run check-work-items` - 成功
  - `mise run test-tools` - 成功
  - `mise run verify` - 成功（1 回目は理由の文書の「日本語版」が `version を「版」と書かない` の検査で失敗し、言い換えて成功）
