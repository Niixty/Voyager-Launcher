# Validação no Windows

O workflow usa uma pasta temporária separada da instalação. Os testes comuns cobrem a gravação de JSON, herança de layout, preservação do canal alfa do cartucho, falha de substituição e exclusão mútua do atualizador.

O teste de execução abre um aplicativo Windows de exemplo que cria uma janela real. Ele verifica a abertura direta e por atalho criado pelo Windows. Com o tempo de permanência configurado em três segundos, mede quanto tempo decorre entre a exibição da janela e a saída do overlay.

O teste de interface inicia o mesmo gerenciador Wails em um executável de teste. Uma ponte exclusiva da compilação de testes opera o formulário HTML, enquanto o teste seleciona arquivos pelo controle Win32 de nome nos diálogos reais do Windows. O teste exige prévia e salvamento das três imagens, composição do editor com GDI+, salvamento de layout e preferências. As capturas usam a janela do aplicativo com dados sintéticos.

Para a aceitação final no host, usar uma cópia da instalação e o comando Sunshine dessa cópia. Conferir a mesma sessão no monitor e no Moonlight, especialmente as transições de foco, a ausência de quadros vazios, a fluidez do cartucho e o tempo após a abertura do RetroArch. Esses resultados dependem do compositor, das telas e da captura em uso.
