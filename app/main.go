package main

import (
   "os"
   "fmt"
   "math"
   "strconv"
   "regexp"
   "encoding/json"
   glfw "github.com/go-gl/glfw3"
   "github.com/GlenKelley/portal"
   gl "github.com/GlenKelley/go-gl/gl32"
   glm "github.com/Jragonmiris/mathgl"
   gtk "github.com/GlenKelley/go-glutil"
   collada "github.com/GlenKelley/go-collada"
)

func main() {
   fmt.Println("Start")
   receiver := &Receiver{}
   gtk.CreateWindow(640, 480, "gotest", true, receiver)
}

func panicOnErr(err error) {
   if err != nil {
      panic(err)
   }
}

type Receiver struct {
   Data     DataBindings
   Shaders  gtk.ShaderLibrary
   
   SceneLoc SceneBindings
   FillLoc  FillBindings
   
   SceneIndex *gtk.Index
   Portals    []portal.Portal

   LastMousePosition    glm.Vec2d
   HasLastMousePosition bool

   SimulationTime  gtk.GameTime
   Player         Player
   UIState        UIState
   Window         *glfw.Window
   Invalid        bool

   Constants GameConstants
   Controls  gtk.ControlBindings
}

type GameConstants struct {
   PlayerMovementLimit        float64
   PlayerImpulseMomentumLimit float64
   Gravity                    float64
   PlayerPanSensitivity       float64
   PlayerFOV                  float64
   PlayerViewNear             float64
   PlayerViewFar              float64
   Debug                      bool
}
var DefaultConstants = GameConstants{5, 5, -9.8, 7, 70, 0.001, 100, false}


type DataBindings struct {
   // Tex0 gl.Texture
   // Tex1 gl.Texture
   Vao  gl.VertexArrayObject

   Fill *gtk.Geometry
   Scene *gtk.Model
   Portal *gtk.Model

   Projection glm.Mat4d
   Cameraview glm.Mat4d
   Inception glm.Mat4d //translates from world coordinates into portal coords
}

type SceneBindings struct {
   Projection gl.UniformLocation `gl:"projection"`
   Cameraview gl.UniformLocation `gl:"cameraview"`
   Worldview  gl.UniformLocation `gl:"worldview"`
   Portalview gl.UniformLocation `gl:"portalview"`
   Inception gl.UniformLocation `gl:"inception"`

   // Tex0           gl.UniformLocation `gl:"textures[0]"`
   // Tex1           gl.UniformLocation `gl:"textures[1]"`
   ElapsedSeconds gl.UniformLocation `gl:"elapsed"`
   Glow           gl.UniformLocation `gl:"glow"`

   Position gl.AttributeLocation `gl:"position"`
}

type FillBindings struct {
   Position gl.AttributeLocation `gl:"position"`
   Depth gl.UniformLocation `gl:"depth"`
   Color gl.UniformLocation `gl:"color"`
}

type Player struct {
   Position  glm.Vec4d
   Velocity  glm.Vec4d
   PanAxis   glm.Vec4d
   TiltAxis  glm.Vec4d
   OrientationH glm.Quatd
   Orientation glm.Quatd
}

func (p *Player) Transform(m glm.Mat4d) {
   p.Position = m.Mul4x1(p.Position)
   p.Velocity = m.Mul4x1(p.Velocity)
   r := gtk.RotationComponent(m)
   // fmt.Println("r",r)
   q := gtk.Quaternion(r)
   // fmt.Println("q", q)
   p.Orientation = q.Mul(p.Orientation)
   p.OrientationH = q.Mul(p.OrientationH)
}

type UIState struct {
   Impulse  glm.Vec4d
   Movement glm.Vec4d
}


