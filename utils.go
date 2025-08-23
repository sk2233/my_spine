package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
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
