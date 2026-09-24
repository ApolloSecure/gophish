# Tenant data ownership and deletion

This fork supports explicit, nullable tenant ownership for customer phishing data and an idempotent hard-delete operation. Tenant IDs are opaque identifiers supplied by Platform. Gophish does not derive ownership from organisation, campaign, or group names.

## Initial expand phase

Apply the migrations in version order:

1. `20260810000000` adds recipient custom fields.
2. `20260814000000` adds nullable campaign ownership and tenant-filtered campaign queries.
3. `20260817000000` adds the tenant ownership root, nullable ownership for groups, targets, and email requests, target-scoped uniqueness, and cascading child relationships.

All four `tenant_id` columns remain nullable. Existing clients can omit the field and continue to create legacy null-owned data. A supplied tenant ID is validated, its `tenants` row is created idempotently in the same transaction as the directly owned record, and group targets inherit the group's tenant. Target reuse is limited to the exact `(tenant_id, email)` pair; a null-owned target and tenant-owned targets with the same email remain distinct.

Campaign creation resolves groups using an exact nullable ownership match. A tenant campaign therefore cannot attach a group belonging to another tenant or a legacy null-owned group. Group creation and update similarly reject target ownership mismatches.

## Purge contract

`DELETE /api/tenants/{tenantId}` uses the existing bearer API-key middleware. The path value must be non-empty, at most 255 bytes, and contain no surrounding whitespace.

The response is HTTP 200 after a successful commit:

```json
{
  "tenant_id": "organisation-id",
  "deleted": {
    "campaigns": 1,
    "results": 25,
    "events": 30,
    "mail_logs": 25,
    "groups": 2,
    "group_targets": 25,
    "targets": 25,
    "email_requests": 1,
    "tenant": 1
  }
}
```

Counts contain no recipient data. An unknown or already-purged tenant succeeds with zero counts. Invalid identifiers return HTTP 400. Any database error returns HTTP 500 and rolls the entire transaction back.

The transaction explicitly deletes campaign children and group memberships so their counts can be reported, then deletes directly owned campaigns, groups, targets, email requests, and the tenant row. Every predicate uses the explicit non-null tenant ID. Templates, attachments, pages, SMTP profiles and headers, users, roles, permissions, webhooks, IMAP settings, and migration metadata are not purge targets.

## Database behaviour

- PostgreSQL uses nullable foreign keys with `ON DELETE CASCADE` from the four directly owned tables to `tenants`, plus cascading foreign keys from results, events, and mail logs to campaigns and from group memberships to groups and targets.
- MySQL uses the same foreign-key graph. Historical integer primary-key columns referenced by these constraints are widened to `BIGINT` so parent and child types match.
- SQLite cannot add foreign keys to existing tables with `ALTER TABLE`. The migration installs insert/update enforcement and delete-cascade triggers for equivalent guarantees, including same-tenant group/target membership checks. Application transactions enforce the same group/target rule on every backend.

The migrations preserve existing rows. Existing campaign tenant IDs are copied verbatim into `tenants` before the new foreign keys are installed; no tenant ID is fabricated. Newly added ownership columns default to null. The unique `(tenant_id, email)` target index isolates tenant targets while SQLite, PostgreSQL, and MySQL continue to permit multiple legacy rows because null values are distinct for unique-index purposes.

Operators should check existing child rows for broken campaign/group/target references before applying the PostgreSQL or MySQL foreign keys. A failed preflight or migration must be corrected from authoritative records, not by deleting or guessing ownership.

## Backfill phase

Platform owns the authoritative organisation-to-Gophish mapping. The later backfill must:

1. Set campaign ownership from Platform `gophish_settings.campaignIds`.
2. Populate group and target ownership using an explicitly reviewed mapping process.
3. Detect targets reused across tenants and split or resolve them deliberately.
4. Verify that no active customer-owned campaigns, groups, targets, or persisted email requests remain null-owned.
5. Produce per-table counts and unresolved records for operator review.

There is intentionally no name-based backfill in this repository.

## Contract phase

Only after Platform supplies `tenant_id` for group, campaign, and applicable test-email creation, the backfill is complete, and validation reports no unresolved customer data should a later migration make ownership required and the relevant columns `NOT NULL`. That future deployment can also make tenant IDs mandatory in API requests. Until then, removing null compatibility would break existing Platform callers.

Platform hard deletion is a separate change. It must call the region-local Gophish purge endpoint before deleting `gophish_settings`, retain Platform records when purge fails, and safely retry the idempotent request.
