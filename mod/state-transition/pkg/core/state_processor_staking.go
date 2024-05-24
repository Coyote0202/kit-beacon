// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package core

import (
	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	"github.com/berachain/beacon-kit/mod/errors"
	"github.com/berachain/beacon-kit/mod/primitives"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/version"
	"github.com/davecgh/go-spew/spew"
)

// processOperations processes the operations and ensures they match the
// local state.
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) processOperations(
	st BeaconStateT,
	blk BeaconBlockT,
) error {
	// Verify that outstanding deposits are processed up to the maximum number
	// of deposits.
	deposits := blk.GetBody().GetDeposits()
	index, err := st.GetEth1DepositIndex()
	if err != nil {
		return err
	}
	eth1Data, err := st.GetEth1Data()
	if err != nil {
		return err
	}
	depositCount := min(
		sp.cs.MaxDepositsPerBlock(),
		eth1Data.DepositCount-index,
	)
	_ = depositCount
	// TODO: Update eth1data count and check this.
	// if uint64(len(deposits)) != depositCount {
	// 	return errors.New("deposit count mismatch")
	// }
	return sp.processDeposits(st, deposits)
}

// ProcessDeposits processes the deposits and ensures they match the
// local state.
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) processDeposits(
	st BeaconStateT,
	deposits []*types.Deposit,
) error {
	// Ensure the deposits match the local state.
	for _, dep := range deposits {
		if err := sp.processDeposit(st, dep); err != nil {
			return err
		}
		// TODO: unhood this in better spot later
		if err := st.SetEth1DepositIndex(dep.Index); err != nil {
			return err
		}
	}
	return nil
}

// processDeposit processes the deposit and ensures it matches the local state.
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) processDeposit(
	st BeaconStateT,
	dep *types.Deposit,
) error {
	// TODO: fill this in properly
	// if !sp.isValidMerkleBranch(
	// 	leaf,
	// 	dep.Credentials,
	// 	32 + 1,
	// 	dep.Index,
	// 	st.root,
	// ) {
	// 	return errors.New("invalid merkle branch")
	// }
	idx, err := st.ValidatorIndexByPubkey(dep.Pubkey)
	// If the validator already exists, we update the balance.
	if err == nil {
		var val *types.Validator
		val, err = st.ValidatorByIndex(idx)
		if err != nil {
			return err
		}

		// TODO: Modify balance here and then effective balance once per epoch.
		val.EffectiveBalance = min(val.EffectiveBalance+dep.Amount,
			math.Gwei(sp.cs.MaxEffectiveBalance()))

		return st.UpdateValidatorAtIndex(idx, val)
	}
	// If the validator does not exist, we add the validator.
	// Add the validator to the registry.
	return sp.createValidator(st, dep)
}

// createValidator creates a validator if the deposit is valid.
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) createValidator(
	st BeaconStateT,
	dep *types.Deposit,
) error {
	var (
		genesisValidatorsRoot primitives.Root
		epoch                 math.Epoch
		err                   error
	)

	// Get the genesis validators root to be used to find fork data later.
	genesisValidatorsRoot, err = st.GetGenesisValidatorsRoot()
	if err != nil {
		return err
	}

	// Get the current epoch.
	// Get the current slot.
	slot, err := st.GetSlot()
	if err != nil {
		return err
	}
	epoch = sp.cs.SlotToEpoch(slot)

	// Get the fork data for the current epoch.
	fd := types.NewForkData(
		version.FromUint32[primitives.Version](
			sp.cs.ActiveForkVersionForEpoch(epoch),
		), genesisValidatorsRoot,
	)

	depositMessage := types.DepositMessage{
		Pubkey:      dep.Pubkey,
		Credentials: dep.Credentials,
		Amount:      dep.Amount,
	}
	if err = depositMessage.VerifyCreateValidator(
		fd, dep.Signature, sp.signer.VerifySignature, sp.cs.DomainTypeDeposit(),
	); err != nil {
		return err
	}

	// Add the validator to the registry.
	return sp.addValidatorToRegistry(st, dep)
}

