package styles

type CensysColorScheme struct{}

var _ ColorScheme = CensysColorScheme{}

func (c CensysColorScheme) Signature() Color {
	return Color{Light: "#ed9134", Dark: "#FFAD5B"}
}

func (c CensysColorScheme) Primary() Color {
	return Color{Light: "#000000", Dark: "#FBFAF6"}
}

func (c CensysColorScheme) Secondary() Color {
	return Color{Light: "#53b8b4", Dark: "#B6D5D4"}
}

func (c CensysColorScheme) Tertiary() Color {
	return Color{Light: "#387782", Dark: "#387782"}
}

func (c CensysColorScheme) Info() Color {
	return Color{Light: "#38a7ab", Dark: "#38a7ab"}
}

func (c CensysColorScheme) Warning() Color {
	return Color{Light: "#a39a5f", Dark: "#BCB480"}
}

func (c CensysColorScheme) Danger() Color {
	return Color{Light: "#dc322f", Dark: "#dc322f"}
}

func (c CensysColorScheme) Comment() Color {
	return Color{Light: "#808080", Dark: "#808080"}
}
