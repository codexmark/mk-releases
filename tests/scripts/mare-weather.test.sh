#!/usr/bin/env bash
# shell/mare/scripts/mare-weather: de onde sai a cidade, as duas consultas ao
# Open-Meteo (curl por stub) e o JSON do painel, com os caminhos de erro.
. "$(dirname "$0")/../lib.sh"

clima_sh=$REPO/shell/mare/scripts/mare-weather
city=$XDG_CONFIG_HOME/mare/city
mkdir -p "$(dirname "$city")"

# curl: responde geo.json à busca da cidade e fc.json à previsão; sem o
# arquivo, falha como sem rede (6 = não resolveu o nome).
stub curl '
case "$*" in
  *geocoding-api.open-meteo.com/v1/search*) cat "$SBX/geo.json" 2>/dev/null || exit 6 ;;
  *api.open-meteo.com/v1/forecast*) cat "$SBX/fc.json" 2>/dev/null || exit 6 ;;
  *) exit 3 ;;
esac'
stub timedatectl 'cat "$SBX/fuso" 2>/dev/null'

geo() { # geo <nome> <lat> <lon>
  jq -n --arg n "$1" --argjson lat "$2" --argjson lon "$3" \
    '{results: [{id: 3470127, name: $n, latitude: $lat, longitude: $lon, country_code: "BR", timezone: "America/Sao_Paulo"}], generationtime_ms: 0.5}' >"$SBX/geo.json"
}
previsao() {
  cat >"$SBX/fc.json" <<'EOF'
{"latitude":-19.875,"longitude":-43.875,"timezone":"America/Sao_Paulo",
 "current":{"time":"2025-03-10T14:15","interval":900,"temperature_2m":27.6,"apparent_temperature":29.4,"weather_code":2,"is_day":1},
 "hourly":{"time":["2025-03-10T14:00","2025-03-10T15:00","2025-03-10T16:00","2025-03-10T17:00","2025-03-10T18:00","2025-03-10T19:00","2025-03-10T20:00","2025-03-10T21:00","2025-03-10T22:00","2025-03-10T23:00","2025-03-11T00:00","2025-03-11T01:00"],
   "temperature_2m":[27.6,28.4,28.5,27.2,25.9,24.1,22.8,21.6,21.4,20.9,20.2,19.5],
   "weather_code":[2,3,61,80,95,3,2,1,0,0,45,48]},
 "daily":{"time":["2025-03-10"],"temperature_2m_max":[29.3],"temperature_2m_min":[18.7]}}
EOF
}
run() { bash "$clima_sh"; }
zera() {
  rm -f "$city" "$SBX/geo.json" "$SBX/fc.json" "$SBX/fuso"
  calls_reset
}

t "com tudo respondendo: o JSON do painel, numa linha"
zera
echo "Belo Horizonte" >"$city"
geo "Belo Horizonte" -19.92083 -43.93778
previsao
out=$(run)
eq 1 "$(wc -l <<<"$out")"
eq '{"city":"Belo Horizonte","temp":28,"feels":29,"code":2,"day":true,"max":29,"min":19,"hours":[{"time":"14:00","temp":28,"code":2},{"time":"16:00","temp":29,"code":61},{"time":"18:00","temp":26,"code":95},{"time":"20:00","temp":23,"code":2},{"time":"22:00","temp":21,"code":0},{"time":"00:00","temp":20,"code":45}]}' "$out"

t "a busca da cidade leva o nome do arquivo, um resultado, em português"
called "https://geocoding-api.open-meteo.com/v1/search --data-urlencode name=Belo Horizonte -d count=1 -d language=pt"

t "a previsão é pedida para a latitude e a longitude achadas"
called "https://api.open-meteo.com/v1/forecast -d latitude=-19.92083 -d longitude=-43.93778 -d timezone=auto -d forecast_hours=12 -d forecast_days=1"
called "-d current=temperature_2m,apparent_temperature,weather_code,is_day -d hourly=temperature_2m,weather_code -d daily=temperature_2m_max,temperature_2m_min"

