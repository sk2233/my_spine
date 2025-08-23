package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"time"
)

type IAnimUpdate interface {
	Update(curr float32)
}

func GetIndexByTime(frames []*KeyFrameData, curr float32) int {
	for i := len(frames) - 1; i >= 0; i-- {
		if curr >= frames[i].Time {
			return i
		}
	}
	return -1
}

// TODO 后面可以实现贝塞尔曲线
func Lerp(v1 float32, v2 float32, rate float32) float32 {
	return v1 + (v2-v1)*rate
}

// 旋转角度需要特殊处理
func LerpRotation(r1 float32, r2 float32, rate float32) float32 {
	delta := r2 - r1
	// 修正差值，确保在[-180, 180]范围内（取最短路径）
	for delta < -180 {
		delta += 360
	}
	for delta > 180 {
		delta -= 360
	}
	return r1 + delta*rate
}

type AttachmentAnimUpdate struct {
	Slot      *SlotData
	KeyFrames []*KeyFrameData // 至少 1 个
}

func NewAttachmentAnimUpdate(slot *SlotData, keyFrames []*KeyFrameData) *AttachmentAnimUpdate {
	return &AttachmentAnimUpdate{Slot: slot, KeyFrames: keyFrames}
}

func (a *AttachmentAnimUpdate) Update(curr float32) {
	idx := max(GetIndexByTime(a.KeyFrames, curr), 0)
	a.Slot.CurrAttachment = a.KeyFrames[idx].AttachmentName
}

type RotateAnimUpdate struct {
	Bone      *BoneData
	KeyFrames []*KeyFrameData
}

func (r *RotateAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(r.KeyFrames, curr)
	if idx < 0 {
		r.Bone.CurrRotation += r.KeyFrames[0].Rotation
	} else if idx+1 >= len(r.KeyFrames) {
		r.Bone.CurrRotation += r.KeyFrames[idx].Rotation
	} else {
		pre := r.KeyFrames[idx]
		next := r.KeyFrames[idx+1]
		r.Bone.CurrRotation += LerpRotation(pre.Rotation, next.Rotation, (curr-pre.Time)/(next.Time-pre.Time))
	}
}

func NewRotateAnimUpdate(bone *BoneData, keyFrames []*KeyFrameData) *RotateAnimUpdate {
	return &RotateAnimUpdate{Bone: bone, KeyFrames: keyFrames}
}

type TranslateAnimUpdate struct {
	Bone      *BoneData
	KeyFrames []*KeyFrameData
}

func NewTranslateAnimUpdate(bone *BoneData, keyFrames []*KeyFrameData) *TranslateAnimUpdate {
	return &TranslateAnimUpdate{Bone: bone, KeyFrames: keyFrames}
}

func (t *TranslateAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(t.KeyFrames, curr)
	if idx < 0 {
		t.Bone.CurrPos = t.Bone.CurrPos.Add(t.KeyFrames[0].Offset)
	} else if idx+1 >= len(t.KeyFrames) {
		t.Bone.CurrPos = t.Bone.CurrPos.Add(t.KeyFrames[idx].Offset)
	} else {
		pre := t.KeyFrames[idx]
		next := t.KeyFrames[idx+1]
		rate := (curr - pre.Time) / (next.Time - pre.Time)
		t.Bone.CurrPos = t.Bone.CurrPos.Add(mgl32.Vec2{
			Lerp(pre.Offset.X(), next.Offset.X(), rate),
			Lerp(pre.Offset.Y(), next.Offset.Y(), rate),
		})
	}
}

type DeformAnimUpdate struct {
	Attachment *AttachmentData
	KeyFrames  []*KeyFrameData
}

func (d *DeformAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(d.KeyFrames, curr)
	if idx < 0 {
		d.Attachment.CurrVertices = d.KeyFrames[0].Vertexes
	} else if idx+1 >= len(d.KeyFrames) {
		d.Attachment.CurrVertices = d.KeyFrames[idx].Vertexes
	} else {
		pre := d.KeyFrames[idx]
		next := d.KeyFrames[idx+1]
		rate := (curr - pre.Time) / (next.Time - pre.Time)
		for i := 0; i < len(d.Attachment.CurrVertices); i++ {
			// TODO 为什么少了？
			if i < len(pre.Vertexes) && i < len(next.Vertexes) {
				d.Attachment.CurrVertices[i] = mgl32.Vec2{
					Lerp(pre.Vertexes[i].X(), next.Vertexes[i].X(), rate),
					Lerp(pre.Vertexes[i].Y(), next.Vertexes[i].Y(), rate),
				}
			}
		}
	}
}

