package request

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// binaryPrefix starts a composer line that is sent as a binary message:
// "hex: de ad be ef".
const binaryPrefix = "hex:"

// BinaryMessage reads a composer line written as hex bytes ("hex: de ad
// be ef", spaces free). ok is false when the line is plain text; a hex line
// that does not decode is an error, so nothing bad is ever sent.
func BinaryMessage(text string) (data []byte, ok bool, err error) {
	rest, ok := strings.CutPrefix(text, binaryPrefix)
	if !ok {
		return nil, false, nil
	}
	data, err = hex.DecodeString(strings.Join(strings.Fields(rest), ""))
	if err != nil {
		return nil, true, fmt.Errorf("hex: %v", err)
	}
	return data, true, nil
}
