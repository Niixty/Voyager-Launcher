package loadingprogress

import "strings"

// StageSpec describes the message shown starting at a given percentage.
// The two additional startup milestones (25% and 45%) are also used by
// the real EXE / RetroArch launcher, not only by the 10-second preview.
type StageSpec struct {
    Percent int
    Label string
    Default string
}

var stages = [...]StageSpec{
    {0,"Inicialização","Verificando arquivos do jogo"},
    {15,"Preparação","Assoprando cartucho..."},
    {25,"Iniciando o jogo","Abrindo o jogo"},
    {35,"Ajustes","Trocando bateria..."},
    {45,"Aguardando abertura","Aguardando o jogo abrir"},
    {55,"Procurando janela","Aguardando a janela do jogo..."},
    {70,"Janela identificada","Janela do jogo detectada"},
    {85,"Confirmando imagem","Confirmando imagem do jogo..."},
    {95,"Finalizando","Finalizando carregamento..."},
    {100,"Concluído","Jogo pronto"},
}

func StageDefinitions() []StageSpec {
    out:=make([]StageSpec,len(stages))
    copy(out,stages[:])
    return out
}

func DefaultTexts() []string {
    texts:=make([]string,len(stages))
    for i,s:=range stages {texts[i]=s.Default}
    return texts
}

// NormalizeTexts restores every missing/empty legacy message and caps its
// length, avoiding multiline native overlay text or oversized JSON files.
func NormalizeTexts(input []string) []string {
    texts:=DefaultTexts()
    for i:=range texts {
        if i>=len(input){break}
        value:=strings.Join(strings.Fields(input[i])," ")
        if value==""{continue}
        chars:=[]rune(value)
        if len(chars)>96 {value=string(chars[:96])}
        texts[i]=value
    }
    return texts
}

// StageWithTexts maps the observed progress to the corresponding customized
// message. The empty/nil slice uses factory defaults for older layouts.
func StageWithTexts(percent int,texts []string) string {
    i:=0
    for j,s:=range stages {
        if percent<s.Percent{break}
        i=j
    }
    if i<len(texts) && strings.TrimSpace(texts[i])!="" {return texts[i]}
    return stages[i].Default
}

// Stage preserves the existing API for callers without a custom layout.
func Stage(percent int) string {return StageWithTexts(percent,nil)}
