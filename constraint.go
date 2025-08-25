package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

type ConstraintController struct {
	Bones                []*Bone
	PathConstraints      []*PathConstraint
	TransformConstraints []*TransformConstraint
}

func (c *ConstraintController) Update() {
	// 实际 ik constraint , path constraint , transform constraint 都是有 order 指定顺序的
	// 不过一般是  ik -> path -> transform
	c.updatePathConstraints()
	c.updateTransformConstraints()
}

func (c *ConstraintController) updateTransformConstraints() {
	for _, item := range c.TransformConstraints {
		if item.CurrRotateMix <= 0 && item.CurrOffsetMix <= 0 && item.CurrScaleMix <= 0 {
			continue // 无效值
		}
		if item.Local || item.Relative {
			panic("not implemented")
		}
		target := c.Bones[item.Target]
		rotate := GetRotate(target.Mat2) + item.Rotate
		pos := target.Mat2.Mul2x1(item.Offset).Add(target.WorldPos)
		scale := GetScale(target.Mat2).Add(item.Scale)
		for _, idx := range item.Bones {
			bone := c.Bones[idx]
			if item.CurrRotateMix > 0 {
				oldRotate := GetRotate(bone.Mat2)
				temp := LerpRotation(oldRotate, rotate, item.CurrRotateMix) - oldRotate
				bone.Mat2 = Rotate(temp).Mul2(bone.Mat2)
				bone.Modify = true
			}
			if item.CurrOffsetMix > 0 {
				bone.WorldPos = Vec2Lerp(bone.WorldPos, pos, item.CurrOffsetMix)
				bone.Modify = true
			}
			if item.CurrScaleMix > 0 {
				oldScale := GetScale(bone.Mat2) // oldScale 为 0  时会有除0风险
				temp := Vec2Div(Vec2Lerp(oldScale, scale, item.CurrScaleMix), oldScale)
				bone.Mat2 = Scale(temp).Mul2(bone.Mat2)
				bone.Modify = true
			}
		}
	}
}

func (c *ConstraintController) updatePathConstraints() {
	for _, item := range c.PathConstraints {
		if item.OffsetMix <= 0 && item.RotateMix <= 0 {
			continue // 无效值
		}
		if item.PositionMode != PositionPercent || item.SpaceMode != SpacePercent || item.RotateMode != RotateChainScale {
			panic("not implemented") // 只实现了部分，也可以直接跳过没有实现的
		}
		attachment := item.Attachment
		points := make([]mgl32.Vec2, 0) // 只存储线段点，暂时不管控制点
		if attachment.Weight {          // 2 5 8 ... 3个点一组，前两个是控制点
			for i := 2; i < len(attachment.CurrWeightVertices); i += 3 {
				res := mgl32.Vec2{}
				for _, vec := range attachment.CurrWeightVertices[i] {
					bone := c.Bones[vec.Bone]
					temp := bone.Mat2.Mul2x1(vec.Offset).Add(bone.WorldPos)
					res = res.Add(temp.Mul(vec.Weight))
				}
				points = append(points, res)
			}
		} else {
			bone := c.Bones[item.Bone]
			for i := 2; i < len(attachment.CurrVertices); i += 3 {
				vec := bone.Mat2.Mul2x1(attachment.CurrVertices[i]).Add(bone.WorldPos)
				points = append(points, vec)
			}
		} // 计算每段长度与总长度
		lens := make([]float32, 0)
		allLen := float32(0)
		for i := 1; i < len(points); i++ {
			temp := points[i].Sub(points[i-1]).Len()
			lens = append(lens, temp)
			allLen += temp
		}
		rate := item.Position
		for _, idx := range item.Bones {
			bone := c.Bones[idx] // 计算线段位置处的坐标与旋转角度
			pos, rotate := c.calculatePosAndRotate(points, lens, allLen*rate)
			if item.RotateMix > 0 {
				oldRotate := GetRotate(bone.Mat2)
				rotate = LerpRotation(oldRotate, rotate, item.RotateMix) - oldRotate
				bone.Mat2 = Rotate(rotate).Mul2(bone.Mat2)
				bone.Modify = true
			}
			if item.OffsetMix > 0 {
				bone.WorldPos = Vec2Lerp(bone.WorldPos, pos, item.OffsetMix)
				bone.Modify = true
			}
			rate += item.Space
		}
	}
}

func GetAngle(src, dst mgl32.Vec2) float32 {
	return float32(math.Atan2(float64(dst.Y()-src.Y()), float64(dst.X()-src.X())) * 180 / math.Pi)
}

func (c *ConstraintController) calculatePosAndRotate(points []mgl32.Vec2, lens []float32, pos float32) (mgl32.Vec2, float32) {
	if pos < 0 { // 不在范围内直接截断
		return points[0], GetAngle(points[0], points[1])
	}
	for i := 0; i < len(lens); i++ {
		if pos > lens[i] {
			pos -= lens[i]
			continue
		}
		return Vec2Lerp(points[i], points[i+1], pos/lens[i]), GetAngle(points[i], points[i+1])
	}
	return points[len(points)-1], GetAngle(points[len(points)-2], points[len(points)-1])
}

func NewConstraintController(bones []*Bone, pathConstraints []*PathConstraint, transformConstraints []*TransformConstraint) *ConstraintController {
	return &ConstraintController{Bones: bones, PathConstraints: pathConstraints, TransformConstraints: transformConstraints}
}
