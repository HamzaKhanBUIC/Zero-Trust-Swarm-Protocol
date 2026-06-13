# FOLDER CONTRACT — Template C (Speed Sprint)
# 72-Hour Hackathon | Minimal Viable Structure
# Version 3.0

## THE PRINCIPLE
Ship a working demo. Every folder you create is time you're not coding.
Maximum 3 feature folders. If it doesn't exist in the first 2 hours, it doesn't exist.

---

## THE DIRECTORY TREE

```
project-root/
│
├── 📋 AGENTS.md
├── 📋 AGENT_RULES.md
├── 📋 GEMINI.md
├── 📋 PROJECT_DNA.md               # Filled in 10 minutes, never touched again
│
├── 📁 src/
│   ├── 📁 core/
│   │   └── config.py / config.ts   # ONE file. Env vars only.
│   │
│   ├── 📁 ui/                      # All frontend — NO sub-folders unless unavoidable
│   │   ├── components/
│   │   └── pages/
│   │
│   ├── 📁 api/                     # All backend routes — ONE file if possible
│   │   └── routes.py / routes.ts
│   │
│   └── 📁 [wow_feature]/           # The ONE thing that makes judges say "whoa"
│       └── [flat files, no nesting]
│
├── 📁 docs/
│   └── HACKATHON_SPRINT.md         # Elevator pitch + 3 features. That's it.
│
└── 📁 .github/
    └── workflows/
        └── deploy.yml              # One-click deploy only
```

---

## THE 72-HOUR LAWS

**Law 1 — Flat is Fast**
No folder should have more than 1 level of sub-folders. Nesting = overthinking.

**Law 2 — The Wow Folder**
One folder must be named after your killer feature. This is your demo anchor.
Everything else exists to support it.

**Law 3 — No Migration Files**
Use hardcoded seed data or a single `db_setup.py` script. No numbered migrations.
You have 72 hours, not 72 weeks.

**Law 4 — Mock First, Real Later**
If an API or database isn't ready, create `mock_data.py` in the relevant folder.
A working demo with mock data > broken demo with real data. Always.

---

## SCAFFOLD COMMAND

```bash
mkdir -p src/{core,ui/{components,pages},api,wow_feature} docs .github/workflows

touch src/core/config.py
touch src/api/routes.py
touch PROJECT_DNA.md
echo "# $(basename $(pwd)) - Hackathon Sprint" > docs/HACKATHON_SPRINT.md
```

**Total setup time: Under 2 minutes. Now go build the demo.**
