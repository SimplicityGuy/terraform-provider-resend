# Security

## Reporting a vulnerability

GitHub private vulnerability reporting is not enabled on this repository, and issues are public, so
do not describe a vulnerability in an issue. Email the maintainer at
[robert@simplicityguy.com](mailto:robert@simplicityguy.com) with the affected version and, if you
can, steps to reproduce. The full security and support policy (supported versions, response times)
is still to come.

## Release signing key

Every release's `SHA256SUMS` file is signed with this key; the Terraform and OpenTofu registries
verify the signature when they ingest a release.

| | |
| --- | --- |
| Fingerprint | `30D2 732F D215 D008 9C27  FCCB 9F85 682D 8BB8 41F1` (`30D2732FD215D0089C27FCCB9F85682D8BB841F1`) |
| Key ID | `9F85682D8BB841F1` |
| Algorithm | RSA 4096 (the Terraform Registry rejects ECC keys) |
| User ID | SimplicityGuy Release Signing <robert@simplicityguy.com> |
| Created | 2026-10-28 |
| Expires | 2028-10-28 |
| Public key | [`release-signing-key.asc`](release-signing-key.asc) in the root of this repository |

Check a downloaded copy of the key against the fingerprint above before trusting it:

```bash
gpg --show-keys --fingerprint release-signing-key.asc
```

The private key is used only by the release workflow. If the key is rotated or revoked, the new
fingerprint is published here and in the [README](README.md#verifying-releases). See
[CONTRIBUTING.md](CONTRIBUTING.md#releases) for the rotation and revocation procedure.
