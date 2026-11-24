---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - monitores
  - compositor
fontes:
  - "shell/mare/scripts/mare-monitors"
  - "shell/mare/Monitors.qml"
  - "config/hypr/hyprland.lua"
---

# D13 Monitores por arquivo com vigia

Uma mudança de monitores é gravada num arquivo, aplicada com `hyprctl reload` e vigiada por um processo fora do shell, que a desfaz em 15 segundos se ninguém confirmar.

## Contexto

Configurar telas é o único ajuste que pode deixar a pessoa **sem imagem** para desfazê-lo: uma resolução que o monitor não aceita, a única tela desligada, uma taxa fora dos limites. Os outros ajustes do Mk são aplicados por `hyprctl eval` e desfeitos se o Hyprland recusar; aqui o Hyprland pode aceitar e a tela ficar preta mesmo assim.

## Escolha

- O texto Lua vai para `<raiz>/monitores.lua`, e o Hyprland **recarrega a config**.
- Antes, o arquivo atual é guardado em `monitores.lua.anterior`, com a sessão na primeira linha.
- Um **vigia** (o próprio script, em segundo plano, fora do Maré) espera a resposta: `keep` apaga o anterior; `revert`, o fim da contagem, a queda do Hyprland ou um sinal de término devolvem o arquivo e recarregam.
- Se nem o vigia rodou (queda de energia), o `hyprland.lua` vê a sobra no login seguinte e volta sozinho ([[Recuperação de monitores]]).

## Alternativas

- **`hyprctl eval` com a regra nova**, como os outros ajustes: "por eval o Hyprland 0.57 só reaplica a regra em telas ligadas e sem espelho; uma tela desligada não religa e um espelho não se desfaz".
- **A contagem dentro do Maré**: se o Maré morrer junto, ou a pessoa não enxergar nada, ninguém desfaz. O vigia fora do shell cobre os dois casos.
- **Sem contagem, com um botão "desfazer"** *(não discutida no repositório)*: não serviria a quem ficou sem imagem.

## Consequência

- Três camadas de volta: contagem, sinal de término, próximo login.
- Recarregar a config inteira é mais pesado que um `eval`, e é por isso que o `hyprland.lua` precisa distinguir "teste desta sessão" de "sobra de outra".
- Todas as escritas são atômicas (`mktemp` + `mv`).
- O mesmo mecanismo serve às taxas calculadas por modeline: um modeline que o monitor não aceita volta em 15 s.

## Onde está no código

`shell/mare/scripts/mare-monitors` (o cabeçalho explica o "por que arquivo + reload"); `shell/mare/Monitors.qml`; `config/hypr/hyprland.lua`.

## Ligações

Afeta: [[Monitores]], [[Recuperação de monitores]], [[Ordem de carga]]. Índice: [[Registro de decisões]].
