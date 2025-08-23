package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"
)

type BoneNode struct {
	Bone     *Bone
	Parent   *BoneNode
	Children []*BoneNode
}

func (n *BoneNode) Update() {
	mat3 := mgl32.Ident3()
	mat3[4] = -1         // 颠倒上下 其坐标系与 OpenGL 相比是上下颠倒的
	if n.Parent != nil { // 父节点肯定计算完了
		mat3 = n.Parent.Bone.Mat3
	}
	n.Bone.Mat3 = mat3.Mul3(mgl32.Translate2D(n.Bone.CurrPos.X(), n.Bone.CurrPos.Y())).
		Mul3(mgl32.HomogRotate2D(n.Bone.CurrRotation * math.Pi / 180)).
		Mul3(mgl32.Scale2D(n.Bone.Scale.X(), n.Bone.Scale.Y()))
	for _, child := range n.Children {
		child.Update()
	}
}

type OrderSlot struct {
	Slot *Slot
}

type DrawData struct {
	Attachment *Attachment
	Image      *ebiten.Image
	Mat3       mgl32.Mat3
	Option     *colorm.DrawTrianglesOptions
	ColorM     colorm.ColorM
}

func (d *DrawData) GetImage(g *Game) *ebiten.Image {
	if d.Image == nil { // 必须延迟加载
		d.Image = g.getImage(d.Attachment.Path)
	}
	return d.Image
}

type Game struct {
	// 原始数据
	Atlas *Atlas
	Skel  *Skel
	// 扩展数据
	BoneRoot       *BoneNode
	OrderSlots     []*OrderSlot
	DrawData       map[string]*DrawData
	Image          *ebiten.Image
	Pos            mgl32.Vec2
	AnimIndex      int
	AnimController *AnimController
}

func NewGame(atlas *Atlas, skel *Skel) *Game {
	res := &Game{Atlas: atlas, Skel: skel, Pos: mgl32.Vec2{640, -550}, AnimIndex: 0}
	res.BoneRoot = res.calculateBoneRoot()
	res.OrderSlots = res.calculateOrderSlot()
	res.Image = res.loadImage()
	res.DrawData = res.calculateDrawData()
	res.AnimController = NewAnimController(skel.Animations[res.AnimIndex], skel.Bones, skel.Slots, res.DrawData)
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
		g.AnimController = NewAnimController(anim, g.Skel.Bones, g.Skel.Slots, g.DrawData)
		fmt.Println(anim.Name)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyK) {
		g.AnimIndex = (g.AnimIndex + 1) % len(g.Skel.Animations)
		anim := g.Skel.Animations[g.AnimIndex]
		g.AnimController = NewAnimController(anim, g.Skel.Bones, g.Skel.Slots, g.DrawData)
		fmt.Println(anim.Name)
	}
	// 初始化数据
	for _, slot := range g.Skel.Slots {
		slot.CurrAttachment = slot.Attachment
		slot.CurrColor = slot.Color
	}
	for _, bone := range g.Skel.Bones {
		bone.CurrPos = bone.Pos
		bone.CurrRotation = bone.Rotation
	}
	for _, attachment := range g.Skel.Skin.Attachments {
		attachment.CurrVertices = make([]mgl32.Vec2, len(attachment.Vertices))
		copy(attachment.CurrVertices, attachment.Vertices)
	}
	// 更新数据
	g.AnimController.Update()
	g.BoneRoot.Bone.Pos = g.Pos
	g.BoneRoot.Bone.Scale = mgl32.Vec2{0.5, 0.5}
	g.BoneRoot.Update()
	sort.Slice(g.OrderSlots, func(i, j int) bool {
		return g.OrderSlots[i].Slot.Order < g.OrderSlots[j].Slot.Order
	})
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, slot := range g.OrderSlots {
		g.drawSlot(slot.Slot, screen)
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
	if slot.BoneIndex < 0 || len(slot.CurrAttachment) == 0 {
		return // 无效值
	}
	drawData := g.DrawData[slot.CurrAttachment]
	img := drawData.GetImage(g)
	bound := img.Bounds()
	w, h := float32(bound.Dx()), float32(bound.Dy())
	vertices := make([]ebiten.Vertex, 0)
	indices := make([]uint16, 0)
	// 不同组件的展示是 动画控制的，默认会全部展示
	if drawData.Attachment.AttachmentType == AttachmentRegion {
		// 不仅位置不对，方向也反了
		mat3 := g.Skel.Bones[slot.BoneIndex].Mat3.Mul3(drawData.Mat3)
		v0 := mat3.Mul3x1(mgl32.Vec3{-w / 2, h / 2, 1}).Vec2()
		v1 := mat3.Mul3x1(mgl32.Vec3{w / 2, h / 2, 1}).Vec2()
		v2 := mat3.Mul3x1(mgl32.Vec3{w / 2, -h / 2, 1}).Vec2()
		v3 := mat3.Mul3x1(mgl32.Vec3{-w / 2, -h / 2, 1}).Vec2()
		vertices = append(vertices, NewVertex(v0.X(), v0.Y(), 0, 0))
		vertices = append(vertices, NewVertex(v1.X(), v1.Y(), w, 0))
		vertices = append(vertices, NewVertex(v2.X(), v2.Y(), w, h))
		vertices = append(vertices, NewVertex(v3.X(), v3.Y(), 0, h))
		indices = []uint16{0, 1, 2, 0, 2, 3}
	} else if drawData.Attachment.AttachmentType == AttachmentMesh {
		attachment := drawData.Attachment
		if attachment.Weight {
			for i, uv := range attachment.UV {
				vec := mgl32.Vec2{}
				wv := attachment.WeightVertices[i]
				for _, item := range wv {
					mat3 := g.Skel.Bones[item.BoneIndex].Mat3
					temp := mat3.Mul3x1(item.Offset.Vec3(1)).Vec2()
					vec = vec.Add(temp.Mul(item.Weight))
				}
				vertices = append(vertices, NewVertex(vec.X(), vec.Y(), uv.X()*w, uv.Y()*h))
			}
		} else {
			mat3 := g.Skel.Bones[slot.BoneIndex].Mat3
			for i, uv := range attachment.UV {
				vec := mat3.Mul3x1(attachment.CurrVertices[i].Vec3(1)).Vec2()
				vertices = append(vertices, NewVertex(vec.X(), vec.Y(), uv.X()*w, uv.Y()*h))
			}
		}
		indices = attachment.VertexIndex
	} else if drawData.Attachment.AttachmentType == AttachmentClip {
		attachment := drawData.Attachment // TODO 剪切蒙版实现
		mat3 := g.Skel.Bones[slot.BoneIndex].Mat3
		for _, item := range attachment.CurrVertices {
			vec := mat3.Mul3x1(item.Vec3(1)).Vec2()
			vertices = append(vertices, NewVertex(vec.X(), vec.Y(), 0, 0))
		}
		for i := 2; i < len(vertices); i++ {
			indices = append(indices, 0, uint16(i-1), uint16(i))
		}
	} else {
		panic("unknown attachment type")
	}
	drawData.ColorM.Reset()
	drawData.ColorM.Scale(float64(slot.CurrColor[0]), float64(slot.CurrColor[1]), float64(slot.CurrColor[2]), float64(slot.CurrColor[3]))
	drawData.Option.Blend = BlendMap[slot.BlendMode]
	colorm.DrawTriangles(screen, vertices, indices, img, drawData.ColorM, drawData.Option)
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
		if bone.ParentIndex >= 0 {
			parent := nodes[bone.ParentIndex]
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		}
		nodes = append(nodes, node)
	}
	return nodes[0] // 第一个就是根骨骼
}