func (r *Receiver) ResetKeyBindingDefaults() {
   c := &r.Controls
   c.ResetBindings()
   c.BindKeyPress(glfw.KeyW, r.MoveForward, r.StopMoveForward)
   c.BindKeyPress(glfw.KeyS, r.MoveBackward, r.StopMoveBackward)
   c.BindKeyPress(glfw.KeyA, r.StrafeLeft, r.StopStrafeLeft)
   c.BindKeyPress(glfw.KeyD, r.StrafeRight, r.StopStrafeRight)
   c.BindKeyPress(glfw.KeyE, r.MoveUp, r.StopMoveUp)
   c.BindKeyPress(glfw.KeyQ, r.MoveDown, r.StopMoveDown)
   c.BindKeyPress(glfw.KeySpace, r.Jump, nil)
   c.BindKeyPress(glfw.KeyEscape, r.Quit, nil)
   c.BindKeyPress(glfw.KeyWorld1, r.ToggleDebug, nil)
   c.BindMouseMovement(r.PanView)
}

const (
   PROGRAM_FILL = "fill"
   PROGRAM_SCENE = "scene"
)

func (r *Receiver) Init(window *glfw.Window) {
   r.Window = window
   r.LoadConfiguration("gameconf.json")
   r.Invalid = true
   gtk.Bind(&r.Data)
   // var err error
   // err = gtk.LoadTexture(r.Data.Tex0, "tex4.png")
   // panicOnErr(err)
   // err = gtk.LoadTexture(r.Data.Tex1, "tex3.png")
   // panicOnErr(err)

   r.Shaders = gtk.NewShaderLibrary()
   r.Shaders.LoadProgram(PROGRAM_SCENE, "scene.v.glsl", "scene.f.glsl")
   r.Shaders.LoadProgram(PROGRAM_FILL, "fill.v.glsl", "fill.f.glsl")
   r.Shaders.BindProgramLocations(PROGRAM_SCENE, &r.SceneLoc)
   r.Shaders.BindProgramLocations(PROGRAM_FILL, &r.FillLoc)
   gtk.PanicOnError()
   
   r.Data.Projection = glm.Ident4d()
   r.Data.Cameraview = glm.Ident4d()
   r.Data.Inception = glm.Ident4d()
   
   r.Data.Scene = gtk.EmptyModel("root")
   r.LoadScene("portal.dae")

   quadElements := gtk.MakeElements(portal.QuadElements)
   r.Data.Scene.AddGeometry(NewPlane("plane1", portal.Quad{
         glm.Vec4d{0, 0, 0, 1},
         glm.Vec4d{0, 1, 0, 0},
         glm.Vec4d{1, 0, 0, 0},
         glm.Vec4d{10, 10, 1, 0},
      }, 
      quadElements,
   ))
   r.Data.Fill = NewPlane("plane1", portal.Quad {
         glm.Vec4d{0, 0, 0, 1},
         glm.Vec4d{0, 0, 1, 0},
         glm.Vec4d{1, 0, 0, 0},
         glm.Vec4d{1, 1, 1, 0},
      }, 
      quadElements,
   )
   
   // r.Portals = append(r.Portals, CreatePortals()...)
   r.Data.Portal = gtk.EmptyModel("portals")
   for i, p := range r.Portals {
      r.Data.Portal.AddGeometry(NewPlane(fmt.Sprintf("portal_%d", i), p.EventHorizon, quadElements))
   }
   r.Player = Player{
      glm.Vec4d{0,1,0,1},
      glm.Vec4d{0,0,0,0},
      glm.Vec4d{0,1,0,0},
      glm.Vec4d{1,0,0,0},
      glm.QuatIdentd(),
      glm.QuatIdentd(),
   }
}

