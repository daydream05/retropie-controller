package gamepad

import "fmt"

type Buttons struct {
	A      bool `json:"a"`
	B      bool `json:"b"`
	X      bool `json:"x"`
	Y      bool `json:"y"`
	LB     bool `json:"lb"`
	RB     bool `json:"rb"`
	Start  bool `json:"start"`
	Select bool `json:"select"`
	Up     bool `json:"up"`
	Down   bool `json:"down"`
	Left   bool `json:"left"`
	Right  bool `json:"right"`
}

type backend interface {
	SetButtons(Buttons) error
	ReleaseAll() error
	Close() error
}

type Manager struct {
	devices []backend
}

func NewManager(count int) (*Manager, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	devices := make([]backend, 0, count)
	for i := 0; i < count; i++ {
		device, err := newBackend(i + 1)
		if err != nil {
			for _, created := range devices {
				_ = created.ReleaseAll()
				_ = created.Close()
			}
			return nil, err
		}
		devices = append(devices, device)
	}

	return &Manager{devices: devices}, nil
}

func (m *Manager) Count() int {
	return len(m.devices)
}

func (m *Manager) SetButtons(player int, buttons Buttons) error {
	device, err := m.device(player)
	if err != nil {
		return err
	}
	return device.SetButtons(buttons)
}

func (m *Manager) ReleaseAll(player int) error {
	device, err := m.device(player)
	if err != nil {
		return err
	}
	return device.ReleaseAll()
}

func (m *Manager) ReleaseAllPlayers() error {
	var firstErr error
	for _, device := range m.devices {
		if err := device.ReleaseAll(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) Close() error {
	var firstErr error
	for _, device := range m.devices {
		if err := device.ReleaseAll(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := device.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) device(player int) (backend, error) {
	if player < 1 || player > len(m.devices) {
		return nil, fmt.Errorf("invalid player %d", player)
	}
	return m.devices[player-1], nil
}
