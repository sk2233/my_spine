package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	atlas := ParseAtlas("res/dyn_illust_1012_skadi2/dyn_illust_char_1012_skadi2.atlas")
	skel := ParseSkel("res/dyn_illust_1012_skadi2/dyn_illust_char_1012_skadi2.json")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