func (r *Receiver) LoadScene(filename string) {
   doc, err := collada.LoadDocument(filename)
   panicOnErr(err)
   index, err := gtk.NewIndex(doc)
   panicOnErr(err)
   r.SceneIndex = index
   
   model := gtk.EmptyModel("scene")
   switch doc.Asset.UpAxis {
   case collada.Xup:
   case collada.Yup:
   case collada.Zup:
      model.Transform = glm.HomogRotate3DXd(-90).Mul4(glm.HomogRotate3DZd(90))
   }
   
   portalPattern, _ := regexp.Compile("^Portal_(\\d+)_(\\d+)")
   
   geometryTemplates := make(map[collada.Id][]*gtk.Geometry)
   for id, mesh := range r.SceneIndex.Mesh {
      geoms := make([]*gtk.Geometry, 0)
      for _, pl := range mesh.Polylist {
         matches := portalPattern.FindStringSubmatch(mesh.VerticesId)
         if matches == nil {
            elements := make([]*gtk.DrawElements, 0)
            drawElements := gtk.NewDrawElements(pl.TriangleElements, gl.TRIANGLES)
            if drawElements != nil {
               elements = append(elements, drawElements)
            }
            geometry := gtk.NewGeometry(string(id), pl.VertexData, pl.NormalData, elements)
            geoms = append(geoms, geometry)  
         } else {
            fmt.Println("ignoring Portal")
         }
      }
      if len(geoms) > 0 {
         geometryTemplates[id] = geoms
      } 
   }
   
   portalLink := map[int]int{}
   portals := map[int]portal.Quad{}
   for _, node := range r.SceneIndex.VisualScene.Node {
      matches := portalPattern.FindStringSubmatch(node.Name)
      transform := r.SceneIndex.Transforms[node.Id]
      if matches == nil {
         geoms := make([]*gtk.Geometry, 0)
         for _, geoinstance := range node.InstanceGeometry {
            geoid, _ := geoinstance.Url.Id()
            geoms = append(geoms, geometryTemplates[geoid]...)
         }
         if len(geoms) > 0 {
            child := gtk.NewModel(node.Name, []*gtk.Model{}, geoms, transform)
            model.AddChild(child)
         }
      } else {
         index, err := strconv.Atoi(matches[1])
         if err != nil { panic(err) }
         exit, err := strconv.Atoi(matches[2])
         if err != nil { panic(err) }

         mt := model.Transform.Mul4(transform)
         center := mt.Mul4x1(glm.Vec4d{0,0,0,1})
         normal := mt.Mul4x1(glm.Vec4d{0,0,1,0}).Normalize()
         planev := mt.Mul4x1(glm.Vec4d{1,0,0,0}).Normalize()
         scale := glm.Vec4d{}
         n := 0
         for i := 0; i < 3; i++ {
            sum := 0.0
            for j := 0; j < 3; j++ {
               sum += float64(mt[n] * mt[n])
               n++
            }
            n++
            scale[i] = math.Sqrt(sum)
         }
         
         quad := portal.Quad {
            center,
            normal,
            planev,
            scale,
         }         
         portalLink[index] = exit
         portals[index] = quad
         // fmt.Println("portal node", node.Name, quad)
      }
   }
   r.Data.Scene.AddChild(model)
   
   for id, quad := range portals {
      exit, ok := portals[portalLink[id]]
      if ok {
         up := portal.Cross3D(exit.Normal, exit.PlaneV)
         // fmt.Println("pre ",exit)
         exit = exit.Apply(glm.HomogRotate3Dd(math.Pi, up))
         // fmt.Println("post",exit)
         _, inverse, pva, _ := portal.PortalTransform(quad, exit)
         portal := portal.Portal{quad, inverse, pva}
         r.Portals = append(r.Portals, portal)
      } else {
         fmt.Println("no exit for portal", id, portalLink[id])
      }
   }
}

