# Commercial ownership of insurance operation

The public activation form collects only data assigned to the implementation contact:
policy document (when applicable), financial contact, goods (when applicable), and users.
Insurance operation metadata (operation type, insurer, policy dates, broker, producer) is
owned by Sales under `/admin/post-sales/{contractID}` and is not exposed to public
activation tokens.

The customer can review and submit their data independently of the Sales operation.
The `completed` activation status means the customer has submitted their part,
not necessarily that internal setup is ready.

Internal setup requires BOTH:
- Fully signed contract and activation marked `completed`
- A complete commercial operation when the proposal requires insurance

The same SQL predicate protects the internal readiness query and the internal
status transition, including direct HTTP requests. Legacy onboarding operation data
is used when no post-sale operation record exists, to preserve historical contracts.

When Sales saves an operation after the customer submits, readiness is rechecked
and existing notifications are triggered if the activation becomes eligible.
No existing signed contract or operational history is rewritten.
