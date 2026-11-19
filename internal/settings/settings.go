package settings

import (
 "encoding/json"
 "fmt"
 "io"
 "os"
 "path/filepath"
)

type Config struct {
 Theme string `json:"theme"`
 ShowPercentage bool `json:"showPercentage"`
 ShowDescription bool `json:"showDescription"`
 ShowPhase bool `json:"showPhase"`
 BlurBackground bool `json:"blurBackground"`
 BackgroundDim int `json:"backgroundDim"`
 PostOpenSeconds int `json:"postOpenSeconds"`
}

type Preset struct{ ID, Name string; Accent, Dark uint32 }
var Presets = []Preset{
 {"cinza","Cinza",0xffb6bdc8,0xff3f434d},
 {"roxo","Roxo",0xffae80ff,0xff44315f},
 {"verde","Verde",0xff6bdc99,0xff2b5543},
 {"vermelho","Vermelho",0xffff767c,0xff64353c},
 {"azul","Azul",0xff67adff,0xff294665},
 {"laranja","Laranja",0xffffac60,0xff62462e},
 {"rosa","Rosa",0xffff8ac9,0xff613651},
 {"ciano","Ciano",0xff64dedf,0xff2b555b},
}

func Default() Config {
 return Config{Theme:"cinza",ShowPercentage:true,ShowDescription:true,ShowPhase:true,BlurBackground:true,BackgroundDim:78,PostOpenSeconds:1}
}
func ValidTheme(id string) bool {
 for _,p:=range Presets {if p.ID==id{return true}}
 return false
}
func PresetFor(id string) Preset {
 for _,p:=range Presets {if p.ID==id{return p}}
 return Presets[0]
}
func (c Config) Accent() uint32 {return PresetFor(c.Theme).Accent}
func (c Config) DarkAccent() uint32 {return PresetFor(c.Theme).Dark}
func (c Config) AccentHex() string {return fmt.Sprintf("#%08X",c.Accent())}
func (c Config) Normalize() Config {
 if !ValidTheme(c.Theme) {c.Theme="cinza"}
 if c.BackgroundDim<30||c.BackgroundDim>95 {c.BackgroundDim=78}
 if c.PostOpenSeconds<1||c.PostOpenSeconds>15 {c.PostOpenSeconds=1}
 return c
}

// Load overlays present fields on defaults, leaving backward-compatible booleans.
func Load(path string) Config {
 c:=Default()
 f,err:=os.Open(path)
 if err!=nil{return c}
 defer f.Close()
 b,err:=io.ReadAll(io.LimitReader(f,64*1024))
 if err!=nil||len(b)>=64*1024{return c}
 if err=json.Unmarshal(b,&c);err!=nil{return Default()}
 return c.Normalize()
}

// Save preserves a backup until the new file is safely renamed.
func Save(path string,c Config) error {
 c=c.Normalize()
 dir:=filepath.Dir(path)
 if err:=os.MkdirAll(dir,0755);err!=nil{return err}
 b,err:=json.MarshalIndent(c,"","  ")
 if err!=nil{return err}
 tmp,err:=os.CreateTemp(dir,"settings-*.tmp")
 if err!=nil{return err}
 tempName:=tmp.Name()
 defer os.Remove(tempName)
 if _,err=tmp.Write(append(b,'\n'));err!=nil {tmp.Close();return err}
 if err=tmp.Close();err!=nil{return err}
 if _,err=os.Stat(path);err==nil{
  backup:=path+".backup"
  _=os.Remove(backup)
  if err=os.Rename(path,backup);err!=nil{return err}
  if err=os.Rename(tempName,path);err!=nil{
   _=os.Rename(backup,path)
   return err
  }
  _=os.Remove(backup)
  return nil
 }
 return os.Rename(tempName,path)
}
