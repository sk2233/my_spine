package main

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"math"
	"time"
)

type IAnimUpdate interface {
	Update(curr float32)
}

func GetIndexByTime(frames []*KeyFrame, curr float32) int {
	for i := len(frames) - 1; i >= 0; i-- {
		if curr >= frames[i].Time {
			return i
		}
	}
	return -1
}

func evalX(curve [2]mgl32.Vec2, rate float32) float32 {
	rate2 := rate * rate
	rate3 := rate2 * rate
	invRate := 1 - rate
	invRate2 := invRate * invRate
	invRate3 := invRate2 * invRate
	return rate3*1 + 3*rate2*invRate*curve[1].X() + 3*rate*invRate2*curve[0].X() + invRate3*0
}

func findX(curve [2]mgl32.Vec2, rate float32) float32 {
	e := 0.00001
	start := float32(0.0)
	stop := float32(1.0)
	res := float32(0.5)
	x := evalX(curve, res)
	for math.Abs(float64(rate-x)) > e {
		if rate < x {
			stop = res
		} else {
			start = res
		}
		res = (stop + start) * 0.5
		x = evalX(curve, res)
	}
	return res
}

func evalY(curve [2]mgl32.Vec2, rate float32) float32 {
	rate2 := rate * rate
	rate3 := rate2 * rate
	invRate := 1 - rate
	invRate2 := invRate * invRate
	invRate3 := invRate2 * invRate
	return rate3*1 + 3*rate2*invRate*curve[1].Y() + 3*rate*invRate2*curve[0].Y() + invRate3*0
}

// curve  0~1
func CurveVal(curve *Curve, rate float32) float32 {
	switch curve.Type {
	case CurveLinear:
		return rate
	case CurveStepped:
		return 0
	case CurveBezier:
		temp := findX(curve.Data, rate)
		return evalY(curve.Data, temp)
	default:
		panic(fmt.Sprintf("invalid curve type: %v", curve.Type))
	}
}

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
	Slot      *Slot
	KeyFrames []*KeyFrame // 至少 1 个
}

func NewAttachmentAnimUpdate(slot *Slot, keyFrames []*KeyFrame) *AttachmentAnimUpdate {
	return &AttachmentAnimUpdate{Slot: slot, KeyFrames: keyFrames}
}

func (a *AttachmentAnimUpdate) Update(curr float32) {
	idx := max(GetIndexByTime(a.KeyFrames, curr), 0)
	a.Slot.CurrAttachment = a.KeyFrames[idx].Attachment
}

type RotateAnimUpdate struct {
	Bone      *Bone
	KeyFrames []*KeyFrame
}

func (r *RotateAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(r.KeyFrames, curr)
	if idx < 0 {
		r.Bone.CurrRotate += r.KeyFrames[0].Rotate
	} else if idx+1 >= len(r.KeyFrames) {
		r.Bone.CurrRotate += r.KeyFrames[idx].Rotate
	} else {
		pre := r.KeyFrames[idx]
		next := r.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		r.Bone.CurrRotate += LerpRotation(pre.Rotate, next.Rotate, rate)
	}
}

func NewRotateAnimUpdate(bone *Bone, keyFrames []*KeyFrame) *RotateAnimUpdate {
	return &RotateAnimUpdate{Bone: bone, KeyFrames: keyFrames}
}

type TranslateAnimUpdate struct {
	Bone      *Bone
	KeyFrames []*KeyFrame
}

func NewTranslateAnimUpdate(bone *Bone, keyFrames []*KeyFrame) *TranslateAnimUpdate {
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
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		t.Bone.CurrPos = t.Bone.CurrPos.Add(mgl32.Vec2{
			Lerp(pre.Offset.X(), next.Offset.X(), rate),
			Lerp(pre.Offset.Y(), next.Offset.Y(), rate),
		})
	}
}

type ScaleAnimUpdate struct {
	Bone      *Bone
	KeyFrames []*KeyFrame
}

