1. Download **`{{asset}}`** from the [latest release]({{releases_url}}).
2. Double-click the `.zip` to extract the `.dmg`, then double-click the `.dmg` to mount it.
3. Drag **{{dmg_payload}}** into `{{install_root}}`.
4. Eject the mounted image.
5. Add the tool to your PATH — see [PATH setup](#path-setup-on-macos):

   ```sh
   {{path_stanza}}
   ```
6. Open a new terminal and verify:

   ```sh
   {{verify_cmd}}
   ```
