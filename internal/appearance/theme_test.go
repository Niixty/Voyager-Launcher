package appearance
import("os";"path/filepath";"testing")
func TestParseARGB(t *testing.T){
 if c,ok:=ParseARGB("#FF123456");!ok||c!=0xff123456 {t.Fatal("ARGB failed")}
 if c,ok:=ParseARGB("#123456");!ok||c!=0xff123456 {t.Fatal("RGB failed")}
 if _,ok:=ParseARGB("errado");ok {t.Fatal("accepted invalid color")}
}
func TestLoad(t *testing.T){
 p:=filepath.Join(t.TempDir(),"t.json")
 if err:=os.WriteFile(p,[]byte(`{"brand":"PERSONALIZADO","colors":{"progressFill":"#FF123456","gameTitle":"errado"}}`),0600);err!=nil{t.Fatal(err)}
 theme:=Load(p)
 if theme.Brand!="PERSONALIZADO"||theme.Color("progressFill")!=0xff123456{t.Fatal("custom theme invalid")}
 if theme.Color("gameTitle")!=Default().Color("gameTitle"){t.Fatal("invalid color not ignored")}
}
