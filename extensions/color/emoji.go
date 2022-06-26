package color

// Emoji represents emoji
type Emoji string

const (
	Tada           Emoji = "🎉"
	Fire           Emoji = "🔥"
	Sparkles       Emoji = "✨"
	Exclamation    Emoji = "❗"
	Bulb           Emoji = "💡"
	X              Emoji = "❌"
	HeavyCheckMark Emoji = "✔️"
	Warning        Emoji = "⚠️"
	Play           Emoji = "▶"
)

func emojiByName(name string) Emoji {
	switch name {
	case "Tada":
		return Tada
	case "Fire":
		return Fire
	case "Sparkles":
		return Sparkles
	case "Exclamation":
		return Exclamation
	case "Bulb":
		return Bulb
	case "X":
		return X
	case "HeavyCheckMark":
		return HeavyCheckMark
	case "Warning":
		return Warning
	case "Play":
		return Play
	}
	return ""
}
