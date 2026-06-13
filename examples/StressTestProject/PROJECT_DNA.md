# PROJECT DNA
# Do not delete. This is the immutable fingerprint of the application.

## 1. CORE IDENTITY
- **App Name**: Swarm Stress Test App (Tasks API)
- **Primary Purpose**: Demonstrate parallel agent coordination building a FastAPI task manager backend.
- **Target Audience**: AI Agents and DevOps

## 2. TECH STACK (IMMUTABLE)
*Agents must NEVER change these without explicit Lead Architect approval.*
- **Backend Framework**: Python FastAPI
- **Language**: Python 3.12
- **Database**: SQLite (via SQLAlchemy)
- **Deployment**: Docker & Docker Compose

## 3. ARCHITECTURAL IMMUTABLES
- **Structure**: Template C (Speed Sprint) — Flat `src/` directory.
- **API Paradigm**: RESTful JSON API
- **State Management**: SQLite local file `tasks.db`
- **Auth**: None (internal tool)

## 4. MODULE INVENTORY
| Module Name | Purpose | Status | Owner |
|-------------|---------|--------|-------|
| Database | SQLite connection and Models | Pending | Database Architect |
| Tasks API | CRUD REST Endpoints | Pending | API Engineer |
| Docker | Containerization | Pending | DevOps Engineer |

## 5. KNOWN DEBT & COMPROMISES
- No authentication implemented.
- SQLite used instead of Postgres for speed.