// addValidatorToRegistry adds a validator to the registry.
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) addValidatorToRegistry(
	st BeaconStateT,
	dep *types.Deposit,
) error {
	val := types.NewValidatorFromDeposit(
		dep.Pubkey,
		dep.Credentials,
		dep.Amount,
		math.Gwei(sp.cs.EffectiveBalanceIncrement()),
		math.Gwei(sp.cs.MaxEffectiveBalance()),
	)
	if err := st.AddValidator(val); err != nil {
		return err
	}

	idx, err := st.ValidatorIndexByPubkey(val.Pubkey)
	if err != nil {
		return err
	}
	return st.IncreaseBalance(idx, dep.Amount)
}

// processWithdrawals as per the Ethereum 2.0 specification.
// https://github.com/ethereum/consensus-specs/blob/dev/specs/capella/beacon-chain.md#new-process_withdrawals
//
//nolint:lll
func (sp *StateProcessor[
	BeaconBlockT, BeaconBlockBodyT, BeaconStateT,
	BlobSidecarsT, ContextT,
]) processWithdrawals(
	st BeaconStateT,
	body BeaconBlockBodyT,
) error {
	// Dequeue and verify the logs.
	var (
		nextValidatorIndex math.ValidatorIndex
		payload            = body.GetExecutionPayload()
		payloadWithdrawals = payload.GetWithdrawals()
	)

	// Get the expected withdrawals.
	expectedWithdrawals, err := st.ExpectedWithdrawals()
	if err != nil {
		return err
	}
	numWithdrawals := len(expectedWithdrawals)

	// Ensure the withdrawals have the same length
	if numWithdrawals != len(payloadWithdrawals) {
		return errors.Newf(
			"withdrawals do not match expected length %d, got %d",
			len(expectedWithdrawals), len(payloadWithdrawals),
		)
	}

	// Compare and process each withdrawal.
	for i, wd := range expectedWithdrawals {
		// Ensure the withdrawals match the local state.
		if !wd.Equals(payloadWithdrawals[i]) {
			return errors.Newf(
				"withdrawals do not match expected %s, got %s",
				spew.Sdump(wd), spew.Sdump(payloadWithdrawals[i]),
			)
		}

		// Then we process the withdrawal.
		if err = st.DecreaseBalance(wd.Validator, wd.Amount); err != nil {
			return err
		}
	}

	// Update the next withdrawal index if this block contained withdrawals
	if numWithdrawals != 0 {
		// Next sweep starts after the latest withdrawal's validator index
		if err = st.SetNextWithdrawalIndex(
			(expectedWithdrawals[numWithdrawals-1].Index + 1).Unwrap(),
		); err != nil {
			return err
		}
	}

	totalValidators, err := st.GetTotalValidators()
	if err != nil {
		return err
	}

	// Update the next validator index to start the next withdrawal sweep
	//#nosec:G701 // won't overflow in practice.
	if numWithdrawals == int(sp.cs.MaxWithdrawalsPerPayload()) {
		// Next sweep starts after the latest withdrawal's validator index
		nextValidatorIndex =
			(expectedWithdrawals[len(expectedWithdrawals)-1].Index + 1) %
				math.U64(totalValidators)
	} else {
		// Advance sweep by the max length of the sweep if there was not
		// a full set of withdrawals
		nextValidatorIndex, err = st.GetNextWithdrawalValidatorIndex()
		if err != nil {
			return err
		}
		nextValidatorIndex += math.ValidatorIndex(
			sp.cs.MaxValidatorsPerWithdrawalsSweep())
		nextValidatorIndex %= math.ValidatorIndex(totalValidators)
	}

	return st.SetNextWithdrawalValidatorIndex(nextValidatorIndex)
}
