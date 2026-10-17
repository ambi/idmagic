# Feature: アドミッションコントロールの例

## Rule: REQ-SYSTEM-018 飽和した API プロセスは優先度の低い要求から拒否する

### Background:

- Given 登録済みのすべての経路が `interactive_auth`、`management`、`management_bulk`、`infrastructure` のいずれか 1 つの優先度クラスに分類されている
- And 起動時設定が優先度クラスごとの同時実行の入場上限を持ち、`management_bulk` の上限は `management` の上限以下、`management` の上限はプロセス全体の上限以下である

### Example: EX-SYSTEM-018-01 通常経路

- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- Then System は `Retry-After` と `urn:idmagic:error:service_overloaded` の Problem Details を伴う 503 を返す
- Then 要求はハンドラーへ到達せず、永続状態もドメインイベントも変化しない
- When 同じ状態で APIConsumer が `interactive_auth` の経路へ要求を送る
- Then 要求はハンドラーへ渡り、通常どおり処理される
- When APIConsumer が `infrastructure` に分類された経路へ要求を送る
- Then System は実行中の要求数にかかわらず拒否せず、要求はハンドラーへ渡る
- When 実行中の要求数がどの入場上限にも達していない
- Then System はどの優先度クラスの要求も拒否しない

### Example: EX-SYSTEM-018-02 実行中の要求数が `management_bulk` の上限に達していない

- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- But 実行中の要求数が `management_bulk` の上限に達していない
- Then 要求はハンドラーへ渡り、通常どおり処理される

### Example: EX-SYSTEM-018-03 実行中の要求数がプロセス全体の上限にも達している

- When 実行中の要求数が `management_bulk` の上限に達している状態で、APIConsumer が `management_bulk` の経路へ要求を送る
- Then System は `Retry-After` と `urn:idmagic:error:service_overloaded` の Problem Details を伴う 503 を返す
- Then 要求はハンドラーへ到達せず、永続状態もドメインイベントも変化しない
- When 同じ状態で APIConsumer が `interactive_auth` の経路へ要求を送る
- But 実行中の要求数がプロセス全体の上限にも達している
- Then System は同じ 503 で拒否し、要求はハンドラーへ到達しないので状態を部分的に更新しない

## Rule: REQ-SYSTEM-019 RoutePriorityReference は分類の定義から生成され乖離を検出できる

### Example: EX-SYSTEM-019-01 通常経路

- Given 登録済みの各経路の優先度クラスが、経路の登録と同じ場所に一箇所で定義されている
- When RoutePriorityReference を生成する
- Then 生成物は組み立て済みの経路それぞれについて、経路パターン、メソッド、属する優先度クラスを示す
- Then 生成物は優先度クラスごとに、対応する縮退のステージと、その上限を与える起動時設定のキーを示す
- When 生成物と分類の定義を突き合わせる
- Then Operator は分類の実装を読まずに、どの経路がどの優先度クラスに属するかを参照できる

### Example: EX-SYSTEM-019-02 生成物が定義と一致しない

- Given 登録済みの各経路の優先度クラスが、経路の登録と同じ場所に一箇所で定義されている
- When RoutePriorityReference を生成する
- Then 生成物は組み立て済みの経路それぞれについて、経路パターン、メソッド、属する優先度クラスを示す
- Then 生成物は優先度クラスごとに、対応する縮退のステージと、その上限を与える起動時設定のキーを示す
- When 生成物と分類の定義を突き合わせる
- But 生成物が定義と一致しない
- Then 突き合わせは失敗し、再生成すべきことを報告する
