package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// 缺失 IK 的支持
	atlas := ParseAtlas("res/358_lisa/build_char_358_lisa.atlas")
	skel := ParseSkel("res/358_lisa/build_char_358_lisa.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
