# Wildcard server names (`*` SNI)

This fork lets a REALITY server accept a *family* of SNIs via a single
`serverNames` entry containing one `*`, e.g.:

```
serverNames: ["de-fra-1-*.blook.network"]
```

A client may then connect with any SNI of the form `de-fra-1-<anything>.blook.network`
(see `Config.matchServerName` / `matchWildcardServerName` in `common.go`). This is
used so a client can advertise a fresh, random SNI per connection — spreading
connections across many SNIs without the server having to enumerate them.

Matching rule: the incoming SNI must share the pattern's prefix and suffix, and
the substituted middle must be **1..64 bytes** (`realityWildcardMaxLen`).

## The hard requirement: `*` must match only *identically-served* domains

> **Every domain that a `*` can match MUST be served identically by `dest`.**

In practice this means all matched names must resolve to **the same backend**,
behind the **same TLS certificate** (typically a wildcard cert such as
`*.blook.network`), producing the **same TLS handshake and the same
post-handshake records**.

This is not a stylistic preference — it is required for REALITY's anti-detection
to remain correct. Here is why.

### Why: REALITY mimics `dest` per-SNI, and random SNIs can't be pre-measured

REALITY disguises the server as a real site (`dest`). Part of that disguise is
replaying, to each client, encrypted records *after* the handshake whose
**lengths** match what `dest` actually sends (e.g. session tickets). The server
learns those lengths up front by dialing `dest` once per configured SNI and
recording them — see `DetectPostHandshakeRecordsLens` in `record_detect.go`,
keyed by `dest + " " + serverName + " " + alpn`, and consumed in `tls.go`
(the post-handshake loop).

A wildcard SNI is **random per connection**, so it can never be pre-measured,
and a name like `de-fra-1-*.blook.network` is not itself dialable. This fork
therefore:

1. **Measures the pattern once, via a concrete representative.** When
   `DetectPostHandshakeRecordsLens` encounters a `serverNames` entry containing
   `*`, it dials `dest` using the name with `*` replaced by `wildcard`
   (e.g. `de-fra-1-wildcard.blook.network`) but stores the result under the
   **pattern** key (`record_detect.go`).
2. **Resolves every matched SNI back to the pattern at runtime.** When a client
   connects with `de-fra-1-<random>.blook.network`, the post-handshake lookup
   uses `Config.canonicalServerName`, which maps the random SNI to its wildcard
   pattern, so it finds the single stored measurement (`tls.go`).

So **all SNIs under one wildcard reuse one measurement**. That measurement is
only correct if `dest` would have produced those same lengths for every one of
those SNIs — i.e. if `dest` serves them identically.

### What goes wrong if they are *not* served identically

If `dest` behaves differently per matched name (different vhost, different cert,
different post-handshake records), then:

- the replayed record lengths match only the representative, not the actual SNI;
- a probe that connects to `dest` directly with a specific matched SNI sees a
  different post-handshake fingerprint than the REALITY server replays for that
  SNI — a detectable mismatch.

With a uniform `dest` (one backend + one wildcard cert) there is nothing to
detect: every matched SNI's "real" behavior is the same, so reusing one
measurement is exact.

## Practical guidance

- Point the wildcard at a `dest` you control that serves the whole label space
  uniformly (one backend, wildcard certificate covering `*.<suffix>`).
- Don't use a `*` broad enough to span domains with differing `dest` behavior.
- Keep the random middle within 64 bytes (the matcher rejects longer).
- Note (orthogonal to the above): a single client IP emitting many unique,
  high-entropy SNIs under one domain is itself a potential traffic-analysis
  signal. That is a property of using random SNIs at all, independent of the
  `dest`-uniformity requirement described here.

## Related code

- `common.go` — `matchServerName`, `matchWildcardServerName`, `canonicalServerName`, `realityWildcardMaxLen`
- `record_detect.go` — `DetectPostHandshakeRecordsLens` (representative measurement, pattern-keyed)
- `tls.go` — server-side SNI acceptance and the post-handshake records loop

The client side (generating the random per-connection SNI and bounding how many
concurrent connections share one name) lives in the Xray-core fork, not here:
`transport/internet/reality/sni_wildcard.go`.
