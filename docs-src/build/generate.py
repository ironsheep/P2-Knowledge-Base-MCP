#!/usr/bin/env python3
"""
Generate every install document in the P2 tool fleet from one set of facts.

    python3 build/generate.py            # write out/
    python3 build/generate.py --check    # CI: fail if out/ is stale or non-conformant

Two slices are produced from the same cells:

    out/GETTING-STARTED-<platform>.md   a COLUMN: all tools, one platform
                                        (lives in P2-Knowledge-Base-MCP)
    out/tools/<tool>/INSTALL.md         a ROW: one tool, all platforms
                                        (lives in that tool's own repo)

No path, asset name, or launcher name is ever typed in prose. All of it is
interpolated from data/, which is why the two slices cannot disagree.
"""

import sys
import hashlib
import pathlib
import difflib

import yaml

ROOT = pathlib.Path(__file__).resolve().parent.parent
DATA = ROOT / "data"
FRAG = ROOT / "fragments"
TPL = ROOT / "templates"
# keys below are repo-relative paths; each doc goes where readers look for it
OUT = ROOT.parent

# NOTE: prototype shortcut — these belong in platforms.yml alongside the rest
# of the platform base. Left here only to keep the diff readable.
HOST_CONFIG = {
    "macos": {
        "claude_desktop_config": "~/Library/Application Support/Claude/claude_desktop_config.json",
        "cursor_config": "~/.cursor/mcp.json",
    },
    "linux": {
        "claude_desktop_config": "~/.config/claude/claude_desktop_config.json",
        "cursor_config": "~/.cursor/mcp.json",
    },
    "windows": {
        "claude_desktop_config": r"%APPDATA%\Claude\claude_desktop_config.json",
        "cursor_config": r"%USERPROFILE%\.cursor\mcp.json",
    },
}


# --------------------------------------------------------------------------
# tiny renderer — {{key}} substitution only. Deliberately not a template
# language: if a doc needs logic, that is a signal the fact belongs in data/.
# --------------------------------------------------------------------------
def render(text: str, ctx: dict) -> str:
    for key, val in ctx.items():
        text = text.replace("{{" + key + "}}", str(val))
    return text


def source_fingerprint() -> str:
    """Short hash over every authoring input.

    Deliberately NOT a git commit or a timestamp: those change on every commit
    and would make --check report drift when nothing actually changed. This
    changes only when data/, fragments/, or templates/ change — so a reader can
    tell two documents were generated from the same facts, and CI stays stable.
    """
    h = hashlib.sha256()
    for d in (DATA, FRAG, TPL):
        for f in sorted(d.rglob("*")):
            if f.is_file():
                h.update(f.relative_to(ROOT).as_posix().encode())
                h.update(f.read_bytes())
    return h.hexdigest()[:12]


def banner(fingerprint: str) -> str:
    return (
        "<!-- GENERATED FILE — do not edit.\n"
        "     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.\n"
        f"     source fingerprint: {fingerprint} -->\n\n"
    )


def load():
    platforms = yaml.safe_load((DATA / "platforms.yml").read_text())
    tools = yaml.safe_load((DATA / "tools.yml").read_text())
    env = yaml.safe_load((DATA / "environment.yml").read_text())
    env["class"] = yaml.safe_load((DATA / "class.yml").read_text())["class"]
    accepted = {a["id"] for a in tools.pop("_accepted", [])}
    probes = tools.pop("_extension_probes")
    search_names = tools.pop("_extension_search_names")
    return platforms, tools, env, probes, search_names, accepted


def variant_section(tools, platform, plat, tool_id="p2kb-mcp") -> str:
    """Advanced install variants, rendered at the very END of a guide.

    Deliberately not a fork in the main flow: a reader following the standard
    path should never have to evaluate this. Part 1 carries only a one-line
    pointer, and it is written so the correct action for most readers is to
    stop reading.
    """
    tool = tools.get(tool_id, {})
    out = []
    for vid, v in (tool.get("variants") or {}).items():
        frag = FRAG / "tools" / tool_id / f"variant-{vid}.md"
        if not frag.exists():
            continue
        sep = sep_for(platform)
        ctx = {
            "skip_unless": v["skip_unless"].strip(),
            "variant_asset": v["asset"].replace("{version}", "<version>"),
            "variant_install_dir": v["install_dir"],
            "variant_bin_launcher": f"{v['bin_dir']}{sep}{v['launcher']}",
            "shell_installer": v.get("shell_installer", "install.sh"),
            "releases_url": f"https://github.com/{tool['repo']}/releases/latest",
        }
        out.append(f"## Advanced: {v['name']}\n")
        if platform == "windows" and v.get("windows_caveat"):
            out.append(f"> {v['windows_caveat'].strip()}\n")
        out.append(render(frag.read_text(), ctx))
    return "\n".join(out)


