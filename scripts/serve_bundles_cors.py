#!/usr/bin/env python3
"""Serve Cocos remote bundles with CORS + correct MIME (esp. index.1.0.0.js)."""
from __future__ import annotations

import sys
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


class CORSRequestHandler(SimpleHTTPRequestHandler):
    extensions_map = {
        **getattr(SimpleHTTPRequestHandler, "extensions_map", {}),
        ".js": "application/javascript",
        ".mjs": "application/javascript",
        ".json": "application/json",
        ".wasm": "application/wasm",
        ".png": "image/png",
        ".jpg": "image/jpeg",
        ".jpeg": "image/jpeg",
        ".webp": "image/webp",
        ".css": "text/css",
        ".html": "text/html",
        ".bin": "application/octet-stream",
    }

    def __init__(self, *args, directory: str | None = None, **kwargs):
        super().__init__(*args, directory=directory, **kwargs)

    def guess_type(self, path: str) -> str:
        # index.1.0.0.js -> Python may miss .js; force by suffix
        lower = path.lower()
        if lower.endswith(".js") or lower.endswith(".mjs"):
            return "application/javascript"
        if lower.endswith(".json"):
            return "application/json"
        if lower.endswith(".wasm"):
            return "application/wasm"
        return super().guess_type(path)

    def end_headers(self) -> None:
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "*")
        self.send_header("Access-Control-Max-Age", "86400")
        self.send_header("Cache-Control", "no-cache")
        super().end_headers()

    def do_OPTIONS(self) -> None:  # noqa: N802
        self.send_response(204)
        self.end_headers()

    def log_message(self, fmt: str, *args) -> None:
        sys.stderr.write("%s - %s\n" % (self.address_string(), fmt % args))


def main() -> None:
    if len(sys.argv) < 3:
        print("usage: serve_bundles_cors.py <root> <port>", file=sys.stderr)
        sys.exit(2)
    root = str(Path(sys.argv[1]).resolve())
    port = int(sys.argv[2])

    def handler(*args, **kwargs):
        return CORSRequestHandler(*args, directory=root, **kwargs)

    httpd = ThreadingHTTPServer(("0.0.0.0", port), handler)
    print("Serving %s on http://localhost:%s/ (CORS + JS MIME)" % (root, port), flush=True)
    httpd.serve_forever()


if __name__ == "__main__":
    main()
