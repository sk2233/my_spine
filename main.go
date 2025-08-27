package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// 缺失 IK 的支持
	atlas := ParseAtlas("res/dyn_illust_2025_shu/dyn_illust_char_2025_shu.atlas")
	skel := ParseSkel("res/dyn_illust_2025_shu/dyn_illust_char_2025_shu.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
