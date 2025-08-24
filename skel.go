package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/ebiten/v2"
	"io"
	"math"
	"os"
	"sort"
)

type SkelHeader struct {
	Hash    string // 校验文件
	Version string // 校验版本
	Pos     mgl32.Vec2
	Size    mgl32.Vec2
}

const (
	TransformNormal                 = 0
	TransformOnlyTranslation        = 1
	TransformNoRotationOrReflection = 2
	TransformNoScale                = 3
	TransformNoScaleOrReflection    = 4
)

type Bone struct {
	Name          string
	Parent        int
	Rotate        float32
	Pos           mgl32.Vec2
	Scale         mgl32.Vec2
	Shear         mgl32.Vec2
	Length        float32 // IK 使用的 暂时没用
	TransformMode uint8   // 继承父节点那些变换属性
	SkinRequire   bool    // 多皮肤使用的 暂时没用
	// 运行时数据
	CurrRotate      float32
	CurrPos         mgl32.Vec2
	CurrScale       mgl32.Vec2
	NormalMat3      mgl32.Mat3 // TransformNormal
	TranslationMat3 mgl32.Mat3 // TransformOnlyTranslation
	NoRotateMat3    mgl32.Mat3 // TransformNoRotationOrReflection
	NoScaleMat3     mgl32.Mat3 // TransformNoScale TransformNoScaleOrReflection
}

const (
	BlendNormal   = 0
	BlendAdditive = 1
	BlendMultiply = 2
	BlendScreen   = 3
)

