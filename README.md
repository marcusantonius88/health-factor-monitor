<div align="center">

# Health Factor Monitor

**Monitor de posições de crédito descentralizado em Aave e Kamino diretamente pelo terminal.**

Uma CLI em Go para consultar Health Factor de posições de lending, classificar risco e exibir resultados de forma clara, com tolerância a falhas e arquitetura extensível por providers.

</div>

---

## 🎯 Problema

No ecossistema DeFi, o Health Factor é um indicador central para avaliar o risco de uma posição de crédito. Em protocolos como Aave e Kamino, a saúde da posição pode mudar rapidamente conforme condições de mercado, volatilidade e uso de alavancagem.

Para usuários que operam com múltiplas posições e protocolos, acompanhar esse dado manualmente costuma ser lento, inconsistente e sujeito a falhas de integração.

O Health Factor Monitor nasceu para resolver esse problema, trazendo uma experiência simples e automatizada para:

- Consultar posições configuradas em JSON;
- Validar endpoints e endereços antes da execução;
- Consultar protocolos distintos em um único fluxo;
- Exibir resultado em tabela legível no terminal;
- Manter robustez mesmo quando alguns providers falham.




---

## 🚀 MVP entregue

O MVP já foi implementado com foco em produtividade e confiabilidade.

### Funcionalidades atuais

- Suporte a Aave na rede Ethereum;
- Suporte a Kamino na rede Solana;
- Carregamento e validação de configuração via JSON;
- Verificação de endpoints RPC e endereços de carteira;
- Classificação do Health Factor em:
  - Safe
  - Warning
  - Critical
- Execução por protocolo ou por todas as posições configuradas;
- Tolerância a falhas por posição;
- Saída tabular no terminal com status e erro detalhado;
- Exit code apropriado quando há sucesso parcial ou falha total.

### Como a aplicação funciona

A CLI lê um arquivo de configuração, valida os dados informados e então consulta cada posição com o provider correspondente.

Se uma posição falhar por timeout, RPC inválida ou resposta malformada, o restante continua sendo processado sem interromper a execução do programa.

---

## 🏗️ Arquitetura

A aplicação foi desenhada com separação clara de responsabilidades entre domínio, aplicação e infraestrutura.