func NewScaleAnimUpdate(bone *Bone, keyFrames []*KeyFrame) *ScaleAnimUpdate {
	return &ScaleAnimUpdate{Bone: bone, KeyFrames: keyFrames}
}

func (t *ScaleAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(t.KeyFrames, curr)
	if idx < 0 {
		t.Bone.CurrScale = t.KeyFrames[0].Scale
	} else if idx+1 >= len(t.KeyFrames) {
		t.Bone.CurrScale = t.KeyFrames[idx].Scale
	} else {
		pre := t.KeyFrames[idx]
		next := t.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		t.Bone.CurrScale = mgl32.Vec2{
			Lerp(pre.Scale.X(), next.Scale.X(), rate),
			Lerp(pre.Scale.Y(), next.Scale.Y(), rate),
		}
	}
}

type DeformAnimUpdate struct {
	Attachment *Attachment
	KeyFrames  []*KeyFrame
}

func (d *DeformAnimUpdate) setDeform(deform []mgl32.Vec2, weightDeform [][]mgl32.Vec2) {
	if d.Attachment.Weight {
		for i, items := range d.Attachment.CurrWeightVertices {
			for j, item := range items {
				item.Offset = item.Offset.Add(weightDeform[i][j])
			}
		}
	} else {
		for i := 0; i < len(d.Attachment.CurrVertices); i++ {
			d.Attachment.CurrVertices[i] = d.Attachment.CurrVertices[i].Add(deform[i])
		}
	}
}

func (d *DeformAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(d.KeyFrames, curr)
	if idx < 0 {
		d.setDeform(d.KeyFrames[0].Deform, d.KeyFrames[0].WeightDeform)
	} else if idx+1 >= len(d.KeyFrames) {
		d.setDeform(d.KeyFrames[idx].Deform, d.KeyFrames[idx].WeightDeform)
	} else {
		pre := d.KeyFrames[idx]
		next := d.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		deform := make([]mgl32.Vec2, 0)
		weightDeform := make([][]mgl32.Vec2, 0)
		if d.Attachment.Weight {
			for i, items := range pre.WeightDeform {
				temp := make([]mgl32.Vec2, 0)
				for j, item := range items {
					temp = append(temp, mgl32.Vec2{
						Lerp(item.X(), next.WeightDeform[i][j].X(), rate),
						Lerp(item.Y(), next.WeightDeform[i][j].Y(), rate),
					})
				}
				weightDeform = append(weightDeform, temp)
			}
		} else {
			for i := 0; i < len(pre.Deform); i++ {
				deform = append(deform, mgl32.Vec2{
					Lerp(pre.Deform[i].X(), next.Deform[i].X(), rate),
					Lerp(pre.Deform[i].Y(), next.Deform[i].Y(), rate),
				})
			}
		}
		d.setDeform(deform, weightDeform)
	}
}

func NewDeformAnimUpdate(attachment *Attachment, keyFrames []*KeyFrame) *DeformAnimUpdate {
	return &DeformAnimUpdate{Attachment: attachment, KeyFrames: keyFrames}
}

type DrawOrderAnimUpdate struct {
	Slots     []*Slot
	KeyFrames []*KeyFrame
}

func (d *DrawOrderAnimUpdate) Update(curr float32) {
	idx := max(GetIndexByTime(d.KeyFrames, curr), 0)
	drawOrder := d.KeyFrames[idx].DrawOrder
	for i := 0; i < len(d.Slots); i++ {
		d.Slots[i].CurrOrder = drawOrder[i]
	}
}

func NewDrawOrderAnimUpdate(slots []*Slot, keyFrames []*KeyFrame) *DrawOrderAnimUpdate {
	return &DrawOrderAnimUpdate{Slots: slots, KeyFrames: keyFrames}
}

