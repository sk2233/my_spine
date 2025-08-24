package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type BoneNode struct {
	Bone     *Bone
	Parent   *BoneNode
	Children []*BoneNode
}

func (n *BoneNode) Update() {
	if n.Parent == nil { // 没有父节点局部坐标就是世界坐标
		n.Bone.WorldRotate = n.Bone.LocalRotate
		n.Bone.WorldPos = n.Bone.LocalPos
		n.Bone.WorldScale = n.Bone.LocalScale
	} else {
		parent := n.Parent.Bone
		switch n.Bone.TransformMode {
		case TransformNormal:
			mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
				Mul3(mgl32.HomogRotate2D(parent.WorldRotate * math.Pi / 180)).
				Mul3(mgl32.Scale2D(parent.WorldScale.X(), parent.WorldScale.Y()))
			n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
			n.Bone.WorldRotate = n.Bone.LocalRotate + parent.WorldRotate
			n.Bone.WorldScale = Vec2Mul(n.Bone.LocalScale, parent.WorldScale)
		case TransformOnlyTranslation:
			mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y())
			n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
			n.Bone.WorldRotate = n.Bone.LocalRotate
			n.Bone.WorldScale = n.Bone.LocalScale
		case TransformNoRotationOrReflection:
			mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
				Mul3(mgl32.Scale2D(parent.WorldScale.X(), parent.WorldScale.Y()))
			n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
			n.Bone.WorldRotate = n.Bone.LocalRotate
			n.Bone.WorldScale = Vec2Mul(n.Bone.LocalScale, parent.WorldScale)
		case TransformNoScale, TransformNoScaleOrReflection:
			mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
				Mul3(mgl32.HomogRotate2D(parent.WorldRotate * math.Pi / 180))
			n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
			n.Bone.WorldRotate = n.Bone.LocalRotate + parent.WorldRotate
			n.Bone.WorldScale = n.Bone.LocalScale
		default:
			panic(fmt.Sprintf("invalid mode: %v", n.Bone.TransformMode))
		}
	}
	n.Bone.Mat3 = mgl32.Translate2D(n.Bone.WorldPos.X(), n.Bone.WorldPos.Y()).
		Mul3(mgl32.HomogRotate2D(n.Bone.WorldRotate * math.Pi / 180)).
		Mul3(mgl32.Scale2D(n.Bone.WorldScale.X(), n.Bone.WorldScale.Y()))
	for _, child := range n.Children {
		child.Update()
	}
}

func (n *BoneNode) ApplyModify() {
	if n.Bone.Modify {
		n.Bone.Mat3 = mgl32.Translate2D(n.Bone.WorldPos.X(), n.Bone.WorldPos.Y()).
			Mul3(mgl32.HomogRotate2D(n.Bone.WorldRotate * math.Pi / 180)).
			Mul3(mgl32.Scale2D(n.Bone.WorldScale.X(), n.Bone.WorldScale.Y()))
		n.Bone.Modify = false // 源头节点只更新 Mat3 即可
		for _, item := range n.Children {
			item.updateWorld() // 会清楚子节点的 Modify
		}
	} else { // TODO 父子节点同时被修改怎么办？ 父节点递归更新会丢掉子节点的更新
		for _, item := range n.Children {
			item.ApplyModify()
		}
	}
}

func (n *BoneNode) updateWorld() {
	parent := n.Parent.Bone // 肯定有父节点
	switch n.Bone.TransformMode {
	case TransformNormal:
		mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
			Mul3(mgl32.HomogRotate2D(parent.WorldRotate * math.Pi / 180)).
			Mul3(mgl32.Scale2D(parent.WorldScale.X(), parent.WorldScale.Y()))
		n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
		n.Bone.WorldRotate = n.Bone.LocalRotate + parent.WorldRotate
		n.Bone.WorldScale = Vec2Mul(n.Bone.LocalScale, parent.WorldScale)
	case TransformOnlyTranslation:
		mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y())
		n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
		n.Bone.WorldRotate = n.Bone.LocalRotate
		n.Bone.WorldScale = n.Bone.LocalScale
	case TransformNoRotationOrReflection:
		mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
			Mul3(mgl32.Scale2D(parent.WorldScale.X(), parent.WorldScale.Y()))
		n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
		n.Bone.WorldRotate = n.Bone.LocalRotate
		n.Bone.WorldScale = Vec2Mul(n.Bone.LocalScale, parent.WorldScale)
	case TransformNoScale, TransformNoScaleOrReflection:
		mat3 := mgl32.Translate2D(parent.WorldPos.X(), parent.WorldPos.Y()).
			Mul3(mgl32.HomogRotate2D(parent.WorldRotate * math.Pi / 180))
		n.Bone.WorldPos = mat3.Mul3x1(n.Bone.LocalPos.Vec3(1)).Vec2()
		n.Bone.WorldRotate = n.Bone.LocalRotate + parent.WorldRotate
		n.Bone.WorldScale = n.Bone.LocalScale
	default:
		panic(fmt.Sprintf("invalid mode: %v", n.Bone.TransformMode))
	}
	n.Bone.Mat3 = mgl32.Translate2D(n.Bone.WorldPos.X(), n.Bone.WorldPos.Y()).
		Mul3(mgl32.HomogRotate2D(n.Bone.WorldRotate * math.Pi / 180)).
		Mul3(mgl32.Scale2D(n.Bone.WorldScale.X(), n.Bone.WorldScale.Y()))
	n.Bone.Modify = false
	for _, item := range n.Children {
		item.updateWorld()
	}
}

