package visual

// Palette gives terminal surfaces and foreground colors semantic roles.
type Palette struct {
	Background, Surface, Raised, Selection RGB
	Text, Muted, Border, Accent, Highlight RGB
	Success, Warning, Danger, Secondary    RGB
}

// Ink returns a dark palette with cyan focus and violet secondary accents.
func Ink() Palette {
	return Palette{
		Background: 0x10141c, Surface: 0x171d29, Raised: 0x202a39, Selection: 0x263e4a,
		Text: 0xe4edf7, Muted: 0xa0afc4, Border: 0x394861, Accent: 0x78d6dd, Highlight: 0xd9ffff,
		Success: 0x91d3a3, Warning: 0xf5cb83, Danger: 0xf4a49c, Secondary: 0xc1a8ee,
	}
}

// Theme is a named, reusable terminal palette.
type Theme struct {
	ID      string
	Palette Palette
}

// Themes returns a fresh catalog in picker order.
func Themes() []Theme {
	black := Palette{
		Background: 0x000000, Surface: 0x000000, Raised: 0x141414, Selection: 0x173928,
		Text: 0xffffff, Muted: 0xa0a0a0, Border: 0x454545, Accent: 0x8fe58f, Highlight: 0xf0fff0,
		Success: 0xa4e59b, Warning: 0xffd587, Danger: 0xffa297, Secondary: 0xaac4ff,
	}
	night := Palette{
		Background: 0x161326, Surface: 0x211b36, Raised: 0x302747, Selection: 0x45335f,
		Text: 0xf0e8fa, Muted: 0xb8aad0, Border: 0x55446b, Accent: 0xc8aaff, Highlight: 0xf8edff,
		Success: 0xa1dfb9, Warning: 0xf6ce85, Danger: 0xf4a6ad, Secondary: 0x9fd7fa,
	}
	return []Theme{{"ink", Ink()}, {"black", black}, {"violet", night}}
}

// PaletteFor resolves a saved theme name or falls back to Ink.
func PaletteFor(id string) Palette {
	for _, theme := range Themes() {
		if theme.ID == id {
			return theme.Palette
		}
	}
	return Ink()
}
