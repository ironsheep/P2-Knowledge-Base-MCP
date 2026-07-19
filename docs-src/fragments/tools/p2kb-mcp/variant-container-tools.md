Most people should not read this section. The standard install above is
simpler and does exactly the same job for a single MCP server. This variant
exists for one situation: **{{skip_unless}}**, and want them co-located under
one tree with a shared installer, shared configuration, and rollback on update.

If that is not you, you are already done — go back to Part 2 or stop here.

1. Download **`{{variant_asset}}`** from the [latest release]({{releases_url}}).
   One archive carries the binaries for every platform.
2. Extract and run the installer:

   ```sh
   tar -xzf {{variant_asset}}
   cd container-tools-p2kb-mcp-v*/p2kb-mcp
   sudo ./{{shell_installer}}
   ```
3. Verify:

   ```sh
   {{variant_bin_launcher}} --version
   ```

**You still have to register with your agent.** The installer writes
`{{variant_install_dir}}/etc/mcp.json`, which is the framework's own
configuration — no AI host reads it. Use the same registration step as the
standard install, with this path instead:

```sh
claude mcp add -s user p2kb-mcp -- {{variant_bin_launcher}}
```

**Installer options.** `--target DIR` installs somewhere other than
`{{variant_install_dir}}`; `--uninstall` removes p2kb-mcp, rolling back to the
previously installed version if one is present.

**If `jq` is not installed**, the installer cannot update the shared
`mcp.json` and will warn rather than fail — the install reports success while
that step is silently skipped. Install `jq` first to avoid this.
