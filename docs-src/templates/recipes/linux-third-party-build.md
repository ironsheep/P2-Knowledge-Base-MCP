There is no Linux binary release — {{tool_name}} is built from source.

1. Install the build dependencies:

   ```sh
   sudo apt-get install build-essential xxd bison git tk8.6-dev
   ```
2. Clone and build. The install directory **must not** be the source directory:

   ```sh
   git clone --recursive {{download_url_repo}}
   cd flexprop
   make install INSTALL={{install_dir}}
   ```

   > This step is heavier than anything else in this guide — a compiler
   > toolchain plus a full build. Budget a few minutes.
3. Make the install directory writable by you. {{tool_name}} stores its own
   configuration inside its install folder, so it needs this whether or not
   you use the GUI today:

   ```sh
   sudo chown -R $USER {{install_dir}}
   ```
4. Add `{{bin_dir}}` to your PATH — see [PATH setup](#path-setup-on-linux).
5. Open a new shell and verify with `{{which_cmd}}`.
