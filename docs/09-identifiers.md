# Identifiers and physical addresses

## UUID

Encode the 16 RFC 9562 octets in network order. Bytewise order follows the unsigned 128-bit value and preserves chronological order for UUID versions whose canonical byte layout is time-ordered, including UUIDv7.

Do not reorder UUID fields using host-language struct layout. Text case and hyphens are presentation details and never enter the key.

## ULID and similar identifiers

Encode a canonical 16-byte binary value. Crockford Base32 aliases and case are presentation details. Other fixed-width identifiers use their canonical network-order bytes.

## MAC addresses

Encode a 48-bit MAC address as six bytes and a 64-bit MAC address as eight bytes in display order. Width is part of the schema. EUI-48 and modified EUI-64 forms are not interchangeable unless the application canonicalizes them before encoding.

## Log and object identifiers

- Encode a log sequence number as the unsigned 64-bit scalar `(high << 32) | low`.
- Encode OID-like values at their declared unsigned width.
- Encode transaction snapshot structures as nested tuples only after selecting a stable canonical member order.

## Enum and domain identifiers

Encode an enum by immutable unsigned rank. Encode a domain through its base profile after applying the domain's normalization and validation. Equality aliases must resolve before encoding.

All fixed-byte payloads are complemented for descending order. The field presence tag is not complemented.
