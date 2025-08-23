package main

import (
	"encoding/json"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"sort"
)

// TODO 现在是先通过 spine_json.jar 把 xxx.skel 转换为 xxx.json 再加载的，后面可以直接加载 xxx.skel

type AnimationData struct {
	Name      string          `json:"name"`
	Duration  float32         `json:"duration"`
	Timelines []*TimelineData `json:"timelines"`
}

const (
	AttachmentRegion   = 0
	AttachmentBoundBox = 1
	AttachmentMesh     = 2
	AttachmentLinkMesh = 3
	AttachmentPath     = 4
	AttachmentPoint    = 5
	AttachmentClip     = 6
)

type AttachmentData struct {
	Name           string `json:"name"`
	AttachmentType int    `json:"attachment_type"`
	// 公共
	Path string `json:"path"`
	// ATTACHMENT_REGION
	Scale    mgl32.Vec2 `json:"scale"`
	Rotation float32    `json:"rotation"`
	Offset   mgl32.Vec2 `json:"offset"`
	// ATTACHMENT_MESH
	UV             []mgl32.Vec2          `json:"uv"`
	VertexIndex    []uint16              `json:"vertex_index"`
	Weight         bool                  `json:"weight"`
	Vertices       []mgl32.Vec2          `json:"vertices"`
	WeightVertices [][]*WeightVertexData `json:"weight_vertices"`
	// ATTACHMENT_CLIPPING 会复用上面的 vertices
	EndSlot      int          `json:"end_slot"` // 作用范围 从所在 slot 到 end_slot 两头都包含
	CurrVertices []mgl32.Vec2 `json:"-"`
}

const (
	TransformNormal                 = 0
	TransformOnlyTranslation        = 1
	TransformNoRotationOrReflection = 2
	TransformNoScale                = 3
	TransformNoScaleOrReflection    = 4
)

type BoneData struct {
	Name          string     `json:"name"`
	ParentIndex   int        `json:"parent_index"`
	Scale         mgl32.Vec2 `json:"scale"`
	Rotation      float32    `json:"rotation"`
	Pos           mgl32.Vec2 `json:"pos"`
	TransformMode int        `json:"transform_mode"`
	Mat3          mgl32.Mat3 `json:"-"`
	CurrRotation  float32    `json:"-"`
	CurrPos       mgl32.Vec2 `json:"-"`
}

type HeaderData struct {
	Hash    string     `json:"hash"`
	Version string     `json:"version"`
	Pos     mgl32.Vec2 `json:"pos"`
	Size    mgl32.Vec2 `json:"size"`
}

type KeyFrameData struct {
	// 公共
	Time float32 `json:"time"`
	// SLOT_ATTACHMENT
	AttachmentName string `json:"attachment_name"` // 切换到目标值
	// BONE_ROTATE
	Rotation float32 `json:"rotation"` // 基于原始的偏移
	// BONE_TRANSLATE
	Offset mgl32.Vec2 `json:"offset"` // 基于原始的偏移
	// SLOT_DRAW_ORDER
	DrawOrder []int `json:"draw_order"` // 目标值
	// SLOT_DEFORM
	Vertexes []mgl32.Vec2 `json:"vertexes"` // 目标值
	// SLOT_COLOR
	Color mgl32.Vec4 `json:"color"` // 目标值
}

type SkinData struct {
	Attachments []*AttachmentData `json:"attachments"`
}

const (
	BlendNormal   = 0
	BlendAdditive = 1
	BlendMultiply = 2
	BlendScreen   = 3
)

var (
	BlendMap = map[int]ebiten.Blend{
		BlendNormal:   ebiten.BlendSourceOver,
		BlendAdditive: ebiten.BlendLighter,
		BlendMultiply: {
			// 源因子：前景颜色乘以背景颜色（取背景颜色作为混合因子）
			BlendFactorSourceRGB:   ebiten.BlendFactorDestinationColor,
			BlendFactorSourceAlpha: ebiten.BlendFactorDestinationAlpha,
			// 目标因子：不保留背景原有颜色（乘以 0）
			BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
			BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
			// 混合操作：加法（仅保留源×目标的结果）
			BlendOperationRGB:   ebiten.BlendOperationAdd,
			BlendOperationAlpha: ebiten.BlendOperationAdd,
		},
		BlendScreen: {
			// 源因子：完全使用前景颜色和透明度
			BlendFactorSourceRGB:   ebiten.BlendFactorOne,
			BlendFactorSourceAlpha: ebiten.BlendFactorOne,
			// 目标因子：背景颜色乘以 (1 - 前景颜色)
			BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceColor,
			BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
			// 混合操作：加法（叠加计算结果）
			BlendOperationRGB:   ebiten.BlendOperationAdd,
			BlendOperationAlpha: ebiten.BlendOperationAdd,
		},
	}
)

type SlotData struct {
	Name           string     `json:"name"`
	BoneIndex      int        `json:"bone_index"`
	Color          mgl32.Vec4 `json:"color"`
	Attachment     string     `json:"attachment"`
	BlendMode      int        `json:"blend_mode"`
	Order          int        `json:"-"`
	CurrAttachment string     `json:"-"`
	CurrColor      mgl32.Vec4 `json:"-"`
}

const (
	TimelineRotate                 = 0
	TimelineTranslate              = 1
	TimelineScale                  = 2
	TimelineShear                  = 3
	TimelineAttachment             = 4
	TimelineColor                  = 5
	TimelineDeform                 = 6
	TimelineEvent                  = 7
	TimelineDrawOrder              = 8
	TimelineIkConstraint           = 9
	TimelineTransformConstraint    = 10
	TimelinePathConstraintPosition = 11
	TimelinePathConstraintSpacing  = 12
	TimelinePathConstraintMix      = 13
	TimelineTwoColor               = 14
)

type TimelineData struct {
	TimelineType int             `json:"timeline_type"`
	KeyFrames    []*KeyFrameData `json:"key_frames"`
	// SLOT_XXX  deform  使用
	SlotIndex int `json:"slot_index"`
	// BONE_XXX 使用
	BoneIndex int `json:"bone_index"`
}

type WeightVertexData struct {
	BoneIndex int        `json:"bone_index"` // 受那个骨骼影响
	Offset    mgl32.Vec2 `json:"offset"`     // 相对于骨骼位置偏移的大小
	Weight    float32    `json:"weight"`     // 受骨骼影响的权重
}

type SkelData struct {
	Header     *HeaderData      `json:"header"`
	Bones      []*BoneData      `json:"bones"`      // 只提供位置信息
	Slots      []*SlotData      `json:"slots"`      // 提供绘制顺序
	Skin       *SkinData        `json:"skin"`       // 提供绘制素材
	Animations []*AnimationData `json:"animations"` // 动态变更要绘制的对象
}

func ParseSkelData(path string) *SkelData {
	res := &SkelData{}
	bytes, err := os.ReadFile(BasePath + path)
	HandleErr(err)
	err = json.Unmarshal(bytes, res)
	HandleErr(err)
	for _, animation := range res.Animations {
		for _, timeline := range animation.Timelines {
			sort.Slice(timeline.KeyFrames, func(i, j int) bool { // 保证顺序性
				return timeline.KeyFrames[i].Time < timeline.KeyFrames[j].Time
			})
		}
	}
	return res
}
