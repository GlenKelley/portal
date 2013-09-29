package portal

import (
    "fmt"
    "math"
    gl "github.com/GlenKelley/go-gl32"
    glm "github.com/Jragonmiris/mathgl"
)

type Quad struct {
    Center glm.Vec4d
    Normal glm.Vec4d
    PlaneV glm.Vec4d
    Scale glm.Vec4d
}

func (q *Quad) Mesh() ([]float64, []float64) {
    px := q.PlaneV.Mul(q.Scale[0])
    py := Cross3Dv(q.PlaneV, q.Normal).Mul(q.Scale[1])
    a := q.Center.Sub(px).Sub(py)
    b := q.Center.Add(px).Sub(py)
    c := q.Center.Sub(px).Add(py)
    d := q.Center.Add(px).Add(py)
    o := q.Center
    n := q.Center.Add(q.Normal.Mul(q.Scale[2]*0.2))
    return []float64{
        a[0],a[1],a[2],
        b[0],b[1],b[2],
        c[0],c[1],c[2],
        d[0],d[1],d[2],
        o[0],o[1],o[2],
        n[0],n[1],n[2],
    }, []float64{
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

func (q *Quad) Apply(t glm.Mat4d) Quad {
    return Quad {
        q.Center, //t.Mul4x1(q.Center),
        t.Mul4x1(q.Normal),
        t.Mul4x1(q.PlaneV),
        q.Scale,//t.Mul4x1(q.Scale),
    }
}

type Portal struct {
    EventHorizon Quad
    Transform glm.Mat4d
    Portalview glm.Mat4d
}

func Cross3D(a, b glm.Vec4d) glm.Vec3d {
    a3 := glm.Vec3d{a[0], a[1], a[2]}
    b3 := glm.Vec3d{b[0], b[1], b[2]}
    return a3.Cross(b3)
}

func Cross3Dv(a, b glm.Vec4d) glm.Vec4d {
    c := Cross3D(a,b)
    return glm.Vec4d{c[0],c[1], c[2], 0}
}

func NearZero(v glm.Vec3d) bool {
    return v.ApproxEqual(glm.Vec3d{})
}

func RotationBetweenNormals(n1, n2 glm.Vec4d) glm.Mat4d {
    axis := Cross3D(n1, n2)
    dot := n1.Dot(n2)
    if !NearZero(axis) {
        angle := math.Acos(dot)
        return glm.HomogRotate3Dd(angle, axis.Normalize())
    } else if dot < 0 {
        for e := 0; e < 3; e++ {
            v := glm.Vec4d{}
            v[e] = 1
            cross := Cross3D(n1, v)
            if !NearZero(cross) {
                return glm.HomogRotate3Dd(math.Pi,cross.Normalize())
            }
        }
        panic(fmt.Sprintln("no orthogonal axis found for normal", n1))
    }
    return glm.Ident4d()
}
func PortalTransform(a, b Quad) (glm.Mat4d, glm.Mat4d, glm.Mat4d, glm.Mat4d) {
    zn := glm.Vec4d{0,0,1,0}
    xn := glm.Vec4d{1,0,0,0}
    
    translateAZ := glm.Translate3Dd(-a.Center[0], -a.Center[1], -a.Center[2])
    rotationAZ := RotationBetweenNormals(a.Normal, zn)
    rotationAXZ := RotationBetweenNormals(rotationAZ.Mul4x1(a.PlaneV), xn)
    scaleAZ := glm.Scale3Dd(1.0/a.Scale[0], 1.0/a.Scale[1], 1.0/a.Scale[2])
    
    
    AZ := scaleAZ.Mul4(rotationAXZ).Mul4(rotationAZ).Mul4(translateAZ)
    ZA := AZ.Inv()
    
    translateBZ := glm.Translate3Dd(-b.Center[0], -b.Center[1], -b.Center[2])
    rotationBZ := RotationBetweenNormals(b.Normal, zn)
    rotationBXZ := RotationBetweenNormals(rotationBZ.Mul4x1(b.PlaneV), xn)
    scaleBZ := glm.Scale3Dd(1.0/b.Scale[0], 1.0/b.Scale[1], 1.0/b.Scale[2])
    
    BZ := scaleBZ.Mul4(rotationBXZ).Mul4(rotationBZ).Mul4(translateBZ)
    ZB := BZ.Inv()
    
    AB := ZB.Mul4(AZ)
    BA := ZA.Mul4(BZ)
    return AB, BA, AZ, BZ
}