Arquivos visuais compartilhados e licenciados podem ser colocados aqui. Nao inclua ROMs, saves ou capas privadas.

`VoyagerLauncher.ico` é o ícone oficial do Voyager Launcher, substituído pela arte de alta qualidade enviada pelo autor em novembro de 2026.

O arquivo original foi preservado integralmente (sem redimensionamento ou conversão com perdas) e tem oito resoluções nativas para Windows: **16, 24, 32, 48, 64, 96, 128 e 256 pixels**. SHA-256: `3c6daa274f59acc05cd06ccf036d034ed39052c07f0dd6f3171c1f7e9c468987`.

O GitHub Actions gera `rsrc_windows_amd64.syso` desse mesmo arquivo para **VoyagerLauncher.exe** e **VoyagerUpdater.exe**. O gerenciador Wails incorpora os mesmos bytes com `//go:embed`, incluindo a imagem da interface em `/icon.ico`.

Nota: no Windows, a resolução máxima de um ícone ICO é 256 px; o Explorer escolhe a versão apropriada para cada escala. Após atualizar, atalhos fixados podem exigir desafixar/fixar novamente por conta do cache do Windows.
