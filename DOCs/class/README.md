# AI-Assisted Propeller 2 Development — Zoom Class

> **Disclaimer:** The pricing, plan details, and feature availability referenced in this document and the linked setup guides reflect our best research as of February 2026. These products and their plans change frequently — verify current pricing and features on each vendor's website before purchasing. Your mileage may vary.

## Welcome

You're invited to a live Zoom session where we'll demonstrate how to use an AI coding agent to develop real software for the Parallax Propeller 2 (P2). This isn't a theoretical walkthrough — we'll be writing actual code, talking to actual hardware, and showing you what's possible when an AI agent has access to the P2 Knowledge Base.

## Why Attend?

If you develop for the Propeller 2, you've likely written Spin2 or PASM2 code by hand — looking up instructions, referencing datasheets, and wiring up peripherals one register at a time. That workflow still works. But there's now a faster path.

With an AI coding agent connected to the **P2 Knowledge Base MCP server**, you can:

- **Describe what you want in plain language** and have the agent generate working Spin2/PASM2 code
- **Ask the agent to look up any PASM2 instruction or Spin2 built-in method** — it has the full reference at its fingertips
- **Point the agent at a datasheet or device documentation** and have it figure out the register map, protocol, and initialization sequence
- **Search the OBEX** (Parallax Object Exchange) for existing community drivers and objects to build on
- **Iterate interactively** — review what the agent wrote, ask for changes, add features, fix bugs, all in conversation

The goal of this session is to show you what a configured environment looks like and what you can actually do with it.

## What You'll See

The session is structured as a live demonstration with audience participation. We'll walk through the full workflow of starting a new P2 project from scratch with an AI agent.

### 1. Setting Up a New Project

We'll create a fresh project directory and show how to structure it so the AI agent can work effectively — where source files go, how to give the agent context about your project, and how to configure the compiler toolchain.

### 2. Defining the Hardware Configuration

Every P2 project starts with "what's connected to what." We'll define the pin assignments and hardware configuration — which pins are driving which peripherals, what voltage levels, what protocols. The agent uses this information as context when generating code, so it writes code that matches your actual wiring.

### 3. Having the Agent Read Documentation and Write Code

This is the core of the demonstration. We'll:

- Give the agent a device datasheet or reference document
- Describe what we want the project to do
- Watch the agent generate Spin2/PASM2 code based on the hardware configuration, the datasheet, and the P2 Knowledge Base
- Compile the code with PNut_TS and download it to the board with pnut-term-ts
- See it run on real hardware

We expect to have a couple of participants with their own hardware setups — different boards, different peripherals, different pinouts — so you'll see the agent adapt to different configurations, not just one canned demo.

### 4. What a Configured Environment Enables

Beyond the live demo, we'll discuss the bigger picture: what does it mean to have this environment set up and working? What kinds of projects become easier? How does the agent handle multi-file projects, debugging, and iterating on code? What are the current limitations, and where is this headed?

## Before the Session: Set Up Your Tools

To follow along (or try things yourself during the session), please install the required tools **before** the Zoom call. We've prepared platform-specific setup guides with step-by-step instructions:

| Your Platform | Setup Guide |
|--------------|-------------|
| **Windows** | [SETUP-WINDOWS.md](SETUP-WINDOWS.md) |
| **macOS** | [SETUP-MACOS.md](SETUP-MACOS.md) |
| **Linux / Raspberry Pi** | [SETUP-LINUX.md](SETUP-LINUX.md) |

Each guide covers installation of:

- **PNut_TS** — the Propeller 2 compiler (TypeScript edition)
- **pnut-term-ts** — the terminal downloader for loading binaries onto your P2 board
- **Claude Code** — the AI coding agent (CLI) from Anthropic
- **P2KB-MCP** — the P2 Knowledge Base MCP server that gives the agent P2-specific knowledge
- **Audio/dictation setup** — so you can speak to the agent hands-free if you wish

## Preferred Setup: Claude Code

For this class we prefer that participants use **Claude Code** (Anthropic's terminal CLI). This is the environment we've extensively tested with the P2 Knowledge Base MCP server, and we know it works reliably. The minimum subscription is the **Claude Pro plan at $20/month**.

The setup guides also document alternative paths — Claude Desktop (free tier), GitHub Copilot in VS Code, Cursor, and the standalone OpenAI Codex CLI — for participants who prefer a different tool or want to explore on their own after the session. These alternatives all support MCP and should work with the P2 Knowledge Base, but they are as yet untested.

## What to Bring

- **Your development machine** with the tools installed (see setup guide above)
- **A P2 board USB-connected to your development machine** so we can compile, download code to the P2, and run the debugger during the session
- **A datasheet or reference document** for whatever device you have wired up — the agent can read PDFs and web pages
- **Curiosity** — this is new territory for P2 development, and we're all learning together

## Zoom Details

[TODO: Date, time, and Zoom link]

---

*Questions before the session? Post in the Parallax forums or reach out directly.*