type AttachmentItem struct {
	Attachment *Attachment
	Image      *ebiten.Image
	Option     *colorm.DrawTrianglesOptions
	ColorM     colorm.ColorM
}

type Game struct {
	// 原始数据
	Atlas *Atlas
	Skel  *Skel
	// 扩展数据
	Image       image.Image
	BoneRoot    *BoneNode
	OrderSlots  []*Slot
	Attachments map[string]*AttachmentItem
	Pos         mgl32.Vec2 // 调整位置
	// 动画
	AnimIndex      int
	AnimController *AnimController
	// 约束
	ConstraintController *ConstraintController
}

func NewGame(atlas *Atlas, skel *Skel) *Game {
	res := &Game{Atlas: atlas, Skel: skel, Pos: mgl32.Vec2{640, 100}, AnimIndex: 0}
	res.Image = res.loadImage()
	res.BoneRoot = res.calculateBoneRoot()
	res.OrderSlots = res.calculateOrderSlot()
	res.Attachments = res.calculateAttachments()
	res.AnimController = NewAnimController(skel.Animations[res.AnimIndex], skel, res.Attachments)
	res.ConstraintController = NewConstraintController(skel.Bones, skel.PathConstraints, skel.TransformConstraints)
	res.fillPathAttachment()
	return res
}

func (g *Game) Update() error {
	// 按键控制
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.Pos[1]--
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.Pos[1]++
	} else if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.Pos[0]--
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.Pos[0]++
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		fmt.Println(g.Pos)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		g.AnimIndex = (g.AnimIndex - 1 + len(g.Skel.Animations)) % len(g.Skel.Animations)
		anim := g.Skel.Animations[g.AnimIndex]
		g.AnimController = NewAnimController(anim, g.Skel, g.Attachments)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyK) {
		g.AnimIndex = (g.AnimIndex + 1) % len(g.Skel.Animations)
		anim := g.Skel.Animations[g.AnimIndex]
		g.AnimController = NewAnimController(anim, g.Skel, g.Attachments)
	}
	g.BoneRoot.Bone.Pos = g.Pos
	// 初始化数据  运行时数据默认为初始状态，防止动画没有改动为 零值
	for i, slot := range g.Skel.Slots {
		slot.CurrOrder = i
		slot.CurrAttachment = slot.Attachment
		slot.CurrColor = slot.Color
		slot.CurrDarkColor = slot.DarkColor
	}
	for _, bone := range g.Skel.Bones {
		bone.LocalRotate = bone.Rotate
		bone.LocalPos = bone.Pos
		bone.LocalScale = bone.Scale
	}
	for _, attachment := range g.Skel.Skin.Attachments {
		if attachment.Weight {
			attachment.CurrWeightVertices = make([][]*WeightVertex, 0)
			for _, items := range attachment.WeightVertices {
				temp := make([]*WeightVertex, 0)
				for _, item := range items {
					temp = append(temp, &WeightVertex{
						Bone:   item.Bone,
						Offset: item.Offset,
						Weight: item.Weight,
					})
				}
				attachment.CurrWeightVertices = append(attachment.CurrWeightVertices, temp)
			}
		} else {
			attachment.CurrVertices = make([]mgl32.Vec2, len(attachment.Vertices))
			copy(attachment.CurrVertices, attachment.Vertices)
		}
	}
	for _, item := range g.Skel.TransformConstraints {
		item.CurrScaleMix = item.ScaleMix
		item.CurrRotateMix = item.RotateMix
		item.CurrOffsetMix = item.OffsetMix
	}
	// 更新数据
	// 应用动画 都是局部坐标系下的对象或者坐标系无关对象
	g.AnimController.Update()
	// 计算出世界坐标 世界旋转 世界缩放 与 世界矩阵
	g.BoneRoot.Update()
	// 对世界坐标下的对象应用约束
	g.ConstraintController.Update()
	g.BoneRoot.ApplyModify() // 应用世界坐标的修改
	sort.Slice(g.OrderSlots, func(i, j int) bool {
		return g.OrderSlots[i].CurrOrder < g.OrderSlots[j].CurrOrder
	})
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, slot := range g.OrderSlots {
		g.drawSlot(slot, screen)
	}
	ebitenutil.DebugPrint(screen, g.AnimController.GetAnimName())
}