def prereq_block(env, tool_ids) -> str:
    """Prerequisites for a given set of tools, rendered where they are needed.

    Emitted per-part rather than in one list up front, so Part 1 never mentions
    VS Code or a PropPlug — keeping Part 1 a true standalone prefix.
    """
    items = [p for p in env["prerequisites"]
             if set(p["for"]) & set(tool_ids)]
    if not items:
        return ""
    lines = ["**Before you start, you need:**\n"]
    for p in items:
        name = f"[{p['name']}]({p['url']})" if p.get("url") else f"**{p['name']}**"
        lines.append(f"- {name} — {p['detail'].strip()}")
    return "\n".join(lines) + "\n"


def sep_for(platform: str) -> str:
    return "\\" if platform == "windows" else "/"


def context(tool_id, tool, platform, plat) -> dict:
    p = tool["platforms"][platform]
    sep = sep_for(platform)
    install_dir = p.get("install_dir", "")
    bin_dir = p.get("bin_dir", "")
    launcher = p.get("launcher", "")
    folder = install_dir.split(sep)[-1] if install_dir else ""
    bin_launcher = f"{bin_dir}{sep}{launcher}" if bin_dir else launcher

    asset = (
        p.get("asset", "")
        .replace("{arch}", "<arch>")
        .replace("{version}", "<version>")
        .replace("{packed}", "<version>")
    )

    ctx = {
        "tool_name": tool["name"],
        "tool_role": tool["role"],
        "blurb": tool.get("blurb", "").strip(),
        "repo": tool["repo"],
        "releases_url": f"https://github.com/{tool['repo']}/releases/latest",
        "download_url": tool.get("download_url", ""),
        "download_url_repo": (f"https://github.com/{tool['repo']}"
                              if tool.get("repo") else ""),
        "extra_files": ", ".join(f"`{f}`" for f in p.get("extra_files", [])),
        "marketplace_id": tool.get("marketplace_id", ""),
        "platform_name": plat["name"],
        "install_root": plat["install_root"],
        "install_dir": install_dir,
        "bin_dir": bin_dir,
        "launcher": launcher,
        "folder": folder,
        "asset": asset,
        "dmg_payload": f"`{folder}`",
        "bin_dir_launcher": bin_launcher,
        "install_dir_bin_launcher": bin_launcher,
        "bin_dir_launcher_json": bin_launcher.replace("\\", "\\\\"),
        "path_stanza": plat.get("path_stanza", "").replace("{bin_dir}", bin_dir),
        "verify_cmd": plat.get("verify_cmd", "").replace("{launcher}", launcher),
        "which_cmd": plat.get("which_cmd", "").replace("{launcher}", launcher),
        "quarantine_cmd": plat.get("quarantine_cmd", "").replace("{install_dir}", install_dir),
        "path_file": plat.get("path_file", ""),
    }
    ctx.update(HOST_CONFIG.get(platform, {}))
    return ctx


def recipe_for(tool_id, tool, platform, ctx) -> str:
    """Per-tool override if present, else the recipe for this (platform, archive)."""
    override = FRAG / "tools" / tool_id / "recipe.md"
    if override.exists():
        return render(override.read_text(), ctx)
    archive = tool["platforms"][platform].get("archive")
    if not archive:
        return "_No install steps declared._\n"
    path = TPL / "recipes" / f"{platform}-{archive}.md"
    if not path.exists():
        return f"_MISSING RECIPE: {platform}-{archive}_\n"
    return render(path.read_text(), ctx)


def notes_for(tool_id, platform, ctx) -> str:
    """Optional prose: shared notes, then platform-specific notes."""
    out = []
    for name in ("notes.md", f"notes-{platform}.md"):
        f = FRAG / "tools" / tool_id / name
        if f.exists():
            out.append(render(f.read_text(), ctx).strip())
    return "\n\n".join(out)


