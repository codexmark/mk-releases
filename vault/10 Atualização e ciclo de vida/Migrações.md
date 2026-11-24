---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - migrações
fontes:
  - "tests/migrations/mig.sh"
  - "tests/migrations/regras.test.sh"
  - "migrations/README.md"
  - "bin/mk-update"
  - "installer/mk-install"
  - "bin/new-user"
  - "bin/try-session"
---

# Migrações

Uma migração é um script que o `mk-update` roda **uma vez** para levar a um Mk já instalado uma mudança que fica fora de `~/.config/mk`: em `/etc`, em `/usr/share`, no boot, no `~/.bashrc`.

## Para quê

O `mk-update` só troca `/usr/share/mk` e as cinco pastas de `<raiz>`. Tudo o mais que o instalador faz (tema do boot, tema do login, fontes, skel, serviços) precisa de outro caminho para chegar a quem instalou antes. As migrações são esse caminho.

## Como funciona

| Tipo | Roda | Registro |
|---|---|---|
| `migrations/system/*.sh` | uma vez por máquina, como root (`sudo env MK_SHARE=… bash`) | `/var/lib/mk/migrations/<nome>` |
| `migrations/user/*.sh` | uma vez por usuário, como ele mesmo | `~/.local/state/mk/migrations/<nome>` |

- **Nome**: `AAAAMMDD-assunto.sh`. Rodam em ordem de nome: todas as de `system/`, depois as de `user/`.
- **O nome é a marca de "feita"**: um arquivo vazio com o mesmo nome na pasta de registro. Renomear uma migração já lançada faz ela rodar de novo em todo mundo.
- **Código de saída**: 0 marca como feita, para sempre. Qualquer outro valor para o `mk-update` ali mesmo ("parou em: ajustes"), não marca, e o que vinha depois não roda nessa vez.
- **`MK_SHARE`** é a única variável que o `mk-update` passa. Nas de sistema, `$HOME` e `$USER` não são os de quem atualiza.
- **`MK_ROOT`**: numa migração de sistema, todo caminho do sistema (`/etc`, `/usr`, `/var`, `/boot`) passa por uma raiz, `R=${MK_ROOT:-}`: `"$R/etc/skel"`, não `/etc/skel`. O `mk-update` não passa essa variável, então `$R` é vazio e os caminhos são os de sempre; quem a usa são os testes, que rodam a migração numa raiz falsa. Comandos que mexem no sistema por conta própria (`systemctl`, `mkinitcpio`, `limine-update`) não recebem a raiz: nos testes são dublês.
- **Teste**: `tests/migrations/system/<nome>.test.sh` (ou `user/`), escrito antes da migração: monta o "antes", roda e confere o "depois"; roda de novo e nada muda; num sistema sem nada a fazer, não quebra.
- **Sem perguntas**: rodam no meio da atualização.
- **Instalação nova**: o instalador marca todas como feitas, sem rodar nenhuma.

### A regra `# pacotes:`

Migração de sistema **não instala pacote**. Quem precisa de um declara no cabeçalho:

```bash
# pacotes: cups ipp-usb jq
```

O `mk-update` instala esses junto com a atualização do sistema e roda a migração **depois** dela. Regras:

- só em `system/` (em `user/` a linha é ignorada);
- só nomes dos repositórios oficiais; AUR não serve;
- só é aceito nome que casa com `^[a-z0-9@._+-]+$`; o resto é descartado em silêncio;
- nome que não existe mais nos repositórios sai do pedido com um aviso, e a migração roda mesmo assim;
- a migração tem de **aguentar o pacote faltar**: avisa e sai com 0.

## Por quê assim

- **Por que migração não instala pacote**: elas rodam antes do `pacman -Syu`, com a lista de pacotes velha; instalar sobre ela falha, e um download que falha travaria toda atualização seguinte. Detalhes em [[D03 Migração não instala pacote]].
- **As regras são conferidas por teste**: `tests/migrations/regras.test.sh` passa por todas as migrações e reprova a que instala pacote no corpo, lê o Mk por caminho fixo, não usa a raiz `$R`, faz pergunta, mexe em home sendo de sistema, ou não tem o próprio teste. As quatro migrações mais antigas que chamavam `pacman -S` direto foram convertidas para a linha `# pacotes:`.
- **Idempotência obrigatória**: a marca só é gravada depois que o script termina; uma rodada interrompida no meio repete a migração inteira.
- **Sair com erro só quando tentar de novo faz sentido**: para o que não tem conserto automático, avisar em stderr e sair com 0. Uma migração que sai com erro para a rodada ali, e o que vinha depois (pacotes, AUR) não roda nessa vez.
- **O par no instalador**: como a instalação nova marca tudo como feito sem rodar, o instalador precisa fazer a mesma mudança. É a regra "sempre os dois" ([[Convenções]]).
- **O par no skel**: uma migração de usuário só alcança quem roda o `mk-update`. Para os usuários criados depois, é preciso também uma de sistema que ajuste `/etc/skel`, que numa máquina instalada só muda por migração.
- **As marcas de usuário no skel são as do dia da instalação**: uma home nova ainda recebe, no primeiro `mk-update`, as migrações de usuário lançadas depois. Se o skel já entrega a mudança pronta, a de sistema grava a marca no skel (o caso das boas-vindas).

## Onde está no código

- `migrations/README.md` (o guia para escrever uma)
- `bin/mk-update`: `pending_migrations`, `migration_packages`, `run_migration`
- `installer/mk-install`: o laço que marca todas em `install_desktop`

## Ligações

Depende de: [[mk-update]], [[Pacman e AUR]]. Lista: [[Catálogo de migrações]]. Relacionado: [[Instalador passo a passo]], [[Usuários e sudo]], [[Boas-vindas]], [[AUR na instalação]].
