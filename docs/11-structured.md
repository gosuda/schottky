# Composite, array, and record values

## Composite values

Build component fields into separate caller-owned scratch storage, then embed the resulting key with `Builder.Tuple`. The outer binary escaping makes the nested boundary unambiguous and preserves its bytewise order.

Nested component direction is part of the nested schema. Apply outer descending order only when the composite value as a whole is descending.

## Arrays

A one-dimensional array with element-wise lexicographic order uses this nested layout:

```text
(element_0, element_1, ..., element_n)
```

Each element is a normal field and can be null. A shorter array sorts before an otherwise equal array that extends it because its nested key is a proper prefix.

For multidimensional SQL arrays, prepend dimensions and lower bounds when they participate in equality or ordering:

```text
(rank, length_0, lower_0, ..., elements in row-major order)
```

Use fixed unsigned widths for rank and lengths. Validate that the element count matches the dimensions.

## Records and composite SQL types

Encode attributes in schema order. Attribute names are schema metadata and do not enter the key. Schema evolution that inserts, removes, reorders, or changes an attribute profile requires a new key-schema version.

## Sets and maps

A set needs a canonical member order before encoding. A map needs canonical key order and one duplicate-key policy. Encode canonical entries as nested `(key, value)` tuples. Never rely on Go map iteration order.

## Opaque extension types

An extension can supply an ordered binary token and use `Builder.Tuple` or `Builder.Bytes`. The token producer, version, options, equality semantics, and migration policy become part of the index schema.
