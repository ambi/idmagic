---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 生成ドキュメントサイトの区分、トップページ、体裁を読み手の目的に合わせる

`mise run spec-diff` reports no normative specification change against `main`: the normative ids, their text
and their coverage are untouched, and the tools that read history now map a document's former path onto its
current one so a move is not counted as an addition.
`docs/` is arranged by what each document is, and the sidebar is a plain mirror of it. `docs/contexts/` became
`docs/domain/`, because what a reader needs first is that these are domain design documents, not that they
happen to be a list of Bounded Contexts. The glossary, the adopted standards, the cross-context scenarios and
the repository structure moved under it, since each is the system-wide side of a document every Context holds
under the same name — or, for `structure.md`, is mostly Context internals and inter-context events. The
product overview moved into `docs/design/`, because it is what a reader reads before any design document.
`docs/design/README.md` was an abstract title over an index of the seven areas; it is now the design
division's entrance, and the seven areas sit directly under it. `docs/domain/README.md` is new. Every
division now has an entrance, including the two generated ones, so no division is a name that cannot be
opened.
Sidebar branches are `details` elements: their children are always in the HTML, so every page is reachable,
and only the ancestors of the current page are open, so no unrelated list expands. Indentation is 18px per
level, a leaf aligns with its sibling branch's label, and a link looks the same whether or not it has
children — the disclosure marker says that instead.
The API reference no longer asks Swagger UI to resolve anything: the document is dereferenced when the page is
built, so no `$ref` is left. Reading the bundled Swagger UI 5.32.15 showed why no way of handing it the
document helps — it always sets `baseDoc` from the document URL and resolves even an internal `#/…` pointer by
retrieving that URI, which a page opened as a file cannot do. The published URL then loads nothing, and the
`blob:` URL and `spec` both load but warn when an operation is expanded. Dereferencing removes the retrieval
instead of relocating it. Each expanded node keeps its `$$ref`, so Swagger UI still shows the model name; the
one self-referential schema, a SCIM attribute holding nested attributes, is cut off with a typed placeholder;
and `api/index.html` grows from 1.4 MiB to 6.4 MiB. The dependency bump from 5.32.14 to 5.32.15 was ruled out
— the resolver code is identical in both.
`docs/README.md` is the site's top page rather than a second copy of the same guidance; a title reaches a nav
label, a breadcrumb and `<title>` as plain text; the stylesheet puts prose on the page background with a
760px measure and a sticky heading outline; `docs/operations/README.md` names the runbooks it owns; and a
link to a directory resolves to that directory's `README.md`.
The names say what they hold: `render-spec-docs` became `render-docs`, `check-rendered-spec` became
`check-rendered-docs`, the `specification/` output prefix became `docs/`, `method/` became `format/`, and
`renderSpecificationSite` became `renderDocumentationSite`. `compile-spec`, `check-spec`, `verify-spec` and
`spec-diff` keep their names because they do act on `spec/`.
Finally the words: `正本`, `正準文書`, `正準 Markdown` and the user-facing `正準 scope 値` became `一次情報`,
`一次情報文書` and plain wording wherever they named the documentation system, while the uses that name where
data originates stayed; `版` became `バージョン` outside `docs/domain/` and `backend/`. Two tests enforce both
rules, which `DOCUMENTATION_GUIDE.md` previously only stated.

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
