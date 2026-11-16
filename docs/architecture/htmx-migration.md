# Progressive HTMX migration

## Context

The ViaGate administrative platform uses Go, Chi, and Templ for server-side rendering. The existing `feature/htmx-commercial-platform` experiment contains seven unique commits from 2026-09-01, but diverged from `main` and must not be merged wholesale.

## Architecture

- Keep Go and Templ as the source of truth for presentation and authorization.
- Serve a pinned HTMX 2.0.10 build as a first-party static asset, with its original Zero-Clause BSD license.
- Support ordinary HTML links and forms without JavaScript.
- Prefer server-rendered fragments or bounded `hx-select` replacements; never enable global boosted navigation before verifying existing JavaScript lifecycle behavior.
- Use same-origin HTMX requests and do not cache sensitive admin HTML in HTMX browser history.
- Redirect to the full login page when an authenticated HTMX request expires.
- Preserve existing origin checks and authorization middleware on all requests.

## Phase 1: read-only pipeline pilot

- Enable HTMX only for navigation from the pipeline list to a negotiation's details.
- Replace the main content region without swapping the session menu.
- Keep the ordinary `href` as the progressive-enhancement fallback.
- Keep detail-page actions on standard navigation until event initialization and page-specific asset loading are assessed.

## Subsequent phases (not implemented yet)

1. Test browser history, session expiration, failures, script reinitialization, and responsiveness on the pilot.
2. Introduce reusable Templ partials and explicit response handling in Go for read-only lists and searches.
3. Migrate notifications and administrative forms with permissions, CSRF/origin validation, and a consistent error contract.
4. Convert modals and editable catalog/proposal screens incrementally, retaining native HTML fallbacks.
5. Migrate the remaining administrative modules and evaluate public flows separately.

## Acceptance criteria

- Existing CI, migrations, and authenticated user journeys continue to pass.
- The pilot works with JavaScript enabled and disabled.
- HTMX does not duplicate handlers, leak customer information into history, or bypass permission checks.
- Production builds include the vendored HTMX asset.
- Merge into `main` only after CI and browser validation.

## Branch cleanup

Keep `main`, `feature/htmx-commercial-platform`, and `feature/htmx-progressive-migration` until validation. Remove the other 54 audited obsolete branches using a separate lease-guarded cleanup process. Do not run the older `cleanup_viagate_branches.sh` because it deletes the HTMX experiment and assumes no migration branch exists.
