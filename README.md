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

- consultar posições configuradas em JSON;
- validar endpoints e endereços antes da execução;
- consultar protocolos distintos em um único fluxo;
- exibir resultado em tabela legível no terminal;
- manter robustez mesmo quando alguns providers falham.

---

## 🚀 MVP entregue

O MVP já foi implementado com foco em produtividade e confiabilidade.

### Funcionalidades atuais

- suporte a Aave na rede Ethereum;
- suporte a Kamino na rede Solana;
- carregamento e validação de configuração via JSON;
- verificação de endpoints RPC e endereços de carteira;
- classificação do Health Factor em:
  - safe
  - warning
  - critical
- execução por protocolo ou por todas as posições configuradas;
- tolerância a falhas por posição;
- saída tabular no terminal com status e erro detalhado;
- exit code apropriado quando há sucesso parcial ou falha total.

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

### Camadas

- Domain: entidades e regras de negócio, como Config, HealthFactor, ProviderResult e classificação de risco.
- Application: orchestrator responsável por mapear posições para providers e tratar falhas de forma resiliente.
- Infrastructure: adapters para Aave e Kamino, além do carregador de configuração JSON.
- Interface: CLI que monta as saídas em formato legível e define códigos de saída.

### Principais contratos

A arquitetura segue o padrão de provider interface, permitindo adicionar novos protocolos sem acoplar a lógica do serviço à implementação concreta de cada protocolo.

---

## ⚙️ Stack tecnológica

### Linguagem e runtime

- Go
- CLI terminal-first

### Protocolos suportados

- Aave (Ethereum)
- Kamino (Solana)

### Integrações e formatos

- JSON-RPC para Ethereum
- REST API para Kamino
- validação de endereços Ethereum e Solana
- leitura de configurações em JSON

### Testes

- testes unitários por domínio, providers e service
- validação de comportamento em cenários de sucesso, timeout e erro
- cobertura de regras de classificação, configuração e recuperação de falhas

---

## 🚀 Como usar

### 1. Configuração

Crie um arquivo de configuração com posições e RPC endpoints.

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

### 2. Execução

```bash
go run ./cmd/hfmon -config ./config.json
```

### 3. Filtrar por protocolo

```bash
go run ./cmd/hfmon -config ./config.json -protocol aave
```

### 4. Exemplo de saída

```text
Position      Protocol   Network   Health Factor   Status
Main Wallet   aave       ethereum  2.41             safe
Solana Pos.   kamino     solana    1.22             warning
```

---

## 🧪 Validação e robustez

A aplicação foi construída pensando em casos reais de falha operacional:

- RPC inválida ou indisponível;
- resposta malformada do provider;
- timeout de consulta;
- endereço inválido;
- protocolo não suportado;
- ausência de configuração ou dado obrigatório.

Quando um item falha, o sistema registra o erro e continua processando o restante, sem quebrar a aplicação.

---

## 📁 Estrutura do projeto

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

## 🧭 Documentação e especificação

O projeto foi desenvolvido com abordagem de engenharia guiada por especificação, com documentação formal armazenada em `specs/`.

A estrutura inclui:

- visão do produto;
- objetivos e regras;
- modelo de dados;
- contratos de interfaces;
- plano de implementação;
- checklist de tarefas e critérios de aceitação.

Essa abordagem permite que a solução evolua com clareza, rastreabilidade e menor risco de regressão.

---

## 🤖 Desenvolvimento Assistido por IA

Este projeto foi construído utilizando práticas modernas de desenvolvimento assistido por IA, com foco em produtividade, qualidade e documentação técnica.

| Categoria | Detalhe |
| --- | --- |
| IDE / Agent | OpenCode |
| Modelo principal | DeepSeek V4 Flash Free |
| Metodologia | Spec-Driven Development (SDD) |
| Framework de especificação | Spec Kit |

A utilização do OpenCode como ambiente de desenvolvimento e do DeepSeek V4 Flash Free como modelo principal permitiu acelerar a escrita de código, a revisão de arquitetura, a geração de testes e a organização da documentação técnica.

A metodologia adotada foi o SDD, aplicada a partir do Spec Kit, disponível em https://github.com/github/spec-kit. Essa abordagem organiza o desenvolvimento em especificações, planejamento, validação e implementação incremental, mantendo o projeto alinhado com requisitos, arquitetura e critérios de aceitação.

Em outras palavras, a solução foi pensada e construída com um fluxo de desenvolvimento estruturado: especificar primeiro, validar depois, implementar em pequenos ciclos e revisar continuamente.

---

## 🛣️ Roadmap

### MVP concluído

- [x] inicialização do projeto
- [x] modelo de domínio
- [x] configuração JSON
- [x] validação de endpoints e posições
- [x] provider Aave
- [x] provider Kamino
- [x] classificação de health factor
- [x] CLI com formato tabular
- [x] robustez para falhas individuais

### Próximos passos

- [ ] testes de integração reais com RPC e APIs públicas
- [ ] suporte a mais protocolos
- [ ] execução concorrente por provider
- [ ] histórico de observação de health factor
- [ ] alertas e monitoramento contínuo
- [ ] exportação de resultados para JSON/CSV

---

## 🔒 Status

Projeto em fase de MVP funcional, com arquitetura pronta para extensão e suporte a múltiplos protocolos.

---

## 📄 Licença

Este projeto está licenciado sob a licença MIT.