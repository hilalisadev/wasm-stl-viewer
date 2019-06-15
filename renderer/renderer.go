package renderer

import (
	"fmt"
	"syscall/js"
	"unsafe"

	"github.com/bobcob7/wasm-stl-viewer/gltypes"
	"github.com/go-gl/mathgl/mgl32"
)

type InitialConfig struct {
	Width              int
	Height             int
	SpeedX             float32
	SpeedY             float32
	SpeedZ             float32
	Colors             []float32
	Vertices           []float32
	Indices            []uint16
	FragmentShaderCode string
	VertexShaderCode   string
}

type Renderer struct {
	glContext      js.Value
	glTypes        gltypes.GLTypes
	colors         js.TypedArray
	vertices       js.TypedArray
	indices        js.TypedArray
	colorBuffer    js.Value
	vertexBuffer   js.Value
	indexBuffer    js.Value
	numIndices     int
	fragShader     js.Value
	vertShader     js.Value
	shaderProgram  js.Value
	tmark          float32
	rotation       float32
	movMatrix      mgl32.Mat4
	PositionMatrix js.Value
	ViewMatrix     js.Value
	ModelMatrix    js.Value
	height         int
	width          int
	speedX         float32
	speedY         float32
	speedZ         float32
}

func NewRenderer(gl js.Value, config InitialConfig) (r Renderer) {
	// Get some WebGL bindings
	r.glContext = gl
	r.glTypes.New(r.glContext)
	r.numIndices = len(config.Indices)
	r.movMatrix = mgl32.Ident4()
	r.width = config.Width
	r.height = config.Height
	fmt.Println("Size", r.width, r.height)

	r.speedX = config.SpeedX
	r.speedY = config.SpeedY
	r.speedZ = config.SpeedZ

	// Convert buffers to JS TypedArrays
	r.UpdateColorBuffer(config.Colors)
	r.UpdateVerticesBuffer(config.Vertices)
	r.UpdateIndicesBuffer(config.Indices)

	r.UpdateFragmentShader(config.FragmentShaderCode)
	r.UpdateVertexShader(config.VertexShaderCode)
	r.updateShaderProgram()
	r.attachShaderProgram()

	r.setContextFlags()

	r.createMatrixes()
	return
}

func (r *Renderer) Release() {
	fmt.Println("Renderer.Release")
}

func (r *Renderer) EnableObject() {
	fmt.Println("Renderer.EnableObject")
	r.glContext.Call("bindBuffer", r.glTypes.ElementArrayBuffer, r.indexBuffer)
}

func (r *Renderer) SetSpeedX(x float32) {
	r.speedX = x
}

func (r *Renderer) SetSpeedY(y float32) {
	r.speedY = y
}

func (r *Renderer) SetSpeedZ(z float32) {
	r.speedZ = z
}

func (r *Renderer) GetSpeed() (x, y, z float32) {
	return r.speedX, r.speedY, r.speedZ
}

func (r *Renderer) SetSize(height, width int) {
	r.height = height
	r.width = width
	fmt.Println("Size", r.width, r.height)
}

func (r *Renderer) createMatrixes() {
	ratio := float32(r.width) / float32(r.height)
	fmt.Println("Renderer.createMatrixes")
	// Generate and apply projection matrix
	projMatrix := mgl32.Perspective(mgl32.DegToRad(45.0), ratio, 1, 100.0)
	var projMatrixBuffer *[16]float32
	projMatrixBuffer = (*[16]float32)(unsafe.Pointer(&projMatrix))
	typedProjMatrixBuffer := js.TypedArrayOf([]float32((*projMatrixBuffer)[:]))
	r.glContext.Call("uniformMatrix4fv", r.PositionMatrix, false, typedProjMatrixBuffer)

	// Generate and apply view matrix
	viewMatrix := mgl32.LookAtV(mgl32.Vec3{3.0, 3.0, 3.0}, mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, 1.0, 0.0})
	var viewMatrixBuffer *[16]float32
	viewMatrixBuffer = (*[16]float32)(unsafe.Pointer(&viewMatrix))
	typedViewMatrixBuffer := js.TypedArrayOf([]float32((*viewMatrixBuffer)[:]))
	r.glContext.Call("uniformMatrix4fv", r.ViewMatrix, false, typedViewMatrixBuffer)
}

func (r *Renderer) setContextFlags() {
	fmt.Println("Renderer.setContextFlags")
	// Set WebGL properties
	r.glContext.Call("clearColor", 0.5, 0.5, 0.5, 0.9)    // Color the screen is cleared to
	r.glContext.Call("clearDepth", 1.0)                   // Z value that is set to the Depth buffer every frame
	r.glContext.Call("viewport", 0, 0, r.width, r.height) // Viewport size
	r.glContext.Call("depthFunc", r.glTypes.LEqual)
}

func (r *Renderer) UpdateFragmentShader(shaderCode string) {
	fmt.Println("Renderer.UpdateFragmentShader")
	// Create fragment shader object
	r.fragShader = r.glContext.Call("createShader", r.glTypes.FragmentShader)
	r.glContext.Call("shaderSource", r.fragShader, shaderCode)
	r.glContext.Call("compileShader", r.fragShader)
}

func (r *Renderer) UpdateVertexShader(shaderCode string) {
	fmt.Println("Renderer.UpdateVertexShader")
	// Create vertex shader object
	r.vertShader = r.glContext.Call("createShader", r.glTypes.VertexShader)
	r.glContext.Call("shaderSource", r.vertShader, shaderCode)
	r.glContext.Call("compileShader", r.vertShader)
}

