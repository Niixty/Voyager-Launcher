package appearance

import (
 "encoding/json"
 "io"
 "os"
 "strconv"
 "strings"
)

// Theme defines appearance settings editable without rebuilding the Windows executable.
type Theme struct {
 Brand string `json:"brand"`
 Footer string `json:"footer"`
 LoadingTitle string `json:"loadingTitle"`
 FailureTitle string `json:"failureTitle"`
 Colors map[string]string `json:"colors"`
}

func Default() Theme {
 return Theme{
  Brand:"VOYAGER", Footer:"VOYAGER LAUNCHER",
  LoadingTitle:"Iniciando jogo", FailureTitle:"Falha ao iniciar",
  Colors:map[string]string{
   "background":"#FF111216","dimOverlay":"#C8080A0C","brandDot":"#FFACB4BF","brandText":"#FFD5D8DC","coverBorder":"#88999CA0",
   "platform":"#FFB6BABF","gameTitle":"#FFF6F7F8","description":"#FFC1C5C8","heading":"#FFEAECED","headingError":"#FFEEA8A8",
   "percentage":"#FFE2E5EA","progressTrack":"#FF454A50","progressFill":"#FFACB7C1","progressError":"#FFC77878","status":"#FFA6AAB0","footer":"#FF7B8088",
  },
 }
}

// Load overlays valid user values onto defaults, never requiring a theme file to run.
func Load(path string) Theme {
 result:=Default()
 file,err:=os.Open(path)
 if err!=nil { return result }
 defer file.Close()
 var input Theme
 if err:=json.NewDecoder(io.LimitReader(file,64*1024)).Decode(&input);err!=nil { return result }
 if s:=strings.TrimSpace(input.Brand);s!=""&&len(s)<=60 {result.Brand=s}
 if s:=strings.TrimSpace(input.Footer);s!=""&&len(s)<=80 {result.Footer=s}
 if s:=strings.TrimSpace(input.LoadingTitle);s!=""&&len(s)<=80 {result.LoadingTitle=s}
 if s:=strings.TrimSpace(input.FailureTitle);s!=""&&len(s)<=80 {result.FailureTitle=s}
 for key,value:=range input.Colors {
  if _,ok:=result.Colors[key];!ok {continue}
  if _,valid:=ParseARGB(value);valid {result.Colors[key]=value}
 }
 return result
}

// ParseARGB understands #AARRGGBB and opaque #RRGGBB.
func ParseARGB(input string)(uint32,bool){
 code:=strings.TrimPrefix(strings.TrimSpace(input),"#")
 if len(code)!=6&&len(code)!=8 {return 0,false}
 value,err:=strconv.ParseUint(code,16,32)
 if err!=nil {return 0,false}
 if len(code)==6 {value|=0xff000000}
 return uint32(value),true
}
func(t Theme)Color(key string)uint32{
 if c,ok:=ParseARGB(t.Colors[key]);ok{return c}
 c,_:=ParseARGB(Default().Colors[key]);return c
}
