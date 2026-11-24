# MkCalc: o contrato entre o motor e a janela

O MkCalc é a calculadora do Mk (`mkcalc`): uma janela pequena, toda operável
pelo teclado, que mostra o resultado enquanto se digita e guarda as contas
confirmadas. Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mkcalc-motor`), que faz a
  conta, com aritmética exata, e guarda o histórico;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que mostra o que o
  motor diz. O que cada tecla faz com o texto da expressão (apagar, trocar o
  sinal, continuar do resultado, arrumar o que foi colado) mora em
  `qml/Calc.js`, testado no node. **A janela nunca faz conta**: todo número
  que aparece no visor veio do motor.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. O que falha
  escreve `{"erro": "frase em português"}`. A janela mostra a frase.
- Códigos de saída: **0** deu certo; **1** erro de uso ou de disco
  (subcomando que não existe, histórico que não deu para gravar); **2** a
  conta foi recusada (expressão que não fecha, divisão por zero, número grande
  demais). No modo `servir` o processo só sai quando o stdin fecha (0).
- O motor não roda nenhum outro programa, não usa a rede e só escreve num
  arquivo: o do histórico.
- Variáveis de ambiente do motor: `XDG_STATE_HOME` (onde fica o histórico;
  padrão `~/.local/state`) e `MKCALC_HISTORICO_LIMITE` (quantas contas o
  histórico guarda; padrão 200).

## A conta

### `mkcalc-motor calcular [--guardar] <expressão>`

```json
{"expressao": "200 + 10%", "resultado": "220", "cru": "220", "aproximado": false}
```

- **`expressao`**: a expressão como o motor a leu, arrumada: operadores `+ -
  × ÷ ^ √ %`, um espaço de cada lado de `+ - × ÷` (o `^` fica colado: `2^-3`),
  o `×` subentendido escrito (`2(3)` vira `2 × (3)`), vírgula decimal e
  **sem** separador de milhar. É o texto que a janela mostra depois
  do `=`, e o motor lê de volta do mesmo jeito.
- **`resultado`**: para mostrar, em pt-BR: milhar com ponto, decimal com
  vírgula, a parte decimal sem agrupamento (`1.234.567,891234`).
- **`cru`**: para copiar e para voltar para a expressão: sem separador de
  milhar e **com vírgula decimal** (`1234567,891234`). A vírgula, e não o
  ponto, porque é o que uma planilha em português espera e porque `1,234`
  colado de volta continua sendo um vírgula dois três quatro.
- O sinal de menos é sempre o hífen (`-`), nos três campos.
- **`aproximado`**: `false` quando `resultado` é o valor exato, com todos os
  algarismos. `true` quando ele foi arredondado (veja "Exato e aproximado").
- Com `--guardar` a conta entra no histórico, e a resposta traz também
  `"itens"` (o histórico inteiro, já com ela). Se não deu para gravar, a conta
  vem do mesmo jeito, com `"aviso": "frase"` e sem `itens`.

Quando a conta não sai:

```json
{"erro": "divisão por zero", "incompleta": false}
```

**`incompleta`** é `true` quando a expressão só não terminou de ser escrita
(`2 +`, `(3 × 4`, `√`, vazia): a janela não mostra isso como erro na prévia,
só espera a próxima tecla. As frases:

| Frase | Quando |
|---|---|
| `expressão vazia` | nada para calcular (incompleta) |
| `expressão incompleta` | acabou esperando um número: `2 +`, `()`, `√` (incompleta) |
| `parêntese sem par` | sobrou `(` aberto (incompleta) ou `)` fechando nada |
| `divisão por zero` | `1 ÷ 0`, `0^-1` |
| `raiz de número negativo` | `√-4`, `(-8)^0,5` |
| `número mal formado: «1.2.3»` | separadores que não fazem um número |
| `operador fora de lugar: «×»` | `2 + × 3`, `× 3` |
| `falta um operador antes de «3»` | `2 3`, `(1 + 2) 3` |
| `não entendi «abc»` | um caractere ou palavra que não é de conta |
| `expoente grande demais` | o resultado passaria do limite: `9^9^9` |
| `número grande demais` | o resultado passou do limite |
| `expoente com casas demais` | expoente fracionário com denominador acima de 1.000 |
| `expressão longa demais` | mais de 5.000 caracteres |

### O que a expressão aceita

- Números; `+`, `-`, `×`, `÷`, `^`; parênteses; menos (e mais) na frente de um
  número ou parêntese; `%`; raiz quadrada.
- Sinônimos: `*`, `x`, `X` e `·` são `×`; `/` e `:` são `÷`; `**` é `^`; `−` e
  `–` são `-`; `raiz` e `sqrt` são `√` (`raiz(9)`, `raiz 9`, `√9`). Espaços
  são ignorados.
- Precedência, da mais forte para a mais fraca: `√` e `%` (grudam no número
  ou no parêntese vizinho), `^` (associa à direita: `2^3^2` é `2^9`), o menos
  na frente (`-2^2` é `-4`; no expoente pode: `2^-2`), `×` e `÷` (da esquerda
  para a direita), `+` e `-`.
- Um parêntese ou uma raiz logo depois de um número, de `)` ou de `%`
  multiplica: `2(3 + 4)` é `2 × (3 + 4)`, `2√9` é `2 × √9`.
- `0^0` é `1`.
- Expoente fracionário vale quando o denominador não passa de 1.000:
  `8^(1 ÷ 3)` é `2`, `2^0,5` é `√2`.

### Percentual, como numa calculadora de mesa

- Com `+` ou `-` pendente, quando o que vem depois do sinal é **só** um número
  (ou parêntese) com `%`, ele é a porcentagem do total até ali:
  `200 + 10%` é `220`, `200 - 10%` é `180`, `100 + 50 + 10%` é `165`,
  `200 + (5 + 5)%` é `220`.
- Em todo outro lugar (com `×`, `÷`, `^`, sozinho, dentro de parênteses
  próprios, começando a conta) o `%` divide por 100: `200 × 10%` é `20`,
  `50%` é `0,5`, `200 + 10% × 2` é `200,2`, `200 + (10%)` é `200,1`.

### Vírgula, ponto e separador de milhar

Um número é uma sequência de algarismos, pontos e vírgulas. A regra é uma só,
e é a mesma no motor e no `Calc.js` (os dois são testados com os mesmos casos,
`src/internal/conta/testdata/numeros.json`):

1. **Um separador só, uma vez só, é o decimal**: `1,5` e `1.5` são um e meio,
   `,5` é `0,5` e `5,` é `5`. A vírgula é sempre decimal (`1,234` é um
   vírgula dois três quatro). O ponto tem uma exceção, que é como se escreve
   em português: **com exatamente três algarismos depois e de um a três antes,
   sem zero na frente, ele é milhar**. `1.234` é 1234, `12.345` é 12345 e
   `999.999` é 999999; `1.5`, `3.14`, `0.123`, `1.2345` e `1234.567` são
   decimais.
2. **O mesmo separador repetido é milhar**, e os grupos têm de estar certos
   (um a três algarismos, depois sempre três): `1.234.567` e `1,234,567` são
   1234567; `1.23.456` é número mal formado.
3. **Com os dois, o último que aparece é o decimal** e o outro é milhar (com
   os grupos certos): `1.234,56` e `1,234.56` são 1234,56; `1,234.567,8` é
   número mal formado.

A janela não tem essa dúvida ao digitar: o ponto do teclado vira vírgula na
hora (digitar `1.234` escreve `1,234`). A regra vale para o que é colado, que
a janela arruma antes de pôr na expressão, e para o que se escreve direto no
`mkcalc-motor calcular`. O `resultado` de uma conta (`1.234`, `1.234,5`)
colado de volta é lido como o número que era.

### Exato e aproximado

A conta é feita com frações exatas (`math/big`): `0,1 + 0,2` é `0,3`,
`1 ÷ 3 × 3` é `1`, e um número grande não perde algarismo.

- O resultado sai **inteiro e exato** (`aproximado: false`) quando é um
  decimal finito com até **40 casas**.
- Senão sai **arredondado para 12 casas decimais** (`aproximado: true`), o
  meio para longe do zero: `2 ÷ 3` é `0,666666666667`. Num número menor que 1
  em módulo as 12 casas contam a partir do primeiro algarismo que não é zero
  (`1 ÷ 3000` é `0,000333333333333`), para um número pequeno não virar zero.
- A raiz que não é exata (`√2`) é calculada com 40 casas e marca tudo que vem
  depois como aproximado: `√2 × √2` é `2`, aproximado. A raiz exata é exata:
  `√16` é `4`, `√0,25` é `0,5`, `√(4 ÷ 9) × 3` é `2`. Numa conta aproximada, o
  que sobra abaixo de 10⁻³⁰ é ruído do arredondamento e vira `0`.

### Limites (para a janela nunca travar)

- Nenhum número da conta passa de 10.240 bits (cerca de **3.000
  algarismos**), nem no numerador nem no denominador. A potência é recusada
  **antes** de ser calculada, pelo tamanho que teria: `9^9^9` responde
  `expoente grande demais` na hora.
- A expressão tem até 5.000 caracteres no motor e 3.500 na janela: o maior
  resultado possível cabe de volta na expressão, para continuar a conta. Por
  isso um resultado aproximado nunca sai com mais de 3.082 casas decimais (o
  que cabe em 10.240 bits).

## O histórico

Fica em `${XDG_STATE_HOME:-~/.local/state}/mk/mkcalc/historico.json`:

```json
{"versao": 1, "itens": [
  {"id": 2, "expressao": "200 + 10%", "resultado": "220", "cru": "220", "aproximado": false, "quando_s": 1795000000}
]}
```

- O mais novo vem primeiro. `id` é um inteiro que só cresce; `quando_s` é a
  hora em segundos desde 1970.
- Guarda as últimas 200 contas (`MKCALC_HISTORICO_LIMITE`); a mais velha sai.
- Confirmar de novo a conta que já é a mais nova só atualiza a hora.
- A escrita é atômica (arquivo temporário na mesma pasta e `rename`).
- Um arquivo que não dá para ler (corrompido, de outra versão) não derruba
  nada: o histórico começa vazio e a próxima conta o regrava.

### `mkcalc-motor historico`, `mkcalc-motor apagar <id>`, `mkcalc-motor limpar`

Os três respondem o histórico como ficou:

```json
{"itens": [{"id": 2, "expressao": "200 + 10%", "resultado": "220", "cru": "220", "aproximado": false, "quando_s": 1795000000}]}
```

`apagar` com um `id` que não existe responde
`{"erro": "essa conta não está mais no histórico"}` (saída 1).

## O modo servidor

### `mkcalc-motor servir`

Para a janela calcular a prévia a cada tecla sem abrir um processo por conta.
Lê **um pedido JSON por linha** no stdin e responde **uma linha JSON** por
pedido no stdout, na ordem. Toda resposta repete o `id` do pedido (um número
ou um texto; `null` se o pedido não trouxe ou não deu para ler), que é como a
janela casa as duas.

| Pedido | Resposta |
|---|---|
| `{"id": 1, "op": "calcular", "expr": "2+2"}` | a de `calcular`, mais `id` |
| `{"id": 2, "op": "calcular", "expr": "2+2", "guardar": true}` | a de `calcular --guardar`, mais `id` |
| `{"id": 3, "op": "historico"}` | `{"id": 3, "itens": […]}` |
| `{"id": 4, "op": "apagar", "item": 2}` | `{"id": 4, "itens": […]}` |
| `{"id": 5, "op": "limpar"}` | `{"id": 5, "itens": []}` |

Um erro é `{"id": …, "erro": "frase"}` (com `incompleta` quando é de conta) e
o servidor continua de pé: linha que não é JSON, `op` que não existe, conta
recusada. Linha vazia é ignorada. Uma linha com mais de 64 KiB encerra o
servidor (saída 1).

## A janela

- Título e classe: **`MkCalc`** (é pelo título que a regra do Hyprland a faz
  flutuar e que o lançador a acha para focar).
- `bin/mkcalc [expressão]` abre a janela (já com a expressão, se veio uma);
  sem expressão e com a janela aberta, só a traz para a frente.
- `MKCALC_MOTOR` troca o binário; `MKCALC_EXEMPLO=pasta` faz a janela usar
  respostas gravadas (`tests/apps/mkcalc-exemplo/`) em vez do motor:
  `calcular.json` é um objeto que leva da expressão pedida à resposta (a chave
  `"*"` vale para as outras), e `historico.json`, `apagar.json` e
  `limpar.json` são a resposta de cada pedido.
- A janela fala com o motor só pelo `servir`, com os pedidos da tabela acima.

### O teclado

| Tecla | Faz |
|---|---|
| `0`-`9` | o algarismo (depois de `=`, começa conta nova) |
| `,` `.` | a vírgula decimal |
| `+` `-` `*` `x` `/` `:` `^` | o operador (depois de `=`, continua do resultado) |
| `%` `(` `)` | eles mesmos |
| `R` | `√` |
| `S` | troca o sinal do último número |
| `Enter` `=` | calcula e guarda no histórico |
| `Backspace` | apaga o último caractere (depois de `=`, volta a editar a expressão) |
| `Esc` | fecha o histórico se está aberto; senão limpa; com tudo vazio, fecha a janela |
| `C` `Delete` | limpa |
| `Ctrl+C` | copia o resultado (o `cru`) |
| `Ctrl+V` | cola um número ou uma expressão |
| `Ctrl+H` | abre e fecha o histórico |

O teclado numérico vale com e sem NumLock: sem ele as teclas chegam como
`Insert`, `End`, setas, `Home`, `PageUp`… e a janela as lê como `0`-`9` e a
vírgula.
