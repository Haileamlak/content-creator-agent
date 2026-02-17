<p align="center">
  <img src="logo.png" alt="Conca Logo" width="200px">
</p>

**Conca** is a sophisticated, production-grade autonomous agent built in Go, designed to scale from a CLI utility to a comprehensive SaaS platform. It researches trends, generates platform-optimized content, manages multiple brand identities, and optimizes performance using live analytics and semantic memory.

---

## Key Capabilities

- **Autonomous Agent Loop**: A complete `Plan -> Generate -> Evaluate -> Publish` cycle that operates without human intervention.
- **SaaS-Ready API**: A robust HTTP API powered by Chi, featuring JWT authentication and multi-tenant isolation.
- **Enterprise-Grade Storage**: Flexible data layer supporting **PostgreSQL** for massive scale or local JSON files for rapid development.
- **Semantic Memory (RAG)**: Uses Gemini Embeddings and a local Vector Store to learn from past successes and maintain brand consistency semantically.
- **Resilient Trend Research**: Leverages **NewsAPI**, **NewsData.io**, and **DuckDuckGo** scrapers to find what's viral NOW.
- **Live Analytics Optimization**: Automatically tracks engagement metrics (likes, shares, comments) to update internal performance scores.
- **Multi-Platform Posting**: Direct integrations with **Twitter/X (v2)** and **LinkedIn (ugcPosts)**.

---

## High-Level Architecture

The system is built on a pragmatic, flat package structure designed for high velocity and low technical debt:

- `/api`: RESTful handlers, JWT middleware, and server orchestration.
- `/agent`: The "Brain" – core autonomous logic and the planning loop.
- `/tools`: Heavy-lifters (LLM, Search Engines, Social Clients, Analytics).
- `/memory`: Dual-layer persistence (Relational via Postgres/Files + Semantic via Vector DB).
- `/cmd/server`: The SaaS gateway.
- `/cmd/cli`: The developer's power tool.

---

## Getting Started

### Prerequisites
- **Go 1.21+**
- **Gemini API Key** (Required for LLM & Embeddings)
- **PostgreSQL** (Optional, falls back to JSON)

### Installation
```bash
git clone https://github.com/Haileamlak/conca.git
cd conca
go mod download
```

### Environment Configuration
Create a `.env` file or export the following:
```bash
export GEMINI_API_KEY="your-key"
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname" # Optional
export JWT_SECRET="your-secure-secret"
```

---

## Multi-Mode Operation

### 1. SaaS Mode (HTTP API)
Run the agent as a multi-tenant service:
```bash
go run cmd/server/main.go --port 8080
```
**Available Endpoints:**
* `POST /api/auth/register` - Create a user account
* `POST /api/brands` - Define a new brand voice/industry
* `POST /api/brands/{id}/run` - Trigger an autonomous content cycle
* `GET /api/brands/{id}/posts` - Review post history and analytics

### 2. Daemon Mode (CLI)
Continuous background operation for a specific brand:
```bash
go run cmd/main.go --config config/tech_startup.json --daemon --interval 4h
```

### 3. Analytics Synchronization
Sync performance data from social platforms to local memory:
```bash
go run cmd/main.go --sync
```

---

## License
MIT License - see [LICENSE](LICENSE) for details.
