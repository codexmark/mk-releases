package coleta

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"mkpulso/internal/base"
)

// ContadorDisco são os bytes lidos e gravados num disco desde o boot.
type ContadorDisco struct {
	Nome             string
	Leitura, Escrita uint64
}

// Disco é um item de "discos" do retrato.
type Disco struct {
	Nome       string  `json:"nome"`
	LeituraBps float64 `json:"leitura_bps"`
	EscritaBps float64 `json:"escrita_bps"`
}

// O que nunca é um disco de verdade: laços, discos de memória, o mapeador
// (dm-, md: a leitura já foi contada no disco de baixo), CD e disquete.
var naoEDisco = []string{"loop", "ram", "zram", "dm-", "md", "sr", "fd", "nbd"}

var nomeDeDisco = regexp.MustCompile(`^(sd[a-z]+|vd[a-z]+|xvd[a-z]+|hd[a-z]+|nvme\d+n\d+|mmcblk\d+)$`)

// DiscoInteiro diz se o nome é de um disco inteiro (não partição, laço nem
// disco de memória). Vale o que está em /sys/block, onde partição não
// aparece; sem /sys/block, o formato do nome.
func DiscoInteiro(nome string) bool {
	for _, p := range naoEDisco {
		if strings.HasPrefix(nome, p) {
			return false
		}
	}
	if nome == "" || strings.ContainsAny(nome, "/.") {
		return false
	}
	if info, err := os.Stat(base.Sys("block")); err == nil && info.IsDir() {
		_, err := os.Stat(base.Sys("block", nome))
		return err == nil
	}
	return nomeDeDisco.MatchString(nome)
}

// LerDiscos lê /proc/diskstats, só os discos inteiros, em ordem de nome.
func LerDiscos() []ContadorDisco {
	b, err := os.ReadFile(base.Proc("diskstats"))
	if err != nil {
		return nil
	}
	var ds []ContadorDisco
	for _, linha := range strings.Split(string(b), "\n") {
		c := strings.Fields(linha)
		if len(c) < 10 || !DiscoInteiro(c[2]) {
			continue
		}
		// os campos 6 e 10 são setores de 512 bytes
		ds = append(ds, ContadorDisco{Nome: c[2], Leitura: numero(c[5]) * 512, Escrita: numero(c[9]) * 512})
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].Nome < ds[j].Nome })
	return ds
}

// MedirDiscos dá a taxa de cada disco entre duas amostras. Disco que não
// estava na anterior (ou sem anterior) sai com 0.
func MedirDiscos(antes, depois *Amostra) []Disco {
	ds := []Disco{}
	if depois == nil {
		return ds
	}
	velhos := map[string]ContadorDisco{}
	dt := 0.0
	if antes != nil {
		dt = (depois.Quando - antes.Quando).Seconds()
		for _, d := range antes.Discos {
			velhos[d.Nome] = d
		}
	}
	for _, d := range depois.Discos {
		item := Disco{Nome: d.Nome}
		if v, ok := velhos[d.Nome]; ok && dt > 0 {
			item.LeituraBps = Arred(subiu(v.Leitura, d.Leitura)/dt, 0)
			item.EscritaBps = Arred(subiu(v.Escrita, d.Escrita)/dt, 0)
		}
		ds = append(ds, item)
	}
	return ds
}

// Montagem é um item de "montagens" do retrato.
type Montagem struct {
	Onde    string `json:"onde"`
	FS      string `json:"fs"`
	TotalKB *int64 `json:"total_kb"`
	UsadoKB *int64 `json:"usado_kb"`
}

// Sistemas de arquivos de verdade (que guardam dados num disco).
var fsDeVerdade = map[string]bool{
	"ext2": true, "ext3": true, "ext4": true, "btrfs": true, "xfs": true, "f2fs": true,
	"vfat": true, "exfat": true, "ntfs": true, "ntfs3": true, "fuseblk": true, "zfs": true,
	"bcachefs": true, "jfs": true, "reiserfs": true, "nilfs2": true, "hfsplus": true, "apfs": true,
}

// Statfs devolve o tamanho e o uso, em kB, do sistema de arquivos montado
// num caminho. É uma variável para os testes trocarem (a emenda).
var Statfs = func(onde string) (totalKB, usadoKB uint64, err error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(onde, &s); err != nil {
		return 0, 0, err
	}
	bloco := uint64(s.Bsize)
	return s.Blocks * bloco / 1024, (s.Blocks - s.Bfree) * bloco / 1024, nil
}

// LerMontagens lê /proc/self/mounts: só sistemas de arquivos de verdade, e
// cada dispositivo uma vez (os subvolumes do btrfs repetem o mesmo disco; vale
// a primeira montagem, que costuma ser a raiz).
func LerMontagens() []Montagem {
	ms := []Montagem{}
	b, err := os.ReadFile(base.Proc("self", "mounts"))
	if err != nil {
		return ms
	}
	visto := map[string]bool{}
	for _, linha := range strings.Split(string(b), "\n") {
		c := strings.Fields(linha)
		if len(c) < 3 || !fsDeVerdade[c[2]] || visto[c[0]] {
			continue
		}
		visto[c[0]] = true
		m := Montagem{Onde: semEscapeOctal(c[1]), FS: c[2]}
		if total, usado, err := Statfs(m.Onde); err == nil {
			m.TotalKB = Ptr(int64(total))
			m.UsadoKB = Ptr(int64(usado))
		}
		ms = append(ms, m)
	}
	return ms
}

// semEscapeOctal desfaz os \040 (espaço), \011 (tab) etc. do mounts.
func semEscapeOctal(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			n := 0
			ok := true
			for _, d := range s[i+1 : i+4] {
				if d < '0' || d > '7' {
					ok = false
					break
				}
				n = n*8 + int(d-'0')
			}
			if ok && n < 256 {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return strings.ToValidUTF8(b.String(), "�")
}
