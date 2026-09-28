package blinkstick

import "testing"

func TestNewInfo(t *testing.T) {
	tests := []struct {
		name        string
		di          deviceInfo
		wantModel   Model
		wantVersion string
	}{
		{"nano", deviceInfo{serial: "BS072777-3.0", release: 0x0202}, Nano, "3.0"},
		{"square", deviceInfo{serial: "BS073788-3.1", release: 0x0201}, Square, "3.1"},
		{"pro", deviceInfo{serial: "BS000001-2.0", release: 0x0200}, unknownModel, "2.0"},
		{"flex", deviceInfo{serial: "BS000002-3.0", release: 0x0203}, unknownModel, "3.0"},
		{"v1 with square release", deviceInfo{serial: "BS000003-1.0", release: 0x0201}, unknownModel, "1.0"},
		{"garbage serial", deviceInfo{serial: "garbage", release: 0x0202}, unknownModel, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newInfo(tt.di)
			if got.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", got.Model, tt.wantModel)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVersion)
			}
			if got.Serial != tt.di.serial {
				t.Errorf("Serial = %q, want %q", got.Serial, tt.di.serial)
			}
		})
	}
}

func TestNewInfoCopiesStrings(t *testing.T) {
	got := newInfo(deviceInfo{
		serial:       "BS072777-3.0",
		manufacturer: "Agile Innovative Ltd",
		product:      "BlinkStick Nano",
		release:      0x0202,
	})
	if got.Manufacturer != "Agile Innovative Ltd" || got.Product != "BlinkStick Nano" {
		t.Errorf("got %+v", got)
	}
}

func TestModelLEDs(t *testing.T) {
	if Nano.LEDs != 2 || Square.LEDs != 8 || unknownModel.LEDs != 0 {
		t.Errorf("LEDs: Nano %d, Square %d, unknown %d", Nano.LEDs, Square.LEDs, unknownModel.LEDs)
	}
}
