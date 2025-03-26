package hcl

type maskType int

const (
	FullMask maskType = iota
	PartialMask
	Default
)

type MaskConfig struct {
	Field     string
	MaskType  maskType
	ShowFirst int
	ShowLast  int
}
