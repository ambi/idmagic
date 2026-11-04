// 改修していない MCP サーバーの例である。認可のコードを持たず、ゲートウェイの背後で動く。
// 単独で起動すると、受けたツールの呼び出しを標準出力へ一行ずつ書く。
import { createMcpHandler, McpServer } from "@modelcontextprotocol/server";
import * as z from "zod/v4";

export type DocumentServer = {
  url: string;
  /** 実行したツールの呼び出しの記録。ゲートウェイが止めた呼び出しはここに現れない。 */
  calls: string[];
  stop: () => Promise<void>;
};

const documents = new Map([
  ["handbook", "就業規則：始業は 9 時である。"],
  ["salary-2026", "給与テーブル：機密。"],
]);

export function startDocumentServer(port = 0, log: (line: string) => void = () => {}): DocumentServer {
  const calls: string[] = [];
  const record = (line: string) => {
    calls.push(line);
    log(line);
  };
  const handler = createMcpHandler(
    () => {
      const server = new McpServer({ name: "documents", version: "1.0.0" });
      const target = { inputSchema: z.object({ document_id: z.string() }) };
      server.registerTool("read_document", { title: "文書の閲覧", ...target }, async ({ document_id }) => {
        record(`read_document ${document_id}`);
        return { content: [{ type: "text", text: documents.get(document_id) ?? "文書がありません。" }] };
      });
      server.registerTool(
        "write_document",
        { title: "文書の更新", inputSchema: z.object({ document_id: z.string(), body: z.string() }) },
        async ({ document_id, body }) => {
          record(`write_document ${document_id}`);
          documents.set(document_id, body);
          return { content: [{ type: "text", text: "更新しました。" }] };
        },
      );
      server.registerTool("delete_document", { title: "文書の削除", ...target }, async ({ document_id }) => {
        record(`delete_document ${document_id}`);
        documents.delete(document_id);
        return { content: [{ type: "text", text: "削除しました。" }] };
      });
      server.registerTool("publish_document", { title: "文書の社外公開", ...target }, async ({ document_id }) => {
        record(`publish_document ${document_id}`);
        return { content: [{ type: "text", text: "公開しました。" }] };
      });
      return server;
    },
    { legacy: "reject" },
  );
  const server = Bun.serve({ hostname: "127.0.0.1", port, fetch: (request) => handler.fetch(request) });
  return {
    url: `http://127.0.0.1:${server.port}/`,
    calls,
    stop: async () => {
      await handler.close();
      await server.stop(true);
    },
  };
}

if (import.meta.main) {
  const port = Number(process.env.PORT ?? "3001");
  const server = startDocumentServer(port, (line) => console.log(`executed: ${line}`));
  console.log(`MCP server: ${server.url}`);
}
