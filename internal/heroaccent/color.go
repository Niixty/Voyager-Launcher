package heroaccent

import (
    "image"
    _ "image/jpeg"
    _ "image/png"
    "math"
    "os"
)

// Neutral is used when no hero image is present or when it is grayscale.
const Neutral uint32 = 0xFFACB7C1

// AdjustBrightness brightens or darkens the same dominant Hero color without
// resampling the Hero. 100% is the original accent; values over 100% gently
// mix toward white, avoiding hue clipping from a naive RGB multiplier.
func AdjustBrightness(rgb uint32, percent int) uint32 {
    if rgb==0 {rgb=Neutral}
    if percent<0 {percent=0}
    if percent>200 {percent=200}
    alpha:=rgb&0xff000000
    if percent==100 {return rgb}
    channel:=func(shift uint) uint32 {
        v:=int((rgb>>shift)&0xff)
        if percent<=100 {
            return uint32((v*percent+50)/100)
        }
        return uint32(v+((255-v)*(percent-100)+50)/100)
    }
    return alpha | channel(16)<<16 | channel(8)<<8 | channel(0)
}

// FromFile calculates the accent once, when the loading begins.
func FromFile(path string) uint32 {
    if path == "" { return Neutral }
    f, err := os.Open(path)
    if err != nil { return Neutral }
    defer f.Close()
    im, _, err := image.Decode(f)
    if err != nil { return Neutral }
    return FromImage(im)
}

// FromImage selects the MOST FREQUENT usable chromatic color family in the
// Hero (not the most saturated isolated pixel). Dark/grey/white pixels are
// ignored; the selected hue is gently brightened for readable UI text and
// the translucent cartridge/Grid progress animation.
func FromImage(im image.Image) uint32 {
    if im == nil { return Neutral }
    bounds := im.Bounds()
    if bounds.Empty() { return Neutral }
    const bands = 24
    const steps = 64
    var histogram [bands]float64
    var sumSin, sumCos [bands]float64
    var sumSat, sumValue [bands]float64
    w, h := bounds.Dx(), bounds.Dy()
    for y := 0; y < steps; y++ {
        py := bounds.Min.Y + (y*2+1)*h/(steps*2)
        for x := 0; x < steps; x++ {
            px := bounds.Min.X + (x*2+1)*w/(steps*2)
            r, g, b, a := im.At(px, py).RGBA()
            if a < 0xC000 { continue }
            rf, gf, bf := float64(r)/65535, float64(g)/65535, float64(b)/65535
            if a < 65535 {
                alpha := float64(a)/65535
                rf, gf, bf = math.Min(1, rf/alpha),math.Min(1,gf/alpha),math.Min(1,bf/alpha)
            }
            hue, saturation, value := toHSV(rf,gf,bf)
            if saturation < 0.19 || value < 0.105 || value > 0.985 { continue }
            bin := int(hue*bands)%bands
            // Frequency, not vibrance, chooses the winner. Otherwise a small
            // vivid red object can override a mostly blue Hero image.
            weight := 1.0
            histogram[bin]+=weight
            angle := hue*2*math.Pi
            sumSin[bin]+=math.Sin(angle)*weight
            sumCos[bin]+=math.Cos(angle)*weight
            sumSat[bin]+=saturation*weight
            sumValue[bin]+=value*weight
        }
    }
    best,bestWeight:=-1,0.0
    for i:=0;i<bands;i++{
        weight:=histogram[(i+bands-1)%bands]*0.55+histogram[i]+histogram[(i+1)%bands]*0.55
        if weight>bestWeight {best,bestWeight=i,weight}
    }
    if best<0 || bestWeight==0 {return Neutral}
    var total,sinH,cosH,saturation,value float64
    for offset:=-1;offset<=1;offset++{
        i:=(best+offset+bands)%bands
        factor:=1.0
        if offset!=0 {factor=0.55}
        total+=histogram[i]*factor
        sinH+=sumSin[i]*factor
        cosH+=sumCos[i]*factor
        saturation+=sumSat[i]*factor
        value+=sumValue[i]*factor
    }
    if total<=0 {return Neutral}
    hue:=math.Atan2(sinH,cosH)/(2*math.Pi)
    if hue<0 {hue++}
    saturation=clamp(saturation/total,0.5,0.83)
    value=clamp(value/total,0.65,0.94)
    r,g,b:=fromHSV(hue,saturation,value)
    return 0xff000000|uint32(math.Round(r*255))<<16|uint32(math.Round(g*255))<<8|uint32(math.Round(b*255))
}
func clamp(v,lo,hi float64)float64{
    if v<lo{return lo}
    if v>hi{return hi}
    return v
}
func toHSV(r,g,b float64)(h,s,v float64){
    max:=math.Max(r,math.Max(g,b))
    min:=math.Min(r,math.Min(g,b))
    delta:=max
    delta-=min
    v=max
    if max==0{return 0,0,0}
    s=delta/max
    if delta==0{return 0,0,v}
    switch max{
    case r: h=math.Mod((g-b)/delta+6,6)
    case g: h=(b-r)/delta+2
    default:h=(r-g)/delta+4
    }
    return h/6,s,v
}
func fromHSV(h,s,v float64)(r,g,b float64){
    h*=6
    chroma:=v*s
    secondary:=chroma*(1-math.Abs(math.Mod(h,2)-1))
    x,y,z:=0.0,0.0,0.0
    switch int(h)%6{
    case 0:x,y=chroma,secondary
    case 1:x,y=secondary,chroma
    case 2:y,z=chroma,secondary
    case 3:y,z=secondary,chroma
    case 4:x,z=secondary,chroma
    default:x,z=chroma,secondary
    }
    m:=v-chroma
    return x+m,y+m,z+m
}
