package loadingmotion

import (
    "math"
    "testing"
)

func TestSettleStartsWithoutTeleport(t *testing.T){
    for _,offset:=range []float64{-11,-2,0,6,11}{
        first:=Settle(offset,0)
        if first.Offset!=offset || first.Scale!=1 {
            t.Fatalf("95%% teleported from offset %v: %+v",offset,first)
        }
        next:=Settle(offset,0.000001)
        if math.Abs(next.Offset-first.Offset)>0.00001 || math.Abs(next.Scale-first.Scale)>0.00001 {
            t.Fatalf("first transition frame jumped: %v -> %v",first,next)
        }
        middle:=Settle(offset,0.18)
        if middle.Scale<=1||middle.Scale>=1.035 {t.Fatalf("transition not smooth: %+v",middle)}
        settled:=Settle(offset,1)
        if math.Abs(settled.Offset)>1e-8 || math.Abs(settled.Scale-1.035)>1e-8 {
            t.Fatalf("not centered: %+v",settled)
        }
    }
}
func Test100PercentHopBeginsAndEndsContinuously(t *testing.T){
    base:=Settle(-7,0.32)
    start:=Celebrate(base,0)
    if start!=base {t.Fatalf("size teleported at 100%%: %+v -> %+v",base,start)}
    tiny:=Celebrate(base,0.000001)
    if math.Abs(start.Scale-tiny.Scale)>0.00001 || math.Abs(start.Offset-tiny.Offset)>0.00001 {
        t.Fatalf("100%% has an instantaneous jump")
    }
    middle:=Celebrate(base,0.23)
    if middle.Scale<=base.Scale || middle.Offset>=base.Offset {t.Fatalf("final hop missing: %+v",middle)}
    end:=Celebrate(base,0.46)
    after:=Celebrate(base,0.7)
    if end!=base || after!=base {t.Fatalf("animation does not settle: %+v %+v",end,after)}
}
