# P2KB MCP Server Update Specification: Alias Resolution

**Purpose**: This document specifies the changes needed to the P2KB MCP server to support the new alias resolution system in the p2kb-index.json.

**Date**: 2026-01-17
**Sprint Reference**: sprint-reference-system-fix.md

---

## Summary of Changes

The p2kb-index.json has been enhanced to version 3.3.0 with a new `aliases` section that enables resolution of common names (instruction mnemonics, method names, pattern IDs, symbols) to their canonical P2KB index keys.

---

## Index Structure Changes

### New `system` Metadata Fields

```json
{
  "system": {
    "version": "3.3.0",
    "total_aliases": 933,
    // ... existing fields
  }
}
```

The `total_aliases` field indicates the number of aliases available.

### New `aliases` Section

The index now includes an `aliases` object that maps common identifiers to canonical index keys:

```json
{
  "aliases": {
    "ADD": "p2kbPasm2Add",
    "COGINIT": "p2kbPasm2Coginit",
    "WAITMS": "p2kbSpin2Waitms",
    "_CLKFREQ": "p2kbSpin2SpecialConfigurationSymbols",
    "motor_controller": "p2kbSpin2Spin2MotorController",
    // ... 933 total entries
  }
}
```

### Alias Categories

Aliases are harvested from multiple sources in YAML files:

1. **Instruction Mnemonics** (PASM2): `ADD`, `MOV`, `JMP`, etc.
2. **Method Names** (Spin2): `WAITMS`, `COGINIT`, `PINWRITE`, etc.
3. **Pattern IDs**: `motor_controller`, `state_machine`, etc.
4. **Explicit Aliases**: Values from `aliases:` field in YAML
5. **Symbol Names**: `_CLKFREQ`, `_CLKMODE`, `_XTLFREQ`, etc.

---

## Required MCP Server Changes

### 1. Load Aliases on Index Initialization

When the MCP server loads `p2kb-index.json`, it must:

```python
def load_index(self, index_path: str):
    with open(index_path) as f:
        index_data = json.load(f)

    self.entries = index_data.get("entries", {})
    self.categories = index_data.get("categories", {})
    self.aliases = index_data.get("aliases", {})  # NEW
```

### 2. Implement Alias Resolution in Key Lookup

All key lookup operations must first check the alias table:

```python
def resolve_key(self, key: str) -> Optional[str]:
    """
    Resolve a key that might be:
    1. A canonical P2KB key (p2kbPasm2Add)
    2. An alias (ADD, add)
    3. An unknown key
    """
    # Direct lookup first
    if key in self.entries:
        return key

    # Check aliases (case-insensitive)
    alias_key = self.aliases.get(key.upper())
    if alias_key and alias_key in self.entries:
        return alias_key

    # Try lowercase for case-insensitive matching
    alias_key = self.aliases.get(key.lower())
    if alias_key and alias_key in self.entries:
        return alias_key

    return None
```

### 3. Update All Public API Methods

Every MCP tool that looks up documentation must use `resolve_key()`:

```python
def get_documentation(self, key: str) -> dict:
    resolved = self.resolve_key(key)
    if not resolved:
        return {"error": f"Unknown key: {key}"}
    return self.entries[resolved]

def search_entries(self, query: str) -> list:
    # Include alias matches in search results
    results = []

    # Check if query matches an alias
    if query.upper() in self.aliases:
        canonical = self.aliases[query.upper()]
        if canonical in self.entries:
            results.append(self.entries[canonical])

    # Continue with regular search...
```

### 4. Provide Alias Information in Responses

When returning entry data, include resolved alias information:

```python
def get_documentation(self, key: str) -> dict:
    resolved = self.resolve_key(key)
    if not resolved:
        return {"error": f"Unknown key: {key}"}

    entry = self.entries[resolved].copy()

    # Add resolution metadata if aliased
    if key != resolved:
        entry["_resolved_from"] = key
        entry["_canonical_key"] = resolved

    return entry
```

---

## Known Alias Conflicts

Some aliases map to multiple entries (both PASM2 and Spin2 have instructions with the same name). The first match wins during index generation. Current conflicts (27 instruction pairs):

- `ABS`, `AKPIN`, `CALL`, `COGATN`, `COGID`, `COGINIT`, `COGSTOP`
- `GETCT`, `GETRND`, `HUBSET`, `LOCKNEW`, `LOCKREL`, `LOCKRET`, `LOCKTRY`
- `MOVBYTS`, `POLLATN`, `QEXP`, `QLOG`, `RDPIN`, `RQPIN`, `WAITATN`
- `WRPIN`, `WXPIN`, `WYPIN`

The MCP server may want to provide context-aware resolution in the future (e.g., if user is asking about PASM2 code, prefer PASM2 entries).

---

## Backward Compatibility

The aliases section is additive. Existing behavior of looking up canonical keys (`p2kbPasm2Add`) continues to work. The alias resolution is a new capability that allows more natural queries.

---

## Testing Requirements

1. **Direct Key Lookup**: `get_documentation("p2kbPasm2Add")` returns entry
2. **Alias Resolution**: `get_documentation("ADD")` resolves and returns entry
3. **Case Insensitivity**: `get_documentation("add")` resolves and returns entry
4. **Unknown Key**: `get_documentation("NOTAKEY")` returns appropriate error
5. **Conflict Handling**: `get_documentation("ABS")` returns first match (PASM2)

---

## Implementation Priority

1. **Phase 1**: Load aliases and implement basic resolution (required)
2. **Phase 2**: Include resolution metadata in responses (recommended)
3. **Phase 3**: Context-aware conflict resolution (optional future enhancement)

---

## Files Affected

- `p2kb-index.json` - Enhanced with aliases section (already done)
- MCP server index loading code - Must parse aliases
- MCP server lookup methods - Must use resolve_key()
- MCP server response formatting - May include resolution metadata
