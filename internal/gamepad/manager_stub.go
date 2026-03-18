//go:build !linux

package gamepad

type stubBackend struct {
	buttons Buttons
}

func newBackend(_ int) (backend, error) {
	return &stubBackend{}, nil
}

func (b *stubBackend) SetButtons(buttons Buttons) error {
	b.buttons = buttons
	return nil
}

func (b *stubBackend) ReleaseAll() error {
	b.buttons = Buttons{}
	return nil
}

func (b *stubBackend) Close() error {
	return nil
}