func CreatePortals() []portal.Portal {
   a := portal.Quad{
      glm.Vec4d{0, 1, -5, 1},
      glm.Vec4d{0, 0, 1, 0},
      glm.Vec4d{1, 0, 0, 0},
      glm.Vec4d{1, 1, 1, 0},
   }
   r := glm.HomogRotate3DYd(90)
   b := portal.Quad{
      glm.Vec4d{-2, 1, 0, 1},
      r.Mul4x1(glm.Vec4d{0,0,1,0}),
      r.Mul4x1(glm.Vec4d{1,0,0,0}),
      glm.Vec4d{1, 1, 1, 0},
   }
   transform, inverse, pva, pvb := portal.PortalTransform(a, b)
   pa := portal.Portal{a, inverse, pva}
   pb := portal.Portal{b, transform, pvb}
   return []portal.Portal{pa, pb}
}

func NewPlane(name string, q portal.Quad, elements []*gtk.DrawElements) *gtk.Geometry {
   vs, ns := q.Mesh()
   geometry := gtk.NewGeometry(name, vs,ns,elements)
   return geometry
}

func (r *Receiver) LoadConfiguration(confFile string) {
   r.Constants = DefaultConstants
   r.ResetKeyBindingDefaults()
   file, err := os.Open(confFile)
   if err == nil {
      defer file.Close()
      decoder := json.NewDecoder(file)
      root := map[string]interface{}{}
      err = decoder.Decode(&root)
      panicOnErr(err)
      if constants, ok := root["constants"]; ok {
         bytes, err := json.Marshal(constants)
         panicOnErr(err)
         err = json.Unmarshal(bytes, &r.Constants)
         panicOnErr(err)
      }
      if controls, ok := root["controls"]; ok {
         sc := make(map[string]string)
         for k, v := range controls.(map[string]interface{}) {
            sc[k] = v.(string)
         }
         r.Controls.Apply(r, sc)
      }
   }
}

func (r *Receiver) Draw(window *glfw.Window) {
   // fmt.Println("render", r.SimulationTime.Elapsed)
   bg := gtk.SoftBlack
   gl.ClearColor(bg[0], bg[1], bg[2], bg[3])
   gl.Enable(gl.DEPTH_TEST)
   gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
   
   r.Shaders.UseProgram(PROGRAM_FILL)
   gl.Uniform4fv(r.FillLoc.Color, 1, &gtk.SkyBlue[0])
   gl.Uniform1f(r.FillLoc.Depth, gl.Float(r.Constants.PlayerViewFar))
   r.Shaders.UseProgram(PROGRAM_SCENE)
   gl.Uniform1f(r.SceneLoc.ElapsedSeconds, gl.Float(r.SimulationTime.Elapsed))
   gl.Uniform1f(r.SceneLoc.Glow, 0)
   gl.UniformMatrix4fv(r.SceneLoc.Projection, 1, gl.FALSE, gtk.MatArray(r.Data.Projection))
   gl.UniformMatrix4fv(r.SceneLoc.Cameraview, 1, gl.FALSE, gtk.MatArray(r.Data.Cameraview))
   gl.UniformMatrix4fv(r.SceneLoc.Inception, 1, gl.FALSE, gtk.MatArray(r.Data.Inception))
   mv := glm.Ident4d()
   gl.UniformMatrix4fv(r.SceneLoc.Portalview, 1, gl.FALSE, gtk.MatArray(mv))
   gl.UniformMatrix4fv(r.SceneLoc.Worldview, 1, gl.FALSE, gtk.MatArray(mv))
   // gtk.AttachTexture(r.SceneLoc.Tex0, gl.TEXTURE0, gl.TEXTURE_2D, r.Data.Tex0)
   // gtk.AttachTexture(r.SceneLoc.Tex1, gl.TEXTURE1, gl.TEXTURE_2D, r.Data.Tex1)
   gtk.PanicOnError()

   r.DrawPortalScene(mv, 0, 1)
   r.Invalid = false
}

