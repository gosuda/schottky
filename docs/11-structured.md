# Composite, array, and record values

## Composite values

Build component fields into separate caller-owned scratch storage, then embed the resulting key with `Builder.Tuple`. The outer binary escaping makes the nested boundary unambiguous and preserves its bytewise order.

Nested component direction is part of the nested schema. Apply outer descending order only when the composite value as a whole is descending.

## Arrays

Build two caller-owned nested keys:

```text
elements = (element_0, element_1, ..., element_n)
array = (Tuple(elements), element_count, rank, dimensions..., lower_bounds...)
```

Encode elements in row-major order. An element uses its normal ascending profile with `AscNullsLast`, so a null element sorts after every non-null element. The nested element tuple makes a shorter equal-prefix element sequence sort first before metadata is considered.

After equal elements, compare signed `Int32` metadata in this order:

1. element count;
2. rank;
3. dimension lengths in dimension order;
4. lower bounds in dimension order.

Validate that the product of non-negative dimension lengths equals the element count and that every array uses the same element profile. Rank, dimensions, and lower bounds may be omitted only when the index schema guarantees they are identical for every compared array.

## Records and composite SQL types

Encode attributes in schema order inside one nested key. Each attribute uses its normal ascending profile with `AscNullsLast`; a null attribute sorts after every non-null attribute. Apply descending order only to the outer `Tuple` field.

Attribute names and named record identities do not enter the key. Reject differing logical arity or attribute profiles before encoding. Schema evolution that inserts, removes, reorders, or changes an attribute profile requires a new key-schema version.

## Sets and maps

A set needs a canonical member order before encoding. A map needs canonical key order and one duplicate-key policy. Encode canonical entries as nested `(key, value)` tuples. Never rely on Go map iteration order.

## Opaque extension types

An extension can supply an ordered binary token and use `Builder.Tuple` or `Builder.Bytes`. The token producer, version, options, equality semantics, and migration policy become part of the index schema.
