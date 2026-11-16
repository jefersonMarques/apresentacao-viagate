# Customer contracting and commercial post-sale redesign

Status: **draft PR, not approved for production deployment**.

## Scope already implemented

- Optional company address and dedicated signer fields in Sales proposal editor, independently from the negotiation contact.
- Signer identification prefilled in the public proposal acceptance form; the authority declaration remains unchecked.
- Customer contracting wizard reduced to company verification and review; the proposal acceptance and signing PDF are separate steps.
- Post-signature contact confirmation: signer self or team contact (required name and phone, optional e-mail), with transactional history.
- Sales post-sale listing and customer detail, scoped to proposal ownership or cross-client permissions.
- Sales can update the implementation contact and six operational/insurance fields, with immutable modification history. Existing signed document snapshots are not rewritten.
- The existing activation form provides policy-document upload, finance, transported goods, system users and the new operation form.
- Sales can create a limited-scope, expiring, revocable post-sale link, share or copy it, or enqueue an invitation email.
- Activation completeness validates policy document, insurer, dates, operation type, goods, finance and users where relevant. Internal setup remains gated by a fully signed contract.
- Existing route that issues an activation link from a signer token now requires an already signed contract.

## Contract template blocker

The existing published template includes operation and insurance variables such as `{operation.type}` and `{insurance.insurer}`. The new onboarding flow intentionally does not collect these fields prior to signature.

The generator refuses to create a new PDF when an insurance-dependent proposal uses a contract template requiring uncollected insurance values. It does **not** replace clauses or change accepted proposal version references. The definitive revised commercial contract needs review and approval before these new journeys can be used for insurance-dependent proposals.

## Compatibility and audit

- Signed PDFs, cryptographic hashes and existing contract template versions must remain untouched.
- Published proposals pin their original template version. Existing accepted, unsigned proposals with insurance-dependent legacy templates require a dedicated migration/continuation decision (legacy completion flow or explicit new accepted proposal); never silently replace the bound template.
- Legacy operation values continue to be read from onboarding when the new post-sale operation row is absent.
- Post-sale contact and operation histories append actor metadata and timestamps. Link-based actions are recorded as access-by-link, not independently verified identity.
- Temporary sharing links expire after 30 days; issuing a new Sales link revokes previously Sales-issued links for the same contract, not all historical owner links.
- No global HTMX boost is enabled. PR #43 is a separate progressive HTMX pilot; any integration must preserve existing scripts and fallback behavior.

## Remaining before merge or release

1. Obtain and approve a revised contract template and decide on version handling for already-published proposals.
2. Add browser E2E coverage for acceptance, login/session expiration, PDF signing, post-sale forms and activation permission/authorization, including historic proposal compatibility.
3. Evaluate a dedicated authenticated staff editor for all operational sections rather than using the public form via a staff-generated link.
4. Validate the full new workflow against a realistic database and approved contract template.
5. Review operational permissions and whether the notification and menu counters should include only owned post-sale tasks.
6. Coordinate deployment so schema changes, templates and server code roll out atomically. Never change existing signed evidence.

## Release safety

This PR remains a draft. CI success proves compilation, unit tests, migrations and existing integration checks; it does **not** certify that the end-to-end contracting workflow is ready for production.
