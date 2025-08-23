package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TODO 看播放效果，可能需要支持 路径约束 其实际就是路径动画

func main() {
	atlas := ParseAtlas("res/dyn_illust_1012_skadi2/dyn_illust_char_1012_skadi2.atlas")
	skel := ParseSkel("res/dyn_illust_1012_skadi2/dyn_illust_char_1012_skadi2.skel")

	ebiten.SetWindowSize(1280, 720)
	err := ebiten.RunGame(NewGame(atlas, skel))
	HandleErr(err)
}
