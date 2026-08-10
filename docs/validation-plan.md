# TetherBound product validation plan

Status: active discovery plan  
Last updated: 2026-08-10

The purpose of this phase is to determine whether TetherBound solves a painful,
repeatable problem before investing heavily in the hosted control plane. The
landing site is a research instrument first and a marketing site second.

## 1. Questions we need to answer

### Problem

1. Do people regularly want an AI assistant to inspect machines it cannot
   currently reach?
2. Are they already copying logs, command output, screenshots, or monitoring
   results into ChatGPT manually?
3. Is private-network connectivity the main blocker, or are identity, approval,
   audit, and trust larger blockers?
4. Is the problem frequent enough to justify installing and maintaining a
   connector?

### Security

1. Does “typed capabilities instead of a shell” materially change willingness to
   use the product?
2. Which read capabilities feel acceptable on development, staging, homelab,
   and production systems?
3. Which local-policy guarantees are required before users trust a hosted
   service?
4. Do users require zero raw-output retention, configurable retention, or a fully
   self-hosted path?
5. Are users willing to run a small open-source host agent on Windows/macOS to
   avoid passwords, WinRM, or unrestricted SSH?

### Product

1. Is the Community edition valuable on its own?
2. Which parts of the current setup create enough friction to justify Cloud?
3. Is a persistent customer-side edge connector acceptable?
4. Which AI clients matter: ChatGPT, Codex, other MCP clients, or all of them?
5. Is mixed-platform support a requirement or merely nice to have?

### Commercial

Do not lead interviews with pricing. After the problem and preferred product
shape are understood, test:

1. Would a user pay to avoid the current tunnel/auth/connector lifecycle work?
2. Is value primarily per user, per workspace, per connector, per host, or per
   managed environment?
3. Does willingness to pay correlate more strongly with convenience or with
   governance/audit?

## 2. Initial hypotheses

| Hypothesis | Evidence that supports it | Evidence that weakens it |
| --- | --- | --- |
| AI users want visibility into private systems from the chat surface they already use. | Repeated manual copying of logs/commands; users ask for access before being prompted. | Users prefer existing monitoring dashboards or terminal workflows and see no benefit to AI access. |
| Safety is the differentiator, not SSH connectivity alone. | Users react positively to no-shell, exact tools, local policy, and non-root/agent design. | Users only want an unrestricted command executor and do not value the boundaries. |
| Mixed Linux/Windows support expands the real market materially. | Prospects have mixed fleets and identify Windows as a blocker. | Nearly all target users operate Linux-only environments. |
| Community drives trust and adoption. | Users inspect code, self-host first, and later ask for managed convenience. | Users primarily want a hosted service and will not run an edge process. |
| Cloud can charge for reduced operational friction and governance. | Users dislike developer tunnels/runtime keys and ask for central health, policy, audit, or team access. | Users are satisfied with the DIY path once bootstrap exists. |
| An outbound customer-controlled edge is an acceptable hosted trust model. | Users prefer it over a cloud node inside their tailnet and accept always-on edge software. | Users reject any hosted relay or insist on a fully local path only. |

## 3. Who to interview first

Target 10–15 qualitative conversations before optimizing for traffic.

Prioritize:

1. self-hosters with at least 3 machines;
2. developers with a VPS plus local or remote development machines;
3. people with mixed Linux and Windows fleets;
4. small teams operating staging/production services;
5. consultants or agencies that manage multiple isolated environments;
6. existing heavy ChatGPT/Codex users who already use Tailscale or another
   private-network overlay.

Avoid over-indexing on people who do not currently use AI for technical work.
Their feedback can be useful later but will muddy the initial problem signal.

## 4. Interview script

Keep the first half product-neutral.

### Existing behavior

- Tell me about the machines you personally manage.
- When something is wrong, where do you usually start?
- How do you use ChatGPT or another AI assistant during that process today?
- What information do you copy into the assistant manually?
- Was there a recent time you wished the assistant could inspect something
  directly?
- What stopped you from connecting it?

### Trust boundary

Only after understanding current behavior, explain the concept:

> TetherBound gives an AI assistant a small set of read-only infrastructure
> capabilities through a customer-controlled connector. It does not expose a
> generic shell. Hosts, services, log sources, output size, and operation time
> are explicitly bounded.

Then ask:

- Which part of that design makes you more comfortable, if any?
- What still worries you?
- Would you use it on a homelab machine? Development? Production?
- What is the most privileged read capability you would permit?
- Would you install an open-source agent on Windows/macOS to avoid storing a
  password or opening WinRM?
- What would you need to audit after an AI request?

### Community vs Cloud

Show both architectures without suggesting a preferred answer.

