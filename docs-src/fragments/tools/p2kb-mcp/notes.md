### Register it with your agent

Pick the host you actually use. This is the only step that differs by host.

**Claude Code** — one command, no file editing:

```sh
claude mcp add -s user p2kb-mcp -- {{bin_dir_launcher}}
```

**Claude Desktop** — edit `{{claude_desktop_config}}` and add the `p2kb-mcp`
entry inside the existing `mcpServers` object (create the file if it does not
exist). Do not replace the whole file if you already have other servers.

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "{{bin_dir_launcher_json}}"
    }
  }
}
```

Restart Claude Desktop afterward.

**Cursor** — same JSON, in `{{cursor_config}}`. Reload the window afterward.

**Codex** — `codex mcp add p2kb-mcp -- {{bin_dir_launcher}}`

### First run

The server downloads its knowledge index from GitHub the first time an agent
calls it, then keeps it fresh automatically. No setup step is needed. On a slow
connection the very first call can take longer than a host's default startup
timeout — if your host reports a timeout, retry once.

### Check that it worked

Ask your agent a P2 question that only the knowledge base can answer, such as
*"What does the PASM2 `RDFAST` instruction do?"* If the tools are registered,
the answer will cite the knowledge base.
