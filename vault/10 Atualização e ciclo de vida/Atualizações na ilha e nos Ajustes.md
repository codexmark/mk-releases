---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - maré
  - ajustes
fontes:
  - "shell/mare/Updates.qml"
  - "shell/mare/scripts/mare-updates"
  - "shell/mare/SettingsUpdatesLast.qml"
  - "docs/atualizacoes.md"
  - "shell/mare/SettingsSchema.js"
---

# Atualizações na ilha e nos Ajustes

O Maré procura atualizações sozinho, sem sudo, e mostra o resultado em dois lugares: um ícone discreto na ilha, só quando importa, e a página Atualizações dos Ajustes, com tudo.

## Para quê

Saber que há o que atualizar sem ser incomodado por isso todo dia.

## Como funciona

**A procura** (`scripts/mare-updates`, JSON):

- pacotes do Arch: `checkupdates`, numa base temporária (não mexe no sistema);
- AUR: `yay -Qua`, com prazo de 30 s;
- o Mk: a versão instalada contra o `VERSION` do canal ([[Canal de lançamentos]]);
- migrações pendentes;
- o resumo da última rodada (`ultima-atualizacao.json`), a data da última atualização e se falta reiniciar.

**Quando roda** (`Updates.qml`):

- 2 minutos depois do login e depois a cada 3 horas, se "Procurar sozinho" (`updates.auto`) estiver ligado;
- ao abrir a página, se a última procura tem mais de 10 minutos;
- no botão "procurar agora";
- sempre que o `ultima-atualizacao.json` muda, ou quando o terminal aberto pelo "atualizar tudo" fecha.

**A página** mostra o número de atualizações (o Mk conta 1, mais as migrações, mais os pacotes), os 12 primeiros pacotes, a linha "última vez: … tudo certo" ou "parou em: …", o botão "ver registro", "reinicie para o kernel novo" e, havendo Mk novo, "o que chega" (as linhas do `NOVIDADES`).

**A ilha** só mostra o ícone com o número (se `updates.island` estiver ligado) quando:

- há versão nova do Mk (o ícone fica em cobre), ou
- há migração pendente, ou
- há pacotes esperando **e** a última atualização foi há 7 dias ou mais, ou nunca.

## Por quê assim

- **"No Arch há pacote novo todo dia"**: um contador que acende sempre não informa nada. A ilha só acende pelo que pede ação.
- **Sem sudo e sem baixar nada**: a procura roda como o usuário, e quem aplica é o `mk-update`, num terminal, "porque o pacman pede a senha".
- **Primeira procura só 2 minutos depois do login**: "para não pesar na entrada".
- **A data da última atualização** é a mais nova entre o `applied` do resumo e o último "starting full system upgrade" do `/var/log/pacman.log`: um `pacman -Syu` feito à mão também conta.
- **Vigiar o `ultima-atualizacao.json`**: o `mk-update` rodado num terminal qualquer atualiza a página e a ilha sem esperar a próxima procura.
- **Erros aparecem na página**: sem internet ou espelho fora, falta do `pacman-contrib`, ou "sem acesso ao canal do Mk".
- **O botão "ver registro"** some quando o arquivo já saiu dos 5 últimos.

## Onde está no código

`shell/mare/Updates.qml` (a propriedade `badge` é a regra da ilha), `scripts/mare-updates`, `SettingsUpdatesLast.qml`.

## Ligações

Depende de: [[mk-update]], [[Canal de lançamentos]], [[Pacman e AUR]], [[Singletons]]. Aparece em: [[Ilha]], [[Páginas dos Ajustes]].
