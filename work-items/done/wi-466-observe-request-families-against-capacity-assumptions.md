---
status: completed
archived_from: 08e544929ac7272bbfe940d595f4c2a5779e3573
---

# 種別ごとの到達率と処理時間を観測し、容量の Planning assumption を Measurement へ置き換えられるようにする

`mise run spec-diff` reports no normative specification change against `main`: the work adds observation,
not behaviour. The semantic difference is that requests can now be read by purpose. Six request families
(`authentication`, `portal`, `management`, `scim`, `shared_signals`, `operations`) are derived from the
`route` label by regular expressions in the two Prometheus rule files, and carried as a `family` label on
seven recording rules covering arrival rate, non-5xx ratio, p50/p99 latency, and in-flight requests. Both
Grafana dashboards show the breakdown directly beneath the route-level panels, so a burning authentication
objective can be read against the same time window's family mix — the observation the C1 reconsideration
condition of the API process-plane separation decision needs and did not have. The classification is
exhaustive by construction: a Go test matches all 505 assembled route patterns against the rule files and
fails when a route matches no family, matches more than one, when the three rules of one family disagree,
or when the two rule files disagree. No service level objective was added, and no planning assumption in
the capacity design was rewritten; both remain out of scope until measurements exist.

全文は `git show 08e544929ac7272bbfe940d595f4c2a5779e3573:<このファイルのパス>` で読む。
