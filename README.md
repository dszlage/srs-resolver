# srs-resolver

**A lightweight SRS resolver for Postfix+Dovecot autoresponders**  
Author: Damian Szlage  
License: GPLv3  
Repository: [https://github.com/dszlage/srs-resolver](https://github.com/dszlage/srs-resolver)  
Release: [https://github.com/dszlage/srs-resolver/releases](https://github.com/dszlage/srs-resolver/releases)  
Latest release: [https://github.com/dszlage/srs-resolver/releases/latest](https://github.com/dszlage/srs-resolver/releases/latest)  

---

## Overview

`srs-resolver` is a minimalistic TCP service for decoding SRS (Sender Rewriting Scheme) addresses like `SRS0=` or `SRS1=` into the original sender's address (e.g. `user@domain.com`).

It was created to solve a real-world problem with autoresponders (e.g. Dovecot Vacation, Postfix pipes) that are unable to reply to an SRS-modified `Return-Path`. This happens when the sender's mail system either cannot or will not reverse SRS addresses.

**This tool works around that limitation by decoding the address locally.**

---

## Why?

When a sender uses SRS (e.g., with postsrsd), a forwarded message arrives with a Return-Path like:
SRS0=hash=time=domain=user@forwarder.tld.

Autoresponders and vacation filters, according to RFC specifications, send replies to the address in the Return-Path. Your server sends a response, but it might bounce if the receiving server is unable or unwilling to resolve SRS addresses — often due to poor or lazy configuration on their end.

`srs-resolver` extracts the original sender (`user@domain`), ensuring that replies are sent to the correct address regardless of the receiving server's SRS support.

---

## Features

- Parses SRS0 and SRS1 formats
- Fallback to a default address if decoding fails (fallback_address in config)
- Works as a secure localhost-only TCP service (Postfix-compatible)
- Simple protocol: `get <address>` → `200 <clean email>` or `500 error`
- Configurable logging and fallback

---

## Installation

--- Requirements: Go 1.16+ (for building from source) ---

Build from source and install:

```bash
mkdir -p ~/srs-resolver
cd ~/srs-resolver
git clone https://github.com/dszlage/srs-resolver.git .

make build
sudo make install

```
Set up systemd (run srs-resolver as a service - daemonize):

```bash
   sudo cp systemd/srs-resolver.service /etc/systemd/system/
   sudo systemctl daemon-reexec
   sudo systemctl daemon-reload
   sudo systemctl enable srs-resolver
   sudo systemctl start srs-resolver
```

Build and run installation script:

```bash
mkdir -p ~/srs-resolver
cd ~/srs-resolver
git clone https://github.com/dszlage/srs-resolver.git .

make
sudo ./install.sh
```

Download precompiled binary (Linux x86_64):

```bash
cd /tmp
wget https://github.com/dszlage/srs-resolver/releases/latest/download/srs-resolver-<version>-linux-amd64.tar.gz
``` 
or 
```bash
wget https://github.com/dszlage/srs-resolver/releases/latest
tar -xzf srs-resolver-<version>-linux-amd64.tar.gz
cd srs-resolver-<version>
sudo ./install.sh
```

---

## Postfix Integration

To make Postfix rewrite SRS addresses for autoresponders, add to /etc/postfix/main.cf:

recipient_canonical_maps = tcp:127.0.0.1:10022  
recipient_canonical_classes = envelope_recipient  

```bash
sudo systemctl reload postfix
```
This ensures that Return-Path headers are decoded before reaching autoresponders like dovecot or vacation.

---

## Example

Request:
get SRS0=xyz=12345=original.com=localpart

Response:
200 localpart@original.com
If Invalid or malformed input:
500 invalid request

If a fallback address is configured, it will respond with:
200 root@domain.com
(or whatever is set in fallback_address)

---

## Testing

Run the included test script:
```bash
./test/test_srs_resolver.sh
```
Make sure the service is running on 127.0.0.1:12345. The script will test:
Valid SRS0/SRS1 addresses
Clean email addresses
Invalid/malformed inputs
Unsupported protocol commands

---

## Security Note

srs-resolver intentionally bypasses parts of RFC SRS behavior in order to make autoresponders functional in edge cases. Use it at your own risk in trusted environments only.
The service only listens on localhost by default
No external network exposure is recommended
It does not perform cryptographic validation of SRS hashes

---

## License

This project is licensed under the GNU GPLv3.
You are free to use, modify, and distribute the code, as long as your modifications are also open-source under the same license.

See LICENSE: https://www.gnu.org/licenses/gpl-3.0.txt

---

## Contributions

Bug reports, suggestions and pull requests are welcome!
You can also open issues or discussions at:

🔗 https://github.com/dszlage/srs-resolver/issues

## Acknowledgements

Inspired by postsrsd, but works in reverse.

Created to fix real production problems with autoresponders and SRS.
