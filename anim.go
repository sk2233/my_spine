package main

import (
	"fmt"
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl32"
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
		r.Bone.LocalRotate = r.Bone.Rotate + r.KeyFrames[0].Rotate
	} else if idx+1 >= len(r.KeyFrames) {
		r.Bone.LocalRotate = r.Bone.Rotate + r.KeyFrames[idx].Rotate
	} else {
		pre := r.KeyFrames[idx]
		next := r.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		r.Bone.LocalRotate = r.Bone.Rotate + LerpRotation(pre.Rotate, next.Rotate, rate)
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
		t.Bone.LocalPos = t.Bone.Pos.Add(t.KeyFrames[0].Offset)
	} else if idx+1 >= len(t.KeyFrames) {
		t.Bone.LocalPos = t.Bone.Pos.Add(t.KeyFrames[idx].Offset)
	} else {
		pre := t.KeyFrames[idx]
		next := t.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		t.Bone.LocalPos = t.Bone.Pos.Add(Vec2Lerp(pre.Offset, next.Offset, rate))
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
		t.Bone.LocalScale = Vec2Mul(t.Bone.Scale, t.KeyFrames[0].Scale)
	} else if idx+1 >= len(t.KeyFrames) {
		t.Bone.LocalScale = Vec2Mul(t.Bone.Scale, t.KeyFrames[idx].Scale)
	} else {
		pre := t.KeyFrames[idx]
		next := t.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		t.Bone.LocalScale = Vec2Mul(t.Bone.Scale, Vec2Lerp(pre.Scale, next.Scale, rate))
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
					temp = append(temp, Vec2Lerp(item, next.WeightDeform[i][j], rate))
				}
				weightDeform = append(weightDeform, temp)
			}
		} else {
			for i := 0; i < len(pre.Deform); i++ {
				deform = append(deform, Vec2Lerp(pre.Deform[i], next.Deform[i], rate))
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
		c.Slot.CurrColor = Vec4Lerp(pre.Color, next.Color, rate)
	}
}

func NewColorAnimUpdate(slot *Slot, keyFrames []*KeyFrame) *ColorAnimUpdate {
	return &ColorAnimUpdate{Slot: slot, KeyFrames: keyFrames}
}

type TransformConstraintAnimUpdate struct {
	TransformConstraint *TransformConstraint
	KeyFrames           []*KeyFrame
}

func (t *TransformConstraintAnimUpdate) Update(curr float32) {
	idx := GetIndexByTime(t.KeyFrames, curr)
	if idx < 0 {
		t.TransformConstraint.CurrRotateMix = t.KeyFrames[0].RotateMix
		t.TransformConstraint.CurrOffsetMix = t.KeyFrames[0].OffsetMix
		t.TransformConstraint.CurrScaleMix = t.KeyFrames[0].ScaleMix
	} else if idx+1 >= len(t.KeyFrames) {
		t.TransformConstraint.CurrRotateMix = t.KeyFrames[idx].RotateMix
		t.TransformConstraint.CurrOffsetMix = t.KeyFrames[idx].OffsetMix
		t.TransformConstraint.CurrScaleMix = t.KeyFrames[idx].ScaleMix
	} else {
		pre := t.KeyFrames[idx]
		next := t.KeyFrames[idx+1]
		rate := CurveVal(pre.Curve, (curr-pre.Time)/(next.Time-pre.Time))
		t.TransformConstraint.CurrRotateMix = Lerp(pre.RotateMix, next.RotateMix, rate)
		t.TransformConstraint.CurrOffsetMix = Lerp(pre.OffsetMix, next.OffsetMix, rate)
		t.TransformConstraint.CurrScaleMix = Lerp(pre.ScaleMix, next.ScaleMix, rate)
	}
}

func NewTransformConstraintAnimUpdate(transformConstraint *TransformConstraint, keyFrames []*KeyFrame) *TransformConstraintAnimUpdate {
	return &TransformConstraintAnimUpdate{TransformConstraint: transformConstraint, KeyFrames: keyFrames}
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
		c.Slot.CurrColor = Vec4Lerp(pre.Color, next.Color, rate)
		c.Slot.CurrDarkColor = Vec4Lerp(pre.DarkColor, next.DarkColor, rate)
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

func NewAnimController(anim *Animation, skel *Skel, attachments map[string]*AttachmentItem) *AnimController {
	updates := make([]IAnimUpdate, 0)
	for i, timeline := range anim.Timelines {
		if len(timeline.KeyFrames) == 0 {
			fmt.Println("error: no keyframes", i)
			continue
		}
		switch timeline.Type {
		case TimelineAttachment:
			updates = append(updates, NewAttachmentAnimUpdate(skel.Slots[timeline.Slot], timeline.KeyFrames))
		case TimelineRotate:
			updates = append(updates, NewRotateAnimUpdate(skel.Bones[timeline.Bone], timeline.KeyFrames))
		case TimelineTranslate:
			updates = append(updates, NewTranslateAnimUpdate(skel.Bones[timeline.Bone], timeline.KeyFrames))
		case TimelineScale:
			updates = append(updates, NewScaleAnimUpdate(skel.Bones[timeline.Bone], timeline.KeyFrames))
		case TimelineDeform:
			attachment := attachments[AttachmentKey(timeline.Attachment, timeline.Slot)]
			updates = append(updates, NewDeformAnimUpdate(attachment.Attachment, timeline.KeyFrames))
		case TimelineDrawOrder:
			updates = append(updates, NewDrawOrderAnimUpdate(skel.Slots, timeline.KeyFrames))
		case TimelineColor:
			updates = append(updates, NewColorAnimUpdate(skel.Slots[timeline.Slot], timeline.KeyFrames))
		case TimelineTwoColor:
			updates = append(updates, NewTwoColorAnimUpdate(skel.Slots[timeline.Slot], timeline.KeyFrames))
		case TimelineTransformConstraint:
			updates = append(updates, NewTransformConstraintAnimUpdate(skel.TransformConstraints[timeline.TransformConstraint], timeline.KeyFrames))
		case TimelineShear:
			// 先不处理斜切
		default:
			panic("unknown timeline type")
		}
	}
	return &AnimController{AnimName: anim.Name, Duration: anim.Duration, Start: time.Now(), AnimUpdates: updates}
}
