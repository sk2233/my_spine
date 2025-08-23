package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"io"
	"math"
	"os"
	"sort"
)

type SkelHeader struct {
	Hash    string
	Version string
	Pos     mgl32.Vec2
	Size    mgl32.Vec2
}

type Bone struct {
	Name          string
	Parent        int
	Rotate        float32
	Pos           mgl32.Vec2
	Scale         mgl32.Vec2
	Shear         mgl32.Vec2
	Length        float32
	TransformMode uint8
	SkinRequire   bool
}

type Slot struct {
	Name             string
	Bone             int
	Color, DarkColor mgl32.Vec4
	Attachment       string
	BlendMode        uint8
}

type WeightVertex struct {
	Bone   int        // 受那个骨骼影响
	Offset mgl32.Vec2 // 相对于骨骼位置偏移的大小
	Weight float32    // 受骨骼影响的权重
}

type Attachment struct {
	Name  string
	Type  uint8
	Path  string
	Color mgl32.Vec4
	// AttachmentRegion
	Rotate float32
	Pos    mgl32.Vec2
	Scale  mgl32.Vec2
	Size   mgl32.Vec2
	// AttachmentMesh
	UVs            []mgl32.Vec2
	Indices        []uint16
	Weight         bool
	Vertices       []mgl32.Vec2
	WeightVertices [][]*WeightVertex
	HullLength     int
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
	DrawOrder map[int]int // 对应槽位与其偏移
	// TimelineDeform
	Start, Len int
	Deform     []mgl32.Vec2 // 从 start 开始 Len 个要调整的大小 不管 WeightVertex 还是非 WeightVertex 都是对应偏移
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
}

type Skel struct {
	Header    *SkelHeader
	Bones     []*Bone
	Slots     []*Slot
	Skin      *Skin // 暂时只支持默认皮肤，不支持换肤
	Animation []*Animation
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
	animations := parseAnimations(reader, strings)
	return &Skel{
		Header:    header,
		Bones:     bones,
		Slots:     slots,
		Skin:      skin,
		Animation: animations,
	}
}

func parseAnimations(reader io.Reader, strings []string) []*Animation {
	count := readInt(reader)
	animations := make([]*Animation, 0)
	for i := 0; i < count; i++ {
		animations = append(animations, parseAnimation(reader, strings))
	}
	return animations
}

func parseAnimation(reader io.Reader, strings []string) *Animation {
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
	if count > 0 {
		panic(fmt.Errorf("invalid timeline count: %v", count))
	}
	//for i := 0; i < count; i++ {
	//	index := readInt(reader)
	//	fCount := readInt(reader)
	//	for j := 0; j < fCount; j++ {
	//		other := readByte(reader, 5*4)
	//		if j < fCount-1 {
	//			readCurve(reader)
	//		}
	//		Use(other)
	//	}
	//	Use(index, fCount)
	//}
	// Path constraint skip
	count = readInt(reader)
	if count > 0 {
		panic(fmt.Errorf("invalid timeline count: %v", count))
	}
	// Deform
	count = readInt(reader)
	for i := 0; i < count; i++ { // 按 skin 分组
		skin := readInt(reader)
		if skin == -1 { // 只支持默认皮肤
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
				fCount := readInt(reader)
				for m := 0; m < fCount; m++ { // 每个 timeline 多帧动画
					keyFrame := &KeyFrame{
						Time: readF4(reader),
						Len:  readInt(reader) / 2,
					}
					deform := make([]mgl32.Vec2, 0)
					if keyFrame.Len > 0 {
						start := readInt(reader)
						keyFrame.Start = start
						for n := 0; n < keyFrame.Len; n++ {
							deform = append(deform, mgl32.Vec2{
								readF4(reader),
								readF4(reader),
							})
						}
					}
					keyFrame.Deform = deform
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
		for i := 0; i < count; i++ {
			time := readF4(reader)
			cCount := readInt(reader)
			offset := make(map[int]int)
			for j := 0; j < cCount; j++ {
				offset[readInt(reader)] = readInt(reader)
			}
			temp.KeyFrames = append(temp.KeyFrames, &KeyFrame{
				Time:      time,
				DrawOrder: offset,
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
	for _, timeline := range timelines {
		sort.Slice(timeline.KeyFrames, func(i, j int) bool {
			return timeline.KeyFrames[i].Time < timeline.KeyFrames[j].Time
		})
	}
	return &Animation{
		Name:      name,
		Timelines: timelines,
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
		Type: attachmentType,
	}
	switch attachmentType {
	case AttachmentRegion:
		res.Path = readRefStr(reader, strings)
		res.Rotate = readF4(reader)
		temp := [3]mgl32.Vec2{}
		readAny(reader, &temp)
		res.Pos = temp[0]
		res.Scale = temp[1]
		res.Size = temp[2]
		res.Color = readClr(reader)
	case AttachmentMesh:
		res.Path = readRefStr(reader, strings)
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
		for i := 0; i < vCount; i++ {
			if res.Weight {
				boneCount := readInt(reader)
				temp := make([]*WeightVertex, 0)
				for j := 0; j < boneCount; j++ {
					temp = append(temp, &WeightVertex{
						Bone:   readInt(reader),
						Offset: mgl32.Vec2{readF4(reader), readF4(reader)},
						Weight: readF4(reader),
					})
				}
				res.WeightVertices = append(res.WeightVertices, temp)
			} else {
				res.Vertices = append(res.Vertices, mgl32.Vec2{
					readF4(reader),
					readF4(reader),
				})
			}
		}
		res.HullLength = readInt(reader)
	default:
		panic(fmt.Sprintf("unknown attachment type: %v", attachmentType))
	}
	return res
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
	if count > 0 {
		panic("not implemented transform constraints")
	}
	//for i := 0; i < count; i++ {
	//	name := readStr(reader)
	//	order := readInt(reader)
	//	skinRequire := readBool(reader)
	//	boneCount := readInt(reader)
	//	for j := 0; j < boneCount; j++ {
	//		bone := readInt(reader)
	//		Use(bone)
	//	}
	//	target := readInt(reader)
	//	other := readByte(reader, 2+10*4)
	//	Use(name, order, skinRequire, target, other)
	//}
	// Path constraints
	count = readInt(reader)
	if count > 0 {
		panic("not implemented path constraints")
	}
}

func parseSlots(reader io.Reader, strings []string) []*Slot {
	res := make([]*Slot, 0)
	count := readInt(reader)
	for i := 0; i < count; i++ {
		res = append(res, parseSlot(reader, strings))
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
