Release v0.2.3 (2018-09-14)
===

### Bug Fixes
* rollup fields were not getting the rolled up values added to the root span

### New Field
* sql and sqlx wrappers get both the DB call being made (eg Select) as well as the name of the function making the call (eg FetchThingsByID)

Release v0.2.2 (2018-09-1)
===

### Bug Fixes
* fix version number inconsistency with a patch bump

Release v0.2.1 (2018-09-14)
===

### Bug Fixes
* fix propagation bug when an incoming request has a serialized beeline trace header

Release v0.2.0 (2018-09-12)
===

This is the second major release of the beeline. It changes the model from "one
current span" to a to a doubly-linked tree of events (now dubbed "spans")
representing a trace.

### Major Changes

* introduces the concept of a span
* adds functions to create new spans in a trace and add fields to specific spans
* adds the ability to create and accept a serialized chunk of data from an upstream service to connect in-process traces in a distributed infrastructure into one large trace.
* adds trace level fields that get copied to every downstream span
* adds rollup fields that sum their values and push them in to the root span
* adds a pre-send hook to modify spans before sending them to Honeycomb
* adds trace-aware deterministic sampling as the default
* adds a sampler hook to manually manage sampling if necessary

### Breaking Changes

* removed `ContextEvent` and `ContextWithEvent` functions; replaced by spans

### Wrapper Changes
* augment the net/http wrapper to wrap `RoundTripper`s and handle outbound HTTP calls
* adding a wrapper for the `pop` package


Release v0.1.2 (2018-08-30)
===

### New Features

* add new sqlx functions to add context to transactions and rollbacks
* add HTTP Headers X-Forwarded-For and X-Forwarded-Proto to events if they exist

### Bug Fixes
* use the passed in context in sqlx instead of creating a background context

Release v0.1.1 (2018-08-20)
===

### Bug Fixes
* Use the right Host header for incoming HTTP requests
* Recognize struct HTTP handlers and add their name
* Fix nil route bug

Release v0.1.0 (2018-05-16)
===

Initial Release
