package reality

import "crypto/mlkem"

// hybridShareVer is the first Xray client whose every hello carries X25519MLKEM768.
var hybridShareVer = []byte{26, 3, 27}

// A floor below hybridShareVer asks for the clients before it, whose hellos have no hybrid share.
func (c *Config) takesLegacyHello() bool {
	return c.MinClientVer != nil && Value(c.MinClientVer...) < Value(hybridShareVer...)
}

// legacyAuthShare is the share a server keyed the auth on before it required the hybrid one:
// the first plain X25519, else the X25519 half of X25519MLKEM768.
func legacyAuthShare(shares []keyShare) []byte {
	for _, share := range shares {
		if share.group == X25519 && len(share.data) == 32 {
			return share.data
		}
	}

	for _, share := range shares {
		if share.group == X25519MLKEM768 && len(share.data) == mlkem.EncapsulationKeySize768+32 {
			return share.data[mlkem.EncapsulationKeySize768:]
		}
	}

	return nil
}