func (r *Receiver) DrawPortalScene(mv glm.Mat4d, stencilLevel int, depth int) {
   s := gtk.Stencil
   if depth == 0 {
      if stencilLevel > 0 {
         gl.Enable(gl.CLIP_DISTANCE0)
      }
      s.Enable().Mask(stencilLevel)
      r.DrawModel(mv, r.Data.Scene, false)
      s.Disable()
      gl.Disable(gl.CLIP_DISTANCE0)
   } else {
      s.Enable().Mask(stencilLevel).NoDraw()
      r.DrawModel(mv, r.Data.Scene, false)
      s.DepthLE().Increment()
      gl.Enable(gl.CULL_FACE)
      r.DrawModel(mv, r.Data.Portal, false)
      
      if r.Constants.Debug {
         r.DrawModel(mv, r.Data.Portal, true)
      }
      
      gl.Disable(gl.CULL_FACE)
      s.Draw().Keep()

      if stencilLevel > 0 {
         gl.Enable(gl.CLIP_DISTANCE0)
      }
      r.DrawModel(mv, r.Data.Scene, false)
      
      if r.Constants.Debug {
         gl.Uniform1f(r.SceneLoc.Glow, 1)
         s.Mask(stencilLevel+1)
         r.DrawModel(mv, r.Data.Portal, true)
         gl.Uniform1f(r.SceneLoc.Glow, 0)
      }
      
      gl.Disable(gl.CLIP_DISTANCE0)
      
      //scene drawn at stencil level
      //portal are at level 1
      r.StepDown(stencilLevel+1)
      r.Shaders.UseProgram(PROGRAM_SCENE)
      s.Enable().Depth().DepthLE().Mask(stencilLevel)
      //scene is at stencil level

      for i, portal := range r.Portals {
         s.NoDraw().Increment()
         gl.UniformMatrix4fv(r.SceneLoc.Worldview, 1, gl.FALSE, gtk.MatArray(mv))
         gl.Enable(gl.CULL_FACE)
         r.DrawGeometry(r.Data.Portal.Geometry[i], r.SceneLoc.Position, false)
         gl.Disable(gl.CULL_FACE)
         s.Draw().Keep()
         
         //need to clear the depth buffer on stencil level1
         r.Shaders.UseProgram(PROGRAM_FILL)
         s.NoDraw().DepthAlways().Mask(stencilLevel+1)
         // color := gtk.DebugPallet.Pick(i)
         // gl.Uniform4fv(r.FillLoc.Color, 1, &color[0])
         r.DrawGeometry(r.Data.Fill, r.FillLoc.Position, false)
         r.Shaders.UseProgram(PROGRAM_SCENE)
         s.Enable().Depth().DepthLE().Mask(stencilLevel)

         gl.UniformMatrix4fv(r.SceneLoc.Portalview, 1, gl.FALSE, gtk.MatArray(portal.Portalview))
         w1 := mv.Mul4(portal.Transform)
         r.DrawPortalScene(w1, stencilLevel+1, depth-1)
         
         r.StepDown(stencilLevel+1)
         r.Shaders.UseProgram(PROGRAM_SCENE)
         s.Enable().Depth().DepthLE().Mask(stencilLevel)
      }
      s.Disable()
   }
}

func (r *Receiver) StepDown(stencilLevel int) {
   s := gtk.Stencil
   r.Shaders.UseProgram(PROGRAM_FILL)
   s.Enable().NoDepth().NoDepthMask().NoDraw().Mask(stencilLevel).Decrement()
   r.DrawGeometry(r.Data.Fill, r.FillLoc.Position, false)
   s.Disable()
}

func (r *Receiver) DrawModel(mv glm.Mat4d, model *gtk.Model, lines bool) {
   mv2 := mv.Mul4(model.Transform)
   gl.UniformMatrix4fv(r.SceneLoc.Worldview, 1, gl.FALSE, gtk.MatArray(mv2))
   for _, geo := range model.Geometry {
      r.DrawGeometry(geo, r.SceneLoc.Position, lines)
   }
   for _, child := range model.Children {
      r.DrawModel(mv2, child, lines)
   }
}