def update_section(tool, platform, plat, ctx) -> str:
    """The update model — stated once, emitted everywhere.

    Default is move-aside. Tools delivered by DMG use Finder's own replace
    flow instead: there is no unpack step to rename around, and the releases
    are retained upstream, so a local backup copy earns nothing.
    """
    p = tool["platforms"][platform]
    if not p.get("install_dir"):
        return ""

    if p.get("upgrade") == "finder-replace":
        body = (
            f"1. Download and open the new `.dmg` as you did when installing.\n"
            f"2. Drag **`{ctx['folder']}`** into `{ctx['install_root']}` again.\n"
            f"3. Finder will say an item of that name already exists. "
            f"Choose **Replace**.\n"
        )
        if p.get("upgrade_warn_keep_both"):
            body += (
                "\n> **Do not choose Keep Both.** It renames the *incoming* app "
                f"to `{ctx['folder'].replace('.app', ' 2.app')}` and leaves the old "
                "one in place under the original name — which is what your PATH "
                "points at. The upgrade appears to succeed while you keep running "
                "the previous version.\n"
            )
        body += (
            f"\nYour PATH entry is unchanged. Confirm with `{ctx['verify_cmd']}`.\n"
        )
        return body

    sep = sep_for(platform)
    d = ctx["install_dir"]
    if platform == "windows":
        body = (
            f"1. Delete any existing `{d}-prior`.\n"
            f"2. Rename `{d}` to `{d}-prior`.\n"
            f"3. Unpack the new release into `{ctx['install_root']}`.\n"
            f"4. Verify with `{ctx['verify_cmd']}`.\n"
        )
    else:
        sudo = "sudo " if tool["platforms"][platform].get("sudo") else ""
        body = (
            "```sh\n"
            f"{sudo}rm -rf {d}-prior\n"
            f"{sudo}mv {d} {d}-prior\n"
            f"# unpack the new release, then:\n"
            f"{sudo}mv {ctx['folder']} {d}\n"
            "```\n"
        )
    return (
        "Move the old folder aside rather than overwriting it. Your PATH entry "
        "does not change, and the previous working build stays one rename away "
        "if you need to go back.\n\n" + body
    )


