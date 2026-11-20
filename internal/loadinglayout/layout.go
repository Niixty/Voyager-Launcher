package loadinglayout

import (
    "encoding/json"
    "errors"
    "fmt"
    "math"
    "os"
    "voyager/native/internal/jsonstore"
)

const VirtualWidth = 1600
const VirtualHeight = 900

const (
    Cartridge = "cartridge"
    Platform = "platform"
    Title = "title"
    Description = "description"
    Heading = "heading"
    Percentage = "percentage"
    Progress = "progress"
    Phase = "phase"
)

type Element struct {
    ID string `json:"id"`
    X float64 `json:"x"`
    Y float64 `json:"y"`
    W float64 `json:"width"`
    H float64 `json:"height"`
    FontSize float32 `json:"fontSize,omitempty"`
    Visible bool `json:"visible"`
    Color string `json:"color,omitempty"`
    // O alinhamento do texto é independente da posição da caixa.
    // Layouts antigos sem esses campos permanecem alinhados à esquerda e ao topo.
    TextAlign string `json:"textAlign,omitempty"`
    VerticalAlign string `json:"verticalAlign,omitempty"`
}

type Layout struct {
    Version int `json:"version"`
    Elements []Element `json:"elements"`
}

func Default() Layout {
    return Layout{Version:1, Elements: []Element{
        {ID:Cartridge,X:165,Y:226,W:305,H:411,Visible:true},
        {ID:Platform,X:526,Y:243,W:810,H:33,FontSize:15,Visible:true,Color:"auto"},
        {ID:Title,X:526,Y:300,W:845,H:120,FontSize:43,Visible:true,Color:"white"},
        {ID:Description,X:526,Y:465,W:790,H:85,FontSize:19,Visible:true,Color:"muted"},
        {ID:Heading,X:526,Y:574,W:615,H:43,FontSize:25,Visible:true,Color:"white"},
        {ID:Percentage,X:1277,Y:580,W:125,H:38,FontSize:21,Visible:true,Color:"white"},
        {ID:Progress,X:526,Y:634,W:865,H:6,Visible:true},
        {ID:Phase,X:526,Y:651,W:865,H:40,FontSize:15,Visible:true,Color:"muted"},
    }}
}

var labels=map[string]string{
    Cartridge:"Cartucho",Platform:"Plataforma",Title:"Título",Description:"Descrição",
    Heading:"Status de carregamento",Percentage:"Porcentagem",Progress:"Barra de progresso",Phase:"Etapa atual",
}
func Name(id string)string {if name,ok:=labels[id];ok{return name};return id}
func ValidID(id string)bool {_,ok:=labels[id];return ok}
func IsText(id string)bool {return id!=Cartridge && id!=Progress}

func (l Layout) Clone() Layout {
    l.Elements=append([]Element(nil),l.Elements...)
    return l
}
func (l *Layout) Find(id string)*Element {
    for i:=range l.Elements {if l.Elements[i].ID==id {return &l.Elements[i]}}
    return nil
}
func (l *Layout) MoveLayer(id string,delta int) {
    for i:=range l.Elements {
        if l.Elements[i].ID!=id {continue}
        dest:=i+delta
        if dest>=0 && dest<len(l.Elements) {l.Elements[i],l.Elements[dest]=l.Elements[dest],l.Elements[i]}
        return
    }
}

func clamp(v,lo,hi float64)float64 {
    if math.IsNaN(v)||math.IsInf(v,0){return lo}
    if v<lo{return lo}
    if v>hi{return hi}
    return v
}
func Snap(v float64,grid float64)float64 {
    if grid<=0{return v}
    return math.Round(v/grid)*grid
}
func (e *Element) Normalize(){
    e.W=clamp(e.W,16,1600)
    e.H=clamp(e.H,6,900)
    e.X=clamp(e.X,0,VirtualWidth-e.W)
    e.Y=clamp(e.Y,0,VirtualHeight-e.H)
    e.FontSize=float32(clamp(float64(e.FontSize),9,90))
    if !IsText(e.ID) {
        e.FontSize=0
        e.TextAlign=""
        e.VerticalAlign=""
    } else {
        switch e.TextAlign {case "left","center","right":default:e.TextAlign="left"}
        switch e.VerticalAlign {case "top","middle","bottom":default:e.VerticalAlign="top"}
    }
    if e.Color!="hero" && e.Color!="white" && e.Color!="muted" && e.Color!="auto" {
        e.Color="auto"
    }
}
func (l Layout) Normalize() Layout {
    defaults:=Default()
    result:=Layout{Version:1,Elements:make([]Element,0,len(defaults.Elements))}
    seen:=make(map[string]bool)
    for _,node:=range l.Elements {
        if !ValidID(node.ID)||seen[node.ID]{continue}
        seen[node.ID]=true
        if orig:=defaults.Find(node.ID);orig!=nil {
            if node.W<=0||node.H<=0 {node.W,node.H=orig.W,orig.H}
            if node.FontSize<=0 {node.FontSize=orig.FontSize}
        }
        node.Normalize()
        result.Elements=append(result.Elements,node)
    }
    for _,node:=range defaults.Elements {
        if !seen[node.ID]{result.Elements=append(result.Elements,node)}
    }
    return result
}

func Load(path string)(Layout,error) {
    content,err:=os.ReadFile(path)
    if errors.Is(err,os.ErrNotExist){return Default(),nil}
    if err!=nil{return Default(),err}
    if len(content)>64*1024{return Default(),fmt.Errorf("layout muito grande")}
    var l Layout
    if err=json.Unmarshal(content,&l);err!=nil{return Default(),err}
    return l.Normalize(),nil
}
func Save(path string,l Layout)error {
    l=l.Normalize()
    data,err:=json.MarshalIndent(l,"","  ")
    if err!=nil{return err}
    return jsonstore.Write(path,append(data,'\n'))
}