func (r *Renderer) updateShaderProgram() {
	fmt.Println("Renderer.updateShaderProgram")
	if r.fragShader == js.Undefined() || r.vertShader == js.Undefined() {
		return
	}
	r.shaderProgram = r.glContext.Call("createProgram")
	r.glContext.Call("attachShader", r.shaderProgram, r.vertShader)
	r.glContext.Call("attachShader", r.shaderProgram, r.fragShader)
	r.glContext.Call("linkProgram", r.shaderProgram)
}

func (r *Renderer) attachShaderProgram() {
	fmt.Println("Renderer.attachShaderProgram")
	// Associate attributes to vertex shader
	r.PositionMatrix = r.glContext.Call("getUniformLocation", r.shaderProgram, "Pmatrix")
	r.ViewMatrix = r.glContext.Call("getUniformLocation", r.shaderProgram, "Vmatrix")
	r.ModelMatrix = r.glContext.Call("getUniformLocation", r.shaderProgram, "Mmatrix")

	r.glContext.Call("bindBuffer", r.glTypes.ArrayBuffer, r.vertexBuffer)
	position := r.glContext.Call("getAttribLocation", r.shaderProgram, "position")
	r.glContext.Call("vertexAttribPointer", position, 3, r.glTypes.Float, false, 0, 0)
	r.glContext.Call("enableVertexAttribArray", position)

	r.glContext.Call("bindBuffer", r.glTypes.ArrayBuffer, r.colorBuffer)
	color := r.glContext.Call("getAttribLocation", r.shaderProgram, "color")
	r.glContext.Call("vertexAttribPointer", color, 3, r.glTypes.Float, false, 0, 0)
	r.glContext.Call("enableVertexAttribArray", color)

	r.glContext.Call("useProgram", r.shaderProgram)
}

func (r *Renderer) UpdateColorBuffer(buffer []float32) {
	fmt.Println("Renderer.UpdateColorBuffer")
	r.colors = js.TypedArrayOf(buffer)
	if r.colorBuffer != js.Undefined() {
		// Delete previous color buffer
		fmt.Println("Deleting color buffer first")
		r.glContext.Call("deleteBuffer", r.colorBuffer)
	}
	// Create color buffer
	r.colorBuffer = r.glContext.Call("createBuffer")
	r.glContext.Call("bindBuffer", r.glTypes.ArrayBuffer, r.colorBuffer)
	r.glContext.Call("bufferData", r.glTypes.ArrayBuffer, r.colors, r.glTypes.StaticDraw)
}

func (r *Renderer) UpdateVerticesBuffer(buffer []float32) {
	fmt.Println("Renderer.UpdateVerticesBuffer")
	r.vertices = js.TypedArrayOf(buffer)
	if r.vertexBuffer != js.Undefined() {
		// Delete previous vertex buffer
		fmt.Println("Deleting vertex buffer first")
		r.glContext.Call("deleteBuffer", r.vertexBuffer)
	}
	// Create vertex buffer
	r.vertexBuffer = r.glContext.Call("createBuffer")
	r.glContext.Call("bindBuffer", r.glTypes.ArrayBuffer, r.vertexBuffer)
	r.glContext.Call("bufferData", r.glTypes.ArrayBuffer, r.vertices, r.glTypes.StaticDraw)
}

func (r *Renderer) UpdateIndicesBuffer(buffer []uint16) {
	fmt.Println("Renderer.UpdateIndicesBuffer")
	r.indices = js.TypedArrayOf(buffer)
	if r.indexBuffer != js.Undefined() {
		// Delete previous index buffer
		fmt.Println("Deleting index buffer first")
		r.glContext.Call("deleteBuffer", r.indexBuffer)
	}
	// Create index buffer
	r.indexBuffer = r.glContext.Call("createBuffer")
	r.glContext.Call("bindBuffer", r.glTypes.ElementArrayBuffer, r.indexBuffer)
	r.glContext.Call("bufferData", r.glTypes.ElementArrayBuffer, r.indices, r.glTypes.StaticDraw)
}

func (r *Renderer) Render(this js.Value, args []js.Value) interface{} {
	// Calculate rotation rate
	now := float32(args[0].Float())
	tdiff := now - r.tmark
	r.tmark = now
	r.rotation = r.rotation + float32(tdiff)/500

	// Do new model matrix calculations
	r.movMatrix = mgl32.HomogRotate3DX(r.speedX * r.rotation)
	r.movMatrix = r.movMatrix.Mul4(mgl32.HomogRotate3DY(r.speedY * r.rotation))
	r.movMatrix = r.movMatrix.Mul4(mgl32.HomogRotate3DZ(r.speedZ * r.rotation))

	// Convert model matrix to a JS TypedArray
	var modelMatrixBuffer *[16]float32
	modelMatrixBuffer = (*[16]float32)(unsafe.Pointer(&r.movMatrix))
	typedModelMatrixBuffer := js.TypedArrayOf([]float32((*modelMatrixBuffer)[:]))

	// Apply the model matrix
	r.glContext.Call("uniformMatrix4fv", r.ModelMatrix, false, typedModelMatrixBuffer)

	// Clear the screen
	r.glContext.Call("enable", r.glTypes.DepthTest)
	r.glContext.Call("clear", r.glTypes.ColorBufferBit)
	r.glContext.Call("clear", r.glTypes.DepthBufferBit)

	// Draw the cube
	r.glContext.Call("drawElements", r.glTypes.Triangles, r.numIndices, r.glTypes.UnsignedShort, 0)

	return nil
}