# P2KB MCP API Reference

This document describes all MCP tools provided by the P2KB MCP server (v1.1.0+).

## Overview

P2KB MCP provides 6 tools for accessing the Propeller 2 Knowledge Base and OBEX:

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content using natural language or exact key |
| `p2kb_find` | Explore and discover documentation |
| `p2kb_obex_get` | Get OBEX object by search or ID |
| `p2kb_obex_find` | Explore OBEX objects |
| `p2kb_version` | Server version and status |
| `p2kb_refresh` | Refresh index and invalidate stale cache |

---

## Documentation Tools

### p2kb_get

Fetch P2 Knowledge Base content using natural language or exact key.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Natural language query or exact key |

**Query Examples:**

- `"mov instruction"` - Natural language
- `"spin2 pinwrite"` - Natural language
- `"cog memory"` - Natural language
- `"p2kbPasm2Mov"` - Exact key

**Returns (content found):**

```json
{
  "type": "content",
  "key": "p2kbPasm2Mov",
  "content": "--- YAML content ---",
  "categories": ["pasm2_data", "pasm2_math"],
  "related": ["p2kbPasm2Loc", "p2kbPasm2Rdlong"]
}
```

**Returns (multiple matches):**

```json
{
  "type": "suggestions",
  "query": "mov",
  "message": "Multiple matches found. Please be more specific or use an exact key.",
  "suggestions": [
    {"key": "p2kbPasm2Mov", "score": 0.9, "category": "pasm2_data"},
    {"key": "p2kbPasm2Movbyts", "score": 0.8, "category": "pasm2_data"}
  ]
}
```

**Example:**

```json
{
  "name": "p2kb_get",
  "arguments": {
    "query": "mov instruction"
  }
}
```

---

### p2kb_find

Explore and discover P2KB documentation.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `term` | string | No | - | Search term |
| `category` | string | No | - | Category to browse |
| `limit` | integer | No | 50 | Max results |

**Behavior:**

- **No parameters**: Returns list of all categories with counts
- **term only**: Searches for matching keys
- **category only**: Lists all keys in that category
- **term + category**: Searches within category

**Returns (no parameters - categories):**

```json
{
  "type": "categories",
  "categories": {
    "pasm2_branch": 37,
    "pasm2_math": 45,
    "spin2_pin": 12
  },
  "total_categories": 47,
  "total_entries": 970
}
```

**Returns (with category):**

```json
{
  "type": "keys",
  "category": "pasm2_math",
  "keys": ["p2kbPasm2Add", "p2kbPasm2Sub", "p2kbPasm2Mul"],
  "count": 45
}
```

**Returns (with term):**

```json
{
  "type": "keys",
  "term": "mov",
  "keys": ["p2kbPasm2Mov", "p2kbPasm2Movbyts"],
  "count": 2
}
```

**Example:**

```json
{
  "name": "p2kb_find",
  "arguments": {
    "category": "pasm2_math"
  }
}
```

---

## OBEX Tools

### p2kb_obex_get

Get OBEX (Parallax Object Exchange) object by search term or numeric ID.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Natural language search or numeric object ID |

**Query Examples:**

- `"led driver"` - Natural language search
- `"i2c sensor"` - Natural language search
- `"2811"` - Numeric object ID
- `"OB4047"` - Object ID with prefix (stripped automatically)

**Returns (object found):**

```json
{
  "type": "obex_object",
  "object_id": "2811",
  "title": "Park transformation",
  "author": "ManAtWork",
  "category": "motors",
  "description": "CORDIC-based park transformation for motor control",
  "languages": ["SPIN2", "PASM2"],
  "tags": ["motor", "cordic", "servo"],
  "download_url": "https://obex.parallax.com/...",
  "obex_page": "https://obex.parallax.com/obex/park-transformation/",
  "download_instructions": {
    "suggested_directory": "OBEX/park-transformation",
    "filename": "OB2811.zip",
    "command": "curl -L -o OB2811.zip 'https://obex.parallax.com/...'"
  },
  "metadata": {
    "version": "",
    "file_size": "16 B",
    "quality": 5,
    "created_date": "2020-05-09 12:00:00"
  }
}
```

**Returns (multiple matches):**

```json
{
  "type": "suggestions",
  "query": "led",
  "message": "Multiple OBEX objects found. Specify an object_id or refine your search.",
  "suggestions": [
    {"object_id": "4047", "title": "WS2812B LED Driver", "author": "...", "category": "drivers"},
    {"object_id": "5274", "title": "NeoPixel Controller", "author": "...", "category": "drivers"}
  ]
}
```

**Example:**

```json
{
  "name": "p2kb_obex_get",
  "arguments": {
    "query": "2811"
  }
}
```

---

### p2kb_obex_find

Explore OBEX objects by category, author, or search term.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `term` | string | No | - | Search term |
| `category` | string | No | - | Category filter (drivers, misc, display, demos, audio, motors, communication, sensors, tools) |
| `author` | string | No | - | Author name filter |
| `limit` | integer | No | 20 | Max results |

**Behavior:**

- **No parameters**: Returns overview with categories and top authors
- **term**: Searches all objects
- **category**: Lists objects in category
- **author**: Lists objects by author

**Returns (no parameters - overview):**

