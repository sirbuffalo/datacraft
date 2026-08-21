# mccomp web editor

This directory is a static browser application. It needs an HTTP server because
browsers do not load WebAssembly correctly from a local `file://` URL.

Build the WebAssembly module from the project root:

```sh
GOOS=js GOARCH=wasm go build -o web/compiler.wasm ./web/wasm
```

Then serve `web/` with any static file server and open its local URL.
No backend is required. Compilation and ZIP creation happen in the browser.