func NewVertex(dx, dy, sx, sy float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:   dx,
		DstY:   dy,
		SrcX:   sx,
		SrcY:   sy,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	}
}

func (g *Game) drawSlot(slot *Slot, screen *ebiten.Image) {
	if slot.Bone < 0 || len(slot.CurrAttachment) == 0 {
		return // 无效值
	}
	item := g.Attachments[AttachmentKey(slot.CurrAttachment, slot.Index)]
	if item.Image == nil {
		return // 无需绘制
	}
	bound := item.Image.Bounds()
	w, h := float32(bound.Dx()), float32(bound.Dy())
	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	attachment := item.Attachment
	currClr := Vec4Mul(slot.CurrColor, slot.CurrDarkColor)
	// 不同组件的展示是 动画控制的，默认会全部展示
	if attachment.Type == AttachmentRegion {
		mat3 := g.Skel.Bones[slot.Bone].Mat3.Mul3(mgl32.Translate2D(attachment.Pos.X(), attachment.Pos.Y())).
			Mul3(mgl32.HomogRotate2D(attachment.Rotate * math.Pi / 180)).
			Mul3(mgl32.Scale2D(attachment.Scale.X(), attachment.Scale.Y()))
		v0 := mat3.Mul3x1(mgl32.Vec3{-w / 2, h / 2, 1}).Vec2()
		v1 := mat3.Mul3x1(mgl32.Vec3{w / 2, h / 2, 1}).Vec2()
		v2 := mat3.Mul3x1(mgl32.Vec3{w / 2, -h / 2, 1}).Vec2()
		v3 := mat3.Mul3x1(mgl32.Vec3{-w / 2, -h / 2, 1}).Vec2()
		vertices = append(vertices, NewVertex(v0.X(), v0.Y(), 0, 0))
		vertices = append(vertices, NewVertex(v1.X(), v1.Y(), w, 0))
		vertices = append(vertices, NewVertex(v2.X(), v2.Y(), w, h))
		vertices = append(vertices, NewVertex(v3.X(), v3.Y(), 0, h))
		indices = []uint16{0, 1, 2, 0, 2, 3}
		currClr = Vec4Mul(currClr, attachment.Color)
	} else if attachment.Type == AttachmentMesh {
		if attachment.Weight {
			for i, uv := range attachment.UVs {
				res := mgl32.Vec2{}
				wv := attachment.CurrWeightVertices[i]
				for _, vec := range wv {
					mat3 := g.Skel.Bones[vec.Bone].Mat3
					temp := mat3.Mul3x1(vec.Offset.Vec3(1)).Vec2()
					res = res.Add(temp.Mul(vec.Weight))
				}
				vertices = append(vertices, NewVertex(res.X(), res.Y(), uv.X()*w, uv.Y()*h))
			}
		} else {
			mat3 := g.Skel.Bones[slot.Bone].Mat3
			for i, uv := range attachment.UVs {
				vec := mat3.Mul3x1(attachment.CurrVertices[i].Vec3(1)).Vec2()
				vertices = append(vertices, NewVertex(vec.X(), vec.Y(), uv.X()*w, uv.Y()*h))
			}
		}
		indices = attachment.Indices
		currClr = Vec4Mul(currClr, attachment.Color)
	} else if attachment.Type == AttachmentClip {
		if attachment.Weight { // 不需要 UV
			for _, wv := range attachment.CurrWeightVertices {
				res := mgl32.Vec2{}
				for _, vec := range wv {
					mat3 := g.Skel.Bones[vec.Bone].Mat3
					temp := mat3.Mul3x1(vec.Offset.Vec3(1)).Vec2()
					res = res.Add(temp.Mul(vec.Weight))
				}
				vertices = append(vertices, NewVertex(res.X(), res.Y(), 0, 0))
			}
		} else {
			mat3 := g.Skel.Bones[slot.Bone].Mat3
			for _, vertex := range attachment.CurrVertices {
				vec := mat3.Mul3x1(vertex.Vec3(1)).Vec2()
				vertices = append(vertices, NewVertex(vec.X(), vec.Y(), 0, 0))
			}
		}
		for i := 2; i < len(vertices); i++ {
			indices = append(indices, 0, uint16(i-1), uint16(i))
		}
	} else {
		panic("unknown attachment type")
	}
	item.ColorM.Reset()
	item.ColorM.Scale(float64(currClr[0]), float64(currClr[1]), float64(currClr[2]), float64(currClr[3]))
	item.Option.Blend = BlendMap[slot.BlendMode]
	colorm.DrawTriangles(screen, vertices, indices, item.Image, item.ColorM, item.Option)
}

