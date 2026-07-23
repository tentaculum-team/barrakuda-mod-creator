package execute_init

import (
	"fmt"

	"barrakudaModKit/internal/manifest"
)

// CreateTypescriptMcp scaffolds a minimal Typescript MCP server: a stdio JSON-RPC MCP
// implementation hand-rolled with no SDK, wired through the same
// domain/repository/service layering used by auth/payments, exposing one example
// tool: greet.
func CreateTypescriptMcp(name string) (string, error) {
	manifestJSON, err := mcpManifestJSON(name, []manifest.Tool{
		{
			Name: "greet", Description: "Greets someone by name and remembers how many times they've been greeted.",
			InputSchema:  greetInputSchema,
			Capabilities: []string{},
		},
	})
	if err != nil {
		return "", err
	}

	return writeFiles(name, map[string]string{
		"manifest.json": manifestJSON,
		"package.json":  fmt.Sprintf(tsMcpPackageJSON, name),
		"tsconfig.json": tsMcpTsconfig,
		"README.md":     fmt.Sprintf(tsMcpReadme, name),

		"src/domain/greeting.ts": tsMcpDomain,

		"src/repository/greetingRepository.ts": tsMcpRepository,

		"src/service/greetingService.ts": tsMcpService,

		"src/mcp/server.ts": fmt.Sprintf(tsMcpServer, name),

		"src/index.ts": tsMcpIndex,
	})
}

const tsMcpPackageJSON = `{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "devDependencies": {
    "@types/node": "^22",
    "typescript": "^5"
  }
}
`

const tsMcpTsconfig = `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "CommonJS",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true
  }
}
`

const tsMcpReadme = `# %s

Minimal MCP server over stdio. The MCP protocol itself (JSON-RPC, newline-delimited)
is hand-rolled in src/mcp/server.ts — no SDK — so it's readable top to bottom.

## Layout

- src/domain     — the Greeting type
- src/repository — in-memory greet counter (swap for a real DB when needed)
- src/service    — business logic (greet)
- src/mcp        — the MCP protocol: the stdio loop
- src/index.ts   — wiring + entrypoint

## Run

    npm install
    npm run build
    npm start

## Try it

Paste these lines (one JSON object per line) into the running process's stdin:

    {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}
    {"jsonrpc":"2.0","method":"notifications/initialized"}
    {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"greet","arguments":{"name":"Ada"}}}
`

const tsMcpDomain = `export interface Greeting {
  name: string;
  count: number;
  message: string;
}
`

const tsMcpRepository = `// In-memory only — restarts reset the count. Swap for a real database when needed.
export class GreetingRepository {
  private counts = new Map<string, number>();

  increment(name: string): number {
    const next = (this.counts.get(name) ?? 0) + 1;
    this.counts.set(name, next);
    return next;
  }
}
`

const tsMcpService = `import { Greeting } from "../domain/greeting";
import { GreetingRepository } from "../repository/greetingRepository";

export class GreetingService {
  private repo: GreetingRepository;

  constructor(repo: GreetingRepository) {
    this.repo = repo;
  }

  greet(name: string): Greeting {
    if (!name) {
      throw new Error("name is required");
    }

    const count = this.repo.increment(name);
    const message = ` + "`Hello, ${name}! I've greeted you ${count} time(s).`" + `;

    return { name, count, message };
  }
}
`

const tsMcpServer = `import * as readline from "node:readline";

// Just enough of MCP's JSON-RPC shapes to serve one tool over stdio.
// See https://modelcontextprotocol.io/specification for the full spec.

interface JsonRpcRequest {
  jsonrpc: string;
  id?: number | string;
  method: string;
  params?: { name: string; arguments?: { name: string } };
}

interface ToolContent {
  type: "text";
  text: string;
}

const PROTOCOL_VERSION = "2025-06-18";

const greetTool = {
  name: "greet",
  description: "Greets someone by name and remembers how many times they've been greeted.",
  inputSchema: {
    type: "object",
    properties: {
      name: { type: "string", description: "Who to greet" },
    },
    required: ["name"],
  },
};

// serve reads newline-delimited JSON-RPC requests from stdin and writes responses to
// stdout: initialize, tools/list and tools/call, which is all a client needs to
// discover and call the greet tool.
export function serve(greet: (name: string) => string): void {
  const rl = readline.createInterface({ input: process.stdin });

  rl.on("line", (line) => {
    if (!line.trim()) return;

    let req: JsonRpcRequest;
    try {
      req = JSON.parse(line);
    } catch {
      return;
    }

    if (req.method === "notifications/initialized") return;

    const res: { jsonrpc: string; id?: number | string; result?: unknown; error?: unknown } = {
      jsonrpc: "2.0",
      id: req.id,
    };

    switch (req.method) {
      case "initialize":
        res.result = {
          protocolVersion: PROTOCOL_VERSION,
          capabilities: { tools: {} },
          serverInfo: { name: "%s", version: "0.1.0" },
        };
        break;

      case "tools/list":
        res.result = { tools: [greetTool] };
        break;

      case "tools/call": {
        const args = req.params?.arguments ?? { name: "" };
        try {
          const text = greet(args.name);
          const content: ToolContent = { type: "text", text };
          res.result = { content: [content] };
        } catch (err) {
          const content: ToolContent = { type: "text", text: (err as Error).message };
          res.result = { content: [content], isError: true };
        }
        break;
      }

      default:
        res.error = { code: -32601, message: ` + "`method not found: ${req.method}`" + ` };
    }

    process.stdout.write(JSON.stringify(res) + "\n");
  });
}
`

const tsMcpIndex = `import { GreetingRepository } from "./repository/greetingRepository";
import { GreetingService } from "./service/greetingService";
import { serve } from "./mcp/server";

const service = new GreetingService(new GreetingRepository());

serve((name) => service.greet(name).message);
`
