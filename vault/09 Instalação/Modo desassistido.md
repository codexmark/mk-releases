---
camada: "09 Instalação"
tags:
  - mk
  - instalação
  - instalador
  - testes
fontes:
  - "tests/installer/mk-install.respostas.test.sh"
  - "installer/mk-install"
  - "installer/answers.example"
  - "installer/README.md"
---

# Modo desassistido

`mk-install --answers <arquivo>` instala sem perguntar nada, lendo as respostas de um arquivo `CHAVE=valor`. Foi feito para testes em VM.

## Para quê

Repetir a instalação inteira numa VM sem digitar, quando se mexe no instalador, no boot ou numa migração.

## Como funciona

| Chave | Valor |
|---|---|
| `DISK` | o disco inteiro, por exemplo `/dev/vda` |
| `CONFIRM` | `formatar` (ou o mesmo caminho de `DISK`); sem isso, para antes de apagar |
| `HOSTNAME`, `USERNAME`, `PASSWORD` | como nas perguntas |
| `TIMEZONE`, `KEYMAP`, `LOCALE` | os três valores de região |
| `HARDWARE` | perfis separados por espaço; vazio para nenhum |
| `APPS`, `ENCRYPT` | `sim` ou `nao`, exatamente assim (`yes`, `true` e outros são recusados) |
| `POWEROFF` | `sim` desliga a máquina no fim |

- O arquivo é lido linha a linha, com uma expressão regular (`^[A-Z_]+=…`): **nada é executado**. Uma chave desconhecida ou uma linha fora do formato para o instalador.
- Não há tela de Wi-Fi: a rede já precisa estar no ar.
- As respostas passam pelas mesmas validações, e a revisão é mostrada antes de formatar.
- No fim não oferece reiniciar: desliga se `POWEROFF=sim`, senão volta ao prompt.
- `MK_SRC` e `MK_AUR_REPO` permitem rodar o instalador de fora de `/usr/share/mk`.

## Por quê assim

- **Lido como dado, não com `source`**: um arquivo de respostas é uma entrada; carregá-lo como shell executaria o que estivesse nele, como root, antes de formatar um disco.
- **`CONFIRM` obrigatório**: mesmo sem perguntas, apagar um disco exige uma autorização explícita no arquivo.
- **Sem padrões de região**: neste modo é preciso preencher todas as chaves do exemplo. Com `LOCALE`, `KEYMAP` ou `TIMEZONE` vazios, ou com um valor que não existe no live, o instalador para antes de tocar no disco.
- **`sim`/`nao` ao pé da letra**: "um 'yes' aqui valeria 'nao' em silêncio: o disco ficaria sem LUKS".
- Foi este modo que revelou a pergunta do `limine-mkinitcpio-hook` que travava a instalação ([[Initramfs]]).

> [!warning] Senha em texto puro
> A senha fica em texto puro no arquivo: não use uma senha de verdade e não deixe o arquivo em lugar compartilhado.

## Onde está no código

`installer/mk-install`: `load_answers`, `confirm_wipe`, o fim de `main`; `installer/answers.example`.

## Ligações

Depende de: [[Instalador passo a passo]]. Usado por: [[VM de teste]], [[Testes e TDD]].
