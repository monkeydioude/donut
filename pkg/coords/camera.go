package coords

type Camera struct {
	Coords
	MoveX       int
	MoveY       int
	Zoom        float32
	DefaultZoom float32
}

func (c *Camera) AddX(x int) {
	c.X += x
}

func (c *Camera) AddY(y int) {
	c.Y += y
}

func (c *Camera) CameraX(x int) int {
	return int(float32(c.X+x) * c.Zoom)
}

func (c *Camera) CameraY(y int) int {
	return int(float32(c.Y+y) * c.Zoom)
}

func (c *Camera) MovingX(val int) {
	c.MoveX = val
}

func (c *Camera) MovingY(val int) {
	c.MoveY = val
}

func (c *Camera) AddZoom(val float32) {
	tmp := c.Zoom + val
	if tmp <= 0 {
		return
	}
	c.Zoom = tmp
}

func (c *Camera) NormalizeSize(v int) int {
	return int(c.Zoom * float32(v))
}

func (c *Camera) ResetZoom() {
	c.Zoom = c.DefaultZoom
}

func (c *Camera) Update() {
	c.X += c.MoveX
	c.Y += c.MoveY
}

func NewCamera(x, y int) *Camera {
	return &Camera{
		Coords:      Coords{x, y},
		DefaultZoom: 1,
		Zoom:        1,
	}
}