func NewDeformAnimUpdate(attachment *AttachmentData, keyFrames []*KeyFrameData) *DeformAnimUpdate {
	return &DeformAnimUpdate{Attachment: attachment, KeyFrames: keyFrames}
}

type DrawOrderAnimUpdate struct {
	Slots     []*SlotData
	KeyFrames []*KeyFrameData
}

func (d *DrawOrderAnimUpdate) Update(curr float32) {
	idx := max(GetIndexByTime(d.KeyFrames, curr), 0)
	drawOrder := d.KeyFrames[idx].DrawOrder
	for i := 0; i < len(d.Slots); i++ {
		d.Slots[i].Order = drawOrder[i]
	}
}

func NewDrawOrderAnimUpdate(slots []*SlotData, keyFrames []*KeyFrameData) *DrawOrderAnimUpdate {
	return &DrawOrderAnimUpdate{Slots: slots, KeyFrames: keyFrames}
}

type ColorAnimUpdate struct {
	Slot      *SlotData
	KeyFrames []*KeyFrameData
}

func (c *ColorAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(c.KeyFrames, curr)
	if idx < 0 {
		c.Slot.CurrColor = c.KeyFrames[0].Color
	} else if idx+1 >= len(c.KeyFrames) {
		c.Slot.CurrColor = c.KeyFrames[idx].Color
	} else {
		pre := c.KeyFrames[idx]
		next := c.KeyFrames[idx+1]
		rate := (curr - pre.Time) / (next.Time - pre.Time)
		c.Slot.CurrColor = mgl32.Vec4{
			Lerp(pre.Color[0], next.Color[0], rate),
			Lerp(pre.Color[1], next.Color[1], rate),
			Lerp(pre.Color[2], next.Color[2], rate),
			Lerp(pre.Color[3], next.Color[3], rate),
		}
	}
}

func NewColorAnimUpdate(slot *SlotData, keyFrames []*KeyFrameData) *ColorAnimUpdate {
	return &ColorAnimUpdate{Slot: slot, KeyFrames: keyFrames}
}

type AnimController struct {
	AnimName    string
	Duration    float32
	Start       time.Time
	AnimUpdates []IAnimUpdate
}

func (c *AnimController) Update() {
	curr := float32(time.Now().Sub(c.Start).Seconds())
	for _, update := range c.AnimUpdates {
		update.Update(curr)
	}
	if curr > c.Duration { // 循环播放
		c.Start = time.Now()
	}
}

func (c *AnimController) GetAnimName() string {
	return c.AnimName
}

func NewAnimController(anim *AnimationData, bones []*BoneData, slots []*SlotData, drawData map[string]*DrawData) *AnimController {
	updates := make([]IAnimUpdate, 0)
	for i, timeline := range anim.Timelines {
		if len(timeline.KeyFrames) == 0 {
			fmt.Println("error: no keyframes", i)
			continue
		}
		switch timeline.TimelineType {
		case TimelineAttachment:
			updates = append(updates, NewAttachmentAnimUpdate(slots[timeline.SlotIndex], timeline.KeyFrames))
		case TimelineRotate:
			updates = append(updates, NewRotateAnimUpdate(bones[timeline.BoneIndex], timeline.KeyFrames))
		case TimelineTranslate:
			updates = append(updates, NewTranslateAnimUpdate(bones[timeline.BoneIndex], timeline.KeyFrames))
		case TimelineDeform:
			slot := slots[timeline.SlotIndex]
			if len(slot.Attachment) > 0 { // 可以绘制动画才有意义
				updates = append(updates, NewDeformAnimUpdate(drawData[slot.Attachment].Attachment, timeline.KeyFrames))
			}
		case TimelineDrawOrder:
			updates = append(updates, NewDrawOrderAnimUpdate(slots, timeline.KeyFrames))
		case TimelineColor:
			updates = append(updates, NewColorAnimUpdate(slots[timeline.SlotIndex], timeline.KeyFrames))
		default:
			panic("unknown timeline type")
		}
	}
	return &AnimController{AnimName: anim.Name, Duration: anim.Duration, Start: time.Now(), AnimUpdates: updates}
}
