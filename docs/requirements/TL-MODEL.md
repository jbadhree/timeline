# TL-MODEL: Recorded data model

Status: logical model defined; serialization and physical schema decisions remain before implementation.

## Purpose and canonical context

The [architecture data model](../architecture.md#data-model) defines the entities, identities, relationships, and derived index representation. It refines TL-TYPES, TL-TIME, TL-HISTORY, TL-RECOVERY, and TL-DATABASE without introducing another source of truth.

## Required behavior

An item has stable identity and original registration data, while each accepted revision captures the content and path known at that acceptance. Type definitions and configuration used to validate accepted content are preserved as immutable versions. Acceptance batches group the selected changes; working files and scan diagnostics remain distinct from accepted revisions. The database stores a reconstructible projection of this evidence.

## Acceptance criteria

- TL-MODEL/AC-1: A first acceptance creates one item identity, one initial revision, and an acceptance association with recoverable content and type/configuration evidence. Draft metadata cannot fabricate registration.
- TL-MODEL/AC-2: An update or move adds a revision while preserving identity, original entry time, initial type-definition binding, and earlier content. A removal adds a tombstone revision and retains the preceding content in history.
- TL-MODEL/AC-3: An acceptance batch links each selected item's new revision to its prior revision, if any. Replay rejects missing dependencies, conflicting identity, and malformed relationships instead of inventing records.
- TL-MODEL/AC-4: Changing current type configuration does not reinterpret earlier items or revisions. The definition actually used for validation is recoverable independently of current files.
- TL-MODEL/AC-5: A scan can report malformed or duplicate-ID files without creating accepted items or replacing the last accepted revision. Pending metadata and diagnostics do not become recorded facts.
- TL-MODEL/AC-6: Rebuilding another index target produces equivalent current items, removed-item history, metadata filters, and saved definitions. Derived surrogate keys and physical SQL types do not determine durable identity.

## Verification and status

All criteria are unimplemented. Add real test references here as work proceeds. Before implementing serialization or migrations, settle the open contracts listed in the architecture: IDs/hashes, timestamp precision, dataset binding, acceptance publication format, metadata semantics, and unsupported history transitions. This logical model is not an executable SQL schema.
