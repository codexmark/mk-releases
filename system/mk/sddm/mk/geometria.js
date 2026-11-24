.pragma library

// A geometria da marca "Mk", para a tela de entrada desenhar a construção
// dela (eixos, guias, diagonais e nós) em movimento. GERADO: não edite à mão,
// rode system/mk/art/make-logo.py (lá diz de onde cada número vem).
// Unidades da fonte, y para cima (na tela, y = -altura). vb é o quadro das
// imagens letra-m.png e letra-k.png: [x, y, largura, altura], y para baixo.
var vb = [-520, -1900, 3483, 2350];
var ax = -230;          // o eixo vertical
var ay = 150;          // a linha de cota, abaixo da base (para baixo)
var top = -1640;        // o alto do eixo (para baixo: negativo é acima)
var right = 2663;    // o fim da cota
var kx = 1509;          // onde o k começa: daqui em diante a cota é de cobre
// Alturas: base, vértice do M, altura-x, maiúsculas, ascendente do k.
var heights = [0, 156, 867, 1400, 1469];
// Hastes do M, o vértice dele, a haste do k e o fim do braço.
var verts = [215, 722, 1320, 1729, 2423];
// Diagonais de construção [x1, y1, x2, y2]: duas do M, o braço e a perna do k.
var diags = [[408, 1390, 722, 156], [1180, 1388, 804, 415], [1792, 321, 2423, 867], [1940, 479, 2362, 74]];
// Nós nas quinas [x, y, cor]: 0 = osso (do M), 1 = cobre (do k).
var nodes = [[408, 1390, 0], [722, 156, 0], [1180, 1388, 0], [1729, 1469, 1], [1812, 360, 1], [2057, 600, 1], [2423, 867, 1], [2362, 74, 1]];
