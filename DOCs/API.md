# P2KB MCP API Reference

This document describes all MCP tools provided by the P2KB MCP server.

## Overview

P2KB MCP provides 11 tools for accessing the Propeller 2 Knowledge Base:

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content by key |
| `p2kb_search` | Search for keys |
| `p2kb_browse` | List keys in a category |
| `p2kb_categories` | List all categories |
| `p2kb_version` | Get server version |
| `p2kb_batch_get` | Fetch multiple keys |
| `p2kb_refresh` | Refresh index/cache |
| `p2kb_info` | Check key existence |
| `p2kb_stats` | Knowledge base stats |
| `p2kb_related` | Get related items |
| `p2kb_help` | Usage information |

---

## Core Tools

### p2kb_get

Fetch P2 Knowledge Base content by key.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `key` | string | Yes | The p2kb key (e.g., `p2kbPasm2Mov`) |

**Returns:**

```json
{
  "content": "--- YAML content ---"
}
```

**Errors:**

- Key not found: Returns suggestions for similar keys

**Example:**

```json
{
  "name": "p2kb_get",
  "arguments": {
    "key": "p2kbPasm2Mov"
  }
}
```

---

### p2kb_search

Search for keys matching a term (case-insensitive).

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `term` | string | Yes | - | Search term |
| `limit` | integer | No | 50 | Max results |

**Returns:**

```json
{
  "keys": ["p2kbPasm2Mov", "p2kbPasm2Movbyts"],
  "count": 2,
  "term": "mov"
}
```

**Example:**

```json
{
  "name": "p2kb_search",
  "arguments": {
    "term": "mov",
    "limit": 10
  }
}
```

---

### p2kb_browse

List all keys in a category.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `category` | string | Yes | Category name (e.g., `pasm2_branch`) |

**Returns:**

```json
{
  "category": "pasm2_branch",
  "keys": ["p2kbPasm2Call", "p2kbPasm2Jmp", ...],
  "count": 37
}
```

**Errors:**

- Category not found: Returns list of valid categories

**Example:**

```json
{
  "name": "p2kb_browse",
  "arguments": {
    "category": "pasm2_math"
  }
}
```

---

### p2kb_categories

List all available categories with counts.

**Parameters:** None

**Returns:**

```json
{
  "categories": {
    "pasm2_branch": 37,
    "pasm2_math": 45,
    "architecture_core": 8
  },
  "total_categories": 47,
  "total_entries": 970
}
```

---

### p2kb_version

Get MCP server version.

**Parameters:** None

**Returns:**

```json
{
  "mcp_version": "1.0.0"
}
```

---

## Enhanced Tools

### p2kb_batch_get

Fetch multiple keys in one call.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `keys` | string[] | Yes | Array of keys to fetch |

**Returns:**

```json
{
  "results": {
    "p2kbPasm2Mov": { "content": "..." },
    "p2kbPasm2Add": { "content": "..." },
    "p2kbInvalid": { "error": "Key not found" }
  },
  "success": 2,
  "errors": 1
}
```

**Example:**

```json
{
  "name": "p2kb_batch_get",
  "arguments": {
    "keys": ["p2kbPasm2Mov", "p2kbPasm2Add", "p2kbPasm2Sub"]
  }
}
```

---

### p2kb_refresh

Force refresh of index and optionally invalidate cache.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `invalidate_cache` | boolean | No | false | Clear all cached YAMLs |
| `prefetch_common` | boolean | No | true | Prefetch common keys after refresh |

**Returns:**

```json
{
  "refreshed": true,
  "version": "3.2.1",
  "total_entries": 970
}
```

---

### p2kb_info

Check if a key exists and what categories it belongs to.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `key` | string | Yes | The p2kb key |

**Returns:**

```json
{
  "key": "p2kbPasm2Mov",
  "exists": true,
  "categories": ["pasm2_data", "pasm2_math"]
}
```

---

### p2kb_stats

Get knowledge base statistics.

**Parameters:** None

**Returns:**

```json
{
  "version": "3.2.0",
  "total_entries": 970,
  "total_categories": 47
}
```

---

### p2kb_related

Get related instructions for a key.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `key` | string | Yes | - | The p2kb key |
| `fetch_content` | boolean | No | false | Also fetch related content |

**Returns:**

```json
{
  "key": "p2kbPasm2Mov",
  "related": ["p2kbPasm2Loc", "p2kbPasm2Rdlong"],
  "content": {
    "p2kbPasm2Loc": "..."
  }
}
```

---

### p2kb_help

Return usage information.

**Parameters:** None

**Returns:**

```json
{
  "tools": ["p2kb_get", "p2kb_search", ...],
  "key_prefixes": {
    "p2kbPasm2*": "PASM2 assembly instructions",
    "p2kbSpin2*": "Spin2 methods",
    "p2kbArch*": "Architecture documentation",
    "p2kbGuide*": "Guides and quick references",
    "p2kbHw*": "Hardware specifications"
  }
}
```

---

## Key Naming Convention

| Prefix | Content Type | Examples |
|--------|--------------|----------|
| `p2kbPasm2*` | PASM2 instructions | `p2kbPasm2Mov`, `p2kbPasm2Add` |
| `p2kbSpin2*` | Spin2 methods | `p2kbSpin2Pinwrite`, `p2kbSpin2Waitms` |
| `p2kbArch*` | Architecture | `p2kbArchCog`, `p2kbArchHub` |
| `p2kbGuide*` | Guides | `p2kbGuideQuickQueries` |
| `p2kbHw*` | Hardware | `p2kbHwSmartPin` |

---

## Error Handling

### JSON-RPC Error Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method/tool not found |
| -32602 | Invalid params |
| -32603 | Internal error |
| -32000 | Tool execution failure |

### Key Not Found

When a key lookup fails:

```json
{
  "error": {
    "code": -32000,
    "message": "Key 'p2kbPasm2Mvo' not found",
    "data": {
      "suggestions": ["p2kbPasm2Mov", "p2kbPasm2Movbyts"],
      "hint": "Use p2kb_search to find valid keys"
    }
  }
}
```

---

## MCP Protocol

P2KB MCP uses the Model Context Protocol version `2024-11-05`.

### Initialize Handshake

Request:
```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "serverInfo": {"name": "p2kb-mcp", "version": "1.0.0"}
  }
}
```

### List Tools

Request:
```json
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
```

### Call Tool

Request:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "p2kb_get",
    "arguments": {"key": "p2kbPasm2Mov"}
  }
}
```

---

## Caching Behavior

- **Index TTL**: 24 hours (configurable via `P2KB_INDEX_TTL`)
- **Content cache**: Persistent, invalidated when index mtime changes
- **Cache location**: `~/.p2kb-mcp/` (configurable via `P2KB_CACHE_DIR`)

### Content Filtering

The following metadata fields are removed from cached content to save tokens:

- `last_updated`
- `enhancement_source`
- `documentation_source`
- `documentation_level`
- `manual_extraction_date`
