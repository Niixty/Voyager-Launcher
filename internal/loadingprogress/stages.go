package loadingprogress

// Stage is shared between the real launch overlay and the no-game
// 10-second preview, so the preview looks like an ordinary game startup.
// Messages are playful but should not claim to actually execute any
// cartridge/battery operations.
func Stage(percent int) string {
    switch {
    case percent < 15:
        return "Verificando arquivos do jogo"
    case percent < 35:
        return "Assoprando cartucho..."
    case percent < 55:
        return "Trocando bateria..."
    case percent < 70:
        return "Aguardando a janela do jogo..."
    case percent < 85:
        return "Janela do jogo detectada"
    case percent < 95:
        return "Confirmando imagem do jogo..."
    case percent < 100:
        return "Finalizando carregamento..."
    default:
        return "Jogo pronto"
    }
}
