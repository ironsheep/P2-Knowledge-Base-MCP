# P2KB MCP API Reference

This document describes all MCP tools provided by the P2KB MCP server (v1.1.0+).

## Overview

P2KB MCP provides 7 tools for accessing the Propeller 2 Knowledge Base and OBEX:

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content using natural language or exact key |
| `p2kb_find` | Explore and discover documentation |
| `p2kb_obex_get` | Get OBEX object by search or ID |
| `p2kb_obex_find` | Explore OBEX objects |
| `p2kb_obex_download` | Download and extract an OBEX object's files |
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
  "content": "instruction: MOV\nsyntax: MOV Dest, {#}Src {WC|WZ|WCZ}\n...",
  "categories": ["pasm2_math"]
}
```

`content` is the KB file with its provenance removed (see [Content Filtering](#content-filtering)).
Related keys, when the file lists them, are inside it (`related_instructions`). When the query was an
alias, the result also carries `resolved_from` with the alias used.

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
    "drivers": 57,
    "misc": 42,
    "display": 7
  },
  "total_objects": 130,
  "top_authors": [
    {"name": "Jon McPhalen (jonnymac)", "object_count": 44},
    {"name": "Stephen M Moraco", "object_count": 17}
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

### p2kb_obex_download

Download an OBEX object's zip from Parallax OBEX and extract it under the working directory.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `object_id` | string | Yes | - | Object ID, with or without the `OB` prefix |
| `target_dir` | string | No | `./OBEX/<id>-<slug>/` | Extraction directory; must stay inside the working directory |

**Returns:**

```json
{
  "type": "download_complete",
  "object_id": "2811",
  "title": "Park transformation",
  "extraction_path": "/home/user/project/OBEX/2811-park-transformation",
  "files": ["README.md", "ParkTransformation.spin2"],
  "file_count": 2,
  "total_size": 5304,
  "message": "Successfully downloaded and extracted 2 files to /home/user/project/OBEX/2811-park-transformation"
}
```

**Example:**

```json
{
  "name": "p2kb_obex_download",
  "arguments": {
    "object_id": "2811"
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
  "mcp_version": "1.5.0",
  "index_version": "3.5.0",
  "filter_rule_id": "b431af2a9f515c11fe5fd982642c636c6f8843bf89ed2c3003ee0e28c04877ee",
  "filter_rule_source": "index",
  "filter_engine_version": "1",
  "index": {
    "total_entries": 1131,
    "total_categories": 59,
    "total_aliases": 3091,
    "is_cached": true,
    "age_seconds": 0,
    "needs_refresh": false
  },
  "obex": {
    "total_objects": 130,
    "parsed_objects": 0,
    "cached_bodies": 0
  }
}
```

The `filter_*` fields report the delivery filter in effect (see [Content Filtering](#content-filtering)):

| Field | Value |
|-------|-------|
| `filter_rule_id` | Identity of the rule in effect |
| `filter_rule_source` | `index` (from the current index), `last-good` (the last rule an index carried), or `built-in` |
| `filter_engine_version` | Version of the filter engine; with the rule id it stamps the cache |
| `filter_rule_refused` | Present only when the index carries a rule this server cannot apply: `{"format": <int>, "reason": "<why>"}`. The server keeps filtering under the last-good or built-in rule. |

`obex.parsed_objects` counts objects held parsed in memory; `obex.cached_bodies` counts object files
held in the content cache.

---

### p2kb_refresh

Force refresh of index and invalidate stale cache entries based on index timestamps.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `include_obex` | boolean | No | false | Also drop parsed OBEX objects, so each is parsed again on next use |
| `flush` | boolean | No | false | Wipe the whole content cache instead of only the stale entries |

**Returns (default):**

```json
{
  "refreshed": true,
  "flushed": false,
  "stale_keys_found": 0,
  "cache_entries_invalidated": 0,
  "index_version": "3.5.0",
  "total_entries": 1131
}
```

**Returns (`flush` and `include_obex`):**

```json
{
  "refreshed": true,
  "flushed": true,
  "stale_keys_found": 0,
  "cache_entries_invalidated": 1,
  "index_version": "3.5.0",
  "total_entries": 1131,
  "obex_refreshed": true,
  "obex_cache_entries_cleared": 0
}
```

With `include_obex` and no `flush`, the result also carries `obex_refreshed`.

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

- **Index**: re-checked after its TTL, 5 minutes by default (`P2KB_INDEX_TTL`); `p2kb_refresh` fetches
  it at once, bypassing CDN caches.
- **Content**: cached in memory and on disk. An entry is used while it is at least as new as its
  index entry's `mtime`; each download is checked against the index entry's `sha256` before it is
  filtered and cached.
- **OBEX objects**: OBEX object files are KB files listed in the main index, so they are fetched,
  checked, filtered and cached like any other content.
- **Filter stamp**: the cache records which filter rule and engine produced it. When the rule in
  effect changes, the cached content is discarded once and re-fetched as it is used. The first start
  after upgrading from a server before 1.5.0 does the same.

### Cache Locations

The first that applies:

| Condition | Location |
|-----------|----------|
| `P2KB_CACHE_DIR` is set | that directory |
| Container-tools install (binary at `<root>/bin/platforms/`) | `<root>/var/cache/p2kb-mcp/` |
| Standalone install on Linux or macOS (binary at `<root>/bin/`) | `<root>/.cache/` |
| Windows | `%LOCALAPPDATA%\p2kb-mcp\cache\` |
| The chosen directory cannot be created | `~/.p2kb-mcp/`, with a warning |

Inside it: `index/` holds the cached index and the last-good filter rule; `cache/` holds content and
the filter stamp.

### Smart Cache Invalidation

When `p2kb_refresh` is called:
1. Fresh index is fetched with cache-busting headers
2. Index file timestamps (`mtime`) are compared with cached content
3. Stale entries (older than index) are automatically invalidated
4. Next access will fetch fresh content

### Content Filtering

The KB's YAML files carry provenance (where a fact came from, when it was checked) that is useful to
the KB's maintainers and costs an AI agent tokens. The server removes it before content is cached or
returned, by a rule the KB publishes in its index as `delivery_filter`. The rule and the filter engine
are defined by the KB's interface specification,
[§4 Content filtering](P2KB-MCP-SPECIFICATION.md#4-content-filtering), and the delivery-filter
interface agreement it cites.

The rule in effect is the index's rule when the server can apply it. When the index carries no rule,
or one this server cannot apply, the server keeps the last rule an index carried, or its built-in
rule, so it never filters less than it last did. `p2kb_version` reports which rule is in effect and
where it came from. A change of rule reaches the KB's users with the next index, without a server
release.

---

## Configuration

Environment variables:

| Variable | Default | Effect |
|----------|---------|--------|
| `P2KB_CACHE_DIR` | see [Cache Locations](#cache-locations) | Cache directory |
| `P2KB_INDEX_TTL` | `300` | Seconds before the cached index is re-checked |
| `P2KB_LOG_LEVEL` | `warn` | `debug`, `info`, `warn` or `error`; logs go to stderr. `info` adds tool errors; `debug` adds one line per request (URL, cache-busting, status, bytes, duration) |
| `P2KB_BASE_URL` | `https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main` | Where the index and content are fetched from: the index at `<base>/deliverables/ai/p2kb-index.json.gz`, each file at `<base>/<path>` |

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
| `p2kb_related` | `p2kb_get` content lists related keys |
| `p2kb_help` | This documentation |
| `p2kb_cached` | `p2kb_version` shows cache stats |
| `p2kb_index_status` | `p2kb_version` shows index status |
| `p2kb_obex_search` | `p2kb_obex_find(term="...")` |
| `p2kb_obex_browse` | `p2kb_obex_find(category="...")` |
| `p2kb_obex_authors` | `p2kb_obex_find()` shows top authors |
