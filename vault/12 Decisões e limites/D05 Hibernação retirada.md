---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - energia
  - hibernação
fontes:
  - "bin/mk-hibernate"
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/scripts/mare-sleep"
  - "docs/limites.md"
---

# D05 Hibernação retirada

O Mk não hiberna. A opção entrou e foi retirada em commits do mesmo dia; ficou só o caminho para desfazer.

## Contexto

Por um pedido (a escolha de como a máquina dorme), Ajustes › Energia ganhou "Ao dormir" com três opções: suspender, hibernar, não dormir. Para a hibernação, `bin/mk-hibernate setup` preparava a máquina: um subvolume `/swap` com um swapfile, a linha no `fstab` e `resume=` na linha do kernel, com cópias de segurança e um arquivo de estado.

No teste, a sessão **gravava e não voltava** no boot seguinte (`systemd-hibernate-resume: Unable to resume from device`). A causa não foi achada.

## Escolha

No commit seguinte:

- sai a opção "hibernar" dos Ajustes, do painel de sessão e da tela de bloqueio;
- `mk-hibernate setup` passa a recusar;
- `mk-hibernate remove` e `status` ficam, para quem preparou poder desfazer; o `remove` só desfaz o que está anotado no arquivo de estado;
- um `power.sleep: "hibernar"` que sobrou no `ajustes.json` passa a valer como "não dormir";
- o instalador não instala mais o comando.

## Alternativas

- **Deixar a opção com um aviso** *(não discutida no repositório)*.
- **Investigar até achar a causa antes de retirar** *(não discutida no repositório)*.

O que está registrado é a retirada e o sintoma; o repositório não descreve uma investigação posterior.

## Consequência

- Dois modos de dormir: suspender ou não dormir ([[Energia e suspensão]]).
- O Mk instalado não tem swap ([[Swap e hibernação]]).
- Numa máquina cuja placa não volta da suspensão (a Navi 10), a única opção segura é "não dormir".
- `bin/mk-hibernate` tem 242 linhas para um recurso que não existe: é o preço de não deixar ninguém com um swapfile e um `resume=` órfãos.

> [!note] Onde foi o teste
> A mensagem do commit e `docs/limites.md` dizem "na VM"; o cabeçalho do `bin/mk-hibernate` diz "na máquina de teste".

## Onde está no código

`bin/mk-hibernate` (o cabeçalho), `shell/mare/SettingsSchema.js` (`sleepMode`, `sleepStep`), `shell/mare/scripts/mare-sleep`.

## Ligações

Afeta: [[Swap e hibernação]], [[Energia e suspensão]], [[Ociosidade e bloqueio]]. Índice: [[Registro de decisões]]. Limite: [[Limitações conhecidas]].
