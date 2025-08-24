package main

import "github.com/go-gl/mathgl/mgl32"

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
		if item.Local || item.Relative {
			panic("not implemented")
		}
		target := c.Bones[item.Target]
		rotate := target.WorldRotate + item.Rotate
		offset := target.Mat3.Mul3x1(item.Offset.Vec3(1)).Vec2()
		scale := target.WorldScale.Add(item.Scale)
		for _, idx := range item.Bones {
			bone := c.Bones[idx]
			if item.CurrRotateMix != 0 {
				bone.WorldRotate = rotate*item.CurrRotateMix + bone.WorldRotate*(1-item.CurrRotateMix)
				bone.Modify = true
			}
			if item.CurrOffsetMix != 0 {
				bone.WorldPos = offset.Mul(item.CurrOffsetMix).Add(bone.WorldPos.Mul(1 - item.CurrOffsetMix))
				bone.Modify = true
			}
			if item.CurrScaleMix != 0 {
				bone.WorldScale = scale.Mul(item.CurrScaleMix).Add(bone.WorldScale.Mul(1 - item.CurrScaleMix))
				bone.Modify = true
			}
		}
	}
}

func (c *ConstraintController) updatePathConstraints() {
	for _, item := range c.PathConstraints {
		if item.PositionMode != PositionPercent || item.SpaceMode != SpacePercent || item.RotateMode != RotateChainScale {
			panic("not implemented") // 只实现了部分，也可以直接跳过没有实现的
		}
		attachment := item.Attachment
		vs := make([]mgl32.Vec2, 0) // 只存储线段点，暂时不管控制点
		if attachment.Weight {      // 2 5 8 ... 3个点一组，前两个是控制点
			for i := 2; i < len(attachment.CurrWeightVertices); i += 3 {
				res := mgl32.Vec2{}
				for _, vec := range attachment.CurrWeightVertices[i] {
					temp := c.Bones[vec.Bone].Mat3.Mul3x1(vec.Offset.Vec3(1)).Vec2()
					res = res.Add(temp.Mul(vec.Weight))
				}
				vs = append(vs, res)
			}
		} else {
			mat3 := c.Bones[item.Bone].Mat3
			for i := 2; i < len(attachment.CurrVertices); i += 3 {
				vec := mat3.Mul3x1(attachment.CurrVertices[i].Vec3(1)).Vec2()
				vs = append(vs, vec)
			}
		}
		pos := item.Position
		for _, idx := range item.Bones {
			bone := c.Bones[idx]
			vec := c.calculatePosAndRotate(vs, pos) // 先只管位移
			if item.OffsetMix > 0 {
				bone.WorldPos = vec.Mul(item.OffsetMix).Add(bone.WorldPos.Mul(1 - item.OffsetMix))
				bone.Modify = true
			}
			pos += item.Space
		}
	}
}

func (c *ConstraintController) calculatePosAndRotate(vs []mgl32.Vec2, rate float32) mgl32.Vec2 {
	if len(vs) != 2 { // 先按简单的两个点来
		panic("not two point")
	}
	return mgl32.Vec2{
		Lerp(vs[0].X(), vs[1].X(), rate),
		Lerp(vs[0].Y(), vs[1].Y(), rate),
	}
}

func NewConstraintController(bones []*Bone, pathConstraints []*PathConstraint, transformConstraints []*TransformConstraint) *ConstraintController {
	return &ConstraintController{Bones: bones, PathConstraints: pathConstraints, TransformConstraints: transformConstraints}
}
