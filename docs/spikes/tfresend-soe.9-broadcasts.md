# Spike tfresend-soe.9: Resend broadcasts as a Terraform resource

## Question

resend-go v4.9.1 exposes a Broadcasts service. Broadcasts are one-off sends to a
segment, not standing configuration. Does the provider gain anything from a
`resend_broadcast` resource, a broadcast data source, or neither?

## Method

- Read `resend-go/v4@v4.9.1/broadcasts.go` in the module cache.
- Read the Resend API reference and dashboard pages for broadcasts (fetched as
  markdown from resend.com/docs; the Context7 `resolve-library-id` and
  `query-docs` tools were not exposed in this session, so the docs were fetched
  directly).
- Looked at the resource inventory of the SendGrid provider
  `Trois-Six/terraform-provider-sendgrid` through the GitHub contents API.
  A Mailchimp provider search found only `ijcd/terraform-provider-mailchimp`,
  which ships just a scaffolding example, so there is no Mailchimp evidence.
- Compared against `docs/release-plan.md`, which already schedules
  `resend_template` in v0.3.0.

## Evidence

**What a broadcast is.** A campaign email sent to a segment of contacts. The
SDK struct is `Broadcast` (`broadcasts.go:174`): id, name, segment_id, from,
subject, reply_to, html, text, `status`, `created_at`, `scheduled_at`,
`sent_at`.

**Lifecycle.** Statuses are `draft`, `scheduled`, `queued`, `sent`, `canceled`
(https://resend.com/docs/dashboard/broadcasts/manage-broadcasts.md, "Understand
Broadcast statuses"). The Terraform-relevant transitions:

- `draft` to `scheduled` or `queued` through Send (`POST /broadcasts/{id}/send`,
  optional `scheduled_at`; `broadcasts.go:338`).
- `scheduled` back to `draft` through Cancel (`broadcasts.go:369`;
  https://resend.com/docs/api-reference/broadcasts/cancel-broadcast.md).
- `queued` to `canceled` through Cancel. This stops only the unsent remainder
  and is terminal: "cannot be sent again".

**Create.** `POST /broadcasts`. The SDK requires a segment (or the deprecated
audience), `from` and `subject` client-side (`broadcasts.go:232-245`).
`CreateBroadcastRequest.Send` (`broadcasts.go:30`) sends immediately; it
defaults to false, so a plain create makes a draft. `ScheduledAt` is valid only
with `Send` (`broadcasts.go:34`).

**Update.** `PATCH /broadcasts/{id}` (`broadcasts.go:273`). Any draft or
scheduled broadcast is fully editable. "Once a Broadcast has been sent, only its
name can be updated" (manage-broadcasts.md, "Edit Broadcasts").

**Delete.** `DELETE /broadcasts/{id}` (`broadcasts.go:428`). Allowed only for
`draft` or `scheduled`; sent or queued broadcasts "can't be deleted"
(https://resend.com/docs/api-reference/broadcasts/delete-broadcast.md).
Deleting a scheduled one cancels the delivery.

**Read and list.** Get returns broadcasts created by the API and by the
dashboard (https://resend.com/docs/api-reference/broadcasts/get-broadcast.md).
List is paginated (`broadcasts.go:456`). Further read-only endpoints cover
recipients per event type (`broadcasts.go:491`) and clicked links
(`broadcasts.go:546`). These are delivery metrics, not configuration.

**API-only send restriction.** "You can send broadcasts only if they were
created via the API"
(https://resend.com/docs/api-reference/broadcasts/send-broadcast.md).
Dashboard-authored drafts can be read but not sent through the API.

**Templates.** `CreateBroadcastRequest` has no template field in v4.9.1
(`broadcasts.go:19-35`), so a broadcast cannot reference a template through the
SDK. The reusable parts (layout, copy) are what `resend_template` (v0.3.0)
covers.

**Comparable providers.** The SendGrid provider manages `template` and
`template_version` (design assets) plus account configuration (domain
authentication, webhooks, API keys, subusers, unsubscribe groups). It has no
campaign or single-send resource
(https://github.com/Trois-Six/terraform-provider-sendgrid/tree/main/docs/resources).
The pattern is that reusable assets and account config are managed; sends are
not. This is limited evidence: one provider checked, none found for Mailchimp.

## Verdict

**NO-GO**

A broadcast is an event with a one-way lifecycle, not state that converges.
Terraform's contract (idempotent apply, reversible destroy, drift detection)
breaks at every point once a send is possible:

- **Apply can send to real recipients.** Any path that sends (`send = true`, or
  a later Send call) makes `terraform apply` an irreversible side effect, and
  `queued` cancels leave some recipients already emailed.
- **Destroy and replace fail after sending.** Delete is refused for `sent` and
  `queued` broadcasts, so `destroy`, and any forced replacement of a sent
  broadcast, errors out or must be a state-only removal that orphans the
  record.
- **Drift is by design.** Status moves `draft` to `sent` outside Terraform
  (dashboard click, another Send call, a scheduled time elapsing) and content
  becomes frozen except for `name`. Every content attribute would then be
  permanently non-updatable, forcing either `ForceNew` (which cannot delete) or
  ignored changes.
- **A draft-only resource buys little.** It would never send, but its drafts
  are useless until someone sends them out of band, after which the resource is
  in the broken state above. The value of declaring a one-time email in code is
  also low, since each campaign is new content.
- **A data source buys little.** A broadcast id is not known ahead of time, so
  a lookup by id is rarely useful, and the interesting reads (recipients,
  clicks) are metrics, not config, and unsuited to plan-time evaluation.

The reusable part is covered by `resend_template` (v0.3.0).

## Recommendation

Do not build a broadcast resource or data source; file no follow-up bead. Mark
the SDK Broadcasts service as intentionally unmapped in the v1.0.0-rc.1 schema
review, and note in the provider docs that campaigns are sent through the API,
dashboard or CI, with `resend_template` and segment/contact resources managed
in Terraform.

Revisit only if Resend adds a way to create a broadcast that cannot be sent
from Terraform plus an idempotent, state-based send trigger (for example,
template reference on broadcasts). If revisited, the narrowest viable shape is a draft-only `resend_broadcast`:
it never sets `send`, has no schedule attribute, exposes `status` as computed,
and fails the plan with a clear error when the remote status is not `draft`
(except for `name`). Import by broadcast id. Roughly one medium bead.
