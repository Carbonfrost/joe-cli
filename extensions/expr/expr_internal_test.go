package expr // intentional

func IsVisible(e *Expr) bool {
	return !e.internalFlags().hidden()
}
