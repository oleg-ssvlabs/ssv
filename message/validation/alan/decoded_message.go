package validation

import (
	spectypes "github.com/bloxapp/ssv-spec/alan/types"
)

type DecodedMessage struct {
	*spectypes.SignedSSVMessage

	// Body is the decoded Data.
	Body any // *Message | *PartialSignatureMessages

	// TODO: consider adding important fields like Slot to have quick access to them
}