- Would you run the Community version yourself?
- Which setup step would be most annoying?
- Would you prefer a hosted MCP endpoint if the tailnet identity and final local
  policy remained on your edge connector?
- Is an always-on edge connector acceptable?
- What data would you permit to transit a hosted relay?
- What should never be retained?

### Future actions

Do not assume users want write access.

- Are read-only diagnostics enough to be useful?
- Is there any action you would eventually trust the assistant to prepare?
- Would a separate approval be enough, or would you still refuse automation?

## 5. Landing-page CTAs

Primary CTA:

> **Try Community**

Signal: willingness to install and tolerate technical setup.

Secondary CTA:

> **Join Cloud early access**

Signal: desire for a managed experience.

Validation CTA:

> **Share your use case**

Collect:

- role/use context;
- approximate fleet size;
- operating-system mix;
- current remote-access method;
- AI client(s);
- three concrete questions they would ask;
- environments they might permit;
- largest security concern;
- Community/Cloud preference;
- willingness to do a short interview.

Do not collect infrastructure credentials, private IPs, customer names, logs,
hostnames, or other operational secrets.

## 6. Signal quality

### Strong signals

- A user installs the Community build without direct maintainer help.
- A user asks to connect a second or third machine after the first one works.
- A user names several recurring questions they would ask every week.
- A user asks for Windows/macOS support because it blocks current adoption.
- A user asks for Cloud because the current OpenAI tunnel/setup is too involved.
- A team asks for policy, audit, role separation, or approvals.
- A user voluntarily returns after the first test and keeps using it.
- A user is willing to let the read-only product inspect a nontrivial environment
  after reviewing the security model.

### Medium signals

- GitHub stars, page visits, waitlist signups, or positive comments without an
  installation attempt.
- People say the idea is useful but cannot name a concrete question or machine
  they would connect.
- People request many features before trying the existing read path.

### Weak / vanity signals

- Social likes.
- “Cool idea” reactions.
- Generic interest from people who do not operate infrastructure.
- Feature requests that effectively turn the product into an unrestricted shell.

## 7. Metrics for the validation site

If analytics are added, prefer a privacy-respecting, minimal setup and document
it publicly. Track only what informs product decisions.

Useful events:

- landing page viewed;
- Community CTA clicked;
- Docs / bootstrap clicked;
- Security page viewed;
- Cloud early-access CTA clicked;
- validation issue started/submitted;
- operating-system mix from the voluntary form;
- Community vs Cloud preference.

Do not fingerprint visitors or add session-replay software during early trust
validation. The security product should not begin by surprising visitors with
aggressive tracking.

## 8. Initial success criteria

These are directional gates, not statistical proof.

After 10–15 serious conversations and at least 5 external installation attempts:

### Continue Community regardless of Cloud if

- at least 3 external users successfully connect a real machine;
- at least 2 use the product more than once;
- the security model is understood without extensive explanation;
- no recurring setup failure requires unsafe defaults.

### Build a hosted alpha if

- at least 5 credible prospects explicitly prefer a managed endpoint/connector
  lifecycle over the DIY tunnel workflow;
- at least 3 identify a repeated use case, not merely curiosity;
- users accept the customer-controlled outbound edge model;
- at least 2 would evaluate or pay for team policy/audit or convenience;
- no unresolved trust objection invalidates the relay architecture.

### Prioritize Windows immediately if

- at least 3 credible prospects cite Windows as a concrete blocker, or
- one high-intent design partner has a mixed Windows/Linux fleet and is willing
  to test the agent.

### Reconsider the product shape if

- users overwhelmingly want unrestricted shell execution rather than typed
  operations;
- users do not care about interacting through an AI client once existing
  monitoring/terminal workflows are considered;
- the edge connector itself is rejected by most hosted prospects;
- users require a trust/data model incompatible with the proposed relay;
- repeated usage is absent after successful installation.

## 9. Design-partner program

The first design partners should receive direct maintainer help in exchange for
specific feedback. Keep the cohort intentionally small.

Ideal commitment:

- 30–45 minute initial interview;
- one Community installation attempt;
- connect at least one noncritical real machine;
- use the tool for a week;
- report confusing setup, missing read capabilities, and trust concerns;
- optional Windows-agent test when available;
- follow-up interview.

Do not ask design partners to place the early read-only build on critical
production systems purely for validation.

## 10. Decision log

Record material product-learning decisions in this file or linked issues. Each
entry should include:

- date;
- observation;
- number/type of users supporting it;
- decision;
- whether it changes roadmap, product copy, or architecture.

Avoid rewriting the roadmap from one enthusiastic interview. Look for repeated
behavior and consistent blockers.
