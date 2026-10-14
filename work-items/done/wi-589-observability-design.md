---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 監視・ログ・トレースの設計を、当番担当者が使える水準まで書く

`mise run spec-diff` reports no normative specification change against `main`, which matches
`spec_impact: none`. The semantic difference is in the observability design documents.
Monitoring now states what it wants to detect, the signal it detects it with, how the signal is judged,
which kind of notification it raises, and which runbook it leads to, as one table per detection. A second
table records the detections that have no rule, so a missing detection is a visible row instead of an
absence. Threshold ownership is split by origin (SLO, capacity, observed normal range), and the document
states that the alerts named after error-budget burn actually judge a five-minute instantaneous value.
Logging gains an `event_name` naming rule (a closed result vocabulary, nothing that varies in the name),
a declaration form for sampling with the statement that nothing is sampled today, a per-deployment-profile
table of storage, redundancy, retention, read access, and deletion, and a table of what happens to the
application and to the logs when Loki, Alloy, or the node disk fails.
Tracing now separates the implemented state from the target design and puts the implemented state first:
only `idmagic-api` creates spans, only for the OAuth/OIDC endpoints, only under `OBSERVABILITY=otel`, with
no propagation past the ingress and no store to search. The target design fixes propagation per boundary
(jobs start a new trace linked to the enqueue, with the reason), head sampling over tail sampling with the
cost of that choice stated, span-name cardinality and the three self-declared attributes, and the rule that
retention is shortened before the sampling rate is lowered.
The observability README gains an index of what each child document owns and points each open item at the
child that records it.
A review pass then replaced the vocabulary this layer had coined. `信号` becomes `シグナル`, the term the
Japanese OpenTelemetry documentation uses for a category of telemetry, with a sentence in the README
separating it from an operating-system signal. `相関` is kept, because the Japanese Datadog documentation
uses it for the same idea. `所有境界` is dropped and written out in plain Japanese, because no primary
source has a matching term and it describes which document decides what, not observability. `公開入口`
becomes `エッジ`, the name the deployment architecture already gives that layer, and `重大度` becomes
`ログレベル`. The rename reaches the glossary entry, the documentation guide, and the design documents that
referred to the old words, so no definition is left behind. Tracing also gains a section stating what it
answers that logs cannot, which is why it is a document of its own rather than a section of the logging
design, and its implemented state moves from four prose subsections into one table.

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
