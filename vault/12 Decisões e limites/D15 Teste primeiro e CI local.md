---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - testes
  - qualidade
fontes:
  - "tests/README.md"
  - "tests/hooks/pre-push"
  - "tests/lib.sh"
  - "bin/mk-release"
  - "docs/desenvolvimento.md"
  - "docs/limites.md"
---

# D15 Teste primeiro e CI local

Toda mudança de comportamento começa por um teste que falha, e os testes rodam **na máquina de quem desenvolve**: no `git push` e antes de cada lançamento. Não há CI na nuvem.

## Contexto

Até aqui o projeto se apoiava em três checagens pontuais, no `dev-shot` e na VM. Os defeitos registrados nesta vault mostram o limite disso: um `mk.bash` lançado que derrubava todo terminal, um instalador que descartava o menu de boot por um detalhe de texto, migrações que instalavam pacote antes do `-Syu`. Quase todos são erros de lógica em scripts, que não precisam de tela nem de disco para aparecer. E o Mk publica direto para máquinas instaladas: o que passa pelo `mk-release` chega a todo mundo, sem "desfazer".

## Escolha

- **Teste primeiro**: o caso entra em `tests/` antes do código; um defeito achado em uso vira primeiro um caso que o reproduz. "Código novo sem teste não entra; código antigo que for tocado ganha o teste que faltava."
- **Uma caixa de areia com guardas**: os comandos que mexem na máquina ou saem para a rede falham se chamados sem `stub`; o código de produção aceita raízes por variável (`MK_ROOT`, `SYS`, `PROC`, `ETC`, `live_root`…).
- **CI local, em dois portões**: o gancho `tests/hooks/pre-push` roda `tests/run` antes de cada `git push` (ligado uma vez por clone); o `bin/mk-release` roda `tests/run` antes de publicar e não lança se algo falhar.
- **Sem CI na nuvem**: "não há CI na nuvem: tudo roda aqui".

## Alternativas

- **CI na nuvem** (um serviço rodando os testes a cada push): descartada. O `tests/README.md` registra só o fato. O motivo dado por quem mantém, ao encomendar esta revisão da vault, é o plano gratuito do GitHub; ele **não está escrito no repositório**.
- **Só as checagens pontuais e a VM** (o estado anterior): é o que os defeitos acima mostraram ser pouco.
- **Testes que sobem a sessão de verdade** *(não discutida no repositório)*: a suíte deixa isso, de propósito, para o `dev-shot` e a VM.

## Consequência

- A suíte precisa rodar em segundos, sem root, sem rede e sem sessão gráfica; é isso que a torna viável como portão de `push`.
- O portão do `push` é **opcional por clone** (depende do `git config core.hooksPath`) e pode ser pulado com `git push --no-verify`. O portão que não se pula é o do `mk-release`.
- Não há um registro externo de que os testes passaram: a garantia é a de quem lançou, na máquina dele.
- O código de produção ganhou emendas para ser testável (as variáveis de raiz, os blocos `// --- puro ---` no QML, o instalador carregável como biblioteca).
- O lint passou a impor convenções que antes eram só recomendação, como o `rm -rf` com `${VAR:?}`.
- O que os testes não veem continua existindo e está listado em `docs/limites.md`: o desenho dos painéis, o foco do teclado, o Hyprland de verdade, a instalação num disco.

## Onde está no código

`tests/README.md` ("A regra: teste primeiro", "Onde isto roda"); `tests/hooks/pre-push`; `bin/mk-release` (a linha do `tests/run`).

## Ligações

Afeta: [[Testes e TDD]], [[mk-release]], [[Trabalhar no repositório]], [[Checagens]], [[Convenções]]. Índice: [[Registro de decisões]].