func (r *Receiver) DrawGeometry(geo *gtk.Geometry, vertexAttribute gl.AttributeLocation, lines bool) {
   gl.BindBuffer(gl.ARRAY_BUFFER, geo.VertexBuffer)
   gl.BindVertexArray(r.Data.Vao)
   gl.VertexAttribPointer(vertexAttribute, 3, gl.FLOAT, gl.FALSE, 12, nil)
   gl.EnableVertexAttribArray(vertexAttribute)
   
   for _, elem := range geo.Elements {
      if lines == (elem.DrawType == gl.LINES) {
         gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, elem.Buffer)
         gl.DrawElements(elem.DrawType, gl.Sizei(elem.Count), gl.UNSIGNED_SHORT, nil)
         gtk.PanicOnError()         
      }
   }
   gl.DisableVertexAttribArray(vertexAttribute)
}

func (r *Receiver) Reshape(window *glfw.Window, width, height int) {
   aspectRatio := gtk.WindowAspectRatio(window)
   fov := r.Constants.PlayerFOV
   r.Data.Projection = glm.Perspectived(fov, aspectRatio, r.Constants.PlayerViewNear, r.Constants.PlayerViewFar)
}

func (r *Receiver) MouseClick(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
   boundAction, ok := r.Controls.FindClickAction(button, action)
   if ok {
      boundAction()
   }
}

func (r *Receiver) MouseMove(window *glfw.Window, xpos float64, ypos float64) {
   pos := MouseCoord(window, xpos, ypos)
   boundAction, ok := r.Controls.FindMouseMovementAction()
   if ok {
      delta := pos.Sub(r.LastMousePosition)
      if !r.HasLastMousePosition {
         r.HasLastMousePosition = true
         delta = glm.Vec2d{}
      }
      boundAction(pos, delta)
   }
   r.LastMousePosition = pos
}

func MouseCoord(window *glfw.Window, xpos, ypos float64) glm.Vec2d {
   width, height := window.GetSize()
   return glm.Vec2d{xpos / float64(width), 1 - ypos / float64(height)}
}

func (r *Receiver) KeyPress(window *glfw.Window, k glfw.Key, s int, action glfw.Action, mods glfw.ModifierKey) {
   boundAction, ok := r.Controls.FindKeyAction(k, action)
   if ok {
      boundAction()
   }
}

func (r *Receiver) Scroll(window *glfw.Window, xoff float64, yoff float64) {
}

func (r *Receiver) Simulate(gameTime gtk.GameTime) {
   r.SimulationTime = gameTime
   deltaT := gameTime.Delta.Seconds()
   
   if !r.UIState.Impulse.ApproxEqual(glm.Vec4d{}) {
      regulatedImpulse := r.UIState.Impulse.Normalize().Mul(r.Constants.PlayerImpulseMomentumLimit)
      viewAdjustedImpulse := gtk.ToHomogVec4D(r.Player.OrientationH.Rotate(gtk.ToVec3D(regulatedImpulse)))
      r.Player.Velocity = r.Player.Velocity.Add(viewAdjustedImpulse)
      r.UIState.Impulse = glm.Vec4d{}
   }

   aggregateVelocity := r.Player.Velocity
   if !r.UIState.Movement.ApproxEqual(glm.Vec4d{}) {
      regulatedMovement := r.UIState.Movement.Normalize().Mul(r.Constants.PlayerMovementLimit)
      viewAdjustedMovement := gtk.ToHomogVec4D(r.Player.OrientationH.Rotate(gtk.ToVec3D(regulatedMovement)))
      aggregateVelocity = aggregateVelocity.Add(viewAdjustedMovement)
   }

   dp := aggregateVelocity.Mul(deltaT)
   
   for _, p := range r.Portals {
      portalview := p.Portalview
      pos := portalview.Mul4x1(r.Player.Position)
      v := portalview.Mul4x1(dp)
      
      if pos[2] < 0 && v[2] > 0 {
         t := -pos[2] / v[2]
         hit := pos.Add(v.Mul(t))
         if math.Abs(float64(hit[0])) <= 1 && 
            math.Abs(float64(hit[1])) <= 1 && 
            t > 0 && t <= 1 {
            // fmt.Println("crossed portal", i, pos)
            ti := p.Transform.Inv()
            r.Player.Transform(ti)
            dp = ti.Mul4x1(dp)
            r.Data.Inception = r.Data.Inception.Mul4(p.Transform)
            break
         }
      }
   }

   p0 := r.Player.Position
   r.Player.Position = p0.Add(dp)

   //apply gravity if player is off the ground (ys==0)
   if r.Player.Position[1] > 1 {
      r.Player.Velocity[1] = r.Player.Velocity[1] + r.Constants.Gravity * deltaT
   } else {
      r.Player.Velocity[1] = 0
      if p0[1] >= 1 {
         r.Player.Position[1] = 1
      }
   }

   p := r.Player.Position
   translate := glm.Translate3Dd(-p[0], -p[1], -p[2])
   rotation := r.Player.Orientation.Conjugate().Mat4()
   r.Data.Cameraview = rotation.Mul4(translate)
}

