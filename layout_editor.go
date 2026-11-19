//go:build windows

package main

import (
    "fmt"
    "math"
    "os"
    "path/filepath"
    "strings"
    "unsafe"

    ll "voyager/native/internal/loadinglayout"
    "voyager/native/internal/heroaccent"
)

type layoutEditorState struct {
    gameID string
    doc ll.Layout
    selected string
    undo []ll.Layout
    redo []ll.Layout
    grid bool
    snap bool
    dragging bool
    resizing string
    startedX,startedY float64
    original ll.Element
    before ll.Layout
    background *img
    backdropPath string
    accent uint32
}

func loadingLayoutFile(id string)string {
    return filepath.Join(gameDir(id),"loading_layout.json")
}
func (m *managerState) editorClear() {
    if m.editor.background!=nil {m.editor.background.close()}
    m.editor=layoutEditorState{}
}
func (m *managerState) editorEnsure(){
    if m.activeID=="" || m.creating {return}
    if m.editor.gameID==m.activeID{return}
    m.editorClear()
    layout,err:=ll.Load(loadingLayoutFile(m.activeID))
    if err!=nil { m.status="Layout inválido: usando posições padrão ("+err.Error()+")" }
    m.editor.doc=layout
    m.editor.gameID=m.activeID
    m.editor.selected=ll.Title
    m.editor.grid=true
    m.editor.snap=true
    m.editor.accent=heroaccent.FromFile(m.heroPath)
    if appPrefs.BlurBackground {
        m.editor.background=loadBlurry(m.heroPath)
    }else{
        m.editor.background=loadImg(m.heroPath)
    }
}
func (m *managerState) editorSave(){
    if m.editor.gameID=="" {return}
    if err:=ll.Save(loadingLayoutFile(m.editor.gameID),m.editor.doc);err!=nil{
        m.status="Erro ao salvar layout: "+err.Error()
        messageBox(m.hwnd,m.status,"Editor de loading")
        return
    }
    m.status="Layout salvo para "+m.editor.gameID+" (posição aplicada no loading e PRINT 50%)"
}
func (m *managerState) editorRemember(doc ll.Layout){
    m.editor.undo=append(m.editor.undo,doc.Clone())
    if len(m.editor.undo)>35 {m.editor.undo=m.editor.undo[len(m.editor.undo)-35:]}
    m.editor.redo=nil
}
func (m *managerState) editorEdit(fn func(*ll.Element)) {
    m.editorEnsure()
    if m.editor.gameID=="" {return}
    current:=m.editor.doc.Find(m.editor.selected)
    if current==nil{return}
    previous:=m.editor.doc.Clone()
    fn(current)
    current.Normalize()
    if !editorSame(previous,m.editor.doc){
        m.editorRemember(previous)
        m.editorSave()
    }
    invalid(m.hwnd)
}
func editorSame(a,b ll.Layout)bool {
    if len(a.Elements)!=len(b.Elements){return false}
    for i:=range a.Elements {if a.Elements[i]!=b.Elements[i]{return false}}
    return true
}
func (m *managerState) editorUndo(){
    e:=&m.editor
    if len(e.undo)==0{return}
    e.redo=append(e.redo,e.doc.Clone())
    e.doc=e.undo[len(e.undo)-1]
    e.undo=e.undo[:len(e.undo)-1]
    m.editorSave()
    invalid(m.hwnd)
}
func (m *managerState) editorRedo(){
    e:=&m.editor
    if len(e.redo)==0{return}
    e.undo=append(e.undo,e.doc.Clone())
    e.doc=e.redo[len(e.redo)-1]
    e.redo=e.redo[:len(e.redo)-1]
    m.editorSave()
    invalid(m.hwnd)
}

