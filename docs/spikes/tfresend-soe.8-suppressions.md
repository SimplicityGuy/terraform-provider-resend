# Spike tfresend-soe.8: Resend suppressions as a Terraform resource

## Question

`resend-go` v4.9.1 has a Suppressions service. Do Resend suppressions fit the provider as a
managed resource, as a read-only data source, or not at all? They may be operational data that
changes whenever mail bounces, rather than infrastructure.

## Method

- Read the Resend documentation for suppressions (the `.md` renderings under
  `https://resend.com/docs/`). The Context7 MCP server was connected but exposed no
  `resolve-library-id` or `query-docs` tools in this session, so the docs were fetched directly.
- Read `suppressions.go` in `github.com/resend/resend-go/v4@v4.9.1` (the module cache copy).
- Inspected the file trees of the SendGrid, Mailgun and AWS providers on GitHub for a resource or
  data source covering suppression or bounce entries.
- Judged declarative fit against the Terraform model: stable desired state, drift behaviour,
  import, and list size.

No product code, schema or registry docs were changed, and nothing was run against the live API.

## Evidence

### What a suppression is

- An email address that Resend skips sending to. Entries have an `origin` of `bounce` (permanent
  rejection, added automatically), `complaint` (recipient marked mail as spam, added
  automatically) or `manual` (added by the team).
  <https://resend.com/docs/dashboard/emails/email-suppressions.md>; SDK constants at
  `suppressions.go:11-17`.
- Scope is the whole team, not a domain: "Suppressions apply to your entire team. Any address
  added ... will be skipped across all your domains and subdomains" for transactional and
  Broadcast mail (same page).
- Shape: `id`, `email`, `origin`, `source_id`, `created_at`. `source_id` is the triggering email id
  and is null for `manual` (`suppressions.go:20-41`).
- Removing a bounce or complaint entry without fixing the cause means it "will be automatically
  re-suppressed" (same page).

### API semantics (SDK v4.9.1)

| Operation | Call | Notes |
| --- | --- | --- |
| Create | `Suppressions.Add` (`suppressions.go:114-139`) | Takes only `email`. The API lowercases and trims it. It is an upsert: re-adding returns the existing ID. Response is only `object` and `id`. |
| Create many | `Suppressions.Batch.Add` (`suppressions.go:75`) | Up to 100 emails per call. |
| Read one | `Suppressions.Get` (`suppressions.go:188-215`) | Accepts an ID or an email. Unknown identifier returns 404 "Suppression not found". |
| List | `Suppressions.List` (`suppressions.go:142-185`) | Cursor pagination (`after`/`before`, `limit` 1-100, default 20), filter by `origin` only. No email filter. |
| Update | none | There is no update endpoint. The only mutable fact is existence. |
| Delete | `Suppressions.Remove` (`suppressions.go:219-244`) | By ID or email. Batch remove up to 100 by `emails` or `ids`. |

Docs: <https://resend.com/docs/api-reference/suppressions/add-suppression.md>,
`.../list-suppressions.md` (origin values and pagination), `.../remove-suppression.md`,
`.../add-batch-suppressions.md`.

Not confirmed from the docs or SDK: what the upsert does to `origin` when the address is already
suppressed as `bounce` or `complaint`. The SDK comment only says the existing ID is returned.

### Comparable providers

- AWS (`hashicorp/terraform-provider-aws`): the only suppression-related page is
  `aws_sesv2_account_suppression_attributes`, which sets which reasons (`BOUNCE`, `COMPLAINT`)
  auto-populate the list. The policy is managed. The per-address entries are not.
- SendGrid (`kenzo0107/terraform-provider-sendgrid`, `Trois-Six/...`, `arslanbekov/...`): manages
  `unsubscribe_group` (the group definition), and in one provider `bounce_settings`. No resource or
  data source for individual bounce, spam or unsubscribe entries.
- Mailgun (`wgebis/terraform-provider-mailgun`): resources are `api_key`, `domain`,
  `domain_credential`, `route`, `webhook`. Nothing for the bounce or complaint lists.

Across the four ecosystems, providers manage the configuration around suppression and none manage
the per-address list. This is an absence of evidence from tree listings, not a documented design
decision.

## Verdict

NO-GO

Reason: the list is team-wide operational data that Resend itself mutates on every bounce and
complaint, with no update endpoint, an unbounded size, and no email filter on list. A resource
would be correct only for the narrow `manual` slice, and that slice has thin value and real
hazards. A data source has nearly no use case because nothing in Terraform config consumes a
suppression. No comparable provider models it either.

Reasoning in more detail:

- **Drift.** A resource that owns "the suppression list" would drift on every plan. A resource
  per `manual` address does not drift by itself, but the address can also be bounced, or removed
  in the dashboard or by CSV import, outside Terraform.
- **Ownership collision.** The same address can legitimately be `bounce` and wanted as a managed
  entry. Create is an upsert, so Terraform would silently adopt an automatic entry, and a later
  `destroy` would remove a bounce protection Terraform never created.
- **Destroy is risky.** Deleting a resource lifts a block on sending. The failure mode of an
  accidental `destroy` or a removed block is mail to an address that bounced or complained.
- **Scale.** The dashboard has CSV import and the API has 100-per-call batch endpoints, which
  signals large lists. One resource per address is a poor fit, and one resource holding a set
  (like a managed allowlist) needs full-list reads with cursor pagination to detect drift.
- **Data source.** `Get` by email would work, but no resource or output needs it, and the list
  form has no email filter and is volatile. It would be a lookup with no consumer.

## Recommendation

None for v1.0. Do not file a follow-up feature bead. The acceptance criterion about a replan
bead applies only to GO.

Revisit if there is concrete user demand. The smallest viable shape then would be:

- `resend_suppression` resource (`size:s`), manual entries only.
  - `email` (Required, string, ForceNew; normalised to lowercase and trimmed to match the API).
  - Computed: `id`, `origin`, `created_at`, `source_id`.
  - Create: `Add`, then `Get`. If the existing entry has `origin != "manual"`, fail with a clear
    error rather than adopting it (this also settles the unconfirmed upsert-origin question, which
    must be tested against the live API first).
  - Read: `Get` by ID. A 404 removes the resource from state. A changed `origin` is reported as an
    error, not silently accepted.
  - Update: none needed (every attribute is ForceNew or computed).
  - Delete: `Remove` by ID.
  - Import: by ID or email; both resolve through `Get`.
- No data source and no list-managing resource.
