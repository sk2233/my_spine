package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

func HandleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Use(args ...any) {}

func AttachmentKey(attachment string, slot int) string {
	return fmt.Sprintf("%s-%d", attachment, slot)
}

func Vec4Mul(v1, v2 mgl32.Vec4) mgl32.Vec4 {
	return mgl32.Vec4{v1.X() * v2.X(), v1.Y() * v2.Y(), v1.Z() * v2.Z(), v1.W() * v2.W()}
}

func Vec2Mul(v1, v2 mgl32.Vec2) mgl32.Vec2 {
	return mgl32.Vec2{v1.X() * v2.X(), v1.Y() * v2.Y()}
}

func Vec2Div(v1, v2 mgl32.Vec2) mgl32.Vec2 {
	return mgl32.Vec2{v1.X() / v2.X(), v1.Y() / v2.Y()}
}

func Lerp(v1 float32, v2 float32, rate float32) float32 {
	return v1 + (v2-v1)*rate
}

func Vec2Lerp(v1 mgl32.Vec2, v2 mgl32.Vec2, rate float32) mgl32.Vec2 {
	return mgl32.Vec2{
		v1.X() + (v2.X()-v1.X())*rate,
		v1.Y() + (v2.Y()-v1.Y())*rate,
	}
}

func Vec4Lerp(v1 mgl32.Vec4, v2 mgl32.Vec4, rate float32) mgl32.Vec4 {
	return mgl32.Vec4{
		v1.X() + (v2.X()-v1.X())*rate,
		v1.Y() + (v2.Y()-v1.Y())*rate,
		v1.Z() + (v2.Z()-v1.Z())*rate,
		v1.W() + (v2.W()-v1.W())*rate,
	}
}

// 旋转角度需要特殊处理
func LerpRotate(r1 float32, r2 float32, rate float32) float32 {
	return r1 + AdjustRotate(r2-r1)*rate
}

func AdjustRotate(val float32) float32 {
	for val > 180 { // -180 ~ 180
		val -= 360
	}
	for val < -180 {
		val += 360
	}
	return val
}

func Rotate(angle float32) mgl32.Mat2 {
	return mgl32.HomogRotate2D(angle * math.Pi / 180).Mat2()
}

func Scale(scale mgl32.Vec2) mgl32.Mat2 {
	return mgl32.Scale2D(scale.X(), scale.Y()).Mat2()
}

/*
		Sx*cos , -Sy*sin
		Sx*sin , Sy*cos
		角度提取 tan = Sx*sin / Sx*cos
		缩放提取 Sx = sqrt((Sx*cos)^2 + (Sx*sin)^2)
	            Sy = sqrt((-Sy*sin)^2 + (Sy*cos)^2)
*/
func GetRotate(mat2 mgl32.Mat2) float32 {
	return float32(math.Atan(float64(mat2[2]/mat2[0])) * 180 / math.Pi)
}

func GetScale(mat2 mgl32.Mat2) mgl32.Vec2 {
	return mgl32.Vec2{
		float32(math.Sqrt(float64(mat2[0]*mat2[0] + mat2[2]*mat2[2]))),
		float32(math.Sqrt(float64(mat2[1]*mat2[1] + mat2[3]*mat2[3]))),
	}
}
