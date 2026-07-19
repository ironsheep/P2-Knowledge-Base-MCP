1. Download **`{{asset}}`** from the [latest release]({{releases_url}}).
2. Unpack it and move it into place:

   ```sh
   unzip {{asset}}
   sudo mv {{folder}} {{install_dir}}
   ```
3. Add the tool to your PATH — see [PATH setup](#path-setup-on-linux):

   ```sh
   {{path_stanza}}
   ```
4. Open a new shell and verify:

   ```sh
   {{verify_cmd}}
   ```
