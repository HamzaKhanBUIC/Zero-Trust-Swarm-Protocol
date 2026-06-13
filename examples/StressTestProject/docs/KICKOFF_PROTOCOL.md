# 🚀 THE 2026 UNIVERSAL KICKOFF PROTOCOL
Whenever you start a new massive project, execute these 4 steps before writing a single line of code.

## Step 1: Scaffold the Directory
Run this in your terminal to instantly create the perfect, AI-optimized folder structure:

```bash
mkdir -p src/modules src/core docs/rules tests .github/workflows .cursor/rules
touch AGENTS.md AGENT_RULES.md .cursorrules .windsurfrules GEMINI.md docs/HACKATHON_SPRINT.md
```

## Step 2: Lock the AI Guardrails (Cursor 2026)
1. Copy `AGENT_RULES.md` and `AGENTS.md` from your AI Empire template into the project root.
2. Copy `.cursor/rules/empire-bootstrap.mdc` from the template (Cursor auto-loads it every session).
3. In legacy pointer files (`.cursorrules`, `.windsurfrules`, `GEMINI.md`, `.agentrules`), point agents to **both** `AGENTS.md` and `AGENT_RULES.md`:

> "Before any command or code change, read AGENTS.md and AGENT_RULES.md and strictly adhere to them."

## Step 3: Wire the Autonomous Memory
(Pro-Tip for 2026: Since you are building massive projects, the default JSON memory server can sometimes crash if 5 agents try to write to it at once. Use the SQLite version for enterprise-grade stability).

Run this in your terminal to install the thread-safe memory server:

```bash
npm install -g @pepk/mcp-memory-sqlite
```

Then, link it to your IDE's MCP config file using this command:

```json
{
  "command": "npx",
  "args": ["-y", "@pepk/mcp-memory-sqlite"]
}
```

## Step 4: Fill Out the Project Charter
When you finally have your multi-million dollar idea, you will open docs/PROJECT_CHARTER.md and fill out this exact template. (Save this template for later):

```markdown
# MOONSHOT MASTER BLUEPRINT

## 1. THE GLOBAL VISION
[Write 2 sentences describing the product, its impact, and what it does.]

## 2. THE TECH STACK
- Frontend: [e.g., Next.js 15, Tailwind]
- Backend: [e.g., Python FastAPI, PostgreSQL]
- AI / Orchestration: [e.g., LangGraph, Gemini Flash 2.5]
- Infrastructure: [e.g., Vercel, AWS S3, Docker]

## 3. PHASED EXECUTION PROTOCOL
*The AI is strictly forbidden from working outside the current active phase. The Lead Architect will update the checkbox when a phase is complete.*

- [ ] **Phase 1: Foundation (Weeks 1-2)** - Database schemas, Auth, MCP wiring, scaffolding.
- [ ] **Phase 2: Core Engine (Weeks 3-8)** - The primary backend logic or AI routing. No UI work allowed.
- [ ] **Phase 3: The API Layer (Weeks 9-12)** - Exposing the core engine securely via strict types.
- [ ] **Phase 4: Frontend/UI (Weeks 13-20)** - Connecting the UI to the API. 
- [ ] **Phase 5: Polish & Scale (Weeks 21-24)** - Caching, load testing, security audits.

## 4. ARCHITECTURAL IMMUTABLES
- Every feature must be contained in its own isolated directory under `/src/modules`.
- The database schema is the single source of truth. Do not invent tables without updating the migration files.
- All secrets must be pulled from `.env` dynamically. Never hardcode keys.
```