```text
                 +----------------------+
                 |        CLI           |
                 |      hfmon           |
                 +----------+-----------+
                            |
                            v
                 +----------------------+
                 |   Check Service      |
                 |  orchestration       |
                 +----------+-----------+
                            |
            <div align="center">

            # Health Factor Monitor

            **Monitor decentralized lending positions on Aave and Kamino directly from the terminal.**

            A Go CLI to query lending positions' Health Factor, rank risk, and present results clearly — built with fault tolerance and an extensible provider architecture.

            </div>

            ---

            ## 🎯 Problem

            In the DeFi ecosystem, the Health Factor is a core metric to assess the risk of a credit position. In protocols such as Aave and Kamino, a position's health can change quickly due to market conditions, volatility, and leverage.

            For users managing multiple positions across protocols, tracking this metric manually is slow, error-prone, and often suffers from integration issues.

            Health Factor Monitor solves this by providing a simple, automated experience to:

            - Load positions configured in JSON;
            - Validate endpoints and wallet addresses before execution;
            - Query multiple protocols in a single flow;
            - Display results in a readable terminal table;
            - Maintain robustness even when some providers fail.


            ---

            ## 🚀 Delivered MVP

            The MVP focuses on productivity and reliability and has already been implemented.

            ### Current features

            - Support for Aave on Ethereum;
            - Support for Kamino on Solana;
            - Load and validate configuration from JSON;
            - Verify RPC endpoints and wallet addresses;
            - Health Factor classification:
              - Safe
              - Warning
              <div align="center">

              # Health Factor Monitor

              **Monitor decentralized lending positions on Aave and Kamino directly from the terminal.**

              A Go CLI to query lending positions' Health Factor, rank risk, and present results clearly — built with fault tolerance and an extensible provider architecture.

              </div>

              ---

              ## 🎯 Problem

              In the DeFi ecosystem, the Health Factor is a core metric to assess the risk of a credit position. In protocols such as Aave and Kamino, a position's health can change quickly due to market conditions, volatility, and leverage.

              For users managing multiple positions across protocols, tracking this metric manually is slow, error-prone, and often suffers from integration issues.

              Health Factor Monitor solves this by providing a simple, automated experience to:

              - Load positions configured in JSON;
              - Validate endpoints and wallet addresses before execution;
              - Query multiple protocols in a single flow;
              - Display results in a readable terminal table;
              - Maintain robustness even when some providers fail.


              ---

              ## 🚀 Delivered MVP

              The MVP focuses on productivity and reliability and has already been implemented.

              ### Current features

              - Support for Aave on Ethereum;
              - Support for Kamino on Solana;
              - Load and validate configuration from JSON;
              - Verify RPC endpoints and wallet addresses;
              - Health Factor classification:
                - Safe
                - Warning
                - Critical
              - Run checks per protocol or for all configured positions;
              - Per-position fault tolerance;
              - Tabular terminal output with status and detailed error information;
              - Appropriate exit codes for partial success or total failure.

              ### How the application works

              The CLI reads a configuration file, validates the provided data, and then queries each position using the corresponding provider.

              If a position fails due to timeout, invalid RPC, or malformed response, the rest of the positions are still processed and the program continues running.

              ---

              ## 🏗️ Architecture

              The application is designed with a clear separation of concerns between domain, application, and infrastructure layers.

              ```text
                               +----------------------+
                               |        CLI           |
                               |      hfmon           |
                               +----------+-----------+
                                          |
                                          v
                               +----------------------+
                               |   Check Service      |
                               |  orchestration       |
                               +----------+-----------+
                                          |
                          +---------------+----------------+
                          |                                |
                          v                                v
                +-------------------+        +----------------------+
                | Aave Provider     |        | Kamino Provider      |
                | Ethereum RPC      |        | Kamino API           |
                +-------------------+        +----------------------+
                          |
                          v
                +-------------------+
                | Domain / Models   |
                | Config / Provider |
                | HealthFactor      |
                +-------------------+
              ```

              ### Layers

              - Domain: entities and business rules such as `Config`, `HealthFactor`, `ProviderResult`, and risk classification.
              - Application: orchestrator responsible for mapping positions to providers and handling failures in a resilient way.
              - Infrastructure: adapters for Aave and Kamino, plus the JSON configuration loader.
              - Interface: CLI that renders readable output and defines exit codes.

              ### Main contracts

              The architecture follows a provider interface pattern, enabling adding new protocols without coupling service logic to concrete protocol implementations.

              ---

              ## ⚙️ Technology stack

              ### Language and runtime

              - Go
              - Terminal-first CLI

              ### Supported protocols

              - Aave (Ethereum)
              - Kamino (Solana)

              ### Integrations and formats

              - JSON-RPC for Ethereum
              - REST API for Kamino
              - Ethereum and Solana address validation
              - JSON configuration loading

              ### Tests

              - Unit tests for domain, providers, and service
              - Validation of behavior for success, timeout, and error scenarios
              - Coverage for classification rules, configuration, and failure recovery

              ---

              ## 🚀 How to use

              ### 1. Configuration

              Create a configuration file with positions and RPC endpoints.

              ```json
              {
                "rpc_endpoints": {
                  "ethereum": "https://mainnet.example-rpc.com",
                  "solana": "https://api.mainnet-beta.solana.com"
                },
                "positions": [
                  {
                    "protocol": "aave",
                    "network": "ethereum",
                    "wallet": {
                      "address": "0x1234567890abcdef1234567890abcdef12345678",
                      "alias": "Main Wallet"
                    }
                  },
                  {
                    "protocol": "kamino",
                    "network": "solana",
                    "wallet": {
                      "address": "8v6GJfY...",
                      "alias": "Solana Position"
                    }
                  }
                ]
              }
              ```

              ### 2. Run

              ```bash
              go run ./cmd/hfmon -config ./config.json
              ```

              ### 3. Filter by protocol

              ```bash
              go run ./cmd/hfmon -config ./config.json -protocol aave
              ```

              ### 4. Example output

              ```text
              Position      Protocol   Network   Health Factor   Status
              Main Wallet   aave       ethereum  2.41             safe
              Solana Pos.   kamino     solana    1.22             warning
              ```

              ---

              ## 🧪 Validation and robustness

              The application is built with realistic operational failures in mind:

              - Invalid or unavailable RPC;
              - Malformed provider response;
              - Query timeout;
              - Invalid address;
              - Unsupported protocol;
              - Missing configuration or required data.

              When an item fails, the system logs the error and continues processing the rest without crashing.

              ---

              ## 📁 Project structure

              ```text
              cmd/
                hfmon/
              internal/
                application/
                domain/
                infrastructure/
                  aave/
                  kamino/
                  config/
                interfaces/
                  cli/

              tests/
                integration/
                testdata/

              specs/
              ```

              ---

              ## AI-assisted development

              This project was developed using modern AI-assisted development practices, focusing on productivity, quality, and technical documentation.

              | Category | Detail |
              | --- | --- |
              | IDE / Agent | OpenCode |
              | Main model | DeepSeek V4 Flash Free |
              | Methodology | Spec-Driven Development (SDD) |
              | Specification framework | Spec Kit |

              Using OpenCode as the development environment and DeepSeek V4 Flash Free as the main model accelerated code authoring, architecture review, test generation, and documentation organization.

              The methodology used is SDD, applied via the Spec Kit available at https://github.com/github/spec-kit. This approach organizes development into specifications, planning, validation, and incremental implementation, keeping the project aligned with requirements, architecture, and acceptance criteria.

              In short, the solution was designed and built with a specification-first workflow: specify first, validate second, implement in small cycles, and continuously review.

              ### Documentation and specification

              The project follows a specification-driven engineering approach, with formal documentation stored in `specs/`.

              The documentation includes:

              - Product vision;
              - Goals and rules;
              - Data model;
              - Interface contracts;
              - Implementation plan;
              - Task checklist and acceptance criteria.

              This approach helps the solution evolve with clarity, traceability, and reduced regression risk.

              ---

              ## 🛣️ Roadmap

              ### MVP completed

              - [x] Project initialization
              - [x] Domain model
              - [x] JSON configuration
              - [x] Endpoint and position validation
              - [x] Aave provider
              - [x] Kamino provider
              - [x] Health Factor classification
              - [x] Tabular CLI output
              - [x] Robustness for individual failures


              ### Next steps

              - [ ] Real integration tests against RPCs and public APIs
              - [ ] Support for additional protocols
              - [ ] Concurrent execution per provider
              - [ ] Historical observation of Health Factor
              - [ ] Alerts and continuous monitoring
              - [ ] Export results to JSON/CSV

              ---

              ## 🔒 Status

              Project is in a functional MVP stage, with an architecture ready for extension and multi-protocol support.

              ---

              ## 📄 License

              This project is licensed under the MIT License.