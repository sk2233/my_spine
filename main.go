package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	atlas := ParseAtlas("res/dyn_illust_249_mlyss/dyn_illust_char_249_mlyss.atlas")
	skel := ParseSkel("res/dyn_illust_249_mlyss/dyn_illust_char_249_mlyss.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