var (
	BlendMap = map[uint8]ebiten.Blend{
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

type Slot struct {
	Name             string
	Bone             int
	Color, DarkColor mgl32.Vec4 // Color 最终要 * DarkColor 进行调整
	Attachment       string
	BlendMode        uint8
	Index            int
	// 运行时值
	CurrOrder      int
	CurrAttachment string
	CurrColor      mgl32.Vec4
}

type WeightVertex struct {
	Bone   int        // 受那个骨骼影响
	Offset mgl32.Vec2 // 相对于骨骼位置偏移的大小
	Weight float32    // 受骨骼影响的权重
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

type Attachment struct {
	Name           string
	Slot           int // name + slot 才是唯一的
	Type           uint8
	Path           string
	Color          mgl32.Vec4
	Weight         bool
	Vertices       []mgl32.Vec2
	WeightVertices [][]*WeightVertex
	// AttachmentRegion
	Rotate float32
	Pos    mgl32.Vec2
	Scale  mgl32.Vec2
	Size   mgl32.Vec2 // 用来确定中心点与 UV 计算
	// AttachmentMesh
	UVs        []mgl32.Vec2
	Indices    []uint16
	HullLength int // 凸包体边数 暂不使用
	// AttachmentPath
	Close         bool
	ConstantSpeed bool
	Lengths       []float32
	// AttachmentClip
	EndSlot int
	// 运行时数据
	CurrVertices       []mgl32.Vec2
	CurrWeightVertices [][]*WeightVertex
}

type Skin struct {
	Attachments []*Attachment
}

const (
	CurveLinear  = 0
	CurveStepped = 1
	CurveBezier  = 2
)

type Curve struct {
	Type uint8
	Data [2]mgl32.Vec2
}

type KeyFrame struct {
	Time  float32
	Curve *Curve
	// TimelineAttachment
	Attachment string
	// TimelineColor
	Color mgl32.Vec4
	// TimelineRotate
	Rotate float32
	// TimelineTranslate
	Offset mgl32.Vec2
	// TimelineScale
	Scale mgl32.Vec2
	// Timeline
	Shear mgl32.Vec2
	// TimelineDrawOrder
	DrawOrder []int // 对应槽位的新位置
	// TimelineDeform
	Weight       bool           // 要修改的是不是 WeightVertex
	Deform       []mgl32.Vec2   // 不是 WeightVertex 针对每个点的偏移
	WeightDeform [][]mgl32.Vec2 // 是 WeightVertex 针对每个骨骼偏移的修改
}

const (
	SlotAttachment = 0
	SlotColor      = 1
)

const (
	BoneRotate    = 0
	BoneTranslate = 1
	BoneScale     = 2
	BoneShear     = 3
)

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

type Timeline struct {
	Type       uint8
	Slot       int
	Bone       int
	Attachment string
	KeyFrames  []*KeyFrame
}

type Animation struct {
	Name      string
	Timelines []*Timeline
	Duration  float32
}

type Skel struct {
	Header     *SkelHeader
	Bones      []*Bone
	Slots      []*Slot
	Skin       *Skin // 暂时只支持默认皮肤，不支持换肤
	Animations []*Animation
}

func ParseSkel(path string) *Skel {
	reader, err := os.Open(BasePath + path)
	HandleErr(err)
	header := parseSkelHeader(reader)
	strings := parseStrings(reader)
	bones := parseBones(reader)
	slots := parseSlots(reader, strings)
	skipConstraints(reader) // 暂时不使用约束，跳过
	skin := parseSkin(reader, strings)
	skipEvents(reader, strings)
	animations := parseAnimations(reader, strings, slots, skin)
	return &Skel{
		Header:     header,
		Bones:      bones,
		Slots:      slots,
		Skin:       skin,
		Animations: animations,
	}
}

func parseAnimations(reader io.Reader, strings []string, slots []*Slot, skin *Skin) []*Animation {
	count := readInt(reader)
	animations := make([]*Animation, 0)
	for i := 0; i < count; i++ {
		animations = append(animations, parseAnimation(reader, strings, slots, skin))
	}
	return animations
}

func parseAnimation(reader io.Reader, strings []string, slots []*Slot, skin *Skin) *Animation {
	name := readStr(reader)
	timelines := make([]*Timeline, 0)
	// slot
	sCount := readInt(reader)     // slotTimeline
	for i := 0; i < sCount; i++ { // 多个 slot 分组
		slot := readInt(reader)
		tCount := readInt(reader)
		for j := 0; j < tCount; j++ { // 每个 slot 多个 timeline
			temp := &Timeline{
				Slot: slot,
			}
			type0 := readU8(reader)
			fCount := readInt(reader) // 每个 timeline 多帧动画
			switch type0 {
			case SlotAttachment:
				temp.Type = TimelineAttachment
				for k := 0; k < fCount; k++ {
					temp.KeyFrames = append(temp.KeyFrames, &KeyFrame{
						Time:       readF4(reader),
						Attachment: readRefStr(reader, strings),
					})
				}
			case SlotColor:
				temp.Type = TimelineColor
				for k := 0; k < fCount; k++ {
					keyFrame := &KeyFrame{
						Time:  readF4(reader),
						Color: readClr(reader),
					}
					if k < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
			default:
				panic(fmt.Errorf("unknown slot type: %v", type0))
			}
			timelines = append(timelines, temp)
		}

	}
	// bone
	bCount := readInt(reader)     // boneTimeline
	for i := 0; i < bCount; i++ { // 多个 bone 分组
		bone := readInt(reader)
		tCount := readInt(reader)
		for j := 0; j < tCount; j++ { // 每组多个 timeline
			temp := &Timeline{
				Bone: bone,
			}
			type0 := readU8(reader)
			fCount := readInt(reader) // 每个 timeline 多帧动画
			switch type0 {
			case BoneRotate:
				temp.Type = TimelineRotate
				for k := 0; k < fCount; k++ {
					keyFrame := &KeyFrame{
						Time:   readF4(reader),
						Rotate: readF4(reader),
					}
					if k < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
			case BoneTranslate:
				temp.Type = TimelineTranslate
				for k := 0; k < fCount; k++ {
					keyFrame := &KeyFrame{
						Time: readF4(reader),
						Offset: mgl32.Vec2{
							readF4(reader),
							readF4(reader),
						},
					}
					if k < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
			case BoneScale:
				temp.Type = TimelineScale
				for k := 0; k < fCount; k++ {
					keyFrame := &KeyFrame{
						Time: readF4(reader),
						Scale: mgl32.Vec2{
							readF4(reader),
							readF4(reader),
						},
					}
					if k < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
			case BoneShear:
				temp.Type = TimelineShear
				for k := 0; k < fCount; k++ {
					keyFrame := &KeyFrame{
						Time: readF4(reader),
						Shear: mgl32.Vec2{
							readF4(reader),
							readF4(reader),
						},
					}
					if k < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
			default:
				panic(fmt.Errorf("unknown bone type: %v", type0))
			}
			timelines = append(timelines, temp)
		}
	}
	// IK constraint skip
	count := readInt(reader)
	for i := 0; i < count; i++ {
		index := readInt(reader)
		fCount := readInt(reader)
		for j := 0; j < fCount; j++ {
			other := readByte(reader, 3*4+3)
			if j < fCount-1 {
				readCurve(reader)
			}
			Use(other)
		}
		Use(index, fCount)
	}
	// Transform constraint skip
	count = readInt(reader)
	for i := 0; i < count; i++ {
		index := readInt(reader)
		fCount := readInt(reader)
		for j := 0; j < fCount; j++ {
			other := readByte(reader, 5*4)
			if j < fCount-1 {
				readCurve(reader)
			}
			Use(other)
		}
		Use(index, fCount)
	}
	// Path constraint skip
	count = readInt(reader)
	if count > 0 {
		panic(fmt.Errorf("invalid timeline count: %v", count))
	}
	// Deform
	attachments := make(map[string]*Attachment)
	for _, attachment := range skin.Attachments {
		if attachment.Type == AttachmentMesh || attachment.Type == AttachmentPath || attachment.Type == AttachmentClip {
			attachments[AttachmentKey(attachment.Name, attachment.Slot)] = attachment
		}
	}
	count = readInt(reader)
	for i := 0; i < count; i++ { // 按 skin 分组
		if readInt(reader) != 0 { // 只支持默认皮肤
			panic(fmt.Errorf("invalid skin: %v", skin))
		}
		sCount = readInt(reader)
		for j := 0; j < sCount; j++ { // 按 slot 分组
			slot := readInt(reader)
			aCount := readInt(reader)
			for k := 0; k < aCount; k++ { // 按 attachment 分组
				temp := &Timeline{
					Type:       TimelineDeform,
					Slot:       slot,
					Attachment: readRefStr(reader, strings),
				}
				key := AttachmentKey(temp.Attachment, temp.Slot)
				attachment := attachments[key]
				if attachment == nil {
					panic(fmt.Errorf("not find attachment %s", key))
				}
				size := 0
				if attachment.Weight {
					for _, items := range attachment.WeightVertices {
						size += len(items)
					}
				} else {
					size = len(attachment.Vertices)
				}
				fCount := readInt(reader)
				for m := 0; m < fCount; m++ { // 每个 timeline 多帧动画
					keyFrame := &KeyFrame{
						Time:   readF4(reader),
						Weight: attachment.Weight,
					}
					deform := make([]mgl32.Vec2, size)
					cCount := readInt(reader)
					if cCount > 0 {
						start := readInt(reader)
						end := start + cCount
						for n := start; n < end; n++ {
							deform[n/2][n%2] = readF4(reader)
						}
					}
					if attachment.Weight {
						weightDeform := make([][]mgl32.Vec2, 0)
						idx := 0
						for _, items := range attachment.WeightVertices {
							weightDeform = append(weightDeform, deform[idx:idx+len(items)])
							idx += len(items)
						}
						keyFrame.WeightDeform = weightDeform
					} else {
						keyFrame.Deform = deform
					}
					if m < fCount-1 {
						keyFrame.Curve = readCurve(reader)
					}
					temp.KeyFrames = append(temp.KeyFrames, keyFrame)
				}
				timelines = append(timelines, temp)
			}
		}
	}
	// Draw order
	count = readInt(reader)
	if count > 0 {
		temp := &Timeline{
			Type: TimelineDrawOrder,
		}
		size := len(slots)
		for i := 0; i < count; i++ {
			time := readF4(reader)
			cCount := readInt(reader)
			drawOrder := make([]int, size)
			for j := 0; j < size; j++ {
				drawOrder[j] = -1 // 先全部初始化为 -1
			}
			has := make(map[int]bool)
			for j := 0; j < cCount; j++ { // 先分配有偏移的，剩下的顺序分配
				idx := readInt(reader)
				offset := int(int32(readInt(reader))) // 保留负号
				drawOrder[idx] = idx + offset
				has[idx+offset] = true
			}
			freeIdx := 0
			for idx := 0; idx < size; idx++ {
				if drawOrder[idx] >= 0 {
					continue // 已经分配了
				}
				for has[freeIdx] {
					freeIdx++
				}
				drawOrder[idx] = freeIdx
				has[freeIdx] = true
			}
			temp.KeyFrames = append(temp.KeyFrames, &KeyFrame{
				Time:      time,
				DrawOrder: drawOrder,
			})
		}
		timelines = append(timelines, temp)
	}
	// Event skip
	count = readInt(reader)
	for i := 0; i < count; i++ {
		time := readF4(reader)
		index := readInt(reader)
		intValue := readInt(reader)
		floatValue := readF4(reader)
		if readBool(reader) {
			readStr(reader)
		}
		Use(time, index, intValue, floatValue)
	}
	duration := float32(0)
	for _, timeline := range timelines {
		sort.Slice(timeline.KeyFrames, func(i, j int) bool {
			return timeline.KeyFrames[i].Time < timeline.KeyFrames[j].Time
		})
		duration = max(duration, timeline.KeyFrames[len(timeline.KeyFrames)-1].Time)
	}
	return &Animation{
		Name:      name,
		Timelines: timelines,
		Duration:  duration,
	}
}

func readCurve(reader io.Reader) *Curve {
	res := &Curve{}
	res.Type = readU8(reader)
	switch res.Type {
	case CurveLinear, CurveStepped:
		return res
	case CurveBezier:
		readAny(reader, &res.Data)
		return res
	default:
		panic(fmt.Sprintf("unknown curve type: %v", res.Type))
	}
}

func skipEvents(reader io.Reader, strings []string) {
	count := readInt(reader)
	for i := 0; i < count; i++ {
		name := readRefStr(reader, strings)
		intValue := readInt(reader)
		floatValue := readF4(reader)
		stringValue := readStr(reader)
		audioPath := readStr(reader)
		if len(audioPath) > 0 {
			volume := readF4(reader)
			balance := readF4(reader)
			Use(volume, balance)
		}
		Use(name, intValue, floatValue, stringValue, audioPath)
	}
}

func parseSkin(reader io.Reader, strings []string) *Skin {
	attachments := make([]*Attachment, 0)
	slotCount := readInt(reader)
	for i := 0; i < slotCount; i++ {
		slot := readInt(reader)
		attachmentCount := readInt(reader)
		for j := 0; j < attachmentCount; j++ {
			if temp := parseAttachment(reader, slot, strings); temp != nil {
				attachments = append(attachments, temp)
			}
		}
	}
	if readInt(reader) > 0 {
		panic("not implemented more skin")
	}
	return &Skin{
		Attachments: attachments,
	}
}

func parseAttachment(reader io.Reader, slot int, strings []string) *Attachment {
	defaultName := readRefStr(reader, strings)
	name := readRefStr(reader, strings)
	if len(name) == 0 {
		name = defaultName
	}
	attachmentType := readU8(reader)
	res := &Attachment{
		Name: name,
		Slot: slot,
		Type: attachmentType,
	}
	switch attachmentType {
	case AttachmentRegion:
		res.Path = readRefStr(reader, strings)
		if len(res.Path) == 0 {
			res.Path = name
		}
		res.Rotate = readF4(reader)
		temp := [3]mgl32.Vec2{}
		readAny(reader, &temp)
		res.Pos = temp[0]
		res.Scale = temp[1]
		res.Size = temp[2]
		res.Color = readClr(reader)
	case AttachmentMesh:
		res.Path = readRefStr(reader, strings)
		if len(res.Path) == 0 {
			res.Path = name
		}
		res.Color = readClr(reader)
		vCount := readInt(reader)
		for i := 0; i < vCount; i++ {
			res.UVs = append(res.UVs, mgl32.Vec2{
				readF4(reader),
				readF4(reader),
			})
		}
		iCount := readInt(reader)
		for i := 0; i < iCount; i++ {
			res.Indices = append(res.Indices, readU16(reader))
		}
		res.Weight = readBool(reader)
		res.Vertices, res.WeightVertices = parseVertices(reader, vCount, res.Weight)
		res.HullLength = readInt(reader)
	case AttachmentPath: // 生成运动路径 依赖 Path constraints 与 路径动画 生效
		res.Close = readBool(reader)
		res.ConstantSpeed = readBool(reader)
		count := readInt(reader)
		res.Weight = readBool(reader)
		res.Vertices, res.WeightVertices = parseVertices(reader, count, res.Weight)
		for i := 0; i < count/3; i++ {
			res.Lengths = append(res.Lengths, readF4(reader))
		}
		return res
	case AttachmentClip:
		res.EndSlot = readInt(reader)
		count := readInt(reader)
		res.Weight = readBool(reader)
		res.Vertices, res.WeightVertices = parseVertices(reader, count, res.Weight)
		return res
	default:
		panic(fmt.Sprintf("unknown attachment type: %v", attachmentType))
	}
	return res
}

func parseVertices(reader io.Reader, count int, weight bool) ([]mgl32.Vec2, [][]*WeightVertex) {
	vertices := make([]mgl32.Vec2, 0)
	weightVertices := make([][]*WeightVertex, 0)
	for i := 0; i < count; i++ {
		if weight {
			boneCount := readInt(reader)
			temp := make([]*WeightVertex, 0)
			for j := 0; j < boneCount; j++ {
				temp = append(temp, &WeightVertex{
					Bone:   readInt(reader),
					Offset: mgl32.Vec2{readF4(reader), readF4(reader)},
					Weight: readF4(reader),
				})
			}
			weightVertices = append(weightVertices, temp)
		} else {
			vertices = append(vertices, mgl32.Vec2{
				readF4(reader),
				readF4(reader),
			})
		}
	}
	return vertices, weightVertices
}

func readU16(reader io.Reader) uint16 {
	temp := readByte(reader, 2)
	return binary.BigEndian.Uint16(temp)
}

func skipConstraints(reader io.Reader) {
	// IK constraints
	count := readInt(reader)
	for i := 0; i < count; i++ {
		name := readStr(reader)
		order := readInt(reader)
		skinRequire := readBool(reader)
		boneCount := readInt(reader)
		for j := 0; j < boneCount; j++ {
			bone := readInt(reader)
			Use(bone)
		}
		target := readInt(reader)
		other := readByte(reader, 4*2+4)
		Use(name, order, skinRequire, target, other)
	}
	// Transform constraints
	count = readInt(reader)
	for i := 0; i < count; i++ {
		name := readStr(reader)
		order := readInt(reader)
		skinRequire := readBool(reader)
		boneCount := readInt(reader)
		for j := 0; j < boneCount; j++ {
			bone := readInt(reader)
			Use(bone)
		}
		target := readInt(reader)
		other := readByte(reader, 2+10*4)
		Use(name, order, skinRequire, target, other)
	}
	// Path constraints
	count = readInt(reader)
	for i := 0; i < count; i++ {
		name := readStr(reader)
		order := readInt(reader)
		skinRequire := readBool(reader)
		boneCount := readInt(reader)
		for j := 0; j < boneCount; j++ {
			bone := readInt(reader)
			Use(bone)
		}
		target := readInt(reader)
		other := readByte(reader, 3+5*4)
		Use(name, order, skinRequire, target, other)
	}
}

func parseSlots(reader io.Reader, strings []string) []*Slot {
	res := make([]*Slot, 0)
	count := readInt(reader)
	for i := 0; i < count; i++ {
		slot := parseSlot(reader, strings)
		slot.Index = i
		res = append(res, slot)
	}
	return res
}

func parseSlot(reader io.Reader, strings []string) *Slot {
	name := readStr(reader)
	bone := readInt(reader)
	color := readClr(reader)
	darkColor := readClr(reader)
	attachment := readRefStr(reader, strings)
	blendMode := readU8(reader)
	return &Slot{
		Name:       name,
		Bone:       bone,
		Color:      color,
		DarkColor:  darkColor,
		Attachment: attachment,
		BlendMode:  blendMode,
	}
}

func readRefStr(reader io.Reader, strings []string) string {
	idx := readInt(reader) - 1
	if idx < 0 {
		return ""
	}
	return strings[idx]
}

func readClr(reader io.Reader) mgl32.Vec4 {
	data := readByte(reader, 4)
	return mgl32.Vec4{
		float32(data[0]) / 0xFF,
		float32(data[1]) / 0xFF,
		float32(data[2]) / 0xFF,
		float32(data[3]) / 0xFF,
	}
}

func parseBones(reader io.Reader) []*Bone {
	count := readInt(reader)
	res := make([]*Bone, 0)
	for i := 0; i < count; i++ {
		res = append(res, parseBone(reader, i == 0))
	}
	return res
}

func parseBone(reader io.Reader, first bool) *Bone {
	name := readStr(reader)
	parent := -1
	if !first {
		parent = readInt(reader)
	}
	rotate := readF4(reader)
	temp := [3]mgl32.Vec2{}
	readAny(reader, &temp)
	length := readF4(reader)
	mode := readU8(reader)
	skip := readBool(reader)
	return &Bone{
		Name:          name,
		Parent:        parent,
		Rotate:        rotate,
		Pos:           temp[0],
		Scale:         temp[1],
		Shear:         temp[2],
		Length:        length,
		TransformMode: mode,
		SkinRequire:   skip,
	}
}

func readF4(reader io.Reader) float32 {
	bs := readByte(reader, 4)
	temp := binary.BigEndian.Uint32(bs)
	return math.Float32frombits(temp)
}

func parseStrings(reader io.Reader) []string {
	count := readInt(reader)
	res := make([]string, 0)
	for i := 0; i < count; i++ {
		res = append(res, readStr(reader))
	}
	return res
}

func parseSkelHeader(reader io.Reader) *SkelHeader {
	hash := readStr(reader)
	version := readStr(reader)
	temp := [2]mgl32.Vec2{}
	readAny(reader, &temp)
	if readBool(reader) { // 可有可无的数据，就不要导出了
		panic("not support nonessential")
	}
	return &SkelHeader{
		Hash:    hash,
		Version: version,
		Pos:     temp[0],
		Size:    temp[1],
	}
}

func readBool(reader io.Reader) bool {
	return readU8(reader) == 1
}

func readAny(reader io.Reader, desc any) {
	err := binary.Read(reader, binary.BigEndian, desc)
	HandleErr(err)
}

func readStr(reader io.Reader) string {
	count := readInt(reader)
	if count <= 1 {
		return ""
	}
	temp := readByte(reader, count-1)
	return string(temp)
}

func readInt(reader io.Reader) int {
	temp := readU8(reader)
	res := int(temp & 127)
	if (temp & 128) != 0 {
		temp = readU8(reader)
		res |= int(temp&127) << 7
		if (temp & 128) != 0 {
			temp = readU8(reader)
			res |= int(temp&127) << 14
			if (temp & 128) != 0 {
				temp = readU8(reader)
				res |= int(temp&127) << 21
				if (temp & 128) != 0 {
					temp = readU8(reader)
					res |= int(temp&127) << 28
				}
			}
		}
	}

	return res
}

func readU8(reader io.Reader) uint8 {
	return readByte(reader, 1)[0]
}

func readByte(reader io.Reader, count int) []byte {
	res := make([]byte, count)
	_, err := reader.Read(res)
	HandleErr(err)
	return res
}
