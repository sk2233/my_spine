package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math/rand"
	"testing"
)

func TestRotateAndScale(t *testing.T) {
	mat2 := mgl32.Ident2()
	rotate := float32(0)
	scale := mgl32.Vec2{1, 1}
	for i := 0; i < 100; i++ {
		temp1 := rand.Float32()
		// temp2 temp3 等比缩放是顺序无关的，非等比缩放顺序有关
		temp2 := rand.Float32()/5 + 0.9
		temp3 := rand.Float32()/5 + 0.9
		rotate += temp1
		scale = Vec2Mul(scale, mgl32.Vec2{temp2, temp3})
		mat2 = mat2.Mul2(Rotate(temp1)).Mul2(Scale(mgl32.Vec2{temp2, temp3}))
	}
	temp := Rotate(rotate).Mul2(Scale(scale))
	fmt.Println(mat2, temp)
}

func TestAllMat(t *testing.T) {
	//fmt.Println(GetRotate(Rotate(90)))
	//fmt.Println(GetScale(Scale(mgl32.Vec2{2, 3})))
	m := mgl32.Mat4{}
	m[2] = 2233
	m2 := m
	m2[2] = 2323
	fmt.Println(m)
}
