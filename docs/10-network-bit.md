# Network and bit values

## IP address

Normalize IPv4-mapped IPv6 addresses to IPv4 unless the schema explicitly distinguishes them. Reject IPv6 zone identifiers because a zone is interface-local and not part of a portable address value.

The ascending payload is:

```text
family | address
```

| Family | Byte | Address bytes |
| --- | ---: | ---: |
| IPv4 | `04` | 4 |
| IPv6 | `06` | 16 |

This orders IPv4 before IPv6, then orders addresses as unsigned network-order integers.

## Network prefix

Canonicalize a prefix by clearing host bits. Encode `family | network-address | prefix-length`. The profile orders canonical network addresses first and more-specific lengths second when addresses are equal.

This is a stable Schottky order, not an assertion that every database uses the same `inet` or `cidr` comparator. An adapter requiring exact database index parity must normalize to that database's comparison tuple before encoding.

## Bit strings

For `bit(n)` and `varbit(n)`, clear unused low bits in the final byte. Build a nested tuple containing:

1. packed bits as binary bytes;
2. significant bit length as `Uint64`.

Embed that tuple with `Builder.Tuple`. The representation distinguishes values with identical padded bytes and different bit lengths. Fixed-length domains may omit the length when it is guaranteed by schema.

## Descending order

Complement the complete payload for the field. For nested bit strings, apply direction to the outer tuple field; nested components remain ascending.