# --------------------------------------------------------------------------
# slice 1 — the COLUMN (one platform, all tools) → lives in P2KB-MCP
# --------------------------------------------------------------------------
def build_suite(platform, plat, platforms, tools, env) -> str:
    ordered = sorted(tools.items(), key=lambda kv: kv[1]["order"])
    here = [t for t in ordered if platform in t[1]["platforms"]]
    agentic = [t for t in here if t[1]["tier"] == "agentic"]
    toolchain = [t for t in here if t[1]["tier"] == "toolchain" and not t[1].get("optional")]
    optional = [t for t in here if t[1].get("optional")]

    L = []
    L.append(f"# Getting started on {plat['label']}\n")
    L.append(
        "This page installs the P2 tools in the order that makes each step "
        "work the first time. **Part 1 stands on its own** — stop there if all "
        "you want is an AI agent that knows the P2. Part 2 adds the hardware "
        "toolchain and is purely additive; nothing in Part 1 gets redone.\n"
    )
    others = [p["label"] for k, p in platforms.items() if k != platform]
    L.append(f"> On a different machine? See the guides for {', '.join(others)}.\n")

    L.append("## Before you start\n")
    L.append(f"**Which build do I need?** {plat['arch_detect']}\n")
    if plat.get("privilege_note"):
        L.append(f"**Privileges.** {plat['privilege_note'].strip()}\n")
    if plat.get("root_note"):
        L.append(f"> {plat['root_note'].strip()}\n")
    if plat.get("smartscreen"):
        L.append(f"**SmartScreen.** {plat['smartscreen_note'].strip()}\n")

    L.append("## Part 1 — Agentic P2 development\n")
    pb = prereq_block(env, [t[0] for t in agentic])
    if pb:
        L.append(pb)
    for tid, tool in agentic:
        ctx = context(tid, tool, platform, plat)
        L.append(f"### {tool['name']} — {tool['role']}\n")
        L.append(ctx["blurb"] + "\n")
        L.append(recipe_for(tid, tool, platform, ctx))
        n = notes_for(tid, platform, ctx)
        if n:
            L.append(n + "\n")

    if tools.get("p2kb-mcp", {}).get("variants"):
        L.append(
            "> There is also an advanced *container-tools* install, for people "
            "already running several MCP servers under one shared tree. It is "
            "described at the very end of this page. If that does not describe "
            "you, ignore it — the install above is complete.\n"
        )

    L.append(
        "---\n\n**You can stop here.** Your agent now knows the P2. Continue "
        "below when you want to compile and run code on real hardware.\n\n---\n"
    )

    L.append("## Part 2 — The hardware toolchain\n")
    pb = prereq_block(env, [t[0] for t in toolchain])
    if pb:
        L.append(pb)
    for tid, tool in toolchain:
        ctx = context(tid, tool, platform, plat)
        L.append(f"### {tool['name']} — {tool['role']}\n")
        L.append(ctx["blurb"] + "\n")
        L.append(recipe_for(tid, tool, platform, ctx))
        n = notes_for(tid, platform, ctx)
        if n:
            L.append(n + "\n")

    # Negative install steps — emitted with the tool they affect.
    if any(t[0] == "spin2-extension" for t in toolchain):
        breaks = [c for c in env["conflicts"] if c["severity"] != "obsolete"]
        obsolete = [c for c in env["conflicts"] if c["severity"] == "obsolete"]
        L.append("### Extensions to remove or disable\n")
        L.append(
            "These interfere with the Spin2 extension. Check for them now — the "
            "symptoms are confusing if you meet them later.\n"
        )
        L.append("| Extension | Marketplace ID | Do | Why |\n|---|---|---|---|")
        for c in breaks:
            L.append(
                f"| {c['name']} | `{c['id']}` | {c['action']} | "
                f"{c['symptom'].strip()} |"
            )
        L.append("")
        for c in obsolete:
            L.append(
                f"> **{c['name']}** (`{c['id']}`) is superseded — "
                f"{c['symptom'].strip()} Safe to {c['action']}.\n"
            )
        L.append("### Optional companions\n")
        L.append("Not required. Commonly used alongside the extension.\n")
        L.append("| Extension | Marketplace ID | Why |\n|---|---|---|")
        for c in env["companions"]:
            L.append(f"| {c['name']} | `{c['id']}` | {c['why']} |")
        L.append("")

    # Device access — needed before any download to the P2 will work.
    hw = env["hardware"].get(platform, [])
    if hw and any(t[1].get("needs_serial") for t in toolchain):
        L.append("### Connecting your P2\n")
        for h in hw:
            L.append(f"**{h['title']}.** {h['detail'].strip()}\n")
            if h.get("command"):
                L.append(f"```sh\n{h['command']}\n```\n")
            if h.get("reference"):
                L.append(f"Reference: [FTDI Technical Note 101]({h['reference']})\n")

    if optional:
        names = ", ".join(t[1]["name"] for t in optional)
        L.append("## Part 3 — Other compilers (optional)\n")
        L.append(
            f"PNut-TS above is all you need. {names} are alternative compilers "
            "the Spin2 extension also supports — install one only if you "
            "specifically want it. If you do, **use the locations below**: the "
            "extension looks in these exact places, and these are not always "
            "the installer's default.\n"
        )
        for tid, tool in optional:
            ctx = context(tid, tool, platform, plat)
            L.append(f"### {tool['name']} — {tool['role']}\n")
            L.append(ctx["blurb"] + "\n")
            L.append(recipe_for(tid, tool, platform, ctx))
            if tool.get("components"):
                items = "\n".join(
                    f"- `{c['name']}` — {c['where']}" for c in tool["components"]
                )
                L.append(
                    "The extension locates the rest of the bundle relative to "
                    f"`{ctx['launcher']}`, so keep the folder intact:\n\n{items}\n"
                )
            if tool.get("needs_version_literal"):
                L.append(
                    "> **Version literal.** `pnut_shell.bat` hard-codes the PNut "
                    "version (for example `pnut_v52a`). Edit it to match the "
                    "version you installed, or the extension's build will fail. "
                    "Re-check this after every PNut upgrade.\n"
                )
            n = notes_for(tid, platform, ctx)
            if n:
                L.append(n + "\n")

    # ---- platform reference, emitted once, referenced by every tool above
    anchor = plat["name"].lower().replace(" ", "-")
    L.append(f"## PATH setup on {plat['name']}\n")
    if platform == "windows":
        L.append(
            f"Open **{plat['path_file']}**, edit `Path`, and add one entry per "
            f"tool folder listed above. {plat['path_reload']}\n"
        )
    else:
        L.append(
            f"Add the `export` lines shown above to `{plat['path_file']}` "
            f"(or `{plat.get('path_file_alt', '')}`), then run "
            f"`{plat['path_reload'].replace('{path_file}', plat['path_file'])}` "
            "or open a new terminal.\n"
        )
    if plat.get("path_warning"):
        L.append(f"> {plat['path_warning'].strip()}\n")

    # Describe only the update styles actually present on THIS platform —
    # a blanket "every tool updates the same way" goes stale the moment one
    # tool diverges, and the doc then contradicts its own next section.
    styles = {t[1]["platforms"][platform].get("upgrade", "move-aside") for t in here
              if t[1]["platforms"][platform].get("install_dir")}
    L.append("## Updating\n")
    if styles == {"move-aside"}:
        L.append(
            "Every tool here updates the same way: **move the old folder aside "
            "and unpack the new one.** Your PATH never changes, and the last "
            "working build stays available as `<folder>-prior`.\n"
        )
    else:
        L.append(
            "Two patterns here, depending on how the tool was delivered. Tools "
            "you unpacked from an archive use **move-aside**: rename the old "
            "folder, unpack the new one, and the last working build stays "
            "available as `<folder>-prior`. Tools you dragged from a `.dmg` are "
            "replaced in place through Finder. **Either way your PATH entry "
            "never changes.**\n"
        )
    # Move-aside applies to EVERY tool, including third-party ones. It is our
    # convention for how a user manages an install, not a claim about how the
    # vendor ships. Only the *download* differs for third-party tools.
    for tid, tool in here:
        ctx = context(tid, tool, platform, plat)
        body = update_section(tool, platform, plat, ctx)
        if body:
            L.append(f"### {tool['name']}\n")
            L.append(body)

    L.append("## If something did not work\n")
    for item in env.get("troubleshooting", []):
        if platform not in item.get("platforms", []):
            continue
        L.append(f"**{item['title']}.** {item['detail'].strip()}\n")

    vs = variant_section(tools, platform, plat)
    if vs:
        L.append("---\n")
        L.append(vs)
    return "\n".join(L) + "\n"


