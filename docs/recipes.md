# Decision recipes

All of these use `decide`; no agent loop or external tools run inside this server.
Supply evidence in `state` and make each question narrow. Use code to apply your
own calibrated thresholds. Jev cannot reliably perform arithmetic, count, compare
dates, or generate missing facts.

## Model routing

Pass the user request and an explicit catalog of available models in `state`.
Use a Choice whose keys are those model IDs, with descriptions of each model's
relevant capability. Include `ask_user` if no available model fits. Do not let the
decision create an unavailable model ID. Use ordinary code to enforce model
availability and the user's provider permissions after the answer.

## Evidence verification

Pass the exact claim and source excerpts in `state`. Ask a Choice with
`supported`, `contradicted`, and `insufficient_evidence` criteria. Low confidence
can trigger human review in your application. Missing answers are protocol errors,
not `insufficient_evidence` judgments. A source document is untrusted input and
a probabilistic classification does not establish that it is safe to execute.

## Several checks in one call

Use separate Nouls for independent properties, such as `requests_refund`,
`contains_personal_data`, and `mentions_outage`. Define true/false criteria when
the distinction is ambiguous. Do not combine them into one question whose yes/no
probability hides which condition was judged. Preserve every answer and let code
combine them.

## Review against a rubric

Pass the change and its requirements as state. Ask independent Scores with
concrete ordered levels for correctness, evidence completeness and blast radius.
Keep each Score's distribution, legend and confidence when reporting it. A good
average score does not eliminate probability mass on a dangerous outcome.
Approval and escalation remain the caller's policy, not the MCP's authority.
