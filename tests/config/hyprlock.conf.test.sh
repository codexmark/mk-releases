#!/usr/bin/env bash
# config/hypr/hyprlock.conf: a tela de bloqueio de reserva (quando o Maré não
# está no ar). Formato e o mínimo para alguém conseguir destravar.
. "$(dirname "$0")/../lib.sh"

conf=$REPO/config/hypr/hyprlock.conf
corpo=$(grep -v '^[[:space:]]*#' "$conf")

t "formato: chaves fechadas, uma seção por bloco, só seções do hyprlock"
eq "$(grep -c '{' <<<"$corpo")" "$(grep -c '^[[:space:]]*}' <<<"$corpo")"
while read -r secao; do
  like "$secao" '^(general|background|label|shape|input-field|image)$' "seção"
done < <(sed -n 's/^\([a-z-]*\)[[:space:]]*{.*/\1/p' <<<"$corpo")

t "dentro dos blocos só há linhas 'chave = valor'"
eq "" "$(grep -v '^[[:space:]]*$' <<<"$corpo" | grep -vE '^[a-z-]+[[:space:]]*\{$|^\}$|^    [a-z_]+ =( .*)?$' || true)"

t "tem um fundo e um (só um) campo de senha"
eq 1 "$(grep -c '^background' <<<"$corpo")"
eq 1 "$(grep -c '^input-field' <<<"$corpo")"

t "o campo de senha fica visível mesmo vazio, e avisa a senha errada em português"
has "$corpo" "fade_on_empty = false"
has "$corpo" "fail_text = senha incorreta"

t "a data sai em português"
has "$corpo" 'LC_TIME=pt_BR.UTF-8 date'

t "as fontes vêm nas listas de pacotes do Mk"
pkgs=$(sed -e 's/#.*//' "$REPO"/system/packages/{base,desktop,dev}.packages)
eq $'Inter\nInter SemiBold\nJetBrainsMono Nerd Font' "$(sed -n 's/^    font_family = //p' <<<"$corpo" | sort -u)"
ok grep -qx "inter-font" <<<"$pkgs"
ok grep -qx "ttf-jetbrains-mono-nerd" <<<"$pkgs"

t "é o arquivo que o hypridle.conf aponta para o hyprlock"
has "$(cat "$REPO/config/hypr/hypridle.conf")" "hyprlock -c ~/.config/mk/hypr/hyprlock.conf"
