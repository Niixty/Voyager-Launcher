package loadingprogress

import ("strings"; "testing")

func TestStageLooksLikeNormalGameLoading(t *testing.T) {
    samples:=[]struct{percent int;label string}{
        {0,"Verificando arquivos do jogo"},
        {20,"Assoprando cartucho..."},
        {40,"Trocando bateria..."},
        {55,"Aguardando a janela do jogo..."},
        {70,"Janela do jogo detectada"},
        {90,"Confirmando imagem do jogo..."},
        {96,"Finalizando carregamento..."},
        {100,"Jogo pronto"},
    }
    for _,c:=range samples {
        if got:=Stage(c.percent);got!=c.label {t.Fatalf("%d%%: got %q, expected %q",c.percent,got,c.label)}
    }
}


func TestCustomStagesPreserveAllMessages(t *testing.T) {
    defaults:=DefaultTexts()
    specs:=StageDefinitions()
    if len(defaults)!=10 || len(specs)!=len(defaults) {t.Fatal("dez etapas esperadas")}
    for i,stage:=range specs {
        custom:=DefaultTexts()
        custom[i]="Mensagem personalizada"
        if got:=StageWithTexts(stage.Percent,custom);got!="Mensagem personalizada" {
            t.Fatalf("%d%%: %q",stage.Percent,got)
        }
    }
    // Real launch milestones are individually customizable too.
    if got:=Stage(25);got!="Abrindo o jogo" {t.Fatalf("25%%: %q",got)}
    if got:=Stage(45);got!="Aguardando o jogo abrir" {t.Fatalf("45%%: %q",got)}
}
func TestNormalizeCustomStages(t *testing.T) {
    msgs:=NormalizeTexts([]string{"  Ligando   console\n  ", "", " \t", "Mensagem teste"})
    if len(msgs)!=10 || msgs[0]!="Ligando console" || msgs[1]!="Assoprando cartucho..." || msgs[2]!="Abrindo o jogo" || msgs[3]!="Mensagem teste"{
        t.Fatalf("normalização de textos: %+v",msgs)
    }
    long:=make([]string,10)
    for i:=range long {long[i]="ok"}
    long[4]=strings.Repeat("ç",130)
    got:=NormalizeTexts(long)
    if len([]rune(got[4]))!=96 {t.Fatal("limite Unicode de 96 caracteres")}
}
