# Network and bit values

## IP address and prefix

Reject IPv6 zone identifiers. Preserve IPv4-mapped IPv6 values as IPv6; address family is part of ordering and equality.

`IP` behaves as a full-width `IPPrefix`: `/32` for IPv4 and `/128` for IPv6. `IPPrefix` preserves host bits. `NetworkPrefix` requires a canonical prefix with all host bits cleared and rejects rather than masks a non-canonical value.

Ascending comparison is:

1. family, with IPv4 before IPv6;
2. address bits through the shorter prefix length;
3. prefix length;
4. every address bit, including host bits.

The payload uses a family byte, an escaped packed network-bit prefix, the prefix length, and the complete address. The escaped prefix makes a shorter equal bit prefix sort before an extension without expanding each source bit.

`EncodedIPSize`, `EncodedIPPrefixSize`, and `EncodedNetworkPrefixSize` return exact field sizes. Network encodings are variable-width because zero bytes in the packed prefix are escaped. `MaxIPv4Size` and `MaxIPv6Size` include the presence tag and worst-case escaping.

## Bit strings

For `bit(n)` and `varbit(n)`, clear unused low bits in the final byte. Build a nested tuple containing:

1. packed bits as binary bytes;
2. significant bit length as `Uint64`.

Embed that tuple with `Builder.Tuple`. The representation distinguishes values with identical padded bytes and different bit lengths. Fixed-length domains may omit the length when it is guaranteed by schema.

## Descending order

Complement the complete payload for the field. For nested bit strings, apply direction to the outer tuple field; nested components remain ascending.
