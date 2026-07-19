1. In VS Code, open the Extensions view and search for **spin2**.
2. Install the extension published by **Iron Sheep Productions, LLC** (`{{marketplace_id}}`).
3. Open a `.spin2` file. The extension activates and scans the standard install
   locations for the tools you installed above.
4. Confirm discovery: open **Settings** and search for `spinExtension.toolchain.paths`.
   The entries for the tools you installed should now be filled in.

> **Install this one last.** The extension scans for the compiler and terminal
> when it activates. Installing it after the other tools means one scan finds
> everything; installing it first means it finds nothing and you have to
> re-trigger discovery.
