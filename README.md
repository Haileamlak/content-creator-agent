# Autonomous Content Creator Agent 🚀

A production-ready AI agent built in Go that autonomously researches trends, generates platform-optimized content, maintains brand memory, and synchronizes live performance analytics.

## 🌟 Key Features

- **Autonomous Loop**: `Research` -> `Plan` -> `Generate` -> `Evaluate` -> `Post`.
- **Resilient Research**: Dual-provider trend discovery using NewsAPI with an automatic HTML-scraping fallback to DuckDuckGo.
- **Brand Memory (RAG)**: Maintains a local vector store (semantic memory) to ensure brand consistency and learn from past successful posts.
- **Multi-Platform Support**: Internal clients for **Twitter/X (v2)** and **LinkedIn (ugcPosts)**.
- **Live Analytics loop**: Automatically fetches likes, shares, and comments to update its internal "performance score."
- **Daemon Mode**: Run as a background service with configurable intervals.
- **Self-Critique**: Integrated evaluation step where the agent critiques its own drafts against brand voice guidelines before publishing.

## 🏗 Architecture

The project follows a pragmatic, flat package structure designed for rapid iteration:

- `cmd/`: Entry point and CLI orchestration.
- `agent/`: Core autonomous logic and cycle management.
- `tools/`: External integrations (LLM, Search, Social, Analytics, Embeddings).
- `memory/`: Persistence layers (File-based history and Local Vector Store).
- `models/`: Shared data structures.

## 🚦 Getting Started

### Prerequisites

- Go 1.21+
- [Gemini API Key](https://aistudio.google.com/app/apikey) (Required for LLM and Embeddings)
- (Optional) NewsAPI Key, X API Credentials, LinkedIn Access Token.

### Installation

```bash
git clone https://github.com/Haileamlak/ai-content-creator-agent.git
cd ai-content-creator-agent
go mod download
```

### Configuration

Create or modify a brand profile in `config/`:

```json
{
  "id": "tech_startup",
  "name": "Nebula AI",
  "industry": "Enterprise AI",
  "voice": "Professional, visionary, yet pragmatic",
  "target_audience": "CTOs and Engineering Leaders"
}
```

### Environment Variables

```bash
export GEMINI_API_KEY="your-key"
# Optional
export NEWSAPI_KEY="your-key"
export TWITTER_API_KEY="your-key"
# ... and other social credentials
```

## 🛠 Usage

### Single Run
Executes one cycle (Research -> Post -> Save).
```bash
go run cmd/main.go --config config/tech_startup.json
```

### Daemon Mode
Starts the agent in autonomous mode, running every 4 hours.
```bash
go run cmd/main.go --daemon --interval 4h
```

### Analytics Sync
Manually update performance metrics for all past posts.
```bash
go run cmd/main.go --sync
```

## 📈 Performance Tracking

The agent stores analytics in `data/{brand_id}/history.json` and updates the vector index metadata. This allows the planning phase to semantically retrieve "what worked before" based on real engagement data.

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.
