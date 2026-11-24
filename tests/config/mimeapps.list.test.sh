#!/usr/bin/env bash
# config/mimeapps.list: os apps padrão de um Mk novo. Formato, nada repetido,
# e cada .desktop citado é de um app que o Mk traz (ou trazia).
. "$(dirname "$0")/../lib.sh"

lista=$REPO/config/mimeapps.list
linhas=$(grep -v '^#' "$lista" | grep -v '^[[:space:]]*$')
pacotes=$(sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$REPO/system/packages/apps.packages" | grep -v '^$')

# De que pacote vem cada .desktop. Um app novo na lista entra aqui também.
pacote_de() {
  case $1 in
    chromium.desktop) echo chromium ;;
    org.gnome.Nautilus.desktop) echo nautilus ;;
    org.gnome.TextEditor.desktop) echo gnome-text-editor ;;
    org.gnome.Papers.desktop) echo papers ;;
    org.gnome.Evince.desktop) echo evince ;;
    org.gnome.Loupe.desktop) echo loupe ;;
    imv.desktop) echo imv ;;
    mpv.desktop) echo mpv ;;
    io.bassi.Amberol.desktop) echo amberol ;;
    *) return 1 ;;
  esac
}

t "uma seção só, [Default Applications], antes de qualquer associação"
eq "[Default Applications]" "$(head -1 <<<"$linhas")"
eq 1 "$(grep -c '^\[' <<<"$linhas")"

t "cada linha é tipo/subtipo=app.desktop[;outro.desktop], sem espaço nem ';' sobrando"
while IFS= read -r l; do
  like "$l" '^[a-z0-9.+-]+/[a-z0-9.+-]+=[A-Za-z0-9._-]+\.desktop(;[A-Za-z0-9._-]+\.desktop)*$' "linha"
done < <(tail -n +2 <<<"$linhas")

t "nenhum tipo aparece duas vezes"
eq "" "$(tail -n +2 <<<"$linhas" | cut -d= -f1 | sort | uniq -d)"

t "nenhum app repetido na mesma linha"
while IFS= read -r l; do
  eq "" "$(cut -d= -f2 <<<"$l" | tr ';' '\n' | sort | uniq -d)" "$l"
done < <(tail -n +2 <<<"$linhas")

t "todo .desktop citado é de um pacote conhecido"
while read -r d; do
  pacote_de "$d" >/dev/null || _fail "não sei de que pacote vem $d (acrescente em pacote_de, neste teste)"
done < <(tail -n +2 <<<"$linhas" | cut -d= -f2 | tr ';' '\n' | sort -u)

t "o primeiro app de cada tipo vem com o Mk (system/packages/apps.packages)"
while IFS= read -r l; do
  d=$(cut -d= -f2 <<<"$l" | cut -d';' -f1)
  p=$(pacote_de "$d") || continue
  grep -qx -- "$p" <<<"$pacotes" || _fail "${l%%=*}: $d é do pacote $p, que não está em apps.packages"
done < <(tail -n +2 <<<"$linhas")

t "o segundo app é o que o Mk trazia antes (imv, evince) ou o mpv, como diz o cabeçalho"
has "$(grep '^#' "$lista")" "(imv, evince) ou o mpv"
while read -r d; do
  like "$d" '^(imv|org\.gnome\.Evince|mpv)\.desktop$' "reserva"
done < <(tail -n +2 <<<"$linhas" | cut -d= -f2 | grep ';' | cut -d';' -f2- | tr ';' '\n' | sort -u)

t "os apps de reserva que o Mk não traz mais são os que a migração apps-oferecidos anota"
antigos=$(cat "$REPO/migrations/system/20261125-apps-oferecidos.sh")
while read -r d; do
  p=$(pacote_de "$d") || continue
  grep -qx -- "$p" <<<"$pacotes" || has "$antigos" " $p " "$d"
done < <(tail -n +2 <<<"$linhas" | cut -d= -f2 | tr ';' '\n' | sort -u)

t "o essencial tem dono: navegador, pastas, texto, PDF, imagem, vídeo e música"
for tipo in x-scheme-handler/http x-scheme-handler/https text/html inode/directory text/plain \
  application/pdf image/png image/jpeg video/mp4 audio/mpeg; do
  ok grep -q "^$tipo=" <<<"$linhas"
done

t "http e https abrem no mesmo navegador"
eq "$(sed -n 's|^x-scheme-handler/http=||p' <<<"$linhas")" "$(sed -n 's|^x-scheme-handler/https=||p' <<<"$linhas")"

t "cada família (imagem, vídeo, áudio) abre sempre nos mesmos apps"
for fam in image video audio; do
  eq 1 "$(grep "^$fam/" <<<"$linhas" | cut -d= -f2 | sort -u | wc -l)" "$fam"
done
