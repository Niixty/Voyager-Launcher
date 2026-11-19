//go:build windows

package main

import (
 "fmt"
 "path/filepath"
 "strings"

 "voyager/native/internal/settings"
)

var appPrefs = settings.Default()
func appSettingsPath() string {return filepath.Join(rootDir,"settings.json")}
func loadAppPrefs(){appPrefs=settings.Load(appSettingsPath())}

func (m *managerState) changeAppPrefs(change func(*settings.Config)){
 previous:=appPrefs
 change(&appPrefs)
 appPrefs=appPrefs.Normalize()
 if err:=settings.Save(appSettingsPath(),appPrefs);err!=nil{
  appPrefs=previous
  m.status="Erro ao salvar preferências: "+err.Error()
  messageBox(m.hwnd,m.status,"Voyager - Configurações")
 }else{
  m.status="Configurações salvas automaticamente em app/settings.json"
 }
 loadOverlayTheme()
 invalid(m.hwnd)
}

func (m *managerState) settingsAction(key string) bool {
 if strings.HasPrefix(key,"preftheme:"){
  id:=strings.TrimPrefix(key,"preftheme:")
  if settings.ValidTheme(id){
   m.changeAppPrefs(func(c *settings.Config){c.Theme=id})
  }
  return true
 }
 switch key {
 case "settings":
  m.tab=4
  m.layout()
  invalid(m.hwnd)
 case "prefpercent":
  m.changeAppPrefs(func(c *settings.Config){c.ShowPercentage=!c.ShowPercentage})
 case "prefdescription":
  m.changeAppPrefs(func(c *settings.Config){c.ShowDescription=!c.ShowDescription})
 case "prefphase":
  m.changeAppPrefs(func(c *settings.Config){c.ShowPhase=!c.ShowPhase})
 case "prefblur":
  m.changeAppPrefs(func(c *settings.Config){c.BlurBackground=!c.BlurBackground})
 case "prefdim:50","prefdim:78","prefdim:88":
  var dim int
  switch key {
  case "prefdim:50":dim=50
  case "prefdim:78":dim=78
  case "prefdim:88":dim=88
  }
  m.changeAppPrefs(func(c *settings.Config){c.BackgroundDim=dim})
 case "prefreset":
  m.changeAppPrefs(func(c *settings.Config){*c=settings.Default()})
 case "prefopenapp":
  call(pShellExecute,m.hwnd,ptrStr("open"),ptrStr(rootDir),0,0,SW_SHOWNORMAL)
 case "prefcheck":
  m.status="Consultando a última versão no GitHub..."
  invalid(m.hwnd)
  startUpdateCheck(m.hwnd)
 case "prefpreview":
  m.capturePreview()
 default:
  return false
 }
 return true
}

func (m *managerState) paintSettings(p *painter){
 w:=m.windowW
 left,full:=310,w-345
 title:=func(y int,t string){p.text(left+18,y,full-30,32,t,19,cWhite,true)}
 caption:=func(y int,t string){p.text(left+18,y,full-30,26,t,12,cMuted,false)}
 title(188,"Personalização do Voyager")
 caption(218,"Escolha a cor da interface do gerenciador. O loading usa a cor da arte hero.")
 colWidth:=(full-58)/4
 if colWidth<120 {colWidth=120}
 for i,preset:=range settings.Presets {
  col,row:=i%4,i/4
  x:=left+18+col*(colWidth+8)
  y:=255+row*51
  selected:=appPrefs.Theme==preset.ID
  m.btn(p,"preftheme:"+preset.ID,preset.Name,x,y,colWidth,39,selected)
  p.fill(x+7,y+9,5,21,palette(preset.Accent))
  if selected {p.fill(x,y+36,colWidth,3,palette(preset.Accent))}
 }
 title(364,"Tela de carregamento")
 caption(392,"Opções globais de todos os jogos, inclusive no PRINT 50%.")
 toggle:=func(key,label string,x,y,width int,enabled bool){
  state:="DESLIGADO"
  if enabled {state="LIGADO"}
  m.btn(p,key,label+"  ·  "+state,x,y,width,37,enabled)
 }
 cw:=(full-44)/2
 toggle("prefpercent","PORCENTAGEM",left+18,425,cw,appPrefs.ShowPercentage)
 toggle("prefdescription","DESCRIÇÃO",left+26+cw,425,cw,appPrefs.ShowDescription)
 toggle("prefphase","ETAPA ATUAL",left+18,473,cw,appPrefs.ShowPhase)
 toggle("prefblur","FUNDO DESFOCADO",left+26+cw,473,cw,appPrefs.BlurBackground)
 caption(519,"ESCURECIMENTO DO FUNDO")
 dimW:=(full-52)/3
 for i,choice:=range []struct{value int;label string}{{50,"LEVE"},{78,"NORMAL"},{88,"FORTE"}} {
  m.btn(p,fmt.Sprintf("prefdim:%d",choice.value),choice.label,left+18+i*(dimW+8),544,dimW,38,appPrefs.BackgroundDim==choice.value)
 }
 caption(590,"As preferências são salvas na pasta app sem alterar seus jogos.")
 caption(613,"O loading termina 1 segundo após uma janela estável ser detectada.")
}
func (m *managerState) paintSettingsFooter(p *painter,right,h int){
 bw,gap:=135,9
 m.btn(p,"prefreset","RESTAURAR",right-(bw+gap)*4,h-72,bw,44,false)
 m.btn(p,"prefcheck","ATUALIZAÇÕES",right-(bw+gap)*3,h-72,bw,44,false)
 m.btn(p,"prefpreview","PRINT 50%",right-(bw+gap)*2,h-72,bw,44,false)
 m.btn(p,"prefopenapp","ABRIR APP",right-(bw+gap),h-72,bw,44,false)
}
