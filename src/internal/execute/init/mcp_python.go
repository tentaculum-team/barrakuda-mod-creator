package execute_init

import (
	"fmt"

	"barrakudaModKit/internal/manifest"
)

// CreatePythonMcp scaffolds a minimal Python MCP server: a stdio JSON-RPC MCP
// implementation hand-rolled with no SDK, wired through the same
// domain/repository/service layering used by auth/payments, exposing one example
// tool: greet.
func CreatePythonMcp(name string) (string, error) {
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
		"README.md":     fmt.Sprintf(pyMcpReadme, name),
		"main.py":       pyMcpMain,

		"app/__init__.py": "",

		"app/domain/__init__.py":   "",
		"app/domain/greeting.py":   pyMcpDomain,

		"app/repository/__init__.py":            "",
		"app/repository/greeting_repository.py": pyMcpRepository,

		"app/service/__init__.py":         "",
		"app/service/greeting_service.py": pyMcpService,

		"app/mcp/__init__.py": "",
		"app/mcp/server.py":   fmt.Sprintf(pyMcpServer, name),
	})
}

const pyMcpReadme = `# %s

Minimal MCP server over stdio. No dependencies. The MCP protocol itself
(JSON-RPC, newline-delimited) is hand-rolled in app/mcp/server.py — no SDK — so it's
readable top to bottom.

## Layout

- app/domain     — the Greeting type
- app/repository — in-memory greet counter (swap for a real DB when needed)
- app/service    — business logic (greet)
- app/mcp        — the MCP protocol: the stdio loop
- main.py        — wiring + entrypoint

## Run

Requires Python 3.10+, no dependencies to install.

    python main.py

## Try it

Paste these lines (one JSON object per line) into the running process's stdin:

    {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}
    {"jsonrpc":"2.0","method":"notifications/initialized"}
    {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"greet","arguments":{"name":"Ada"}}}
`

const pyMcpDomain = `from dataclasses import dataclass


@dataclass
class Greeting:
    name: str
    count: int
    message: str
`

const pyMcpRepository = `import threading


class GreetingRepository:
    """In-memory only — restarts reset the count. Swap for a real database when needed."""

    def __init__(self):
        self._lock = threading.Lock()
        self._counts: dict[str, int] = {}

    def increment(self, name: str) -> int:
        with self._lock:
            self._counts[name] = self._counts.get(name, 0) + 1
            return self._counts[name]
`

const pyMcpService = `from app.domain.greeting import Greeting
from app.repository.greeting_repository import GreetingRepository


class GreetingService:
    def __init__(self, repo: GreetingRepository):
        self._repo = repo

    def greet(self, name: str) -> Greeting:
        if not name:
            raise ValueError("name is required")

        count = self._repo.increment(name)
        message = f"Hello, {name}! I've greeted you {count} time(s)."

        return Greeting(name=name, count=count, message=message)
`

const pyMcpServer = `import json
import sys
from typing import Callable

# Just enough of MCP's JSON-RPC shapes to serve one tool over stdio.
# See https://modelcontextprotocol.io/specification for the full spec.

PROTOCOL_VERSION = "2025-06-18"

GREET_TOOL = {
    "name": "greet",
    "description": "Greets someone by name and remembers how many times they've been greeted.",
    "inputSchema": {
        "type": "object",
        "properties": {"name": {"type": "string", "description": "Who to greet"}},
        "required": ["name"],
    },
}


def serve(greet: Callable[[str], str]) -> None:
    """Reads newline-delimited JSON-RPC requests from stdin, writes responses to
    stdout: initialize, tools/list and tools/call, which is all a client needs to
    discover and call the greet tool."""

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            req = json.loads(line)
        except json.JSONDecodeError:
            continue

        method = req.get("method")

        if method == "notifications/initialized":
            continue

        res = {"jsonrpc": "2.0", "id": req.get("id")}

        if method == "initialize":
            res["result"] = {
                "protocolVersion": PROTOCOL_VERSION,
                "capabilities": {"tools": {}},
                "serverInfo": {"name": "%s", "version": "0.1.0"},
            }
        elif method == "tools/list":
            res["result"] = {"tools": [GREET_TOOL]}
        elif method == "tools/call":
            params = req.get("params") or {}
            args = params.get("arguments") or {}
            try:
                text = greet(args.get("name", ""))
                res["result"] = {"content": [{"type": "text", "text": text}]}
            except ValueError as err:
                res["result"] = {
                    "content": [{"type": "text", "text": str(err)}],
                    "isError": True,
                }
        else:
            res["error"] = {"code": -32601, "message": f"method not found: {method}"}

        sys.stdout.write(json.dumps(res) + "\n")
        sys.stdout.flush()
`

const pyMcpMain = `from app.repository.greeting_repository import GreetingRepository
from app.service.greeting_service import GreetingService
from app.mcp.server import serve

if __name__ == "__main__":
    service = GreetingService(GreetingRepository())
    serve(lambda name: service.greet(name).message)
`