func (r *Receiver) OnClose(window *glfw.Window) {
}

func (r *Receiver) IsIdle() bool {
   if !r.UIState.Impulse.ApproxEqual(glm.Vec4d{}) {
      return false
   }
   if !r.UIState.Movement.ApproxEqual(glm.Vec4d{}) {
      return false
   }
   if !r.Player.Velocity.ApproxEqual(glm.Vec4d{}) {
      return false
   }
   // if r.Player.Position[1] > 1 {
   //    return false
   // }
   
   return true
}

func (r *Receiver) Quit() {
   r.Window.SetShouldClose(true)
}

func (r *Receiver) PanView(pos, delta glm.Vec2d) {
   theta := delta.Mul(r.Constants.PlayerFOV * r.Constants.PlayerPanSensitivity)   
   
   turnV := glm.QuatRotated(theta[1], gtk.ToVec3D(r.Player.TiltAxis))
   turnH := glm.QuatRotated(-theta[0], gtk.ToVec3D(r.Player.PanAxis))
   
   r.Player.OrientationH = turnH.Mul(r.Player.OrientationH)
   r.Player.Orientation = turnH.Mul(r.Player.Orientation).Mul(turnV)
   
   r.Invalid = true
}

func (r *Receiver) NeedsRender() bool {
   return !r.IsIdle() || r.Invalid
}

func (r *Receiver) ToggleDebug() {
   r.Invalid = true
   r.Constants.Debug = !r.Constants.Debug
}

func (r *Receiver) MoveUp() {
   r.UIState.Movement[1]++
}

func (r *Receiver) StopMoveUp() {
   r.UIState.Movement[1]--
}

func (r *Receiver) MoveDown() {
   r.UIState.Movement[1]--
}

func (r *Receiver) StopMoveDown() {
   r.UIState.Movement[1]++
}

func (r *Receiver) MoveForward() {
   r.UIState.Movement[2]--
}

func (r *Receiver) StopMoveForward() {
   r.UIState.Movement[2]++
}

func (r *Receiver) MoveBackward() {
   r.UIState.Movement[2]++
}

func (r *Receiver) StopMoveBackward() {
   r.UIState.Movement[2]--
}

func (r *Receiver) StrafeLeft() {
   r.UIState.Movement[0]--
}

func (r *Receiver) StopStrafeLeft() {
   r.UIState.Movement[0]++
}

func (r *Receiver) StrafeRight() {
   r.UIState.Movement[0]++
}

func (r *Receiver) StopStrafeRight() {
   r.UIState.Movement[0]--
}

func (r *Receiver) Jump() {
   r.UIState.Impulse[1]++
}
