package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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
		n.Bone.WorldPos = n.Bone.LocalPos
		n.Bone.Mat2 = Rotate(n.Bone.LocalRotate).Mul2(Scale(n.Bone.LocalScale))
	} else {
		parent := n.Parent.Bone // 坐标计算毕竟是在父坐标系还是会受影响的
		n.Bone.WorldPos = parent.Mat2.Mul2x1(n.Bone.LocalPos).Add(parent.WorldPos)
		switch n.Bone.TransformMode {
		case TransformNormal:
			n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate)).Mul2(Scale(n.Bone.LocalScale))
		case TransformOnlyTranslation:
			n.Bone.Mat2 = Rotate(n.Bone.LocalRotate).Mul2(Scale(n.Bone.LocalScale))
		case TransformNoRotationOrReflection:
			rotate := GetRotate(parent.Mat2) // 移除父对象的旋转量
			n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate - rotate)).Mul2(Scale(n.Bone.LocalScale))
		case TransformNoScale, TransformNoScaleOrReflection:
			scale := GetScale(parent.Mat2) // 移除父对象的缩放量
			n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate)).Mul2(Scale(Vec2Div(n.Bone.LocalScale, scale)))
		default:
			panic(fmt.Sprintf("invalid mode: %v", n.Bone.TransformMode))
		} // 参考原项目必须使用矩阵变换，非等比缩放影响必须使用矩阵累加
	}
	for _, child := range n.Children {
		child.Update()
	}
}

func (n *BoneNode) ApplyModify() {
	if n.Bone.Modify {
		n.Bone.Modify = false // 源头节点只更新 Mat3 即可
		for _, item := range n.Children {
			item.updateWorld() // 会清除子节点的 Modify
		}
	} else { // TODO 父子节点同时被修改怎么办？ 父节点递归更新会丢掉子节点的更新
		for _, item := range n.Children {
			item.ApplyModify()
		}
	}
}

func (n *BoneNode) updateWorld() {
	parent := n.Parent.Bone // 坐标计算毕竟是在父坐标系还是会受影响的
	n.Bone.WorldPos = parent.Mat2.Mul2x1(n.Bone.LocalPos).Add(parent.WorldPos)
	switch n.Bone.TransformMode {
	case TransformNormal:
		n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate)).Mul2(Scale(n.Bone.LocalScale))
	case TransformOnlyTranslation:
		n.Bone.Mat2 = Rotate(n.Bone.LocalRotate).Mul2(Scale(n.Bone.LocalScale))
	case TransformNoRotationOrReflection:
		rotate := GetRotate(parent.Mat2) // 移除父对象的旋转量
		n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate - rotate)).Mul2(Scale(n.Bone.LocalScale))
	case TransformNoScale, TransformNoScaleOrReflection:
		scale := GetScale(parent.Mat2) // 移除父对象的缩放量  TODO 与标注实现不一致
		n.Bone.Mat2 = parent.Mat2.Mul2(Rotate(n.Bone.LocalRotate)).Mul2(Scale(Vec2Div(n.Bone.LocalScale, scale)))
	default:
		panic(fmt.Sprintf("invalid mode: %v", n.Bone.TransformMode))
	}
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
	//g.ConstraintController.Update()
	//g.BoneRoot.ApplyModify() // 应用世界坐标的修改
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
		bone := g.Skel.Bones[slot.Bone]
		worldPos := bone.Mat2.Mul2x1(attachment.Pos).Add(bone.WorldPos)
		mat2 := bone.Mat2.Mul2(Rotate(attachment.Rotate)).Mul2(Scale(attachment.Scale))
		v0 := mat2.Mul2x1(mgl32.Vec2{-w / 2, h / 2}).Add(worldPos)
		v1 := mat2.Mul2x1(mgl32.Vec2{w / 2, h / 2}).Add(worldPos)
		v2 := mat2.Mul2x1(mgl32.Vec2{w / 2, -h / 2}).Add(worldPos)
		v3 := mat2.Mul2x1(mgl32.Vec2{-w / 2, -h / 2}).Add(worldPos)
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
				for _, vec := range attachment.CurrWeightVertices[i] {
					bone := g.Skel.Bones[vec.Bone]
					temp := bone.Mat2.Mul2x1(vec.Offset).Add(bone.WorldPos)
					res = res.Add(temp.Mul(vec.Weight))
				}
				vertices = append(vertices, NewVertex(res.X(), res.Y(), uv.X()*w, uv.Y()*h))
			}
		} else {
			bone := g.Skel.Bones[slot.Bone]
			for i, uv := range attachment.UVs {
				vec := bone.Mat2.Mul2x1(attachment.CurrVertices[i]).Add(bone.WorldPos)
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
					bone := g.Skel.Bones[vec.Bone]
					temp := bone.Mat2.Mul2x1(vec.Offset).Add(bone.WorldPos)
					res = res.Add(temp.Mul(vec.Weight))
				}
				vertices = append(vertices, NewVertex(res.X(), res.Y(), 0, 0))
			}
		} else {
			bone := g.Skel.Bones[slot.Bone]
			for _, vertex := range attachment.CurrVertices {
				vec := bone.Mat2.Mul2x1(vertex).Add(bone.WorldPos)
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

// rotate180 对图像进行180度旋转，返回旋转后的图像
func rotate180(src image.Image) image.Image {
	// 获取原始图像的尺寸
	bound := src.Bounds()
	width, height := bound.Dx(), bound.Dy()
	// 创建目标图像（RGBA格式，支持读写像素）
	dst := image.NewRGBA(bound)
	// 遍历原始图像的每个像素
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dst.Set(width-1-x, height-1-y, src.At(x, y))
		}
	}
	return dst
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
			switch item.Rotate {
			case 0:
				return ebiten.NewImageFromImage(res)
			case 90:
				return ebiten.NewImageFromImage(rotate90(res))
			case 180:
				return ebiten.NewImageFromImage(rotate180(res))
			case 270:
				return ebiten.NewImageFromImage(rotate270(res))
			default:
				panic(fmt.Sprintf("unknown rotate %d", item.Rotate))
			}
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
