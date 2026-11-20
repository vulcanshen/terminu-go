package terminu

// lastToast is the toast a test looks at: the newest, closing or not, else
// the inspect toast; nil for none.
func (a *App) lastToast() *layer {
	if n := len(a.toasts); n > 0 {
		return a.toasts[n-1]
	}
	return a.inspectT
}
