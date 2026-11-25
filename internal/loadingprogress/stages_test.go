package loadingprogress

import "testing"

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
