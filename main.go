package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TODO 看播放效果，可能需要支持 路径约束 其实际就是路径动画

func main() {
	atlas := ParseAtlas("res/249_mlyss/build_char_249_mlyss.atlas")
	skel := ParseSkel("res/249_mlyss/build_char_249_mlyss.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
