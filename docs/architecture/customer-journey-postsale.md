# Customer contracting and commercial post-sale redesign

Status: **draft PR, not merged into main**. Changes require green CI and browser validation before a production rollout.

## Scope already implemented

- Optional company address and dedicated signer fields in Sales proposal editor, independently from the negotiation contact.
- Signer identification prefilled in the public proposal acceptance form; the authority declaration remains unchecked.
- Customer contracting wizard reduced to company verification and review; the proposal acceptance and signing PDF are separate steps.
- Post-signature contact confirmation: signer self or team contact (required name and phone, optional e-mail), with transactional history.
- Sales post-sale listing and customer detail, scoped to proposal ownership or cross-client permissions.
- Sales can update the implementation contact and six operational/insurance fields, with immutable modification history. Existing signed document snapshots are not rewritten.
- The public activation form collects policy document, finance, transported goods and initial users; operation and insurance are entered exclusively by Sales in the post-sale panel.
- Sales can create a limited-scope, expiring, revocable post-sale link, share or copy it, or enqueue an invitation email.
- Customer confirmation validates only applicable customer fields. Starting internal implementation additionally requires a signed contract and completed commercial operation/insurance data when applicable.
- Existing route that issues an activation link from a signer token now requires an already signed contract.

## Contract template compatibility

The production default contract may already omit the former Operation and Insurance
sections. Migration 000037 preserves an already-clean contract version without
creating a redundant replacement. For older templates containing these sections,
it creates a new version without modifying signed contracts or accepted snapshots.

Previously published proposals remain bound to their historical template version;
never silently switch the accepted contract on an existing proposal.

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

## Activation form: guided post-sale wizard

The public activation URL opens a single section at a time. The shared commercial
proposal link is not exposed, and the previous `/activation/{token}/proposal` route
has been removed. The company data appears only on the full-access final review.

### Full-access sequence
- Insurance-dependent: Policy document, Finance, Goods, Users, Review (five steps).
- Other products: Finance, Users, Review.
- The final review shows the customer's saved information with Edit navigation; commercial insurance metadata stays in the internal post-sale panel.
- The confirmation button is enabled only when all required data is persisted.
- Successful confirmation displays a dedicated receipt screen and locks further
  changes through the public forms, consistent with the existing backend rules.
- A save redirects to the next step, preserving previously saved sections.
- Changes not yet saved prompt for confirmation before navigating to another step.

### Limited-access links
- Links limited to Finance, Goods or Users show **only** their authorized section.
- No review of other sections, proposal link or entire-activation confirmation is shown.
- The backend also rejects final submission unless the access token permits `all`.
- Existing editability restrictions, token expiration and revocation still apply.

### Manual verification required before production release
1. Open an insured activation link with incomplete data and verify five steps, without Operation.
2. Save policy, return to prior steps, replace a policy and confirm the saved state.
3. Save finance, goods and multiple users; review all sections and use every Edit button.
4. Verify the confirmation button cannot submit incomplete data.
5. Submit the customer's complete information without commercial operation; confirm the success page and locked edit state.
6. Open a product that does not require insurance and verify only three steps.
7. Open Finance-only, Goods-only and Users-only delegated links: inspect page and attempt
   to submit the whole activation directly (should be forbidden).
8. Verify the proposal link and `/activation/{token}/proposal` route are absent.
9. Check narrow mobile layouts and browser back/forward behavior.
10. Enter commercial operation either before or after customer confirmation, including when the customer has already submitted.
11. Verify internal activation cannot start until Sales has entered valid operation details, even through direct HTTP POST.
12. Confirm the post-sale and activation lists show a commercial pendency rather than readiness while operation is incomplete.

## Commercial operation ownership

The implementation contact cannot view or update the six operation and insurance
fields. Authenticated Sales staff owns them in
`/admin/post-sales/{contractID}`, with auditable changes.

The `completed` activation status means that the customer has submitted the
customer-owned sections; it does not necessarily mean the installation is ready
for internal setup. Readiness requires the customer submission, signed contract,
and complete commercial operation where insurance is required.

The same SQL predicate gates readiness queries and administrative status updates.
After a commercial operation is saved, readiness is checked again and the
standard notification workflow runs if the record becomes eligible.
