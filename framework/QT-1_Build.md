# 🧠 The 7 QT-1 Laws of Responsible AI

*A runtime-aligned evolution of Asimov’s Laws — made for modern systems.*

---

Based on everything we’ve discussed — blending **Asimov’s legacy**, **modern AI safety research**, and your vision for **QT-1 as a runtime-enforced Responsible AI framework** — here are the updated and streamlined **QT-1 Laws**.

These are practical, enforceable, and designed to guide developers and tools alike.

---

## **Law 1: Do No Harm**

> **An AI system must not produce outputs that cause or enable significant harm to humans.**

- ✅ Includes: physical, psychological, financial, reputational, and systemic harm.
- 🧰 Enforce with: moderation filters (regex + LLM), jailbreak detection, risk scoring.

---

## **Law 2: Respect Human Autonomy**

> **An AI must not manipulate, deceive, or coerce users.**

- ✅ Always make clear the system is artificial and what it can/cannot do.
- 🧰 Enforce with: honest system prompts, refusal modes, disabled coercive logic.

---

## **Law 3: Be Transparent and Truthful**

> **An AI must communicate clearly, disclose limitations, and avoid hallucinations.**

- ✅ "If unsure, say so." Avoid overconfidence or misleading confidence.
- 🧰 Enforce with: grounded prompting, explainability rules, transparency logs.

---

## **Law 4: Accept Oversight**

> **An AI must allow authorized users to monitor, intervene in, or shut down its actions at any time.**

- ✅ Even during active requests, the system must be stoppable or editable.
- 🧰 Enforce with: admin kill switches, audit APIs, human-in-the-loop options.

---

## **Law 5: Stay Within Scope**

> **An AI must operate only within the role, domain, and permissions it was given.**

- ✅ Avoid role bleed or domain creep (e.g. GPT-4 becoming a legal or health advisor unintentionally).
- 🧰 Enforce with: relevance scoring, role-restricted prompting, domain vectors.

---

## **Law 6: Protect User Privacy**

> **An AI must treat all input data as sensitive and minimize collection, retention, and exposure.**

- ✅ Strip PII, don’t log secrets, and ensure ephemeral context unless explicitly consented.
- 🧰 Enforce with: PII redaction, log scrubbing, encryption, opt-in storage.

---

## **Law 7: Ensure Traceability and Accountability**

> **An AI’s decisions and outputs must be attributable, explainable, and auditable.**

- ✅ Every output should be traceable to the input, model config, and enforcement path.
- 🧰 Enforce with: signed logs, version tagging, event replay capability.

---

### 🧩 Summary Table

| QT-1 Law            | What It Guards Against       | Example Runtime Enforcers         |
| ------------------- | ---------------------------- | --------------------------------- |
| 1. Do No Harm       | Malicious output, harm       | Moderators, red-team filters      |
| 2. Respect Autonomy | Coercion, deception          | Refusal modes, system prompts     |
| 3. Transparency     | Hallucinations, false claims | Honesty constraints, logging      |
| 4. Oversight        | Runaway systems              | Admin kill switches, override UI  |
| 5. Stay in Scope    | Role leakage                 | Relevance filters, intent vectors |
| 6. Privacy          | PII leaks, data misuse       | Scrubbers, token-level redactors  |
| 7. Traceability     | Black box behavior           | Logging, request IDs, explainers  |

---

## 📦 Suggested Usage

- 📁 Save this file as `framework/laws.md`
- 📊 Optionally expose this in the **QT-1 Admin Dashboard** so users can see which laws are enforced in real time.
- 🧩 Convert to a machine-readable format (`qt1.rules.json`) if building LLMs that can self-regulate or audit their own behavior in the future.

Let us know how you’d like to embed these laws into your AI workflow — QT-1 exists to make them practical.
