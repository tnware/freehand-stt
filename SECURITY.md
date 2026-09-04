# Security policy

Freehand is pre-1.0 software. Only the latest published alpha receives security
fixes.

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting for
[`tnware/freehand-stt`](https://github.com/tnware/freehand-stt/security/advisories/new).
Do not open a public issue for a suspected vulnerability.

Include the affected version, impact, and reproduction steps. Remove API keys,
transcripts, private URLs, machine names, and personal paths from reports and
attachments.

## Release trust

Windows alpha releases are not Authenticode-signed. Download them only from the
official GitHub Releases page and verify the published `SHA256SUMS` entry. The
in-app updater performs the same checksum verification and never installs an
update silently.
