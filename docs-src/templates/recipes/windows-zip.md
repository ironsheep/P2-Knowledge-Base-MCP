1. Download **`{{asset}}`** from the [latest release]({{releases_url}}).
2. Right-click the `.zip` and choose **Extract All**.
3. In the destination box, enter exactly:

   ```
   {{install_root}}
   ```

   > Windows pre-fills a subfolder named after the `.zip`. **Delete that part**
   > of the path. The archive already contains a `{{folder}}` folder, so
   > leaving it produces a doubled path that the Spin2 extension will not find.
4. Confirm you now have `{{install_dir}}`.
5. Add `{{bin_dir}}` to your PATH — see [PATH setup](#path-setup-on-windows).
6. Open a **new** Command Prompt and verify:

   ```
   {{verify_cmd}}
   ```
