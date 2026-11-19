# Third-party notices

## dingtalk-chat-exporter (Python)

The DingTalk V3 database decryption (key derivation from the numeric UID and the
per-account salt, AES-ECB page decryption, WAL frame decryption with checksum
recomputation) and the UID discovery from client logs were ported to Go from
<https://github.com/idealistichacker/dingtalk-chat-exporter>, which is
distributed under the following license:

```
MIT License

Copyright (c) 2026 DingTalk Exporter Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Go dependencies

- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — BSD-3-Clause (pure-Go SQLite)
- and its transitive dependencies listed in `go.sum`, under their respective licenses.