type ColorAnimUpdate struct {
	Slot      *Slot
	KeyFrames []*KeyFrame
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
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		c.Slot.CurrColor = mgl32.Vec4{
			Lerp(pre.Color[0], next.Color[0], rate),
			Lerp(pre.Color[1], next.Color[1], rate),
			Lerp(pre.Color[2], next.Color[2], rate),
			Lerp(pre.Color[3], next.Color[3], rate),
		}
	}
}

func NewColorAnimUpdate(slot *Slot, keyFrames []*KeyFrame) *ColorAnimUpdate {
	return &ColorAnimUpdate{Slot: slot, KeyFrames: keyFrames}
}

type TwoColorAnimUpdate struct {
	Slot      *Slot
	KeyFrames []*KeyFrame
}

func (c *TwoColorAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(c.KeyFrames, curr)
	if idx < 0 {
		c.Slot.CurrColor = c.KeyFrames[0].Color
		c.Slot.CurrDarkColor = c.KeyFrames[0].DarkColor
	} else if idx+1 >= len(c.KeyFrames) {
		c.Slot.CurrColor = c.KeyFrames[idx].Color
		c.Slot.CurrDarkColor = c.KeyFrames[idx].DarkColor
	} else {
		pre := c.KeyFrames[idx]
		next := c.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		c.Slot.CurrColor = mgl32.Vec4{
			Lerp(pre.Color[0], next.Color[0], rate),
			Lerp(pre.Color[1], next.Color[1], rate),
			Lerp(pre.Color[2], next.Color[2], rate),
			Lerp(pre.Color[3], next.Color[3], rate),
		}
		c.Slot.CurrDarkColor = mgl32.Vec4{
			Lerp(pre.DarkColor[0], next.DarkColor[0], rate),
			Lerp(pre.DarkColor[1], next.DarkColor[1], rate),
			Lerp(pre.DarkColor[2], next.DarkColor[2], rate),
			Lerp(pre.DarkColor[3], next.DarkColor[3], rate),
		}
	}
}

func NewTwoColorAnimUpdate(slot *Slot, keyFrames []*KeyFrame) *TwoColorAnimUpdate {
	return &TwoColorAnimUpdate{Slot: slot, KeyFrames: keyFrames}
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

func NewAnimController(anim *Animation, bones []*Bone, slots []*Slot, attachments map[string]*AttachmentItem) *AnimController {
	updates := make([]IAnimUpdate, 0)
	for i, timeline := range anim.Timelines {
		if len(timeline.KeyFrames) == 0 {
			fmt.Println("error: no keyframes", i)
			continue
		}
		switch timeline.Type {
		case TimelineAttachment:
			updates = append(updates, NewAttachmentAnimUpdate(slots[timeline.Slot], timeline.KeyFrames))
		case TimelineRotate:
			updates = append(updates, NewRotateAnimUpdate(bones[timeline.Bone], timeline.KeyFrames))
		case TimelineTranslate:
			updates = append(updates, NewTranslateAnimUpdate(bones[timeline.Bone], timeline.KeyFrames))
		case TimelineScale:
			updates = append(updates, NewScaleAnimUpdate(bones[timeline.Bone], timeline.KeyFrames))
		case TimelineDeform:
			attachment := attachments[AttachmentKey(timeline.Attachment, timeline.Slot)]
			updates = append(updates, NewDeformAnimUpdate(attachment.Attachment, timeline.KeyFrames))
		case TimelineDrawOrder:
			updates = append(updates, NewDrawOrderAnimUpdate(slots, timeline.KeyFrames))
		case TimelineColor:
			updates = append(updates, NewColorAnimUpdate(slots[timeline.Slot], timeline.KeyFrames))
		case TimelineTwoColor:
			updates = append(updates, NewTwoColorAnimUpdate(slots[timeline.Slot], timeline.KeyFrames))
		case TimelineShear:
			// 先不处理斜切
		default:
			panic("unknown timeline type")
		}
	}
	return &AnimController{AnimName: anim.Name, Duration: anim.Duration, Start: time.Now(), AnimUpdates: updates}
}
