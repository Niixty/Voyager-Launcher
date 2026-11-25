package loadingmotion

import "math"

// Pose is the visual transform of the icon in virtual 1600x900 coordinates.
// Settle and Celebrate have zero instantaneous jumps at their endpoints.
type Pose struct {
    Offset float64
    Scale float64
}

func clamp01(v float64)float64{
    if v<=0{return 0}
    if v>=1{return 1}
    return v
}
func Smoothstep(v float64)float64{
    t:=clamp01(v)
    return t*t*(3-2*t)
}
func Settle(initialOffset float64, elapsedSeconds float64) Pose {
    // Ease smoothly from the exact previous bobbing position at 95%.
    t:=Smoothstep(elapsedSeconds/0.36)
    return Pose{Offset:initialOffset*(1-t),Scale:1+0.035*t}
}
func Celebrate(p Pose,elapsedSeconds float64) Pose {
    // sin² starts and ends at both zero height and zero velocity. There is
    // no size/position teleport when 100% first arrives or when hop ends.
    t:=clamp01(elapsedSeconds/0.46)
    pulse:=math.Pow(math.Sin(math.Pi*t),2)
    p.Offset-=9*pulse
    p.Scale+=0.065*pulse
    return p
}
