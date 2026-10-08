# Strict command summaries and portable proof

Command displays are bounded to 200 bytes after whitespace normalization.
Truncation preserves UTF-8 boundaries and never splits a whitespace token
containing an HTTP or HTTPS URL, including its quotes, host, IPv6 address, port,
or path. Long non-URL tokens may be shortened. Displays are never executed.

New truncated `small apply` captures carry `command_summary_version: 2`.
This is a tooling summary-format marker, independent of the v1/v2 protocol
profile. Strict validation requires exact full-command proof for every marked
capture, including commands whose URLs or credentials are entirely beyond the
display cap. Short captures keep the existing metadata contract. Removing a
marker or changing any historical receipt is an audit-history edit, not a repair.

V1 start, end, and dry-run entries keep `command` and `command_summary` as the
same display and bind original bytes through `command_ref` and
`command_sha256`. V2 `command_recorded` payloads use the same summary marker,
ref, and hash; the hash also matches the event's `source_digest`.
No profile migration occurs. Older binaries cannot validate the new v1 marker;
all writers and hosted gates must use a compatible CLI before capturing new
bounded commands.

## Supported local stores

Only these relative command refs are supported:

| Capture | Original command ref | Portable fallback |
| --- | --- | --- |
| V1 | `.small-cache/logs/<replayId>/commands/<timestamp>.txt` | `.small-command-proofs/logs/<replayId>/commands/<timestamp>.txt` |
| V2 | `.small-cache/commands/<sha256>.txt` | `.small-command-proofs/commands/<sha256>.txt` |

The replay id and digest are 64 lowercase hex digits. The v1 filename is the
CLI's UTC `YYYYMMDDTHHMMSS.nnnnnnnnnZ.txt` form and must match the receipt
timestamp and its replay id when present. Proof contains the **exact original
command bytes**, with no added newline. The selected original ref and receipt
hash determine the fallback path and accepted bytes; proof cannot substitute a
different ref. V2 content-addressed filenames must also match the hash.

Reads stay within the workspace using `os.Root`, require a regular file, reject
file symlinks and directory symlink escapes, and are limited to 1 MiB. Absolute
paths, traversal, network refs, unsupported stores, missing proof, and hash or
summary mismatch fail closed with a diagnostic. A missing cache file allows
the mirrored portable file; a corrupt or unreadable cache file fails without
falling back. No network requests occur during proof validation.

Strict validation checks the verified full command for insecure HTTP and
potential secrets. The local HTTP exception is progress/command context only,
with exact hosts `localhost`, `127.0.0.1`, `0.0.0.0`, and `::1`, no userinfo,
and syntactically valid ports. Prefix lookalikes are rejected. Command secret
heuristics include credential assignments, bearer credentials, recognized token
prefixes and private-key headers, as well as existing artifact-value heuristics.
Diagnostics do not echo command bytes or credentials.

## Existing receipts

The pre-correction summarizer could split an allowed local URL into an insecure
looking host. Compatibility is limited to a 200-byte ellipsized CLI summary
that otherwise fails link hygiene. The full original bytes must match both the
receipt hash and the **exact legacy summarizer output**. `command`, when
present, must equal that output too. Full-command checks still reject an
external HTTP URL or credential beyond the cap.

Only proven display fields are substituted in memory during lint. The exact
generated `apply --dry-run (cmd: %q)` wrapper is recognized only on a pending
entry with evidence `Dry-run: no command executed`. Arbitrary notes, evidence,
intent, constraints and other strings retain ordinary enforcement. Check,
lint and verify never rewrite command displays, timestamps, hashes, task state,
or files. Otherwise valid unmarked historical receipts do not gain a universal
cache requirement; their omitted bytes are not asserted to have been verified.

## Selective proof for a fresh checkout

`.small-cache/` is private, ignored local state. Never commit or upload it as a
whole. Select only the command refs needed for the affected receipts, inspect
their exact bytes for secrets, and verify their SHA256 against the receipt.
Supply them as individually reviewed files in `.small-command-proofs/`, which
is outside authoritative `.small/` and may be explicitly committed or supplied
as a narrowly scoped CI artifact. This proof store is not a protocol migration
or replacement history.

For each reviewed non-secret command, prepare the following through the
workspace's existing plan and `small apply` policy:

```sh
ref='.small-cache/logs/<replayId>/commands/<timestamp>.txt'
proof=".small-command-proofs/${ref#.small-cache/}"
mkdir -p "$(dirname "$proof")"
cp "$ref" "$proof"
```

Copy bytes unchanged; do not regenerate the command or add a newline. Then run
the patched CLI's `check --strict --dir <workspace>` in a **fresh checkout with
no cache**. With the selected proof it must pass; without it or with modified
bytes it must fail. Preserve before/after hashes of authoritative audit files.

For LoopExec's reported failure, the only required portable files mirror refs
ending in `20261007T230645.843552000Z.txt` and
`20261007T230805.581332000Z.txt` under replay id
`f6bbda15d790df6d023dea340edd1b85ccac89c82db362652a1b656981e82d96`.
The expected command hashes are
`3a9159b8ed0e83cadee0ccc34e8fb70164815a20f165c86402bbfa7edfd6f607` and
`f967306198f45f0b60b0e9eae2d03399e082d970c7ed26d7da916fc2eaf982c1`.
Prepare these files in the downstream task; this correction does not modify
LoopExec or its CI pin. A pushed development branch is not a stable release.
Stable publication requires maintainer authorization, then compatible binary
pinning and selective proof in downstream CI before deployment can resume.