func (g *Game) Layout(w, h int) (int, int) {
	return w, h
}

func (g *Game) calculateBoneRoot() *BoneNode {
	nodes := make([]*BoneNode, 0)
	for _, bone := range g.Skel.Bones {
		node := &BoneNode{
			Bone: bone,
		}
		if bone.Parent >= 0 {
			parent := nodes[bone.Parent]
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		}
		nodes = append(nodes, node)
	}
	return nodes[0] // 第一个就是根骨骼
}

func (g *Game) calculateOrderSlot() []*Slot {
	res := make([]*Slot, len(g.Skel.Slots)) // 原始的顺序不要动
	copy(res, g.Skel.Slots)
	return res
}

func (g *Game) calculateAttachments() map[string]*AttachmentItem {
	res := make(map[string]*AttachmentItem)
	for _, item := range g.Skel.Skin.Attachments {
		if item.Type == AttachmentMesh || item.Type == AttachmentRegion || item.Type == AttachmentClip {
			res[AttachmentKey(item.Name, item.Slot)] = &AttachmentItem{
				Attachment: item,
				Image:      g.createImage(item.Path),
				Option:     &colorm.DrawTrianglesOptions{},
				ColorM:     colorm.ColorM{},
			}
		} else {
			res[AttachmentKey(item.Name, item.Slot)] = &AttachmentItem{
				Attachment: item,
			}
		}
	}
	return res
}

func rotate90(img *image.RGBA) *image.RGBA {
	// 获取原图尺寸
	bound := img.Bounds()
	width, height := bound.Dx(), bound.Dy()
	// 创建新图像（旋转后宽高互换）
	res := image.NewRGBA(image.Rect(0, 0, height, width))
	// 像素旋转映射
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			res.Set(height-1-y, x, img.At(x, y))
		}
	}
	return res
}

func rotate270(img *image.RGBA) *image.RGBA {
	// 獲取原始圖片的邊界
	bound := img.Bounds()
	width, height := bound.Dx(), bound.Dy()
	// 創建一個新的圖像，旋轉後寬高互換
	newImg := image.NewRGBA(image.Rect(0, 0, height, width))
	// 遍歷每個像素點，按照270度旋轉的規則重新排列
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 270度旋轉的坐標映射: (x, y) -> (y, width-1-x)
			newImg.Set(y, width-1-x, img.At(x, y))
		}
	}
	return newImg
}

var (
	EmptyImage = ebiten.NewImage(1, 1)
)

func init() {
	EmptyImage.Fill(color.White)
}

func (g *Game) createImage(path string) *ebiten.Image {
	if path == "" {
		return EmptyImage
	}
	for _, item := range g.Atlas.Items {
		if item.Name == path {
			res := image.NewRGBA(image.Rect(0, 0, item.OrigW, item.OrigH))
			draw.Draw(res, image.Rect(item.OrigX, item.OrigY, item.OrigX+item.W, item.OrigY+item.H),
				g.Image, image.Pt(item.X, item.Y), draw.Over)
			if item.Rotate == 90 {
				res = rotate90(res)
			} else if item.Rotate == 270 {
				res = rotate270(res)
			}
			return ebiten.NewImageFromImage(res)
		}
	}
	panic(fmt.Sprintf("image %s not found", path))
}

func (g *Game) loadImage() image.Image {
	file, err := os.Open(BasePath + g.Atlas.Image)
	HandleErr(err)
	img, err := png.Decode(file)
	HandleErr(err)
	file.Close()
	return img
}

func (g *Game) fillPathAttachment() {
	for _, item := range g.Skel.PathConstraints {
		slot := g.Skel.Slots[item.Target]
		item.Attachment = g.Attachments[AttachmentKey(slot.Attachment, slot.Index)].Attachment
		item.Bone = slot.Bone
	}
}
