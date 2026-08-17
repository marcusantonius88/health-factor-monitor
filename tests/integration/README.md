# Teste E2E de integração

Este cenário valida o fluxo principal do Health Factor Monitor, cobrindo os passos abaixo:

1. Carregamento de configuração JSON.
2. Validação dos dados de rede e de posições.
3. Inicialização dos providers reais do Aave e Kamino com servidores HTTP locais.
4. Execução do serviço de verificação em `CheckAll`.
5. Renderização do resultado final em tabela via `FormatResults`.

## Objetivo

Garantir que a aplicação consegue:

- ler uma configuração de posições válidas;
- mapear cada posição para o provider correto;
- consultar os dois protocolos em sequência;
- devolver Health Factors positivos;
- classificar os resultados como `safe`;
- manter a saída por terminal consistente e legível.

## Arquivos envolvidos

- `tests/integration/e2e_test.go` — cenário end-to-end.
- `tests/integration/testdata/valid-config.json` — configuração usada para o teste.

## Resultado esperado

O teste deve concluir com sucesso quando:

- o arquivo de configuração é aceito pelo `config.Reader`;
- os dois providers respondem com dados válidos;
- `CheckAll` retorna exatamente 2 resultados;
- cada resultado contém um `HealthFactor` e a classificação `safe`;
- a formatação final inclui os aliases `aave-main` e `kamino-main`.