# --------------------------------------------------------------------------
# slice 2 — the ROW (one tool, all platforms) → lives in that tool's repo
# --------------------------------------------------------------------------
def build_tool(tool_id, tool, platforms, env) -> str:
    L = []
    L.append(f"# Installing {tool['name']}\n")
    L.append(tool.get("blurb", "").strip() + "\n")
    L.append(
        f"> Setting up the whole P2 toolchain? Start with the "
        f"[getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) "
        f"instead — it installs this tool in the right order alongside the rest.\n"
    )
    pb = prereq_block(env, [tool_id])
    if pb:
        L.append(pb)
    for pid, plat in sorted(platforms.items(), key=lambda kv: kv[1]["order"]):
        if pid not in tool["platforms"]:
            continue
        ctx = context(tool_id, tool, pid, plat)
        L.append(f"## {plat['label']}\n")
        L.append(f"**Which build?** {plat['arch_detect']}\n")
        L.append(recipe_for(tool_id, tool, pid, ctx))
        n = notes_for(tool_id, pid, ctx)
        if n:
            L.append(n + "\n")
        body = update_section(tool, pid, plat, ctx)
        if body:
            L.append(f"### Updating on {plat['name']}\n")
            L.append(body)
    return "\n".join(L) + "\n"


# --------------------------------------------------------------------------
# conformance — the checks that make drift loud
# --------------------------------------------------------------------------
def conformance(platforms, tools, probes, search_names, accepted):
    errors, warnings = [], []

    discoverable = tools["spin2-extension"].get("discovers", [])

    for pid in platforms:
        probe_list = probes.get(pid, [])

        for tid in discoverable:
            tool = tools[tid]
            if pid not in tool["platforms"]:
                continue                      # e.g. Parallax PNut is Windows-only
            p = tool["platforms"][pid]

            # 1. is the tool's bin_dir actually probed by the extension?
            if p["bin_dir"] not in probe_list:
                errors.append(
                    f"[{pid}/{tid}] bin_dir {p['bin_dir']!r} is not in the "
                    f"extension's probe list — discovery relies on PATH alone"
                )

            # 2. does the launcher we ship match the name the extension seeks?
            want = search_names.get(pid, {}).get(tid)
            if want and want != p["launcher"]:
                msg = (f"[{pid}/{tid}] we ship {p['launcher']!r} but the "
                       f"extension searches for {want!r} — discovery cannot succeed")
                if f"{pid}/{tid}/launcher" in accepted:
                    warnings.append("ACCEPTED " + msg)
                else:
                    errors.append(msg)

        # 3. probe entries that can never match
        for entry in probe_list:
            if entry.startswith("~"):
                warnings.append(
                    f"[{pid}] probe {entry!r} starts with '~' — Node's path.join "
                    f"does not expand it, so this entry never matches"
                )

        # 4. tools installing outside the platform's declared root
        roots = platforms[pid].get("install_roots", {})
        for tid, tool in tools.items():
            d = tool["platforms"].get(pid, {}).get("install_dir", "")
            if not d:
                continue
            if tool.get("owner") == "third-party":
                continue          # vendor chooses; we document, not dictate
            root = roots.get(tool.get("kind", "cli-tool"))
            if root and not d.startswith(root):
                warnings.append(
                    f"[{pid}/{tid}] kind={tool.get('kind')} installs to {d!r}, "
                    f"outside its declared root {root!r}"
                )

        # 5. asset naming drift between tools on the same platform
        tags = {}
        for tid, tool in tools.items():
            a = tool["platforms"].get(pid, {}).get("asset", "")
            if a:
                for tag in ("win", "windows", "macos", "darwin", "linux"):
                    if f"-{tag}-" in a:
                        tags.setdefault(tag, []).append(tid)
        for a, b in (("win", "windows"), ("macos", "darwin")):
            if a in tags and b in tags:
                warnings.append(
                    f"[{pid}] asset names disagree: {tags[a]} use '-{a}-' while "
                    f"{tags[b]} use '-{b}-'"
                )

    # 6. folder/launcher separator mismatch — the bug generator
    for tid, tool in tools.items():
        for pid, p in tool["platforms"].items():
            d, l = p.get("install_dir", ""), p.get("launcher", "")
            if not d or not l:
                continue
            folder = d.split(sep_for(pid))[-1]
            if "_" in folder and "-" in l:
                warnings.append(
                    f"[{pid}/{tid}] folder {folder!r} uses '_' but launcher "
                    f"{l!r} uses '-' — easy to mistype in prose"
                )
    return errors, warnings