t "as duas consultas falham em erro HTTP e desistem em 10 s"
eq 2 "$(grep -c '^curl -fsS --max-time 10 -G https://' "$SBX/calls")"
eq 2 "$(grep -c '^curl ' "$SBX/calls")"

t "o nome mostrado é o que a busca devolveu, não o que foi digitado"
echo "bh" >"$city"
eq "Belo Horizonte" "$(run | jq -r .city)"

t "do arquivo da cidade, só a primeira linha vale"
calls_reset
printf '%s\n' "Recife" "Olinda" >"$city"
run >/dev/null
called "--data-urlencode name=Recife -d count=1"
not_called "Olinda"

t "day: falso quando é noite"
jq '.current.is_day = 0' "$SBX/fc.json" >"$SBX/fc.tmp" && mv "$SBX/fc.tmp" "$SBX/fc.json"
eq false "$(run | jq -c .day)"

t "as horas vêm de duas em duas, só com HH:MM"
eq '["14:00","16:00","18:00","20:00","22:00","00:00"]' "$(run | jq -c '.hours | map(.time)')"

t "sem arquivo de cidade: sai do fuso do sistema, com espaço no lugar do _"
zera
geo "São Paulo" -23.5475 -46.63611
previsao
echo "America/Sao_Paulo" >"$SBX/fuso"
eq "São Paulo" "$(run | jq -r .city)"
called "timedatectl show -p Timezone --value"
called "--data-urlencode name=Sao Paulo -d count=1"

t "fuso com três partes (America/Argentina/Buenos_Aires): vale a última"
calls_reset
echo "America/Argentina/Buenos_Aires" >"$SBX/fuso"
run >/dev/null
called "--data-urlencode name=Buenos Aires -d count=1"

t "arquivo de cidade vazio conta como sem arquivo"
calls_reset
: >"$city"
echo "America/Bahia" >"$SBX/fuso"
run >/dev/null
called "--data-urlencode name=Bahia -d count=1"

t "com arquivo de cidade, o fuso nem é consultado"
calls_reset
echo "Manaus" >"$city"
run >/dev/null
not_called "timedatectl"

t "sem cidade e sem fuso: erro 'sem cidade', sem ir à rede"
zera
eq '{"error":"sem cidade"}' "$(run)"
not_called "curl"

t "sem rede na busca da cidade: erro 'sem rede', saída 0, e a previsão nem é pedida"
zera
echo "Belo Horizonte" >"$city"
out=$(run 2>/dev/null)
eq 0 $?
eq '{"error":"sem rede"}' "$out"
not_called "forecast"

t "cidade que a busca não conhece: o erro diz qual"
zera
echo "Xyzópolis" >"$city"
echo '{"generationtime_ms":0.3}' >"$SBX/geo.json"
eq '{"error":"cidade não encontrada: Xyzópolis"}' "$(run | jq -c .)"
not_called "forecast"

t "cidade com aspas, barra e cifrão: vai inteira num argumento só e o erro é JSON válido"
zera
hostil='Vila "Nova" \ $(id) & x=1'
echo "$hostil" >"$city"
echo '{"results":[]}' >"$SBX/geo.json"
out=$(run)
eq "cidade não encontrada: $hostil" "$(jq -r .error <<<"$out")"
called "--data-urlencode name=$hostil -d count=1"

t "sem rede na previsão: erro 'sem rede'"
zera
echo "Belo Horizonte" >"$city"
geo "Belo Horizonte" -19.92083 -43.93778
out=$(run 2>/dev/null)
eq 0 $?
eq '{"error":"sem rede"}' "$out"
called "forecast"

t "resposta que não é a previsão (portal cativo): nada de JSON pela metade"
previsao
echo "<html><body>Entre com seu usuário</body></html>" >"$SBX/fc.json"
out=$(run 2>/dev/null)
[[ -z $out ]] || ok jq -e . <<<"$out" >/dev/null
