package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	atlas := ParseAtlas("res/4198_christ/build_char_4198_christ.atlas")
	skel := ParseSkel("res/4198_christ/build_char_4198_christ.json")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
