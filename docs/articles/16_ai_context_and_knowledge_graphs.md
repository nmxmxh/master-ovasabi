# Providing Context for AI Agents: Overcoming Median Bias with Knowledge Graphs

## Introduction

Artificial Intelligence (AI) is rapidly transforming how we build, operate, and reason about complex
systems. Yet, despite its scale and speed, AI often struggles with context, best practices, and
architectural nuance. This article explores the fundamental contrast between AI and human
intelligence, the problem of "median bias" in AI, and how context, knowledge graphs, and curated
references can unlock AI's true productivity for technical teams.

---

## Human Intelligence vs. AI Intelligence: The Context Gap

### Human Intelligence

- **Starts from nothing:** Humans are born with no knowledge and slowly build intelligence through
  experience, mentorship, and exposure to best practices.
- **Contextual learning:** Human experts develop intuition and judgment by learning from
  high-quality examples, mistakes, and domain-specific mentorship.
- **Bias toward excellence:** With the right environment, humans can learn from the best and avoid
  common pitfalls, leading to mastery and innovation.

### AI Intelligence

- **Starts with scale:** AI models (like LLMs) are trained on vast, diverse datasets, absorbing the
  "average" of human knowledge and code.
- **Median bias:** Most training data represents the median, not the best. AI is excellent at
  producing "average" solutions, but struggles with optimal architecture, best practices, or
  domain-specific nuance.
- **Lacks context:** Without explicit context, AI cannot distinguish between good, bad, or outdated
  patterns. It has no "intuition" for what works best in your environment.

---

## The Problem: AI's Median Bias in Practice

- **AI can generate code, docs, and designs at scale, but...**
  - It often proposes "safe" or "common" solutions, not the best ones for your system.
  - It may miss architectural patterns, compliance requirements, or performance optimizations that
    an expert would know.
  - It cannot infer your team's standards, legacy constraints, or business goals without explicit
    context.

**Example:**

- Ask an LLM to design a microservices architecture. It will produce a generic, textbook
  answer—ignoring your team's preferred stack, deployment model, or integration patterns.

---

## The Solution: Context, Knowledge Graphs, and Curated Scope

### 1. **Provide Explicit Context**

- Give the AI agent detailed information about your system, goals, and constraints.
- Use architectural diagrams, service registries, and documentation as input.
- The more context, the more relevant and high-quality the AI's output.

### 2. **Leverage Knowledge Graphs**

- A knowledge graph encodes the relationships, patterns, and best practices of your system.
- It acts as a "memory" for the AI, allowing it to reason about dependencies, compliance, and
  impact.
- With a knowledge graph, AI can:
  - Suggest changes that fit your architecture
  - Warn about breaking changes or anti-patterns
  - Recommend proven solutions from your own history

### 3. **Limit Scope to Good and Tenured References**

- Curate the data and examples the AI can access.
- Prioritize references from your best engineers, successful projects, and trusted sources.
- Exclude outdated, low-quality, or irrelevant patterns.
- This "scope limiting" turns AI from a generic assistant into a domain expert.

---

## Maximizing AI Productivity: A Practical Approach

1. **Build and maintain a system knowledge graph** (e.g., Amadeus KG)
2. **Integrate the knowledge graph with your AI agents**
3. **Feed the AI high-quality, context-rich prompts**
4. **Continuously update the graph with new patterns, migrations, and lessons learned**
5. **Review and curate the AI's outputs, feeding back improvements**

---

## Conclusion

AI's power comes from scale, but its productivity and architectural quality depend on context. By
bridging the gap between median-trained models and your team's best practices—using knowledge
graphs, curated references, and explicit context—you can turn AI from a generic code generator into
a true engineering accelerator.

**Key Takeaway:**

> "AI is only as good as the context and examples you give it. With the right knowledge graph and
> scope, it can become your team's most productive expert."

---

_For more on building self-documenting, AI-ready platforms, see the Amadeus Knowledge Graph
documentation and architecture guides in this repo._
