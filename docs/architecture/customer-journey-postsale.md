# Contracting and post-sale redesign

Status: implementation staged; no production rollout until the contract template is legally reviewed.

## Customer contracting

1. The proposal acceptance form captures signer name, CPF, role (optional), e-mail, phone and a fresh authority declaration. Explicit signer fields provided by Sales appear prefilled; the negotiation contact is a separate person and must not be used as the signer by default.
2. The company step reuses the address and company identification already captured in the proposal editor and asks the customer to review missing values.
3. The review step confirms the contract data and generates the signing PDF.
4. E-mail OTP and explicit consent are required to sign the immutable PDF.
5. After signing, the customer chooses themselves or a team contact for implementation. This is a separate, resumable action that must never revoke the already completed signature.

## Post-sale process

Sales owns the operational intake: operation type, insurer, policy validity, broker, producer, policy document, transported goods, financial contact and initial system users. Sales can edit or invite an individual limited-scope contact; all mutations require audit events and permissions checks. Operations receives the activation only after all applicable data is complete and the contract is signed.

## Contract template dependency

The currently deployed contract template contains sections 1.3 (operation) and 1.4 (insurance); the generator and onboarding backend require these values when the product needs insurance. These clauses and the insurance gate must only be removed when the commercial team supplies and approves the canonical replacement contract. The original signed documents, hashes and historical template versions remain immutable.

## Release safety

Do not deploy this feature until the post-sale data capture, backend readiness rules, migration and revised contract template are deployed together. A draft PR may include isolated milestones while deployment is blocked.
