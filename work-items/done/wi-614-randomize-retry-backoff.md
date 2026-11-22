---
status: completed
---

# 再試行の待ち時間に一様乱数のばらつきを付ける

REQ-JOBS-005 と REQ-SHAREDSIGNALS-006 の再試行時刻を、30 秒から倍々に増えて最大 30 分となる値を上限とし、0 以上上限以下の一様乱数で選ぶ形に変えた。
回数、dead letter、イベント、状態遷移は変えていない。
一様乱数の抽選だけを共有ライブラリ `backend/shared/resilience` の `RandomDelay` に置き、上限の計算は Jobs、SharedSignals、`RetryWithBackoff` の各所有者に残した。
Runner と配送処理は乱数の関数を受け取り、省略時は本番用の乱数を使う。本番の組み立て経路を通るテストでこれを固定した。
