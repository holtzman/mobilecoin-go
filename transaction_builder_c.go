package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"

	account "github.com/jadeydi/mobilecoin-account"
	"github.com/jadeydi/mobilecoin-account/types"
	"google.golang.org/protobuf/proto"
)

// #cgo CFLAGS: -I${SRCDIR}/include
// #cgo darwin LDFLAGS: ${SRCDIR}/include/libmobilecoin.a -framework Security -framework Foundation
// #cgo linux LDFLAGS: ${SRCDIR}/include/libmobilecoin_linux.a -lm -ldl
// #include <stdio.h>
// #include <stdlib.h>
// #include <errno.h>
// #include "libmobilecoin.h"
import "C"

type TxC struct {
	Tx                 []byte
	TxOut              *types.TxOut
	ShareSecretOut     []byte
	ConfirmationOut    []byte
	TxOutChange        *types.TxOut
	ShareSecretChange  []byte
	ConfirmationChange []byte
}

// mc_transaction_builder_create
func MCTransactionBuilderCreateC(inputCs []*InputC, amount, changeAmount, fee, tombstone uint64, tokenID, version uint, recipient, change *account.PublicAddress) (*TxC, error) {
	var fog_resolver *C.McFogResolver

	if recipient != nil && recipient.FogReportUrl != "" {
		signature, err := ParseSignature()
		if err != nil {
			return nil, err
		}

		enclave := signature.MRENCLAVE()
		h := hex.EncodeToString(enclave[:])
		h = "3d6e528ee0574ae3299915ea608b71ddd17cbe855d4f5e1c46df9b0d22b04cdb"
		fog_url_to_mr_enclave_hex := map[string]string{
			"fog://fog.prod.mobilecoinww.com":            h,
			"fog://fog-rpt-prd.namda.net":                h,
			"fog://service.fog.mob.production.namda.net": h,
			"fog://service.fog.mob.staging.namda.net":    "a4764346f91979b4906d4ce26102228efe3aba39216dec1e7d22e6b06f919f11",
		}

		mr_enclave_hex, ok := fog_url_to_mr_enclave_hex[string(recipient.FogReportUrl)]
		if !ok {
			return nil, errors.New("No enclave hex for Address' fog url")
		}

		// Construct a verifier object that is used to verify the report's attestation
		mr_enclave_bytes, err := hex.DecodeString(mr_enclave_hex)
		if err != nil {
			return nil, err
		}

		c_mr_enclave_bytes := C.CBytes(mr_enclave_bytes)
		defer C.free(c_mr_enclave_bytes)

		c_mr_enclave := C.McBuffer{
			buffer: (*C.uchar)(c_mr_enclave_bytes),
			len:    C.ulong(len(mr_enclave_bytes)),
		}

		mr_enclave_verifier, err := C.mc_mr_enclave_verifier_create(&c_mr_enclave)
		if err != nil {
			return nil, err
		}
		if mr_enclave_verifier == nil {
			return nil, errors.New("mc_mr_enclave_verifier_create failed")
		}
		defer C.mc_mr_enclave_verifier_free(mr_enclave_verifier)

		c_advisory_id := C.CString("INTEL-SA-00334")
		defer C.free(unsafe.Pointer(c_advisory_id))
		ret, err := C.mc_mr_enclave_verifier_allow_hardening_advisory(mr_enclave_verifier, c_advisory_id)
		if err != nil {
			return nil, err
		}
		if ret == false {
			return nil, errors.New("mc_mr_enclave_verifier_allow_hardening_advisory failed")
		}

		mc_verifier, err := C.mc_verifier_create()
		if err != nil {
			return nil, err
		}
		defer C.mc_verifier_free(mc_verifier)

		ret, err = C.mc_verifier_add_mr_enclave(mc_verifier, mr_enclave_verifier)
		if err != nil {
			return nil, err
		}
		if ret == false {
			return nil, errors.New("mc_verifier_add_mr_enclave failed")
		}

		fog_resolver, err = C.mc_fog_resolver_create(mc_verifier)
		if err != nil {
			return nil, err
		}
		defer C.mc_fog_resolver_free(fog_resolver)
	}

	memo_builder, err := C.mc_memo_builder_default_create()
	if err != nil {
		return nil, err
	}
	defer C.mc_memo_builder_free(memo_builder)

	transaction_builder, err := C.mc_transaction_builder_create(C.uint64_t(fee), C.uint64_t(tokenID), C.uint64_t(tombstone), fog_resolver, memo_builder, C.uint32_t(version))
	if err != nil {
		return nil, err
	}
	if transaction_builder == nil {
		return nil, errors.New("mc_transaction_builder_create error")
	}
	defer C.mc_transaction_builder_free(transaction_builder)

	// add input
	for _, input := range inputCs {
		view_private_input_buf := input.ViewPrivate.Bytes()
		view_private_key_bytes := C.CBytes(view_private_input_buf)
		defer C.free(view_private_key_bytes)
		view_private_key := &C.McBuffer{
			buffer: (*C.uint8_t)(view_private_key_bytes),
			len:    C.size_t(len(view_private_input_buf)),
		}

		subaddress_spend_private_buf := input.SubAddressSpendPrivate.Bytes()
		subaddress_spend_private_key_bytes := C.CBytes(subaddress_spend_private_buf)
		defer C.free(subaddress_spend_private_key_bytes)
		subaddress_spend_private_key := &C.McBuffer{
			buffer: (*C.uint8_t)(subaddress_spend_private_key_bytes),
			len:    C.size_t(len(subaddress_spend_private_buf)),
		}

		ring, err := C.mc_transaction_builder_ring_create()
		if err != nil {
			return nil, err
		}
		defer C.mc_transaction_builder_ring_free(ring)

		for _, r := range input.TxOutWithProofCs {
			tx_out_buf, err := proto.Marshal(r.TxOut)
			if err != nil {
				return nil, err
			}
			tx_out_proto_bytes := C.CBytes(tx_out_buf)
			defer C.free(tx_out_proto_bytes)
			tx_out_proto := &C.McBuffer{
				buffer: (*C.uint8_t)(tx_out_proto_bytes),
				len:    C.size_t(len(tx_out_buf)),
			}
			membership_proof_buf, err := proto.Marshal(r.TxOutMembershipProof)
			if err != nil {
				return nil, err
			}
			membership_proof_proto_bytes := C.CBytes(membership_proof_buf)
			defer C.free(membership_proof_proto_bytes)
			membership_proof_proto := &C.McBuffer{
				buffer: (*C.uint8_t)(membership_proof_proto_bytes),
				len:    C.size_t(len(membership_proof_buf)),
			}
			b, err := C.mc_transaction_builder_ring_add_element(ring, tx_out_proto, membership_proof_proto)
			if err != nil {
				return nil, err
			} else if !b {
				return nil, errors.New("mc_transaction_builder_ring_add_element failure")
			}
		}

		var out_error *C.McError
		b, err := C.mc_transaction_builder_add_input(transaction_builder, view_private_key, subaddress_spend_private_key, C.size_t(input.RealIndex), ring, &out_error)
		if err != nil {
			return nil, err
		} else if !b {
			if out_error == nil {
				return nil, fmt.Errorf("mc_transaction_builder_add_input failure")
			} else {
				err = fmt.Errorf("mc_transaction_builder_add_input failed: [%d] %s", out_error.error_code, C.GoString(out_error.error_description))
				C.mc_error_free(out_error)
				return nil, err
			}
		}
	}

	// mc_transaction_builder_add_output
	view_public_key_buf := account.HexToBytes(recipient.ViewPublicKey)
	view_public_key_bytes := C.CBytes(view_public_key_buf)
	defer C.free(view_public_key_bytes)
	view_public := &C.McBuffer{
		buffer: (*C.uint8_t)(view_public_key_bytes),
		len:    C.size_t(len(view_public_key_buf)),
	}

	spend_public_key_buf := account.HexToBytes(recipient.SpendPublicKey)
	spend_public_key_bytes := C.CBytes(spend_public_key_buf)
	defer C.free(spend_public_key_bytes)
	spend_public := &C.McBuffer{
		buffer: (*C.uint8_t)(spend_public_key_bytes),
		len:    C.size_t(len(spend_public_key_buf)),
	}

	report_url_recipient_str := C.CString(recipient.FogReportUrl)
	defer C.free(unsafe.Pointer(report_url_recipient_str))
	report_id_recipient_str := C.CString(recipient.FogReportId)
	defer C.free(unsafe.Pointer(report_id_recipient_str))
	sig_buf := account.HexToBytes(recipient.FogAuthoritySig)
	sig_bytes := C.CBytes(sig_buf)
	defer C.free(sig_bytes)
	authority_sig := &C.McBuffer{
		buffer: (*C.uint8_t)(sig_bytes),
		len:    C.size_t(len(sig_buf)),
	}
	fog_info := (*C.McPublicAddressFogInfo)(C.malloc(C.sizeof_McPublicAddressFogInfo))
	defer C.free(unsafe.Pointer(fog_info))
	fog_info.report_url = (*C.char)(report_url_recipient_str)
	fog_info.report_id = (*C.char)(report_id_recipient_str)
	fog_info.authority_sig = authority_sig
	recipient_address := (*C.McPublicAddress)(C.malloc(C.sizeof_McPublicAddress))
	defer C.free(unsafe.Pointer(recipient_address))
	recipient_address.view_public_key = view_public
	recipient_address.spend_public_key = spend_public
	recipient_address.fog_info = fog_info

	secret_recipient_buf := make([]byte, 32)
	secret_recipient_bytes := C.CBytes(secret_recipient_buf)
	defer C.free(secret_recipient_bytes)
	out_tx_out_shared_secret := &C.McMutableBuffer{
		buffer: (*C.uint8_t)(secret_recipient_bytes),
		len:    C.size_t(len(secret_recipient_buf)),
	}
	confirmation_recipient_buf := make([]byte, 32)
	confirmation_recipient_bytes := C.CBytes(confirmation_recipient_buf)
	defer C.free(confirmation_recipient_bytes)
	out_tx_out_confirmation_number := &C.McMutableBuffer{
		buffer: (*C.uint8_t)(confirmation_recipient_bytes),
		len:    C.size_t(len(confirmation_recipient_buf)),
	}

	var rng_callback *C.McRngCallback
	var out_error *C.McError
	mcDataOut, err := C.mc_transaction_builder_add_output(transaction_builder, C.uint64_t(amount), recipient_address, rng_callback, out_tx_out_confirmation_number, out_tx_out_shared_secret, &out_error)
	if err != nil {
		return nil, err
	}
	if out_error != nil {
		err = fmt.Errorf("mc_transaction_builder_add_output failed: [%d] %s", out_error.error_code, C.GoString(out_error.error_description))
		C.mc_error_free(out_error)
		return nil, err
	}
	secret_recipient_buf = C.GoBytes(unsafe.Pointer(out_tx_out_shared_secret.buffer), C.int(len(secret_recipient_buf)))
	confirmation_recipient_buf = C.GoBytes(unsafe.Pointer(out_tx_out_confirmation_number.buffer), C.int(len(confirmation_recipient_buf)))

	defer C.mc_data_free(mcDataOut)
	var tx_out_size_bytes *C.McMutableBuffer
	tx_out_size := C.mc_data_get_bytes(mcDataOut, tx_out_size_bytes)

	tx_out_data_buf := make([]byte, int(tx_out_size))
	tx_out_data_bytes := C.CBytes(tx_out_data_buf)
	defer C.free(tx_out_data_bytes)
	tx_out_data := &C.McMutableBuffer{
		buffer: (*C.uint8_t)(tx_out_data_bytes),
		len:    C.size_t(len(tx_out_data_buf)),
	}
	tx_out_size = C.mc_data_get_bytes(mcDataOut, tx_out_data)
	txOut := &types.TxOut{}
	err = proto.Unmarshal(C.GoBytes(tx_out_data_bytes, C.int(tx_out_size)), txOut)
	if err != nil {
		return nil, err
	}

	// mc_transaction_builder_add_output for change
	secret_change_buf := make([]byte, 32)
	confirmation_change_buf := make([]byte, 32)
	txOutChange := &types.TxOut{}
	if changeAmount > 0 {
		view_public_key_change_buf := account.HexToBytes(change.ViewPublicKey)
		view_public_key_change_bytes := C.CBytes(view_public_key_change_buf)
		defer C.free(view_public_key_change_bytes)
		view_public_change := &C.McBuffer{
			buffer: (*C.uint8_t)(view_public_key_change_bytes),
			len:    C.size_t(len(view_public_key_change_buf)),
		}

		spend_public_key_change_buf := account.HexToBytes(change.SpendPublicKey)
		spend_public_key_change_bytes := C.CBytes(spend_public_key_change_buf)
		defer C.free(spend_public_key_change_bytes)
		spend_public_change := &C.McBuffer{
			buffer: (*C.uint8_t)(spend_public_key_change_bytes),
			len:    C.size_t(len(spend_public_key_change_buf)),
		}

		report_url_change_str := C.CString(change.FogReportUrl)
		defer C.free(unsafe.Pointer(report_url_change_str))
		report_id_change_str := C.CString(change.FogReportId)
		defer C.free(unsafe.Pointer(report_id_change_str))
		sig_change_buf := account.HexToBytes(change.FogAuthoritySig)
		sig_change_bytes := C.CBytes(sig_change_buf)
		defer C.free(sig_change_bytes)
		authority_sig_change := &C.McBuffer{
			buffer: (*C.uint8_t)(sig_change_bytes),
			len:    C.size_t(len(sig_change_buf)),
		}
		fog_info_change := (*C.McPublicAddressFogInfo)(C.malloc(C.sizeof_McPublicAddressFogInfo))
		defer C.free(unsafe.Pointer(fog_info_change))
		fog_info_change.report_url = (*C.char)(report_url_change_str)
		fog_info_change.report_id = (*C.char)(report_id_change_str)
		fog_info_change.authority_sig = authority_sig_change
		change_address := (*C.McPublicAddress)(C.malloc(C.sizeof_McPublicAddress))
		defer C.free(unsafe.Pointer(change_address))
		change_address.view_public_key = view_public_change
		change_address.spend_public_key = spend_public_change
		change_address.fog_info = fog_info_change

		secret_change_bytes := C.CBytes(secret_change_buf)
		defer C.free(secret_change_bytes)
		change_tx_out_shared_secret := &C.McMutableBuffer{
			buffer: (*C.uint8_t)(secret_change_bytes),
			len:    C.size_t(len(secret_change_buf)),
		}
		confirmation_change_bytes := C.CBytes(confirmation_change_buf)
		defer C.free(confirmation_change_bytes)
		change_tx_out_confirmation_number := &C.McMutableBuffer{
			buffer: (*C.uint8_t)(confirmation_change_bytes),
			len:    C.size_t(len(confirmation_change_buf)),
		}

		mcDataChange, err := C.mc_transaction_builder_add_output(transaction_builder, C.uint64_t(changeAmount), change_address, rng_callback, change_tx_out_confirmation_number, change_tx_out_shared_secret, &out_error)
		if err != nil {
			return nil, err
		}
		if out_error != nil {
			err = fmt.Errorf("mc_transaction_builder_add_output change failed: [%d] %s", out_error.error_code, C.GoString(out_error.error_description))
			C.mc_error_free(out_error)
			return nil, err
		}
		secret_change_buf = C.GoBytes(unsafe.Pointer(change_tx_out_shared_secret.buffer), C.int(len(secret_change_buf)))
		confirmation_change_buf = C.GoBytes(unsafe.Pointer(change_tx_out_confirmation_number.buffer), C.int(len(confirmation_change_buf)))

		defer C.mc_data_free(mcDataChange)
		var tx_out_change_size_bytes *C.McMutableBuffer
		data_size := C.mc_data_get_bytes(mcDataChange, tx_out_change_size_bytes)

		tx_out_change_data_buf := make([]byte, int(data_size))
		tx_out_change_data_bytes := C.CBytes(tx_out_change_data_buf)
		defer C.free(tx_out_change_data_bytes)
		tx_out_change_data := &C.McMutableBuffer{
			buffer: (*C.uint8_t)(tx_out_change_data_bytes),
			len:    C.size_t(len(tx_out_change_data_buf)),
		}
		data_size = C.mc_data_get_bytes(mcDataChange, tx_out_change_data)
		err = proto.Unmarshal(C.GoBytes(tx_out_change_data_bytes, C.int(data_size)), txOutChange)
		if err != nil {
			return nil, err
		}
	}

	mcData, err := C.mc_transaction_builder_build(transaction_builder, rng_callback, &out_error)
	if err != nil {
		return nil, err
	}
	if out_error != nil {
		err = fmt.Errorf("mc_transaction_builder_build failed: [%d] %s", out_error.error_code, C.GoString(out_error.error_description))
		C.mc_error_free(out_error)
		return nil, err
	}
	defer C.mc_data_free(mcData)
	var out_size_bytes *C.McMutableBuffer
	data_size := C.mc_data_get_bytes(mcData, out_size_bytes)

	out_data_buf := make([]byte, int(data_size))
	out_data_bytes := C.CBytes(out_data_buf)
	defer C.free(out_data_bytes)
	out_data := &C.McMutableBuffer{
		buffer: (*C.uint8_t)(out_data_bytes),
		len:    C.size_t(len(out_data_buf)),
	}
	data_size = C.mc_data_get_bytes(mcData, out_data)
	return &TxC{
		Tx:                 C.GoBytes(out_data_bytes, C.int(data_size)),
		TxOut:              txOut,
		ShareSecretOut:     secret_recipient_buf,
		ConfirmationOut:    confirmation_recipient_buf,
		TxOutChange:        txOutChange,
		ShareSecretChange:  secret_change_buf,
		ConfirmationChange: confirmation_change_buf,
	}, nil
}