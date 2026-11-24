---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - boas-vindas
  - primeiro-login
fontes:
  - "shell/mare/Welcome.qml"
  - "shell/mare/WelcomeWindow.qml"
  - "migrations/user/20261123-bem-vindo.sh"
  - "migrations/system/20261125-bem-vindo-skel.sh"
  - "shell/mare/scripts/mare-devtools"
---

# Boas-vindas

As boas-vindas são uma folha que aparece uma vez, no primeiro login de cada usuário, e apresenta o Mk em cinco passos: a ilha, a tecla Super, os projetos, as ferramentas de desenvolvimento (opcional) e o cuidado com o sistema.

## Para quê

Ensinar o mínimo para usar um desktop sem menus: o que é a ilha, que o Super abre a busca, que workspaces são projetos.

## Como funciona

- **Quando aparece sozinha** (`Welcome.firstRun`): o arquivo `<raiz>/.bem-vindo` **não** existe, a migração de usuário `20261123-bem-vindo.sh` **consta como feita**, os ajustes já foram lidos e a tela não está bloqueada. Um temporizador de 900 ms dá um instante para a ilha assentar.
- **"Já vi"** é o arquivo `.bem-vindo`. Apagar faz a folha voltar no próximo login.
- **Recolher**: abrir um painel, uma janela ou trocar de workspace no meio do passeio recolhe a folha para uma etiqueta no pé da tela, sem teclado; um clique a traz de volta no mesmo passo.
- **O 4º passo** oferece instalar as ferramentas de desenvolvimento: abre `mare-devtools` num terminal ([[Ferramentas de desenvolvimento]]).
- **Os atalhos mostrados** vêm do `Keybinds.js` mais o que o usuário trocou.
- **Abrir de novo**: busca "boas-vindas", Ajustes › Sobre, `mk boas-vindas` ou `ipc call mare welcome abrir`.
- O [[dev-shot]] cria o `.bem-vindo` na cópia de teste, a não ser com `MK_DEV_WELCOME=1`.

## Por quê assim

- **O `.bem-vindo` fica na raiz**, fora das pastas que a atualização troca: reinstalar o shell não mostra a folha de novo.
- **A porta da migração.** Quem já usava o Mk antes de as boas-vindas existirem não é novato; a migração de usuário grava o `.bem-vindo` para essa gente. Mas o `mk-update` reinicia o Maré **antes** de rodar as migrações, e logo depois pede a senha do sudo no terminal: se a folha subisse nesse meio-tempo, levaria o teclado embora. Por isso a abertura automática só vale quando a migração já consta como feita. Uma instalação nova nasce com todas marcadas.
- **Usuários criados depois**: a migração de sistema `20261125-bem-vindo-skel.sh` grava **só essa marca** no skel. Sem ela, um usuário novo nunca veria a folha sozinho, e o primeiro `mk-update` dele a daria por vista.
- **`late`**: se o Maré subiu com a sessão já andando (a assinatura do Hyprland traz a hora em que ele subiu; mais de 120 s), a folha volta **recolhida**: quem reiniciou o shell costuma estar digitando num terminal.
- **Recolher em vez de voltar por cima**: a folha convida a experimentar o Super e a busca; não pode cobrir o que a pessoa acabou de abrir.
- **Teclas só valem um instante depois de a folha subir**, e nunca por repetição: quem estava digitando quando ela apareceu não a atravessa sem ler.
- **Nunca ensina uma tecla que não existe**: por isso lê os combos de hoje.

## Onde está no código

- `shell/mare/Welcome.qml` (a lógica; os comentários contam cada caso)
- `shell/mare/WelcomeWindow.qml` (o desenho, 1083 linhas)
- `migrations/user/20261123-bem-vindo.sh`, `migrations/system/20261125-bem-vindo-skel.sh`

## Ligações

Depende de: [[Singletons]], [[Migrações]], [[Atalhos]], [[Pasta do usuário]]. Relacionado: [[Clique fora e teclado exclusivo]], [[mk-update]].
