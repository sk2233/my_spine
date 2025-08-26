package main

import "github.com/go-gl/mathgl/mgl32"

const (
	BasePath = "/Users/wepie/Documents/go/my_spine/"
)

var (
	GScaleVec = mgl32.Vec2{0.35, -0.35}
	GScaleMat = Scale(GScaleVec)
)
