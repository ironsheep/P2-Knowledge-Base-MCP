1. Download **`{{asset}}`** from the [latest release]({{releases_url}}).
2. Unpack it and move it into place:

   ```sh
   cd ~/Downloads
   tar -xzf {{asset}}
   sudo mv {{folder}} {{install_dir}}
   ```
3. Clear the quarantine flag so macOS will launch it:

   ```sh
   sudo {{quarantine_cmd}}
   ```
4. Verify:

   ```sh
   {{install_dir_bin_launcher}} --version
   ```
