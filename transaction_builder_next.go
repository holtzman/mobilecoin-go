package api

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	account "github.com/jadeydi/mobilecoin-account"
	"github.com/jadeydi/mobilecoin-account/types"
)

type UTXO struct {
	TransactionHash string
	Index           uint32
	Amount          uint64
	PrivateKey      string
	ScriptPubKey    string
}

type Output struct {
	TransactionHash string
	RawTransaction  string
	SharedSecret    string
	Fee             uint64
	OutputIndex     int64
	OutputHash      string
	ChangeIndex     int64
	ChangeHash      string
	ChangeAmount    uint64
}

func TransactionBuilderBuild(inputs []*UTXO, proofs *Proofs, output string, amount, fee uint64, tombstone uint64, tokenID, version uint, changeStr string) (*Output, error) {
	recipient, err := account.DecodeB58Code(output)
	if err != nil {
		return nil, err
	}
	change, err := account.DecodeB58Code(changeStr)
	if err != nil {
		return nil, err
	}

	var totalAmount uint64 = 0
	unspentList := make([]*UnspentTxOut, len(inputs))
	for i, input := range inputs {
		totalAmount += input.Amount

		data, err := hex.DecodeString(input.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		var txOut TxOut
		err = json.Unmarshal(data, &txOut)
		if err != nil {
			return nil, err
		}

		onetimePrivateKey, err := RecoverOnetimePrivateKey(txOut.PublicKey, input.PrivateKey)
		if err != nil {
			return nil, err
		}
		image := hex.EncodeToString(KeyImageFromPrivate(onetimePrivateKey).Bytes())
		unspentList[i] = &UnspentTxOut{
			TxOut:                   &txOut,
			SubaddressIndex:         0,
			KeyImage:                image,
			Value:                   fmt.Sprint(input.Amount),
			AttemptedSpendHeight:    0,
			AttemptedSpendTombstone: 0,
			MonitorId:               "",
		}
	}

	changeAmount := totalAmount - amount - fee
	if changeAmount < MOB_MINIMUM_FEE {
		changeAmount = 0
		fee += changeAmount
	}
	if changeAmount > 0 && changeAmount < MILLIMOB_TO_PICOMOB {
		return nil, errors.New("invalid change amount")
	}
	if totalAmount != (amount + fee + changeAmount) {
		return nil, errors.New("invalid amount")
	}
	inputCs, err := BuildRingElements(inputs, proofs)
	if err != nil {
		return nil, err
	}

	txC, err := MCTransactionBuilderCreateC(inputCs, amount, changeAmount, fee, tombstone, tokenID, version, recipient, change)
	if err != nil {
		return nil, err
	}

	size := 1
	if changeAmount > 0 {
		size = 2
	}
	outlayList := make([]*Outlay, size)
	outlayIndexToTxOutIndex := make([][]int, size)
	outlayConfirmationNumbers := make([][]int, size)

	outlayList[0] = &Outlay{
		Value:    fmt.Sprint(amount),
		Receiver: recipient,
	}
	outlayIndexToTxOutIndex[0] = []int{0, 0}
	numsOut := make([]int, len(txC.ConfirmationOut))
	for i, b := range txC.ConfirmationOut {
		numsOut[i] = int(b)
	}
	outlayConfirmationNumbers[0] = numsOut

	if size == 2 {
		outlayList[1] = &Outlay{
			Value:    fmt.Sprint(changeAmount),
			Receiver: recipient,
		}
		outlayIndexToTxOutIndex[1] = []int{1, 1}
		numsChange := make([]int, len(txC.ConfirmationChange))
		for i, b := range txC.ConfirmationChange {
			numsChange[i] = int(b)
		}
		outlayConfirmationNumbers[1] = numsChange
	}

	tx := UnmarshalTx(txC.Tx)
	txProposal := TxProposal{
		InputList:                 unspentList,
		OutlayList:                outlayList,
		Tx:                        tx,
		Fee:                       fee,
		OutlayIndexToTxOutIndex:   outlayIndexToTxOutIndex,
		OutlayConfirmationNumbers: outlayConfirmationNumbers,
	}

	script, err := json.Marshal(txProposal)
	if err != nil {
		return nil, err
	}
	return &Output{
		TransactionHash: hex.EncodeToString(txC.TxOut.PublicKey.GetData()),
		RawTransaction:  hex.EncodeToString(script),
		SharedSecret:    hex.EncodeToString(txC.ShareSecretOut),
		Fee:             fee,
		OutputIndex:     0,
		OutputHash:      hex.EncodeToString(txC.TxOut.PublicKey.GetData()),
		ChangeIndex:     0,
		ChangeHash:      hex.EncodeToString(txC.TxOutChange.PublicKey.GetData()),
		ChangeAmount:    changeAmount,
	}, nil
}

func UnmarshalTx(tx *types.Tx) *Tx {
	return &Tx{
		Prefix:    UnmarshalPrefix(tx.Prefix),
		Signature: UnmarshalSignatureRctBulletproofs(tx.Signature),
	}
}

func UnmarshalPrefix(prefix *types.TxPrefix) *TxPrefix {
	ins := make([]*TxIn, len(prefix.Inputs))
	for i, in := range prefix.Inputs {
		ring := make([]*TxOut, len(in.Ring))
		for i, r := range in.Ring {
			ring[i] = UnmarshalTxOut(r)
		}
		proofs := make([]*TxOutMembershipProof, len(in.Proofs))
		for i, p := range in.Proofs {
			proofs[i] = UnmarshalTxOutMembershipProof(p)
		}
		ins[i] = &TxIn{
			Ring:   ring,
			Proofs: proofs,
		}
	}

	outs := make([]*TxOut, len(prefix.Outputs))
	for i, out := range prefix.Outputs {
		outs[i] = UnmarshalTxOut(out)
	}

	return &TxPrefix{
		Inputs:         ins,
		Outputs:        outs,
		Fee:            FeeValue(prefix.Fee),
		TombstoneBlock: TombstoneValue(prefix.TombstoneBlock),
	}
}

func UnmarshalTxOut(out *types.TxOut) *TxOut {
	return &TxOut{
		Amount: &Amount{
			Commitment:    hex.EncodeToString(out.MaskedAmount.Commitment.GetData()),
			MaskedValue:   MaskedValue(out.MaskedAmount.MaskedValue),
			MaskedTokenID: hex.EncodeToString(out.MaskedAmount.MaskedTokenId),
		},
		TargetKey: hex.EncodeToString(out.TargetKey.GetData()),
		PublicKey: hex.EncodeToString(out.PublicKey.GetData()),
		EFogHint:  hex.EncodeToString(out.EFogHint.GetData()),
		EMemo:     hex.EncodeToString(out.EMemo.GetData()),
	}
}

func UnmarshalTxOutMembershipProof(proof *types.TxOutMembershipProof) *TxOutMembershipProof {
	elements := make([]*TxOutMembershipElement, len(proof.Elements))
	for i, e := range proof.Elements {
		elements[i] = &TxOutMembershipElement{
			Range: &Range{
				From: fmt.Sprint(e.Range.From),
				To:   fmt.Sprint(e.Range.To),
			},
			Hash: hex.EncodeToString(e.Hash.GetData()),
		}
	}
	return &TxOutMembershipProof{
		Index:        fmt.Sprint(proof.Index),
		HighestIndex: fmt.Sprint(proof.HighestIndex),
		Elements:     elements,
	}
}

func UnmarshalSignatureRctBulletproofs(signature *types.SignatureRctBulletproofs) *SignatureRctBulletproofs {
	signatures := make([]*RingMLSAG, len(signature.RingSignatures))
	for i, s := range signature.RingSignatures {
		signatures[i] = UnmarshalRingMLSAG(s)
	}
	commitments := make([]string, len(signature.PseudoOutputCommitments))
	for i, c := range signature.PseudoOutputCommitments {
		commitments[i] = hex.EncodeToString(c.GetData())
	}
	return &SignatureRctBulletproofs{
		RingSignatures:          signatures,
		PseudoOutputCommitments: commitments,
		RangeProofs:             hex.EncodeToString(signature.RangeProofBytes),
	}
}

func UnmarshalRingMLSAG(mlsag *types.RingMLSAG) *RingMLSAG {
	responses := make([]string, len(mlsag.Responses))
	for i, resp := range mlsag.Responses {
		responses[i] = hex.EncodeToString(resp.GetData())
	}
	return &RingMLSAG{
		CZero:     hex.EncodeToString(mlsag.CZero.GetData()),
		Responses: responses,
		KeyImage:  hex.EncodeToString(mlsag.KeyImage.GetData()),
	}
}
