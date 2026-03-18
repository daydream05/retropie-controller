//go:build linux

package gamepad

import (
	"fmt"

	"github.com/bendahl/uinput"
)

const (
	btnA      = 304
	btnB      = 305
	btnX      = 307
	btnY      = 308
	btnLB     = 310
	btnRB     = 311
	btnSelect = 314
	btnStart  = 315
)

type linuxBackend struct {
	gamepad uinput.Gamepad
	buttons Buttons
}

func newBackend(player int) (backend, error) {
	device, err := uinput.CreateGamepad("/dev/uinput", []byte(fmt.Sprintf("BrowserPad %d", player)), 0x045e, 0x028e)
	if err != nil {
		return nil, err
	}
	return &linuxBackend{gamepad: device}, nil
}

func (b *linuxBackend) SetButtons(next Buttons) error {
	if err := b.setButton(btnA, b.buttons.A, next.A); err != nil {
		return err
	}
	if err := b.setButton(btnB, b.buttons.B, next.B); err != nil {
		return err
	}
	if err := b.setButton(btnX, b.buttons.X, next.X); err != nil {
		return err
	}
	if err := b.setButton(btnY, b.buttons.Y, next.Y); err != nil {
		return err
	}
	if err := b.setButton(btnLB, b.buttons.LB, next.LB); err != nil {
		return err
	}
	if err := b.setButton(btnRB, b.buttons.RB, next.RB); err != nil {
		return err
	}
	if err := b.setButton(btnStart, b.buttons.Start, next.Start); err != nil {
		return err
	}
	if err := b.setButton(btnSelect, b.buttons.Select, next.Select); err != nil {
		return err
	}
	if err := b.setHat(uinput.HatUp, b.buttons.Up, next.Up); err != nil {
		return err
	}
	if err := b.setHat(uinput.HatDown, b.buttons.Down, next.Down); err != nil {
		return err
	}
	if err := b.setHat(uinput.HatLeft, b.buttons.Left, next.Left); err != nil {
		return err
	}
	if err := b.setHat(uinput.HatRight, b.buttons.Right, next.Right); err != nil {
		return err
	}

	b.buttons = next
	return nil
}

func (b *linuxBackend) ReleaseAll() error {
	if err := b.SetButtons(Buttons{}); err != nil {
		return err
	}
	return nil
}

func (b *linuxBackend) Close() error {
	return b.gamepad.Close()
}

func (b *linuxBackend) setButton(code int, current bool, next bool) error {
	switch {
	case !current && next:
		return b.gamepad.ButtonDown(code)
	case current && !next:
		return b.gamepad.ButtonUp(code)
	default:
		return nil
	}
}

func (b *linuxBackend) setHat(direction uinput.HatDirection, current bool, next bool) error {
	switch {
	case !current && next:
		return b.gamepad.HatPress(direction)
	case current && !next:
		return b.gamepad.HatRelease(direction)
	default:
		return nil
	}
}
