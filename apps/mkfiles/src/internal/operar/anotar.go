package operar

// FezArquivo anota que a tarefa criou um arquivo (ou link): é o que o desfazer
// apaga, se ele não mudou desde então. É por aqui que um trabalho de outro
// pacote (os compactados) entra no desfazer.
func (t *Tarefa) FezArquivo(caminho string) { t.fezArquivo(caminho) }

// FezPasta anota que a tarefa criou uma pasta: o desfazer a apaga se ela
// continua vazia.
func (t *Tarefa) FezPasta(caminho string) { t.fez(feito{tipo: fPasta, caminho: caminho}) }