def build_class(platform, plat, tools, env) -> str:
    """The class slice: same cells, student framing.

    Differences from the self-serve guide are audience, not facts — cost and
    required/optional stated up front, a functional test before the checklist,
    troubleshooting present, and every optional task moved past the finish line.
    """
    c = env["class"]
    L = [f"# {c['title']} — {plat['label']}\n"]
    L.append("Set this up before class so we can start on time.\n")
    for line in c["up_front"]:
        L.append(f"- {line.strip()}")
    L.append("")
    L.append(f"**Which build do I need?** {plat['arch_detect']}\n")
    if plat.get("smartscreen"):
        L.append(f"> {plat['smartscreen_note'].strip()}\n")

    L.append("## Install\n")
    for i, tid in enumerate(c["tools"], 1):
        tool = tools[tid]
        if platform not in tool["platforms"]:
            continue
        ctx = context(tid, tool, platform, plat)
        L.append(f"### {i}. {tool['name']} — {tool['role']}\n")
        L.append(recipe_for(tid, tool, platform, ctx))
        n = notes_for(tid, platform, ctx)
        if n:
            L.append(n + "\n")

    hw = env["hardware"].get(platform, [])
    if hw:
        L.append("### Connecting your P2\n")
        for h in hw:
            L.append(f"**{h['title']}.** {h['detail'].strip()}\n")
            if h.get("command"):
                L.append(f"```sh\n{h['command']}\n```\n")

    pi = c["prove_it"]
    L.append("## Prove it works\n")
    L.append(f"Ask your AI:\n\n> *{pi['prompt']}*\n")
    L.append(pi["expect"].strip() + "\n")
    L.append("Then confirm all four:\n")
    for item in pi["checklist"]:
        L.append(f"- [ ] {item}")
    L.append("\n**That's it — you're ready for class.**\n")

    L.append("## If something did not work\n")
    for item in env.get("troubleshooting", []):
        if platform in item.get("platforms", []):
            L.append(f"**{item['title']}.** {item['detail'].strip()}\n")
    L.append(
        "Still stuck? Come to class anyway and arrive 15 minutes early &mdash; "
        "we will sort it out together.\n"
    )

    L.append("---\n")
    L.append("## Optional, after class\n")
    for o in c["optional_after"]:
        L.append(f"**{o['title']}.** {o['detail'].strip()}\n")
    return "\n".join(L) + "\n"


