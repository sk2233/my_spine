package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	atlas := ParseAtlas("res/003_kalts/build_char_003_kalts.atlas")
	skel := ParseSkel("res/003_kalts/build_char_003_kalts.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
