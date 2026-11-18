package reality

import (
	"bytes"
	"crypto/mlkem"
	"testing"
)

func TestOnlyALoweredFloorTakesLegacyHellos(t *testing.T) {
	for _, c := range []struct {
		floor []byte
		takes bool
	}{
		{nil, false},
		{[]byte{26, 3, 27}, false},
		{[]byte{26, 11, 9}, false},
		{[]byte{26, 3, 26}, true},
		{[]byte{0, 0, 0}, true},
	} {
		if got := (&Config{MinClientVer: c.floor}).takesLegacyHello(); got != c.takes {
			t.Errorf("floor %v: takes legacy hellos = %v, want %v", c.floor, got, c.takes)
		}
	}
}

func TestLegacyAuthShare(t *testing.T) {
	plain := bytes.Repeat([]byte{1}, 32)
	hybrid := append(bytes.Repeat([]byte{2}, mlkem.EncapsulationKeySize768), bytes.Repeat([]byte{3}, 32)...)

	for name, c := range map[string]struct {
		shares []keyShare
		want   []byte
	}{
		"a hello of an old client":    {[]keyShare{{group: X25519, data: plain}}, plain},
		"plain before hybrid":         {[]keyShare{{group: X25519, data: plain}, {group: X25519MLKEM768, data: hybrid}}, plain},
		"hybrid alone":                {[]keyShare{{group: X25519MLKEM768, data: hybrid}}, hybrid[mlkem.EncapsulationKeySize768:]},
		"no share REALITY can key on": {[]keyShare{{group: CurveP256, data: plain}}, nil},
	} {
		if got := legacyAuthShare(c.shares); !bytes.Equal(got, c.want) {
			t.Errorf("%s: got %x, want %x", name, got, c.want)
		}
	}
}
