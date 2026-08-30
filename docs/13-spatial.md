# Spatial values

Spatial types have no database-independent natural total order. Every spatial index schema must name its comparison tuple.

## Scalar tuple profiles

Use canonical `Float64` fields and reject coordinates outside the domain's finite-value policy.

| Type | Recommended tuple |
| --- | --- |
| point | `(x, y)` |
| line segment | `(min_endpoint, max_endpoint)` after endpoint canonicalization |
| box | `(min_x, min_y, max_x, max_y)` |
| circle | `(center_x, center_y, radius)` with non-negative radius |
| line | normalized coefficients `(a, b, c)` with one sign and scale convention |

These tuples provide deterministic B-tree order; they do not encode distance or overlap semantics.

## Paths and polygons

Canonicalize orientation, starting vertex, closure, duplicate adjacent points, coordinate reference system, and invalid geometry policy before encoding. Build a nested tuple containing profile rank, vertex count when required, and row-major point fields.

A polygon profile that treats rotations or reversed rings as equal must select the lexicographically least canonical ring before encoding. That operation may require scratch storage and is intentionally outside the core encoder.

## Spatial extensions

For PostGIS or vector extensions, prefer an extension-owned sortable token when exact operator-class parity is required. Record extension version, operator class, coordinate system, dimensionality, NaN policy, and canonicalization rules in schema metadata.
