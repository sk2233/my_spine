package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TODO 注意角度要控制在  -180 ～ 180 之间  全部检查一遍

func main() {
	// 缺失 IK 的支持
	atlas := ParseAtlas("res/003_kalts/build_char_003_kalts.atlas")
	skel := ParseSkel("res/003_kalts/build_char_003_kalts.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