func (g *Game) calculateOrderSlot() []*OrderSlot {
	res := make([]*OrderSlot, 0)
	for i, slot := range g.Skel.Slots {
		slot.Order = i // 给个默认值
		res = append(res, &OrderSlot{slot})
	}
	return res
}

func (g *Game) calculateDrawData() map[string]*DrawData {
	res := make(map[string]*DrawData)
	for _, item := range g.Skel.Skin.Attachments {
		res[item.Name] = &DrawData{
			Attachment: item,
			Option: &colorm.DrawTrianglesOptions{
				ColorScaleMode: 0,
			},
			ColorM: colorm.ColorM{},
		}
		if item.AttachmentType == AttachmentRegion {
			res[item.Name].Mat3 = mgl32.Translate2D(item.Offset.X(), item.Offset.Y()).
				Mul3(mgl32.HomogRotate2D(item.Rotation * math.Pi / 180)).
				Mul3(mgl32.Scale2D(item.Scale.X(), item.Scale.Y()))
		}
	}
	return res
}

func rotate90(img *ebiten.Image) *ebiten.Image {
	// 获取原图尺寸
	bound := img.Bounds()
	width, height := bound.Dx(), bound.Dy()
	// 创建新图像（旋转后宽高互换）
	res := ebiten.NewImage(height, width)
	// 像素旋转映射
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			res.Set(height-1-y, x, img.At(x, y))
		}
	}
	return res
}

func rotate270(img *ebiten.Image) *ebiten.Image {
	// 獲取原始圖片的邊界
	bound := img.Bounds()
	width, height := bound.Dx(), bound.Dy()
	// 創建一個新的圖像，旋轉後寬高互換
	newImg := ebiten.NewImage(height, width)
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

func (g *Game) getImage(path string) *ebiten.Image {
	if path == "" {
		return EmptyImage
	}
	for _, item := range g.Atlas.Items {
		if item.Name == path {
			res := ebiten.NewImage(item.OrigW, item.OrigH)
			draw.Draw(res, image.Rect(item.OrigX, item.OrigY, item.OrigX+item.W, item.OrigY+item.H),
				g.Image, image.Pt(item.X, item.Y), draw.Over)
			if item.Rotate == 90 {
				res = rotate90(res)
			} else if item.Rotate == 270 {
				res = rotate270(res)
			}
			return res
		}
	}
	panic(fmt.Sprintf("image %s not found", path))
}

func (g *Game) loadImage() *ebiten.Image {
	file, err := os.Open(BasePath + g.Atlas.Image)
	HandleErr(err)
	img, err := png.Decode(file)
	HandleErr(err)
	file.Close()
	return ebiten.NewImageFromImage(img)
}
