package portal

import (
    // "fmt"
    "math"
    gl "github.com/GlenKelley/go-gl32"
    glm "github.com/Jragonmiris/mathgl"
)

type Quad struct {
    Center glm.Vec4f
    Normal glm.Vec4f
    PlaneV glm.Vec4f
    Size glm.Vec2f
}

func (q *Quad) Mesh() ([]float32, []float32) {
    px := q.PlaneV.Mul(q.Size[0])
    py := Cross3Dv(q.PlaneV, q.Normal).Mul(q.Size[1])
    a := q.Center.Sub(px).Sub(py)
    b := q.Center.Add(px).Sub(py)
    c := q.Center.Sub(px).Add(py)
    d := q.Center.Add(px).Add(py)
    o := q.Center
    n := q.Center.Add(q.Normal.Mul(0.2))
    return []float32{
        a[0],a[1],a[2],
        b[0],b[1],b[2],
        c[0],c[1],c[2],
        d[0],d[1],d[2],
        o[0],o[1],o[2],
        n[0],n[1],n[2],
    }, []float32{
        n[0],n[1],n[2],
        n[0],n[1],n[2],
        n[0],n[1],n[2],
        n[0],n[1],n[2],
        n[0],n[1],n[2],
        n[0],n[1],n[2],
    }
}

var QuadElements = map[gl.Enum][]int16 {
    gl.TRIANGLE_STRIP:[]int16{0,1,2,3},
    gl.LINES:[]int16{0,1,1,3,3,2,2,0,0,3,1,2,4,5},
}

func (q *Quad) Apply(t glm.Mat4f) Quad {
    return Quad {
        t.Mul4x1(q.Center),
        t.Mul4x1(q.Normal),
        t.Mul4x1(q.PlaneV),
        q.Size,
    }
}

type Portal struct {
    EventHorizon Quad
    Transform glm.Mat4f
    Portalview glm.Mat4f
}

func Cross3D(a, b glm.Vec4f) glm.Vec3f {
    a3 := glm.Vec3f{a[0], a[1], a[2]}
    b3 := glm.Vec3f{b[0], b[1], b[2]}
    return a3.Cross(b3)
}

func Cross3Dv(a, b glm.Vec4f) glm.Vec4f {
    c := Cross3D(a,b)
    return glm.Vec4f{c[0],c[1], c[2], 0}
}

func NearZero(v glm.Vec3f) bool {
    return v.ApproxEqual(glm.Vec3f{})
}

func RotationBetweenNormals(n1, n2 glm.Vec4f) glm.Mat4f {
    axis := Cross3D(n1, n2)
    if !NearZero(axis) {
        dot := n1.Dot(n2)
        angle := float32(math.Acos(float64(dot)))
        return glm.HomogRotate3D(angle, axis)
    }
    return glm.Ident4f()
}
func PortalTransform(a, b Quad) (glm.Mat4f, glm.Mat4f, glm.Mat4f, glm.Mat4f) {
    ca := glm.Translate3D(-a.Center[0], -a.Center[1], -a.Center[2])
    cb := glm.Translate3D(b.Center[0], b.Center[1], b.Center[2])
    
    rotate1 := RotationBetweenNormals(a.Normal, b.Normal)
    rotate2 := RotationBetweenNormals(rotate1.Mul4x1(a.PlaneV), b.PlaneV)
    transform := cb.Mul4(rotate2).Mul4(rotate1).Mul4(ca)
    inverse := transform.Inv()
    
    zn := glm.Vec4f{0,0,1,0}
    fromA := RotationBetweenNormals(a.Normal, zn)
    fromB := RotationBetweenNormals(b.Normal, zn)
    
    transformA := fromA.Mul4(ca)
    transformB := fromB.Mul4(cb.Inv())
    
    return transform, inverse, transformA, transformB
}