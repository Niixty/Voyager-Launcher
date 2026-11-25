package loadingprogress

import (
	"image"
	"image/color"
	"math"
)

// Blend shades only existing image pixels. Transparent pixels (including holes)
// remain untouched. The tinted region advances horizontally from left to right.
func Blend(src *image.NRGBA, percent int, accent color.NRGBA) *image.NRGBA {
    if src == nil {return nil}
    w,h:=src.Rect.Dx(),src.Rect.Dy()
    dst:=image.NewNRGBA(image.Rect(0,0,w,h))
    copy(dst.Pix,src.Pix)
    p:=math.Max(0,math.Min(100,float64(percent)))/100
    // Gradient centered at the advancing boundary. At exactly 0% and 100%
    // the image is entirely darkened / entirely colored, respectively.
    edge:=p*float64(w)
    feather:=math.Max(12,math.Min(96,float64(w)*0.12))
    brightness:=0.79+0.17*p
    if p>=0.95{brightness+=0.09*(p-0.95)/0.05}
    strength:=0.13+0.14*p
    if p>=0.95{strength+=0.045}
    for y:=0;y<h;y++ {
        for x:=0;x<w;x++ {
            i:=y*dst.Stride+x*4
            if src.Pix[i+3]==0 {continue}
            mix:=0.0
            if p>=1 {mix=1} else if p>0 {
                mix=math.Max(0,math.Min(1,(edge+feather*0.5-float64(x)-0.5)/feather))
                mix=mix*mix*(3-2*mix) // smoothstep, no hard vertical cut
            }
            bright:=brightness-0.05*(1-p)+mix*(0.11+0.045*p)
            tint:=strength*mix
            for c:=0;c<3;c++ {
                value:=float64(src.Pix[i+c])*bright
                if tint>0 {
                    var channel uint8
                    switch c{case 0:channel=accent.R;case 1:channel=accent.G;default:channel=accent.B}
                    value=value*(1-tint)+float64(channel)*tint
                }
                dst.Pix[i+c]=uint8(math.Max(0,math.Min(255,math.Round(value))))
            }
        }
    }
    return dst
}

// Halo blurs the source silhouette, not its rectangular PNG bounds.
// Output includes radius pixels of transparent padding on every side.
func Halo(src *image.NRGBA, radius int, accent color.NRGBA) *image.NRGBA {
	if src==nil{return nil}
	if radius<2{radius=2}
	if radius>64{radius=64}
	w:=src.Rect.Dx()+radius*2
	h:=src.Rect.Dy()+radius*2
	alpha:=make([]int,w*h)
	for y:=0;y<src.Rect.Dy();y++ {
		for x:=0;x<src.Rect.Dx();x++ {
			alpha[(y+radius)*w+x+radius]=int(src.Pix[y*src.Stride+x*4+3])
		}
	}
	// Two box passes approximate a soft Gaussian but avoid O(radius) per pixel.
	blur:=func(input []int,r int) []int {
		tmp:=make([]int,w*h)
		output:=make([]int,w*h)
		for y:=0;y<h;y++ {
			sum:=0
			for dx:=-r;dx<=r;dx++ {if dx>=0&&dx<w{sum+=input[y*w+dx]}}
			for x:=0;x<w;x++ {
				tmp[y*w+x]=sum/(2*r+1)
				sub:=x-r
				add:=x+r+1
				if sub>=0 {sum-=input[y*w+sub]}
				if add<w {sum+=input[y*w+add]}
			}
		}
		for x:=0;x<w;x++ {
			sum:=0
			for dy:=-r;dy<=r;dy++ {if dy>=0&&dy<h{sum+=tmp[dy*w+x]}}
			for y:=0;y<h;y++ {
				output[y*w+x]=sum/(2*r+1)
				sub:=y-r
				add:=y+r+1
				if sub>=0 {sum-=tmp[sub*w+x]}
				if add<h {sum+=tmp[add*w+x]}
			}
		}
		return output
	}
	alpha=blur(alpha,(radius+1)/2)
	alpha=blur(alpha,(radius+1)/2)
	dst:=image.NewNRGBA(image.Rect(0,0,w,h))
	for y:=0;y<h;y++ {
		for x:=0;x<w;x++ {
			i:=y*dst.Stride+x*4
			a:=alpha[y*w+x]
			if a<=0 {continue}
			dst.Pix[i]=accent.R
			dst.Pix[i+1]=accent.G
			dst.Pix[i+2]=accent.B
			dst.Pix[i+3]=uint8(math.Min(180,float64(a)*0.84))
		}
	}
	return dst
}
