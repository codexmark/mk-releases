---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - janelas
fontes:
  - "config/hypr/mare/rules.lua"
---

# Regras de janela

O Mk tem quatro regras de janela, todas em `rules.lua`: utilitários flutuam, diálogos de arquivo flutuam, os apps do Mk flutuam, e nenhum app se maximiza por conta própria.

## Para quê

Num gerenciador em mosaico, um diálogo pequeno que entra no mosaico empurra todas as outras janelas. As regras tiram do mosaico o que é passageiro.

## Como funciona

| Regra | Casa com | Efeito |
|---|---|---|
| `float-utilities` | classe `pavucontrol`, `nm-connection-editor`, `blueman-manager`, `xdg-desktop-portal-gtk`, `org.gnome.Calculator` | flutua e centraliza |
| `float-file-dialogs` | título "Open File", "Save File", "Save As", "Choose Files", "Open Folder", "Abrir arquivo", "Salvar arquivo" | flutua e centraliza |
| `float-mkpulso` | título exatamente `MkPulso`, `MkPacotes`, `MkInstalar`, `MkComprimir`, `MkCalc` ou `MkSom` (o MkCorte e o MkEscrita são janelas de trabalho e entram lado a lado) | flutua e centraliza |
| `no-maximize-requests` | qualquer classe | `suppress_event = "maximize"` |

São chamadas a `hl.window_rule({ name, match, … })`.

## Por quê assim

- O arquivo traz um comentário só: "diálogos e utilitários flutuam e centralizam". O resto do comportamento das janelas vem do layout e dos [[Atalhos]].
- Diálogos de arquivo são reconhecidos **pelo título**, em inglês e em português; os utilitários, pela classe.
- `suppress_event = "maximize"` para todas as janelas: o efeito é ignorar o pedido de um app para se maximizar. O arquivo não comenta além do nome da regra (`no-maximize-requests`); **motivo não registrado**.

> [!note] O que não está aqui
> As camadas do Maré (ilha, painéis, bloqueio) não são janelas: são *layer surfaces* e não passam por estas regras. Veja [[Maré]].

- **Os apps do Mk flutuam**: são "utilitários que se abrem por cima do trabalho, no tamanho deles. Lado a lado com outras janelas viram uma coluna estreita e as tabelas não cabem" (visto na VM com o MkPulso ao lado de dois terminais). A regra casa pelo título, e ficou com o nome do primeiro app que a pediu ([[Apps do Mk (motor em Go, janela em QML)]]).

## Onde está no código

`config/hypr/mare/rules.lua` (20 linhas).

## Ligações

Depende de: [[Hyprland em Lua]]. Carregado em: [[Ordem de carga]].
