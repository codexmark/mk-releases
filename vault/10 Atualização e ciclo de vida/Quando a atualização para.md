---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - recuperação
  - diagnóstico
fontes:
  - "docs/limites.md"
  - "docs/atualizacoes.md"
  - "bin/mk-update"
---

# Quando a atualização para

O que fazer quando o `mk-update` para no meio. A resposta curta: veja onde parou e, quase sempre, rode `mk-update` de novo.

## Para quê

Cada passo do atualizador foi escrito para aguentar repetição; esta nota diz o que esperar de cada ponto de parada.

## Como funciona

**Primeiro, onde parou.** Em Ajustes › Atualizações, a linha "parou em: …" e o botão "ver registro". Ou:

```bash
jq . ~/.local/state/mk/ultima-atualizacao.json
less -R +G "$(jq -r .log ~/.local/state/mk/ultima-atualizacao.json)"
```

| Parou em | O que costuma ser | O que fazer |
|---|---|---|
| `mk` | sem rede ou sem acesso ao canal durante o clone; `sudo` recusado | `mk-update` de novo; enquanto `<raiz>/VERSION` não muda, a versão nova continua sendo oferecida |
| `ajustes` | uma migração saiu com erro | o registro diz qual; ela continua pendente. Corrija a causa e rode de novo. Se o defeito é da migração, o conserto é um lançamento novo |
| `sistema` | o pacman falhou (espelho, chave, conflito de arquivo, disco cheio), ou parou porque precisava escolher entre provedores | leia o erro, resolva, rode de novo; na escolha de provedor, rode o `sudo pacman -Syu …` que a mensagem mostra, responda e rode `mk-update` de novo. Se parou depois do `-Sy`, o sistema está com a lista nova e os pacotes antigos: atualize antes de instalar qualquer pacote avulso |
| `aur` | um pacote do AUR não compilou | o sistema já está atualizado; tente mais tarde ou trate o pacote com o `yay` |
| `início` | parou antes do primeiro passo | `mk-update` de novo |
| (nem começa) | "já há uma atualização rodando" | há outra rodada do mesmo usuário; espere ela terminar ou feche a janela dela |

Por que repetir funciona: a troca do Mk se refaz inteira, migração feita não roda outra vez, a trava do pacman é oferecida para remoção, e o pacman continua de onde dá.

**Casos à parte** (a documentação os marca como "deduzidos do código e não ensaiados"):

- **`/usr/share/mk` sumiu** (interrompido entre apagar a pasta e renomear a nova): a cópia nova está em `/usr/share/mk.novo`; `sudo mv /usr/share/mk.novo /usr/share/mk`. Se a pasta antiga ainda existe pela metade, apague-a antes, senão o `mv` põe a nova dentro da velha.
- **Sem `/usr/share/mk`, o `mk-update` até roda** (mora em `/usr/bin` e usa o `mare-updates` de `<raiz>/shell/scripts`), mas só atualiza o Arch e o AUR: o canal padrão e as migrações são lidos de lá.
- **O Mk trocou mas o `mk-update` novo quebra**: o Arch ainda atualiza com `sudo pacman -Syu`. Se ele nem começa, roda-se o de um clone do canal já consertado (`bash <clone>/bin/mk-update`), que troca o Mk e passa a vez ao novo.
- **O Maré não voltou depois da troca**: saia da sessão e entre de novo.
- **O sistema não inicia depois da atualização**: é o caso dos snapshots ([[Voltar atrás pelo menu de boot]]), com as ressalvas de lá.

## Por quê assim

- **Idempotência em toda parte** é o que torna "rode de novo" uma resposta honesta ([[Convenções]]).
- **O resumo é gravado ao sair, com sucesso ou não** (`trap finish EXIT`), inclusive quando a janela é fechada no meio; por isso sempre há um "parou em".
- **A janela frágil é pequena, mas existe**: entre o `rm -rf /usr/share/mk` e o `mv` da pasta nova. Como a cópia para `.novo` vem antes, o trecho perigoso é só um apagar e um renomear, e a pasta nova está inteira ao lado.
- **Não há rollback do próprio Mk**: a única volta é o snapshot do sistema, que não cobre a home.

## Onde está no código

`docs/atualizacoes.md` (seção "Quando a atualização para no meio"); `bin/mk-update` (`write_summary`, `finish`, os `trap`).

## Ligações

Depende de: [[mk-update]], [[Migrações]]. Relacionado: [[Voltar atrás pelo menu de boot]], [[Atualizações na ilha e nos Ajustes]], [[Limitações conhecidas]].