```json
{
  "type": "overview",
  "categories": {
    "drivers": 49,
    "misc": 34,
    "display": 7
  },
  "total_objects": 113,
  "top_authors": [
    {"name": "Jon McPhalen", "object_count": 44},
    {"name": "Stephen M Moraco", "object_count": 15}
  ]
}
```

**Returns (with category or term):**

```json
{
  "type": "objects",
  "category": "drivers",
  "objects": [
    {"object_id": "2811", "title": "...", "author": "...", "description": "..."},
    {"object_id": "4047", "title": "...", "author": "...", "description": "..."}
  ],
  "count": 49
}
```

**Example:**

```json
{
  "name": "p2kb_obex_find",
  "arguments": {
    "category": "drivers",
    "limit": 10
  }
}
```

---

## System Tools

### p2kb_version

Get MCP server version and status information.

**Parameters:** None

**Returns:**

```json
{
  "mcp_version": "0.3.0",
  "index_version": "3.2.0",
  "index": {
    "total_entries": 970,
    "total_categories": 47,
    "is_cached": true,
    "age_seconds": 3600,
    "needs_refresh": false
  },
  "obex": {
    "total_objects": 113,
    "cached_memory": 10,
    "cached_disk": 50,
    "stale_cache_entries": 0
  }
}
```

---

### p2kb_refresh

Force refresh of index and invalidate stale cache entries based on index timestamps.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `include_obex` | boolean | No | false | Also refresh OBEX index |

**Returns:**

```json
{
  "refreshed": true,
  "stale_keys_found": 5,
  "cache_entries_invalidated": 5,
  "index_version": "3.2.1",
  "total_entries": 970,
  "obex_refreshed": true
}
```

**Example:**

```json
{
  "name": "p2kb_refresh",
  "arguments": {
    "include_obex": true
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

## Natural Language Query Matching

The `p2kb_get` tool supports natural language queries using token matching:

1. **Query tokenization**: "MOV instruction" → ["mov", "instruction"]
2. **Key tokenization**: `p2kbPasm2Mov` → ["p2kb", "pasm2", "mov"]
3. **Scoring**: Tokens are matched and scored
4. **High-confidence match**: Returns content directly
5. **Ambiguous match**: Returns suggestions

### Examples

| Query | Matches |
|-------|---------|
| `mov` | p2kbPasm2Mov, p2kbPasm2Movbyts |
| `pasm2 add` | p2kbPasm2Add |
| `spin2 pinwrite` | p2kbSpin2Pinwrite |
| `cog memory` | p2kbArchCogMemory |

---

## OBEX Search Term Expansion

The OBEX tools automatically expand search terms to related concepts:

| Term | Expanded To |
|------|-------------|
| `i2c` | i2c, iic, twi, two-wire |
| `led` | led, pixel, ws2812, rgb, neopixel, strip |
| `motor` | motor, servo, stepper, pwm, drive |
| `sensor` | sensor, detector, measure, monitor |
| `display` | display, lcd, oled, screen, graphics |

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

### Error Response Example

```json
{
  "error": {
    "code": -32000,
    "message": "No matches found",
    "data": {
      "query": "xyz nonexistent",
      "hint": "Try using p2kb_find to explore available documentation"
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
    "serverInfo": {"name": "p2kb-mcp", "version": "0.3.0"}
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
    "arguments": {"query": "mov instruction"}
  }
}
```

---

## Caching Behavior

- **Index TTL**: 24 hours (configurable via `P2KB_INDEX_TTL`)
- **Content cache**: Persistent, invalidated based on index mtime comparison
- **OBEX cache**: TTL-based, 24 hours per object
- **Cache location**: Platform-specific (see below)

### Cache Locations

| Platform | Location |
|----------|----------|
| Linux | `~/.cache/p2kb-mcp/` |
| macOS | `~/Library/Caches/p2kb-mcp/` |
| Windows | `%LocalAppData%\p2kb-mcp\` |

### Smart Cache Invalidation

When `p2kb_refresh` is called:
1. Fresh index is fetched with cache-busting headers
2. Index file timestamps (`mtime`) are compared with cached content
3. Stale entries (older than index) are automatically invalidated
4. Next access will fetch fresh content

### Content Filtering

The following metadata fields are removed from cached content to save tokens:

- `last_updated`
- `enhancement_source`
- `documentation_source`
- `documentation_level`
- `manual_extraction_date`

---

## Migration from v1.0.x

The following tools were removed and replaced:

| Old Tool | Replacement |
|----------|-------------|
| `p2kb_search` | `p2kb_find(term="...")` |
| `p2kb_browse` | `p2kb_find(category="...")` |
| `p2kb_categories` | `p2kb_find()` (no params) |
| `p2kb_batch_get` | Multiple `p2kb_get` calls |
| `p2kb_info` | `p2kb_get` returns categories |
| `p2kb_stats` | `p2kb_version` |
| `p2kb_related` | `p2kb_get` returns related items |
| `p2kb_help` | This documentation |
| `p2kb_cached` | `p2kb_version` shows cache stats |
| `p2kb_index_status` | `p2kb_version` shows index status |
| `p2kb_obex_search` | `p2kb_obex_find(term="...")` |
| `p2kb_obex_browse` | `p2kb_obex_find(category="...")` |
| `p2kb_obex_authors` | `p2kb_obex_find()` shows top authors |