func (m *managerState) editorCanvas()(int,int,int,int){
    full:=m.windowW-345
    width:=full-225
    byHeight:=int(float64(m.windowH-470)*16/9)
    if byHeight<width{width=byHeight}
    if width<170{width=170}
    return 326,246,width,int(float64(width)*9/16)
}
func (m *managerState) editorVirtualAt(x,y int)(float64,float64){
    cx,cy,cw,ch:=m.editorCanvas()
    return float64(x-cx)*float64(ll.VirtualWidth)/float64(cw),
        float64(y-cy)*float64(ll.VirtualHeight)/float64(ch)
}
func (m *managerState) editorPixelRect(elem ll.Element)(int,int,int,int){
    x,y,w,h:=m.editorCanvas()
    return x+int(math.Round(elem.X*float64(w)/1600)),
        y+int(math.Round(elem.Y*float64(h)/900)),
        int(math.Max(1,math.Round(elem.W*float64(w)/1600))),
        int(math.Max(1,math.Round(elem.H*float64(h)/900)))
}
func editorInRect(x,y,rx,ry,rw,rh int)bool{
    return x>=rx && y>=ry && x<rx+rw && y<ry+rh
}
func (m *managerState) editorMouseDown(x,y int)bool{
    if m.tab!=5{return false}
    cx,cy,cw,ch:=m.editorCanvas()
    if !editorInRect(x,y,cx,cy,cw,ch){return false}
    m.editorEnsure()
    e:=&m.editor
    if e.gameID==""{return true}
    selected:=e.doc.Find(e.selected)
    if selected!=nil && selected.Visible {
        rx,ry,rw,rh:=m.editorPixelRect(*selected)
        // The four corner handles allow independent sizing on both axes.
        for _,handle:=range []struct{name string;x,y int}{
            {"tl",rx,ry},{"tr",rx+rw,ry},{"bl",rx,ry+rh},{"br",rx+rw,ry+rh},
        }{
            if math.Abs(float64(x-handle.x))<=9 && math.Abs(float64(y-handle.y))<=9{
                e.resizing=handle.name
                e.dragging=true
                e.startedX,e.startedY=m.editorVirtualAt(x,y)
                e.original=*selected
                e.before=e.doc.Clone()
                call(pSetFocus,m.hwnd)
                call(pSetCapture,m.hwnd)
                invalid(m.hwnd)
                return true
            }
        }
    }
    for i:=len(e.doc.Elements)-1;i>=0;i--{
        node:=e.doc.Elements[i]
        if !node.Visible{continue}
        rx,ry,rw,rh:=m.editorPixelRect(node)
        if editorInRect(x,y,rx,ry,rw,rh){
            e.selected=node.ID
            e.resizing=""
            e.dragging=true
            e.startedX,e.startedY=m.editorVirtualAt(x,y)
            e.original=node
            e.before=e.doc.Clone()
            call(pSetFocus,m.hwnd)
            call(pSetCapture,m.hwnd)
            invalid(m.hwnd)
            return true
        }
    }
    e.selected=""
    invalid(m.hwnd)
    return true
}
func (m *managerState) editorMouseMove(x,y int){
    e:=&m.editor
    if m.tab!=5 || !e.dragging{return}
    node:=e.doc.Find(e.selected)
    if node==nil{return}
    vx,vy:=m.editorVirtualAt(x,y)
    dx,dy:=vx-e.startedX,vy-e.startedY
    src:=e.original
    if e.resizing=="" {
        node.X=src.X+dx
        node.Y=src.Y+dy
        if e.snap {node.X=ll.Snap(node.X,10);node.Y=ll.Snap(node.Y,10)}
    }else{
        left,top,right,bottom:=src.X,src.Y,src.X+src.W,src.Y+src.H
        if strings.Contains(e.resizing,"l"){left+=dx}else{right+=dx}
        if strings.Contains(e.resizing,"t"){top+=dy}else{bottom+=dy}
        if e.snap {
            if strings.Contains(e.resizing,"l"){left=ll.Snap(left,10)}else{right=ll.Snap(right,10)}
            if strings.Contains(e.resizing,"t"){top=ll.Snap(top,10)}else{bottom=ll.Snap(bottom,10)}
        }
        if right-left<16{
            if strings.Contains(e.resizing,"l"){left=right-16}else{right=left+16}
        }
        if bottom-top<6{
            if strings.Contains(e.resizing,"t"){top=bottom-6}else{bottom=top+6}
        }
        node.X,node.Y,node.W,node.H=left,top,right-left,bottom-top
    }
    node.Normalize()
    invalid(m.hwnd)
}
func (m *managerState) editorMouseUp(){
    e:=&m.editor
    if !e.dragging{return}
    e.dragging=false
    call(pReleaseCapture)
    if !editorSame(e.before,e.doc){
        m.editorRemember(e.before)
        m.editorSave()
    }
    invalid(m.hwnd)
}
func (m *managerState) editorWheel(x,y int,delta int16)bool{
    if m.tab!=5{return false}
    cx,cy,cw,ch:=m.editorCanvas()
    if !editorInRect(x,y,cx,cy,cw,ch){return false}
    factor:=0.95
    if delta>0{factor=1.05}
    m.editorEdit(func(e *ll.Element){
        newW,newH:=e.W*factor,e.H*factor
        e.X+=(e.W-newW)/2
        e.Y+=(e.H-newH)/2
        e.W,e.H=newW,newH
    })
    return true
}
func (m *managerState) editorKey(key uintptr)bool{
    if m.tab!=5{return false}
    if call(pGetKeyState,0x11)&0x8000!=0{
        if key=='Z'{m.editorUndo();return true}
        if key=='Y'{m.editorRedo();return true}
    }
    if key==0x2E {
        m.editorEdit(func(e *ll.Element){e.Visible=!e.Visible})
        return true
    }
    delta:=1.0
    if call(pGetKeyState,0x10)&0x8000!=0 {delta=10}
    x,y:=0.0,0.0
    switch key{
    case 0x25:x=-delta
    case 0x27:x=delta
    case 0x26:y=-delta
    case 0x28:y=delta
    default:return false
    }
    m.editorEdit(func(e *ll.Element){e.X+=x;e.Y+=y})
    return true
}
func (m *managerState) editorAction(key string)bool{
    if !strings.HasPrefix(key,"layout:"){return false}
    m.editorEnsure()
    e:=&m.editor
    action:=strings.TrimPrefix(key,"layout:")
    if action=="save" {m.editorSave();invalid(m.hwnd);return true}
    if action=="grid" {e.grid=!e.grid;invalid(m.hwnd);return true}
    if action=="snap" {e.snap=!e.snap;invalid(m.hwnd);return true}
    if action=="undo" {m.editorUndo();return true}
    if action=="redo" {m.editorRedo();return true}
    if action=="preview" {m.capturePreview();return true}
    if action=="edittext" {m.tab=0;m.layout();invalid(m.hwnd);return true}
    if action=="resetall" {
        if e.gameID==""{return true}
        m.editorRemember(e.doc)
        e.doc=ll.Default()
        e.selected=ll.Title
        m.editorSave();invalid(m.hwnd)
        return true
    }
    if action=="next"||action=="prev"{
        index:=0
        for i,node:=range e.doc.Elements{if node.ID==e.selected{index=i;break}}
        if action=="next"{index=(index+1)%len(e.doc.Elements)}else{index=(index-1+len(e.doc.Elements))%len(e.doc.Elements)}
        e.selected=e.doc.Elements[index].ID
        invalid(m.hwnd);return true
    }
    if action=="front"||action=="back" {
        old:=e.doc.Clone()
        delta:=-1
        if action=="front"{delta=1}
        e.doc.MoveLayer(e.selected,delta)
        if !editorSame(old,e.doc){m.editorRemember(old);m.editorSave()}
        invalid(m.hwnd);return true
    }
    if action=="reset"{
        defaults:=ll.Default()
        node:=defaults.Find(e.selected)
        if node!=nil{
            replacement:=*node
            m.editorEdit(func(current *ll.Element){*current=replacement})
        }
        return true
    }
    switch action{
    case "visible":
        m.editorEdit(func(n *ll.Element){n.Visible=!n.Visible})
    case "color":
        m.editorEdit(func(n *ll.Element){
            options:=[]string{"auto","white","muted","hero"}
            index:=0
            for i,v:=range options{if n.Color==v{index=i;break}}
            n.Color=options[(index+1)%len(options)]
        })
    case "left","centerx","right","top","centery","bottom":
        m.editorEdit(func(n *ll.Element){
            switch action {
            case "left":n.X=40
            case "centerx":n.X=(1600-n.W)/2
            case "right":n.X=1560-n.W
            case "top":n.Y=40
            case "centery":n.Y=(900-n.H)/2
            case "bottom":n.Y=860-n.H
            }
        })
    case "small","big":
        factor:=0.9
        if action=="big"{factor=1.1}
        m.editorEdit(func(n *ll.Element){
            nw,nh:=n.W*factor,n.H*factor
            n.X+=(n.W-nw)/2;n.Y+=(n.H-nh)/2;n.W,n.H=nw,nh
        })
    case "fontminus","fontplus":
        delta:=float32(-2)
        if action=="fontplus"{delta=2}
        m.editorEdit(func(n *ll.Element){if ll.IsText(n.ID){n.FontSize+=delta}})
    default:return false
    }
    return true
}
func (m *managerState) paintEditor(p *painter){
    p.text(326,186,m.windowW-370,35,"Editor visual do loading",21,cWhite,true)
    p.text(326,213,m.windowW-370,27,"Arraste os elementos; use os cantos para redimensionar. Cada jogo guarda seu próprio layout.",11,cMuted,false)
    m.editorEnsure()
    if m.editor.gameID==""{
        p.text(334,300,m.windowW-400,80,"Salve um jogo na aba JOGO para começar a editar a tela de loading.",16,cWhite,false)
        return
    }
    cx,cy,cw,ch:=m.editorCanvas()
    p.fill(cx-3,cy-3,cw+6,ch+6,cEdge)
    c:=m.readUI()
    if c.Title==""{c.Title=m.editor.gameID}
    cartridge:=m.cartridge
    if cartridge==nil{cartridge=m.grid}
    bg:=m.editor.background
    if bg==nil{bg=m.hero}
    drawOverlay(p,c,cx,cy,cw,ch,50,"Preparando a tela do jogo","",cartridge,bg,m.editor.accent,0,m.editor.doc,false)
    if m.editor.grid {
        for i:=100;i<1600;i+=100 {
            x:=cx+i*cw/1600
            p.fill(x,cy,1,ch,0x50ffffff)
        }
        for i:=100;i<900;i+=100 {
            y:=cy+i*ch/900
            p.fill(cx,y,cw,1,0x50ffffff)
        }
    }
    // Visibility outlines and selected resize handles.
    for _,node:=range m.editor.doc.Elements {
        if !node.Visible {continue}
        x,y,w,h:=m.editorPixelRect(node)
        border:=palette(0x609a9eaa)
        if node.ID==m.editor.selected {border=palette(appPrefs.Accent())}
        p.fill(x,y,w,1,border)
        p.fill(x,y+h-1,w,1,border)
        p.fill(x,y,1,h,border)
        p.fill(x+w-1,y,1,h,border)
        if node.ID==m.editor.selected {
            for _,pt:=range []struct{x,y int}{{x,y},{x+w,y},{x,y+h},{x+w,y+h}}{
                p.fill(pt.x-4,pt.y-4,9,9,palette(appPrefs.Accent()))
            }
        }
    }
    if m.editor.snap {p.text(cx+10,cy+ch-21,128,18,"SNAP 10px",10,palette(0xffd8e8dc),true)}
    // Toolbar immediately below the preview.
    toolbarY:=cy+ch+14
    toolbarW:=(cw-24)/4
    for i,b:=range []struct{id,name string;selected bool}{
        {"grid","GRADE",m.editor.grid},{"snap","SNAP 10px",m.editor.snap},
        {"undo","DESFAZER",false},{"redo","REFAZER",false},
    }{
        m.btn(p,"layout:"+b.id,b.name,cx+i*(toolbarW+8),toolbarY,toolbarW,34,b.selected)
    }
    p.text(cx,toolbarY+44,cw,33,"Setas = mover 1 px   •   Shift + setas = 10 px   •   Roda = tamanho   •   Del = ocultar   •   Ctrl+Z/Y",10,cMuted,false)
    // Property panel.
    sx:=cx+cw+17
    sw:=m.windowW-35-sx
    if sw<80{return}
    p.fill(sx,cy,sw,m.windowH-344,cField)
    p.text(sx+10,cy+10,sw-20,25,"PROPRIEDADES",12,cMuted,true)
    current:=m.editor.doc.Find(m.editor.selected)
    if current==nil {
        p.text(sx+10,cy+49,sw-20,46,"Clique em um elemento para selecioná-lo.",12,cWhite,false)
        m.btn(p,"layout:prev","ANTERIOR",sx+8,cy+104,(sw-24)/2,33,false)
        m.btn(p,"layout:next","PRÓXIMO",sx+16+(sw-24)/2,cy+104,(sw-24)/2,33,false)
        return
    }
    p.text(sx+10,cy+44,sw-20,48,ll.Name(current.ID),17,cWhite,true)
    p.text(sx+10,cy+84,sw-20,24,fmt.Sprintf("X %.0f   Y %.0f",current.X,current.Y),11,cMuted,false)
    p.text(sx+10,cy+105,sw-20,24,fmt.Sprintf("L %.0f   A %.0f",current.W,current.H),11,cMuted,false)
    w2:=(sw-24)/2
    two:=func(y int,a,al,b,bl string){
        m.btn(p,"layout:"+a,al,sx+8,cy+y,w2,32,false)
        m.btn(p,"layout:"+b,bl,sx+16+w2,cy+y,w2,32,false)
    }
    two(141,"prev","ANTERIOR","next","PRÓXIMO")
    visibility:="MOSTRAR"
    if current.Visible {visibility="OCULTAR"}
    two(183,"visible",visibility,"reset","RESTAURAR")
    two(225,"small","TAM. -","big","TAM. +")
    if ll.IsText(current.ID) {
        two(267,"fontminus","FONTE -","fontplus","FONTE +")
    } else {
        two(267,"back","ATRÁS","front","FRENTE")
    }
    p.text(sx+10,cy+308,sw-20,20,"ALINHAMENTO",10,cMuted,true)
    w3:=(sw-32)/3
    for i,b:=range []struct{id,label string}{
        {"left","ESQ"},{"centerx","CENTRO"},{"right","DIR"},
    }{
        m.btn(p,"layout:"+b.id,b.label,sx+8+i*(w3+8),cy+334,w3,30,false)
    }
    for i,b:=range []struct{id,label string}{
        {"top","TOPO"},{"centery","MEIO"},{"bottom","BASE"},
    }{
        m.btn(p,"layout:"+b.id,b.label,sx+8+i*(w3+8),cy+371,w3,30,false)
    }
    if ll.IsText(current.ID) {
        p.text(sx+10,cy+409,sw-20,21,"COR: "+strings.ToUpper(current.Color),11,cMuted,false)
        m.btn(p,"layout:color","ALTERAR COR",sx+8,cy+434,sw-16,30,false)
    }
}
func (m *managerState) paintEditorFooter(p *painter,right,h int){
    width:=138
    gap:=9
    opts:=[]struct{id,text string;sel bool}{
        {"resetall","REDEFINIR",false},{"edittext","EDITAR TEXTOS",false},
        {"preview","PRINT 50%",false},{"save","SALVAR LAYOUT",true},
    }
    for i,o:=range opts{
        m.btn(p,"layout:"+o.id,o.text,right-(len(opts)-i)*(width+gap),h-72,width,44,o.sel)
    }
}
