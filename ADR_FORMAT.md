# ADR archive policy

New ADRs are no longer created. Existing files under `decisions/` are immutable historical records and remain available to explain old choices and keep links valid.

For a new change:

- put change-specific motivation, considered alternatives, risks, and implementation history in the work item;
- put normative behavior in TypeSpec or `requirements.md`;
- put durable current design and its concise rationale in `ARCHITECTURE.md`;
- update those current-state documents directly when a former decision is replaced.

Do not renumber, rewrite, or delete archived ADRs as part of ordinary feature work.
