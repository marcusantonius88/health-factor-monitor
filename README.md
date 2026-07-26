<div align="center">

# Health Factor Monitor

**Monitor DeFi lending positions across multiple protocols from your terminal.**

*A provider-based, extensible CLI built with Go and developed using Spec-Driven Development (SDD).*

</div>

---

## 📖 Overview

Health Factor Monitor is a command-line application written in Go that monitors the **Health Factor** of lending positions across multiple DeFi protocols.

The project is designed around a **provider-based architecture**, making it straightforward to support additional protocols without changing the application's core.

The initial MVP focuses on:

- Aave (Ethereum)
- Kamino (Solana)

Future providers can be added by implementing the same provider interface.

---

## ✨ Goals

- Simple and fast CLI experience
- Clean Architecture principles
- Provider-based design
- Easy protocol extensibility
- Reliable error handling
- Production-quality Go code
- Testable components
- Incremental development through Spec-Driven Development

---

## 🏗️ Architecture

The application follows a layered architecture that separates domain logic from protocol implementations.

```text
                 +--------------------+
                 |     CLI (hfmon)    |
                 +---------+----------+
                           |
                           v
                 +--------------------+
                 |    Check Service   |
                 +---------+----------+
                           |
             +-------------+-------------+
             |                           |
             v                           v
      Aave Provider              Kamino Provider
             |                           |
             v                           v
     Ethereum RPC/API            Solana RPC/API
```

Each protocol is responsible only for retrieving its own Health Factor.

The application layer orchestrates execution while remaining independent of blockchain-specific details.

---

## 📁 Project Structure

```text
cmd/
└── hfmon/

internal/
├── application/
├── domain/
├── infrastructure/
│   ├── aave/
│   ├── kamino/
│   └── config/
└── interfaces/
    └── cli/

tests/
├── integration/
└── testdata/

specs/
```

---

## 🚀 Current Status

This project is currently under active development.

Current progress:

- ✅ Project bootstrap
- ⏳ Domain model
- ⏳ Configuration loader
- ⏳ Aave provider
- ⏳ CLI
- ⏳ Kamino provider
- ⏳ Integration tests

---

## 🛣️ Roadmap

### MVP

- [x] Project initialization
- [ ] Domain model
- [ ] Configuration management
- [ ] Aave Health Factor provider
- [ ] Health classification
- [ ] CLI interface
- [ ] Kamino provider
- [ ] Integration tests

### Future

- Additional lending protocols
- Concurrent provider execution
- Historical Health Factor tracking
- Alerting system
- Telegram integration
- Metrics & observability
- Export to JSON / CSV

---

## 🧪 Development Process

This project is being developed using **Spec-Driven Development (SDD)**.

Instead of writing code first, every feature follows a structured engineering workflow:

```text
Specification
        ↓
Planning
        ↓
Research
        ↓
Task Breakdown
        ↓
Implementation
        ↓
Review
        ↓
Commit
```

Every implementation task is developed incrementally:

1. Implement
2. Validate
3. Review
4. Commit

This approach keeps the architecture consistent and allows each feature to evolve independently.

---

## 📚 Documentation

Project documentation is located under the `specs/` directory.

It includes:

- Product Specification
- Implementation Plan
- Technical Research
- Data Model
- Contracts
- Task Breakdown

These artifacts document not only **what** was built, but also **why** each architectural decision was made.

---

## 🛠️ Technologies

- Go
- JSON-RPC
- Ethereum
- Solana
- Aave
- Kamino
- Spec Kit
- OpenCode

---

## 🤝 Contributing

Contributions, suggestions, and discussions are always welcome.

If you'd like to improve the project, feel free to open an issue or submit a pull request.

---

## 📄 License

This project is licensed under the MIT License.