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
- Validação de endereços Ethereum e Solana
- Leitura de configurações em JSON

### Testes

- Testes unitários por domínio, providers e service
- Validação de comportamento em cenários de sucesso, timeout e erro
- Cobertura de regras de classificação, configuração e recuperação de falhas

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
- Resposta malformada do provider;
- Timeout de consulta;
- Endereço inválido;
- Protocolo não suportado;
- Ausência de configuração ou dado obrigatório.

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

## � Desenvolvimento Assistido por IA

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

### 🧭 Documentação e especificação

O projeto foi desenvolvido com abordagem de engenharia guiada por especificação, com documentação formal armazenada em `specs/`.

A estrutura inclui:

- Visão do produto;
- Objetivos e regras;
- Modelo de dados;
- Contratos de interfaces;
- Plano de implementação;
- Checklist de tarefas e critérios de aceitação.
Essa abordagem permite que a solução evolua com clareza, rastreabilidade e menor risco de regressão.

---

## 🛣️ Roadmap

### MVP concluído

- [x] Inicialização do projeto
- [x] Modelo de domínio
- [x] Configuração JSON
- [x] Validação de endpoints e posições
- [x] Provider Aave
- [x] Provider Kamino
- [x] Classificação de health factor
- [x] CLI com formato tabular
- [x] Robustez para falhas individuais


### Próximos passos

- [ ] Testes de integração reais com RPC e APIs públicas
- [ ] Suporte a mais protocolos
- [ ] Execução concorrente por provider
- [ ] Histórico de observação de health factor
- [ ] Alertas e monitoramento contínuo
- [ ] Exportação de resultados para JSON/CSV

---

## 🔒 Status

Projeto em fase de MVP funcional, com arquitetura pronta para extensão e suporte a múltiplos protocolos.

---

## 📄 Licença

Este projeto está licenciado sob a licença MIT.