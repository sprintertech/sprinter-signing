// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package comm

// MessageType represents message type identificator
type MessageType int64

const (
	// TssKeyGenMsg message type used for communicating key generation.
	TssKeyGenMsg MessageType = iota
	// TssKeySignMsg message type used for communicating signature for specific message.
	TssKeySignMsg
	// TssInitiateMsg message type sent by the leader to signify preparation for a tss process.
	TssInitiateMsg
	// TssStartMsg message type sent by a leader to signify the start of a tss process after parties sent the ready message.
	TssStartMsg
	// TssFailMsg message type sent by parties after an communication or tss error happens during process.
	TssFailMsg
	// TssReadyMsg message type sent by coordinator sent if the process fails.
	TssReadyMsg
	// TssReshareMsg message type used for resharing tss messages.
	TssReshareMsg
	// CoordinatorElectionMsg message type used to communicate that new election process needs to start.
	CoordinatorElectionMsg
	// CoordinatorAliveMsg  message type used to respond on CoordinatorElectionMsg message, signaling that peer is alive and ready for new election process.
	CoordinatorAliveMsg
	// CoordinatorLeaveMsg message type used to communicate that peer is going offline and will not participate in the future.
	CoordinatorLeaveMsg
	// CoordinatorSelectMsg message type used to communicate that sender has pronounced itself as a leader.
	CoordinatorSelectMsg
	// CoordinatorPingMsg message type used to check if the current coordinator is alive.
	CoordinatorPingMsg
	// CoordinatorPingResponseMsg message type used to respond on CoordinatorPingMsg message.
	CoordinatorPingResponseMsg
	// SignatureMsg message type is used to the share the signature to all relayers
	SignatureMsg
	// AcrossMsg message type is used for the process coordinator to share across data
	AcrossMsg
	// MayanMsg message type is used for the process coordinator to share mayan data
	MayanMsg
	// LighterMsg message type is used for the process coordinator to share lighter data
	LighterMsg
	// LifiEscrowMsg message type is used for the process coordinator to share lifi data
	LifiEscrowMsg
	// Rhinestone message type is used for the process coordinator to share rhinestone data
	RhinestoneMsg
	// LifiUnlockMsg message type is used for the process coordinator to share lifi unlock data
	LifiUnlockMsg
	// Unknown message type
	Unknown
)

const (
	SignatureSessionID  = "signatures"
	AcrossSessionID     = "across"
	MayanSessionID      = "mayan"
	LifiEscrowSessionID = "lifi-escrow"
	RhinestoneSessionID = "rhinestone"
	LighterSessionID    = "lighter"
	LifiUnlockSessionID = "lifi-unlock"
)

// String implements fmt.Stringer
func (msgType MessageType) String() string {
	switch msgType {
	case TssKeyGenMsg:
		return "TssKeyGenMsg"
	case TssKeySignMsg:
		return "TssKeySignMsg"
	case TssInitiateMsg:
		return "TssInitiateMsg"
	case TssStartMsg:
		return "TssStartMsg"
	case TssFailMsg:
		return "TssFailMsg"
	case TssReadyMsg:
		return "TssReadyMsg"
	case TssReshareMsg:
		return "TssReshareMsg"
	case CoordinatorElectionMsg:
		return "CoordinatorElectionMsg"
	case CoordinatorAliveMsg:
		return "CoordinatorAliveMsg"
	case CoordinatorLeaveMsg:
		return "CoordinatorLeaveMsg"
	case CoordinatorSelectMsg:
		return "CoordinatorSelectMsg"
	case CoordinatorPingMsg:
		return "CoordinatorPingMsg"
	case CoordinatorPingResponseMsg:
		return "CoordinatorPingResponseMsg"
	case AcrossMsg:
		return "AcrossMsg"
	case MayanMsg:
		return "MayanMsg"
	case RhinestoneMsg:
		return "RhinestoneMsg"
	case LifiEscrowMsg:
		return "LifiEscrowMsg"
	case LifiUnlockMsg:
		return "LifiUnlockMsg"
	case LighterMsg:
		return "LighterMsg"
	default:
		return "UnknownMsg"
	}
}
