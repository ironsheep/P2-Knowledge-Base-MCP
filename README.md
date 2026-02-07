# P2 Knowledge Base MCP

[![CI](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml)
[![Release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Turn your AI assistant into an interactive reference manual for the Parallax Propeller 2.**

P2KB MCP connects Claude, Codex, and Cursor to the [Propeller 2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base) and [OBEX (Parallax Object Exchange)](https://obex.parallax.com/), giving your AI direct access to PASM2 instructions, Spin2 methods, P2 hardware architecture, and community code objects.

---

## What You Can Do

Once installed, just ask your AI in plain English. Here are some examples of what becomes possible:

### PASM2 Assembly Reference

Your AI becomes an interactive PASM2 manual — ask about any instruction and get syntax, operands, flag effects, encoding details, and usage examples.

> *"Explain the MOV instruction — what flags does it affect?"*
>
> *"What's the difference between RDLONG and RDLUT?"*
>
> *"Show me all the conditional execution prefixes for PASM2."*
>
> *"How does WAITX work? Can you show an example of a microsecond delay loop?"*

### Spin2 Language Help

Get details on any built-in Spin2 method — parameters, return values, and how to use them.

> *"How do I use PINSTART to configure a Smart Pin?"*
>
> *"What are the parameters for the Spin2 SEND method?"*
>
> *"List all the Spin2 string-handling methods."*
>
> *"Show me how to set up a serial port in Spin2."*

### P2 Hardware & Architecture

Ask about COGs, HUB memory, Smart Pins, clock configuration, and other hardware topics.

> *"How does the P2 HUB memory bus work?"*
>
> *"Explain the Smart Pin modes available for PWM output."*
>
> *"How many COGs does the P2 have and how do they share resources?"*
>
> *"What are the P2 clock configuration options?"*

### OBEX Community Code

Search and discover the ~113 community objects in the Parallax Object Exchange — drivers, libraries, and examples contributed by the P2 community.

> *"Find me an OBEX object for driving WS2812B LEDs."*
>
> *"Are there any I2C sensor drivers in the OBEX?"*
>
> *"Show me what motor control objects are available."*
>
> *"What OBEX objects has Jon McPhalen published?"*

### Cross-Topic Questions

Combine knowledge areas — your AI can pull from multiple parts of the knowledge base in a single conversation.

> *"I need to bit-bang SPI in PASM2. Show me the relevant instructions and then find an OBEX object I could use instead."*
>
> *"Help me understand how to launch a PASM2 COG from Spin2 — what Spin2 methods and PASM2 instructions are involved?"*

---

## Installation

P2KB MCP supports Claude (Desktop and Code), Cursor, and Codex. Choose the guide for your AI tool:

| Guide | For |
|-------|-----|
| **[INSTALL.md](INSTALL.md)** | **Claude Desktop** and **Claude Code** — platform-specific instructions for Windows, macOS, and Linux |
| **[INSTALL-CURSOR.md](INSTALL-CURSOR.md)** | **Cursor** IDE users |
| **[INSTALL-CODEX.md](INSTALL-CODEX.md)** | **OpenAI Codex** CLI and IDE extension users |
| **[INSTALL-ADVANCED.md](INSTALL-ADVANCED.md)** | **Advanced users** who manage multiple MCPs with the container-tools framework |

Each guide walks you through three steps: download the binary, install it, and connect it to your AI tool.

---

## Pair Programming with PNut-TS

Take it further: when [PNut-TS](https://github.com/ironsheep/PNut-TS) (the cross-platform Spin2/PASM2 compiler) and [PNut-Term-TS](https://github.com/ironsheep/PNut-Term-TS) (the P2 downloader and debug terminal) are installed on your machine, your AI assistant becomes a full **pair programmer** — not just a reference, but an active collaborator that can write, compile, download, and debug P2 code alongside you.

With all three tools available, you give high-level engineering intent and your AI handles the rest:

> *"I need a FIFO to queue data from one COG to another. Create it as a standalone Spin2 object."*
>
> *"Write regression tests for the FIFO — test empty, full, overflow, and underflow conditions."*
>
> *"Run the regression tests on the P2 and make sure the FIFO is completely working."*
>
> *"Now add a PASM2 driver COG that pushes sensor readings into the FIFO at 10 kHz."*

Your AI looks up the correct PASM2 instructions and Spin2 methods from the Knowledge Base, writes the code, compiles with `pnut-ts`, downloads to the P2 with `pnut-term-ts`, reads the debug output, and iterates until the tests pass — all autonomously within a single conversation.

Or start from the hardware side — hand your AI a new peripheral and let it plan the architecture:

> *"Here's the datasheet for the BME280 sensor. What interface options does it support, and what data rates can we achieve?"*
>
> *"Which P2 features are the best fit for talking to this device — Smart Pin SPI, bit-banged, or something else?"*
>
> *"Show me the performance I could achieve with each approach, given P2 clock and instruction timing."*
>
> *"Write up an architecture plan for the driver."*

Your AI researches the hardware documentation online, maps the interface requirements against P2 capabilities from the Knowledge Base — Smart Pin modes, instruction cycle timing, streamer bandwidth — and produces an engineering plan showing what's achievable before writing a single line of code.

Then you wire up the hardware and hand it back to your AI:

> *"I've wired the BME280 to the P2 — SDA is on pin 40, SCL on pin 41, power on 3.3V. Build the driver and verify it reads valid sensor data."*

From there, your AI generates the code with the correct pin assignments, compiles, downloads to the P2, and validates against real hardware — closing the loop from datasheet to working driver.

| Tool | What It Provides |
|------|-----------------|
| **P2KB MCP** | Accurate PASM2, Spin2, and hardware knowledge so the AI writes correct P2 code |
| **[PNut-TS](https://github.com/ironsheep/PNut-TS)** | Cross-platform Spin2/PASM2 compiler (Windows, macOS, Linux, RPi) |
| **[PNut-Term-TS](https://github.com/ironsheep/PNut-Term-TS)** | P2 downloader + serial terminal + debug display (Windows, macOS, Linux, RPi) |

PNut-TS and PNut-Term-TS are command-line tools that Claude Code, Codex, and Cursor can invoke directly. No special configuration is needed beyond having them on your PATH.

**Tip:** After installing PNut-TS and PNut-Term-TS, ask your AI to learn them:

> *"Run `pnut-ts --help` and `pnut-term-ts --help` so you know how to use the P2 compiler and terminal tools."*

This teaches your AI the full command syntax, flags, and options so it can use both tools effectively throughout your session.

### AI-Ready: Headless Mode

PNut-Term-TS includes a **headless mode** designed specifically for AI agent and CI workflows. This lets your AI download code to the P2, capture serial/debug output, and determine when a run is complete — all without a GUI:

```bash
# Download to RAM, run until the program prints END_SESSION
pnut-term-ts --headless -r program.bin --end-marker

# Download to RAM, run for 30 seconds, then exit
pnut-term-ts --headless -r program.bin --timeout 30

# Wait for a custom end phrase in the output
pnut-term-ts --headless -r test.bin --end-marker "TEST_DONE"

# Download to FLASH for persistent storage
pnut-term-ts --headless -f program.bin --timeout 10
```

In headless mode, all serial and debug output from your Spin2 program is captured to a log file. When the run completes, your AI reads the log, sees what worked and what didn't, makes code corrections, and recompiles — a full compile-download-test-fix loop running autonomously on real hardware.

---

## How It Works

P2KB MCP is an [MCP server](https://modelcontextprotocol.io/) — a small program that runs locally on your machine. When your AI tool needs Propeller 2 information, it queries the MCP server, which fetches and caches content from the P2 Knowledge Base on GitHub. You don't need to download any documentation manually; the server handles it automatically.

The knowledge base covers:

| Area | Content |
|------|---------|
| **PASM2** | All assembly instructions with syntax, encoding, flag effects, and examples |
| **Spin2** | Built-in methods with parameters, return values, and usage |
| **Architecture** | COG, HUB, Smart Pins, and hardware documentation |
| **Guides** | Quick reference cards and getting-started material |
| **OBEX** | ~113 community code objects — drivers, libraries, and demos |

---

## Documentation

- [Changelog](CHANGELOG.md) — Version history
- [API Reference](DOCs/API.md) — MCP tool specifications (for developers)
- [Testing & Coverage](DOCs/TESTING.md) — Test strategy and metrics

## License

MIT License — see [LICENSE](LICENSE) for details.

## Related

- [P2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base) — Documentation source
- [OBEX](https://obex.parallax.com/) — Parallax Object Exchange
- [Propeller 2](https://www.parallax.com/propeller-2/) — The microcontroller
- [Model Context Protocol](https://modelcontextprotocol.io/) — MCP specification

---

*Iron Sheep Productions, LLC*
