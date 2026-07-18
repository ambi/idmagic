---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-120: イベント relay を `Publisher` 抽象で transport 中立化し、Pub/Sub を build タグ隔離の任意アダプタとする

## コンテキスト

[[ADR-094-transactional-event-log-and-audit-projection]] で採用した transactional
outbox は、アプリが `outbox` テーブルへ書き、別プロセス `idmagic-relay` が
skip-locked で drain して外部ストリームへ転送する。転送先は当初 Kafka
（`twmb/franz-go`）にハードコードされ、`idmagic-relay` は broker が常時必要だった。

これがデプロイのコスト構造を歪めた。マネージド Kafka は「最小 3 ブローカーの
クラスタ固定費」があり、outbox のイベント量（監査・ライフサイクル等の低〜中量）に
対して過大で、より重要でデータ量も大きいセッション/一時状態用の Valkey よりも
月額が高くなる、という逆転が起きる。加えて次の運用要件が明確になった。

- **ローカル/オンプレでも動かす**。broker を用意できない/したくない環境がある。
- **クラウド先で安く動かす**。GCP ではサーバレスな Pub/Sub がフロア無しで安い。
- ただし **コードベースを特定クラウド（GCP）に限定・依存させたくない**。

調査の結果、この差し替えは安全に行えることが分かった: Kafka 依存は relay の
1 ファイルに閉じ、リポジトリ内に Kafka コンシューマは無く、トピック名は行ごとに
`outbox.topic` へ保存され broker 非依存、`published_to` は自由文字列カラムである。

## 決定

relay を **transport 中立の `Publisher` 抽象**に整理する。drain ループ
（skip-locked 読み出し・ack・失敗時 `attempts`/`last_error` 更新）は転送先を
知らず、`Publisher.Publish` と `Publisher.Name()` にのみ依存する。

- **default は `kafka`**。既存挙動を変えない。オンプレ/自ホストは自ホスト
  Redpanda/Kafka を指すだけで動く。
- **`log` sink** を追加する。broker を用意せず outbox を drain したいローカル/
  オンプレ最小構成向けに、イベントをログ出力して ack する。消費側が要るように
  なったら `kafka`/`pubsub` へ切り替える。
- **Pub/Sub アダプタは build タグ `//go:build pubsub` で隔離**する。既定ビルド
  （ローカル/オンプレ/dev）は GCP SDK を一切コンパイルしない。GCP デプロイ時のみ
  `-tags pubsub` でビルドし `RELAY_SINK=pubsub` を選ぶ。非タグビルドで `pubsub`
  を要求したら明示エラーにする。
- 選択は `RELAY_SINK`（`kafka` | `pubsub` | `log`, default `kafka`）で行う。
- delivery semantics は不変（at-least-once、per-aggregate ordering)。ordering は
  Kafka では partition key、Pub/Sub では ordering key に同じ `partitionKey` を割る。

これにより「特定クラウドへの依存」は build タグ付きの 1 アダプタに閉じ込められ、
既定のコードパス・バイナリはどのクラウドにも縛られない。

## 却下した代替案

- **Kafka のままマネージドを使い続ける**: コスト逆転が解消せず、ローカル/オンプレの
  broker 必須要件も残る。
- **Pub/Sub を無条件（build タグ無し）でコンパイルし runtime 選択のみにする**: 実装は
  単純だが GCP SDK が全ビルド・全環境に常時入り、「クラウド非依存」要件を満たさない。
- **Pub/Sub 実装を別 Go module に切り出す**: 依存分離は最も厳密だが、単一 module の
  モジュラーモノリス構成（[[ADR-092-backend-and-frontend-top-level-directories]]）に
  対して過剰で、build タグで要件は十分満たせる。
- **Kafka を Pub/Sub で完全置換（抽象なし）**: 将来 Kafka 前提の外部連携が現れたときに
  戻せず、vendor lock-in を Kafka→GCP に付け替えるだけになる。

## 影響

- **SCL 変更なし**。転送先は adapter 層の関心で、`spec/scl.yaml` は event transport を
  規定していない。振る舞い契約（at-least-once/ordering）も不変。
- 変更は relay バイナリと共有 eventsink adapter に限定される。アプリ/writer 側、
  `outbox` テーブル、`OutboxEventSink`、イベント種別→トピックの対応、全ドメイン
  モジュールは不変。`published_to` カラムに `kafka`/`pubsub`/`log` が入るようになる
  （既存の自由文字列カラムで表現でき、スキーマ変更なし）。
- 運用: `RELAY_SINK` と、`kafka` の `KAFKA_BROKERS` / `pubsub` の `PUBSUB_PROJECT` が
  新たな構成軸になる。GCP イメージは `-tags pubsub` でビルドする（`ARCHITECTURE.md`
  のデプロイ構成、`infra/deploy/gcp/` を参照）。dev の compose は default `kafka`
  （Redpanda）のまま GCP 依存を持ち込まない。
- relay drain ループは従来テストが無かったため、本 ADR の実装で fake `Publisher` を
  用いた test-first の回帰テストを整備する（[[ADR-119-test-first-discipline-for-behavior-layers]]）。
