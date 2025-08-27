package main

import "github.com/go-gl/mathgl/mgl32"

const (
	BasePath = "/Users/wepie/Documents/go/my_spine/"
)

const (
	GSignX = 1
	GSignY = -1
	GScale = 0.6
)

var (
	GScaleMat = Scale(mgl32.Vec2{GSignX * GScale, GSignY * GScale})
)
