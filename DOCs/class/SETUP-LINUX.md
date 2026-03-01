# Pre-Class Setup Guide — Linux

Welcome! Before our Zoom session, please install the tools listed below so we can hit the ground running. Set aside about 30 minutes—most steps are quick downloads and single-line installs.

---

## 1. PNut_TS Compiler

PNut_TS is the TypeScript-based Propeller 2 compiler (the name stands for **P**-**N**-**u**-**t** **T**ype**S**cript).

**Install instructions:** [Install/Update PNut_TS on RPi/Linux](https://github.com/ironsheep/P2-vscode-langserv-extension/blob/main/TASKS-User-RPi.md#installing-pnut-ts-on-rpilinux)

## 2. PNut_TS Downloader (pnut-term-ts)

The PNut_TS terminal downloader (`pnut-term-ts`) fetches P2 binaries to your board.

**Install instructions:** [Install/Update pnut-term-ts on RPi/Linux](https://github.com/ironsheep/P2-vscode-langserv-extension/blob/main/TASKS-User-RPi.md#installing-pnut-term-ts-on-rpilinux)

## 3. Claude Code CLI

Claude Code is Anthropic's command-line interface for Claude.

**Install instructions:** Follow the [Claude Code Quickstart Guide](https://code.claude.com/docs/en/quickstart) — it covers prerequisites and installation for all platforms.

After install, verify it works by opening a terminal and running:

```bash
claude --version
```

### Claude Subscription Required

Claude has a **Free** tier, but it only covers the web/desktop/mobile chat — it does **not** include Claude Code (the CLI). To use Claude Code you need at least a **Pro** subscription. See [Claude Pricing](https://claude.com/pricing) for full details. Here are the relevant plans:

| Plan | Cost | Claude Code CLI | Models Available | Usage Limit | Best For |
|------|------|:-:|-----------------|-------------|----------|
| **Free** | $0 | No — Desktop only | Sonnet 4.6, Haiku 4.5 | ~5-10 tool calls/hour; smaller context window | Light lookups via Claude Desktop. No terminal workflow. |
| **Pro** | $20/month | Yes | Sonnet 4.6, Haiku 4.5 | ~45 messages per 5-hour window | Learning, light use. You may hit limits during extended coding sessions. |
| **Max 5x** | $100/month | Yes | Sonnet 4.6, Haiku 4.5, **Opus 4.6** | ~225 messages per 5-hour window | Regular daily development with occasional Opus use. |
| **Max 20x** | $200/month | Yes | Sonnet 4.6, Haiku 4.5, **Opus 4.6** | ~900 messages per 5-hour window | Heavy daily use; effectively unlimited for most sessions. |

**What this means in practice:**

- **Pro ($20/mo)** is the minimum for our class. Sonnet 4.6 is very capable and handles most coding tasks well. Expect roughly 1-2 hours of active back-and-forth before hitting the 5-hour window limit, depending on task complexity.
- **Opus 4.6** is the most capable model (deeper reasoning, better multi-file work) but is only available on Max plans.
- **Haiku 4.5** is the fastest and cheapest model — great for quick lookups and simple tasks.
- All models share a **1 million token context window**, so Claude can hold large codebases in memory during a session.

> **For this class:** The **Pro plan ($20/mo)** with Sonnet is sufficient. You can always upgrade later if you find yourself wanting more.
>
> **Free tier alternative:** If you'd prefer not to subscribe, you can skip the Claude Code CLI install and instead use the [Claude Desktop](https://claude.ai/download) app (free) with the P2KB-MCP server connected to it (see Section 4, Option B). The free tier supports MCP servers, file access, and tool execution — so you can still look up PASM2 instructions, search OBEX, etc. However, you'll be limited to roughly 5-10 tool calls per hour and a smaller context window. You also won't have the CLI terminal workflow we'll demo in class.

## 4. P2 Knowledge Base MCP Server

The P2KB-MCP server gives Claude access to PASM2 instructions, Spin2 methods, and OBEX objects.

### Download

Go to the [Releases page](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases) and download the Linux package:

| Your System | Download File |
|-------------|--------------|
| 64-bit (x86_64) | `p2kb-mcp-vX.X.X-linux-amd64.tar.gz` |
| ARM64 (aarch64) | `p2kb-mcp-vX.X.X-linux-arm64.tar.gz` |

> **Which one?** Run `uname -m` in a terminal — `x86_64` means 64-bit, `aarch64` means ARM64.

### Install

In your terminal, run:

```bash
cd ~/Downloads
tar -xzf p2kb-mcp-v*-linux-*.tar.gz
sudo mv p2kb-mcp /opt/p2kb-mcp
```

Verify:

```bash
/opt/p2kb-mcp/bin/p2kb-mcp --version
```

### Connect to Claude — Option A: Claude Code (CLI)

If you have a **Pro or Max** subscription, run this in your terminal:

```bash
claude mcp add -s user p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp --mode stdio
```

This registers the MCP server for all your Claude Code sessions. Next time you start Claude Code, the P2 Knowledge Base tools will be available.

### Connect to Claude — Option B: Claude Desktop (Free tier)

If you're using the **free tier**, install [Claude Desktop](https://claude.ai/download) and edit its configuration file to add the MCP server.

1. Open the config file at `~/.config/claude/claude_desktop_config.json` in a text editor (create it if it doesn't exist)
2. Add (or merge into) the `mcpServers` section:

   ```json
   {
     "mcpServers": {
       "p2kb-mcp": {
         "command": "/opt/p2kb-mcp/bin/p2kb-mcp",
         "args": ["--mode", "stdio"]
       }
     }
   }
   ```

   > If you already have other MCP servers in the file, add just the `"p2kb-mcp": { ... }` block inside the existing `"mcpServers"` object.

3. **Restart Claude Desktop** after saving the config file.

---

## 5. Audio Setup (Speech-to-Text for Terminal Use)

Linux doesn't have a single standard dictation feature, but **Speech Note** is a straightforward, offline, GUI-based tool that works on any distro supporting Flatpak.

### Verify Microphone Access

1. Confirm your user is in the `audio` group:

   ```bash
   sudo usermod -a -G audio $USER
   ```

   Then **log out and back in** if you weren't already in the group.

2. Ensure **PulseAudio** or **PipeWire** is running (standard on most modern distros).

### Install Speech Note (Flatpak)

1. Make sure Flatpak is installed and Flathub is configured. On Debian/Ubuntu:

   ```bash
   sudo apt install flatpak
   flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo
   ```

2. Install Speech Note from your software center (search "Speech Note") or via the command line.

### Configure Speech Note

1. Launch **Speech Note** from the app menu
2. On first run, download the English language model (note: models can be ~2 GB)
3. In settings, choose the correct **microphone device**
4. Set the listening mode — **"always on"** keeps listening until you stop it; **"press and hold"** is also available

### Using Speech Note with Claude Code

1. Open your preferred terminal (GNOME Terminal, Konsole, etc.) and run `claude`
2. Place the cursor where you'd type a command
3. Switch to Speech Note, click **Listen** (or use its hotkey if configured)
4. Speak — after a short lag, Speech Note transcribes into its window
5. Copy the transcribed text and paste it into the terminal
6. Press **Enter** to send it

> **For more technical users:** [nerd-dictation](https://github.com/ideasman42/nerd-dictation) is a Vosk-based Python script that pipes transcribed text directly into the active window, skipping the copy/paste step. It requires more setup but is very effective.

## 6. Wispr (Optional — AI Voice Input)

[Wispr](https://www.wispr.com/) is an AI-powered voice-to-text tool that listens to your microphone and pastes transcribed text directly into whatever application has focus — including the terminal. This is entirely optional, but if you'd like hands-free input to Claude Code, give it a try.

- Visit [wispr.com](https://www.wispr.com/) to sign up for a free trial or subscription
- Install the Wispr desktop app and follow their setup wizard
- **Note:** Wispr's Linux support may be limited — check their site for current platform availability

---

## Checklist

Before the class session, confirm each of the following:

- [ ] PNut_TS compiler installed and runs
- [ ] PNut_TS downloader installed and runs
- [ ] Claude Code CLI installed (`claude --version` shows a version)
- [ ] P2KB-MCP server installed and connected to Claude Code
- [ ] Microphone working (test with `arecord -d 3 /tmp/test.wav && aplay /tmp/test.wav` or your desktop's Sound settings)
- [ ] (Optional) Wispr installed and working

See you on Zoom!

---

## Appendix: Alternative AI Coding Tools

Our class uses **Claude Code** (Anthropic's terminal CLI), and the P2 Knowledge Base MCP server has been **extensively tested with Claude Code**. However, there are other AI coding tools that also support MCP servers — meaning they can also connect to the P2 Knowledge Base. These alternatives are **as yet untested with P2KB-MCP but should work**, as they all follow the same MCP standard. If you already use one of these, or are curious, here's a quick overview.

### GitHub Copilot in VS Code

If you already use **VS Code with the Spin2 extension**, [GitHub Copilot](https://github.com/features/copilot) is the most natural addition — it works right inside your existing editor. Copilot supports MCP servers in VS Code (v1.99+), so you can connect the P2 Knowledge Base without leaving the IDE you already use for P2 development.

**Install:** Install the "GitHub Copilot" extension from the VS Code marketplace.

**Platform support:** Windows, macOS, Linux (anywhere VS Code runs).

**MCP support:** Yes — configure via `.vscode/mcp.json` in your project or through VS Code's user settings. MCP is enabled by default (`chat.mcp.enabled`). MCP is available on **all tiers, including Free**.

**Copilot also includes OpenAI Codex:** On paid tiers, Copilot includes access to the OpenAI Codex coding agent (GPT-5.3-Codex) and a Copilot CLI for terminal use — no separate OpenAI/ChatGPT subscription needed.

| Plan | Cost | Completions | Premium Requests/month | Copilot CLI | Notes |
|------|------|-------------|----------------------|:-----------:|-------|
| Free | $0 | 2,000/month | 50 | No | MCP supported. Good for light use and lookups. |
| Pro | $10/month | Unlimited | 300 | Yes | Includes Codex agent and Copilot CLI. |
| Pro+ | $39/month | Unlimited | 1,500 | Yes | Access to all models including Claude Opus 4, o3. |

> **If you're already a VS Code + Spin2 user:** Copilot Free gives you MCP support at no cost — just install the extension, configure the P2KB-MCP server in `.vscode/mcp.json`, and you can query PASM2 instructions and OBEX objects directly from Copilot Chat. Paid tiers add the Codex agent for autonomous multi-file coding and a terminal CLI.

### OpenAI Codex CLI (Standalone)

[Codex CLI](https://developers.openai.com/codex/cli) is OpenAI's **standalone** terminal-based coding agent — similar in concept to Claude Code but powered by OpenAI models (default: GPT-5.3-Codex). It's open source, built in Rust, and supports MCP servers via `~/.codex/config.toml`.

> **Note:** This is a separate product from the Codex agent built into GitHub Copilot (above). The standalone Codex CLI requires its own **ChatGPT Plus** subscription — it does not use your Copilot subscription.

**Install:** `npm i -g @openai/codex` (requires Node.js). See the [Codex CLI docs](https://developers.openai.com/codex/cli) for full setup instructions.

**Platform support:** macOS and Linux fully supported; Windows is experimental (WSL recommended).

**MCP support:** Yes — both STDIO and streaming HTTP servers. Configure in `~/.codex/config.toml` or via `codex mcp add`. See our [P2KB-MCP Codex setup guide](../../INSTALL-CODEX.md) for details.

| Plan | Cost | Notes |
|------|------|-------|
| ChatGPT Plus | $20/month | Includes Codex CLI access |
| ChatGPT Pro | $200/month | Higher limits |
| ChatGPT Business | Per-seat pricing | Team features |

### Cursor IDE

[Cursor](https://www.cursor.com/) is a VS Code-based IDE with built-in AI coding assistance. It supports MCP servers natively and offers agentic multi-file editing, background agents, and tab completions.

**Install:** Download from [cursor.com/downloads](https://www.cursor.com/downloads)

**Platform support:** Windows, macOS, Linux.

**MCP support:** Yes — configure via Cursor Settings UI or `~/.cursor/mcp.json`. See our [P2KB-MCP Cursor setup guide](../../INSTALL-CURSOR.md) for details.

| Plan | Cost | Notes |
|------|------|-------|
| Hobby (Free) | $0 | Limited completions and agent requests; includes MCP support |
| Pro | $20/month | Unlimited completions, extended agent requests |
| Pro+ | $60/month | 3x Pro usage |
| Ultra | $200/month | 20x Pro usage |

> **Note:** Cursor's free Hobby tier includes MCP support — so you can connect the P2 Knowledge Base at no cost, though with limited request counts.
