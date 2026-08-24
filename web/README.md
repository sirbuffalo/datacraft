# DataCraft web editor

This directory is a static browser application. It needs an HTTP server because
browsers do not load WebAssembly correctly from a local `file://` URL.

Build the WebAssembly module from the project root:

```sh
GOOS=js GOARCH=wasm go build -o web/compiler.wasm ./web/wasm
```

Then serve `web/` with any static file server and open its local URL.
No backend is required. Compilation and ZIP creation happen in the browser.

The editor autosaves its current project in IndexedDB. Existing `localStorage`
saves are migrated automatically once. **Save project** exports
`datacraft.toml` and `src/*.dcraft` in a source ZIP; **Open project** imports that
format entirely in the browser.

**Load example** replaces the current browser project, after confirmation, with a
known-working typed example and saves it locally.

The file toolbar supports adding, renaming, and deleting `.dcraft` files. These
changes are included in autosave and source ZIP exports.