def build_standard(platforms, tools) -> str:
    """The stated per-platform guidelines, generated from the same base the
    docs are generated from — so the rules and the instructions cannot drift."""
    L = ["# P2 tool installation standard\n"]
    L.append(
        "Rules every tool in the fleet conforms to. A tool does not invent its "
        "own install root, PATH mechanism, or update story — it declares a "
        "platform and inherits what follows. Generated from `data/platforms.yml`; "
        "do not edit by hand.\n"
    )
    for pid, plat in sorted(platforms.items(), key=lambda kv: kv[1]["order"]):
        L.append(f"## {plat['label']}\n")
        rows = [
            ("Install root", f"`{plat['install_root']}`"),
            ("Architectures", ", ".join(f"`{a}`" for a in plat["archs"])),
            ("PATH is set in", f"`{plat['path_file']}`"),
            ("Elevation", plat.get("privilege_note", "").strip() or "not required"),
            ("Update model", f"`{plat['update_style']}`"),
            ("Verify with", f"`{plat['verify_cmd'].replace('{launcher}', '<tool>')}`"),
        ]
        L.append("| Rule | Value |\n|---|---|")
        L += [f"| {k} | {v} |" for k, v in rows]
        L.append("")
        if plat.get("path_policy"):
            L.append(f"**PATH policy.** {plat['path_policy'].strip()}\n")
        L.append("Tools installed here:\n")
        L.append(
            "| Tool | Folder | PATH element | Launcher |\n|---|---|---|---|"
        )
        for tid, tool in sorted(tools.items(), key=lambda kv: kv[1]["order"]):
            p = tool["platforms"].get(pid, {})
            if not p.get("install_dir"):
                continue
            # The PATH element is bin_dir, which is NOT always install_dir —
            # showing both is what makes the .app nesting impossible to miss.
            elem = f"`{p['bin_dir']}`" if tool.get("needs_path") else "— (host-invoked)"
            L.append(
                f"| {tool['name']} | `{p['install_dir']}` | {elem} | `{p['launcher']}` |"
            )
        L.append("")
    return "\n".join(L) + "\n"


def main():
    check = "--check" in sys.argv
    platforms, tools, env, probes, search_names, accepted = load()

    generated = {"DOCs/setup/PLATFORM-STANDARD.md": build_standard(platforms, tools)}
    for pid, plat in platforms.items():
        generated[f"GETTING-STARTED-{plat['name']}.md"] = build_suite(
            pid, plat, platforms, tools, env
        )
    for tid, tool in tools.items():
        generated[f"DOCs/setup/tools/{tid}/INSTALL.md"] = build_tool(tid, tool, platforms, env)
    for pid, plat in platforms.items():
        generated[f"DOCs/class/SETUP-{plat['name'].upper()}.md"] = build_class(
            pid, plat, tools, env
        )

    fp = source_fingerprint()
    generated = {rel: banner(fp) + body for rel, body in generated.items()}

    errors, warnings = conformance(platforms, tools, probes, search_names, accepted)

    if check:
        stale = []
        for rel, body in generated.items():
            f = OUT / rel
            if not f.exists() or f.read_text() != body:
                stale.append(rel)
                if f.exists():
                    d = difflib.unified_diff(
                        f.read_text().splitlines(), body.splitlines(),
                        fromfile=f"committed/{rel}", tofile=f"generated/{rel}",
                        lineterm="", n=1,
                    )
                    print("\n".join(list(d)[:20]))
        for w in warnings:
            print(f"WARN  {w}")
        for e in errors:
            print(f"ERROR {e}")
        if stale:
            print(f"\nSTALE: {len(stale)} file(s) differ from data/ — re-run generate.py")
        if stale or errors:
            sys.exit(1)
        print(f"OK — {len(generated)} docs current, {len(warnings)} warning(s)")
        return

    for rel, body in generated.items():
        f = OUT / rel
        f.parent.mkdir(parents=True, exist_ok=True)
        f.write_text(body)
    print(f"wrote {len(generated)} docs to {OUT}")
    for w in warnings:
        print(f"WARN  {w}")
    for e in errors:
        print(f"ERROR {e}")


if __name__ == "__main__":
    main()
