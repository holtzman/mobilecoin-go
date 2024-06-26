package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	account "github.com/MixinNetwork/mobilecoin-account"
	"github.com/MixinNetwork/mobilecoin-account/types"
	"github.com/bwesterb/go-ristretto"
)

type TxOutWithProofC struct {
	TxOut                *types.TxOut
	TxOutMembershipProof *types.TxOutMembershipProof
}

type InputC struct {
	ViewPrivate            *ristretto.Scalar
	SpendPrivate           *ristretto.Scalar
	SubAddressSpendPrivate *ristretto.Scalar
	RealIndex              int
	TxOutWithProofCs       []*TxOutWithProofC
}

// mc_transaction_builder_ring_add_element
func BuildRingElements(utxos []*UTXO, proofs *Proofs) ([]*InputC, error) {
	if len(proofs.Ring) == 0 || len(proofs.Ring) != len(proofs.Rings) {
		return nil, fmt.Errorf("Invalid proofs ring len %d, rings len %d", len(proofs.Rings), len(proofs.Ring))
	}

	inputSet := make(map[string]*UTXO)
	for _, utxo := range utxos {
		if len(utxo.ScriptPubKey) < 8 {
			return nil, fmt.Errorf("MobileCoin invalid script pub key %s", utxo.ScriptPubKey)
		}
		data, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		var txOut TxOut
		err = json.Unmarshal(data, &txOut)
		if err != nil {
			return nil, err
		}
		inputSet[txOut.PublicKey] = utxo
	}

	var inputCs []*InputC
	for i, itemi := range proofs.Ring {
		index := 0
		ring := proofs.Rings[i]
		for j, itemj := range ring {
			if itemi.TxOut.PublicKey == itemj.TxOut.PublicKey {
				index = j
				break
			}
		}
		if len(ring) == 0 {
			ring = append(ring, itemi)
		}
		if index == 0 {
			ring[index] = itemi
		}

		txOutWithProofCs := make([]*TxOutWithProofC, len(ring))
		for j, _ := range ring {
			txOutWithProofCs[j] = &TxOutWithProofC{
				TxOut:                MarshalTxOut(ring[j].TxOut),
				TxOutMembershipProof: MarshalTxOutMembershipProof(ring[j].Proof),
			}
		}

		if inputSet[itemi.TxOut.PublicKey] == nil {
			return nil, fmt.Errorf("UTXO did not find")
		}
		utxo := inputSet[itemi.TxOut.PublicKey]
		acc, err := account.NewAccountKey(utxo.PrivateKey[:64], utxo.PrivateKey[64:])
		if err != nil {
			return nil, err
		}
		inputCs = append(inputCs, &InputC{
			ViewPrivate:            account.HexToScalar(utxo.PrivateKey[:64]),
			SpendPrivate:           account.HexToScalar(utxo.PrivateKey[64:]),
			SubAddressSpendPrivate: acc.SubaddressSpendPrivateKey(0),
			RealIndex:              index,
			TxOutWithProofCs:       txOutWithProofCs,
		})
	}
	return inputCs, nil
}

func MarshalTxOut(input *TxOut) *types.TxOut {
	out := &types.TxOut{
		TargetKey: &types.CompressedRistretto{
			Data: account.HexToPoint(input.TargetKey).Bytes(),
		},
		PublicKey: &types.CompressedRistretto{
			Data: account.HexToPoint(input.PublicKey).Bytes(),
		},
	}
	switch input.Amount.Version {
	case 0, 1:
		out.MaskedAmount = &types.TxOut_MaskedAmountV1{
			MaskedAmountV1: &types.MaskedAmount{
				Commitment: &types.CompressedRistretto{
					Data: account.HexToBytes(input.Amount.Commitment),
				},
				MaskedValue:   uint64(input.Amount.MaskedValue),
				MaskedTokenId: account.HexToBytes(input.Amount.MaskedTokenID),
			},
		}
	case 2:
		out.MaskedAmount = &types.TxOut_MaskedAmountV2{
			MaskedAmountV2: &types.MaskedAmount{
				Commitment: &types.CompressedRistretto{
					Data: account.HexToBytes(input.Amount.Commitment),
				},
				MaskedValue:   uint64(input.Amount.MaskedValue),
				MaskedTokenId: account.HexToBytes(input.Amount.MaskedTokenID),
			},
		}
	default:
		panic(fmt.Sprintf("Invalid amount version %d", input.Amount.Version))
	}
	if input.EFogHint != "" {
		out.EFogHint = &types.EncryptedFogHint{
			Data: account.HexToBytes(input.EFogHint),
		}
	}
	if input.EMemo != "" {
		out.EMemo = &types.EncryptedMemo{
			Data: account.HexToBytes(input.EMemo),
		}
	}
	return out
}

func MarshalTxOutMembershipProof(proof *TxOutMembershipProof) *types.TxOutMembershipProof {
	var elements []*types.TxOutMembershipElement
	for _, e := range proof.Elements {
		elements = append(elements, &types.TxOutMembershipElement{
			Range: &types.Range{
				From: stringToUint64(e.Range.From),
				To:   stringToUint64(e.Range.To),
			},
			Hash: &types.TxOutMembershipHash{
				Data: account.HexToBytes(e.Hash),
			},
		})
	}
	return &types.TxOutMembershipProof{
		Index:        stringToUint64(proof.Index),
		HighestIndex: stringToUint64(proof.HighestIndex),
		Elements:     elements,
	}
}

func stringToUint64(v string) uint64 {
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}
