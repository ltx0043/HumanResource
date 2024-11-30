package changeset

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/smartcontractkit/chainlink/deployment"
	commonchangeset "github.com/smartcontractkit/chainlink/deployment/common/changeset"
	commontypes "github.com/smartcontractkit/chainlink/deployment/common/types"

	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func Test_NewAcceptOwnershipChangeset(t *testing.T) {
	e := NewMemoryEnvironmentWithJobs(t, logger.TestLogger(t), 2, 4)
	state, err := LoadOnchainState(e.Env)
	require.NoError(t, err)

	allChains := maps.Keys(e.Env.Chains)
	source := allChains[0]
	dest := allChains[1]

	newAddresses := deployment.NewMemoryAddressBook()
	err = deployPrerequisiteChainContracts(e.Env, newAddresses, allChains, nil)
	require.NoError(t, err)
	require.NoError(t, e.Env.ExistingAddresses.Merge(newAddresses))

	mcmConfig := commontypes.MCMSWithTimelockConfig{
		Canceller:         commonchangeset.SingleGroupMCMS(t),
		Bypasser:          commonchangeset.SingleGroupMCMS(t),
		Proposer:          commonchangeset.SingleGroupMCMS(t),
		TimelockExecutors: e.Env.AllDeployerKeys(),
		TimelockMinDelay:  big.NewInt(0),
	}
	out, err := commonchangeset.DeployMCMSWithTimelock(e.Env, map[uint64]commontypes.MCMSWithTimelockConfig{
		source: mcmConfig,
		dest:   mcmConfig,
	})
	require.NoError(t, err)
	require.NoError(t, e.Env.ExistingAddresses.Merge(out.AddressBook))
	newAddresses = deployment.NewMemoryAddressBook()
	tokenConfig := NewTestTokenConfig(state.Chains[e.FeedChainSel].USDFeeds)
	ocrParams := make(map[uint64]CCIPOCRParams)
	for _, chain := range allChains {
		ocrParams[chain] = DefaultOCRParams(e.FeedChainSel, nil)
	}
	err = deployCCIPContracts(e.Env, newAddresses, NewChainsConfig{
		HomeChainSel:   e.HomeChainSel,
		FeedChainSel:   e.FeedChainSel,
		ChainsToDeploy: allChains,
		TokenConfig:    tokenConfig,
		OCRSecrets:     deployment.XXXGenerateTestOCRSecrets(),
		OCRParams:      ocrParams,
	})
	require.NoError(t, err)

	// at this point we have the initial deploys done, now we need to transfer ownership
	// to the timelock contract
	state, err = LoadOnchainState(e.Env)
	require.NoError(t, err)

	// compose the transfer ownership and accept ownership changesets
	_, err = commonchangeset.ApplyChangesets(t, e.Env, map[uint64]*gethwrappers.RBACTimelock{
		source: state.Chains[source].Timelock,
		dest:   state.Chains[dest].Timelock,
	}, []commonchangeset.ChangesetApplication{
		// note this doesn't have proposals.
		{
			Changeset: commonchangeset.WrapChangeSet(commonchangeset.NewTransferOwnershipChangeset),
			Config:    genTestTransferOwnershipConfig(e, allChains, state),
		},
		// this has proposals, ApplyChangesets will sign & execute them.
		// in practice, signing and executing are separated processes.
		{
			Changeset: commonchangeset.WrapChangeSet(commonchangeset.NewAcceptOwnershipChangeset),
			Config:    genTestAcceptOwnershipConfig(e, allChains, state),
		},
	})
	require.NoError(t, err)

	assertTimelockOwnership(t, e, allChains, state)
}

func genTestTransferOwnershipConfig(
	e DeployedEnv,
	chains []uint64,
	state CCIPOnChainState,
) commonchangeset.TransferOwnershipConfig {
	var (
		timelocksPerChain = make(map[uint64]common.Address)
		contracts         = make(map[uint64][]commonchangeset.OwnershipTransferrer)
	)

	// chain contracts
	for _, chain := range chains {
		timelocksPerChain[chain] = state.Chains[chain].Timelock.Address()
		contracts[chain] = []commonchangeset.OwnershipTransferrer{
			state.Chains[chain].OnRamp,
			state.Chains[chain].OffRamp,
			state.Chains[chain].FeeQuoter,
			state.Chains[chain].NonceManager,
			state.Chains[chain].RMNRemote,
		}
	}

	// home chain
	homeChainTimelockAddress := state.Chains[e.HomeChainSel].Timelock.Address()
	timelocksPerChain[e.HomeChainSel] = homeChainTimelockAddress
	contracts[e.HomeChainSel] = append(contracts[e.HomeChainSel],
		state.Chains[e.HomeChainSel].CapabilityRegistry,
		state.Chains[e.HomeChainSel].CCIPHome,
		state.Chains[e.HomeChainSel].RMNHome,
	)

	return commonchangeset.TransferOwnershipConfig{
		TimelocksPerChain: timelocksPerChain,
		Contracts:         contracts,
	}
}

func genTestAcceptOwnershipConfig(
	e DeployedEnv,
	chains []uint64,
	state CCIPOnChainState,
) commonchangeset.AcceptOwnershipConfig {
	var (
		timelocksPerChain = make(map[uint64]common.Address)
		proposerMCMses    = make(map[uint64]*gethwrappers.ManyChainMultiSig)
		contracts         = make(map[uint64][]commonchangeset.OwnershipAcceptor)
	)
	for _, chain := range chains {
		timelocksPerChain[chain] = state.Chains[chain].Timelock.Address()
		proposerMCMses[chain] = state.Chains[chain].ProposerMcm
		contracts[chain] = []commonchangeset.OwnershipAcceptor{
			state.Chains[chain].OnRamp,
			state.Chains[chain].OffRamp,
			state.Chains[chain].FeeQuoter,
			state.Chains[chain].NonceManager,
			state.Chains[chain].RMNRemote,
		}
	}

	// add home chain contracts.
	// this overwrite should be fine.
	timelocksPerChain[e.HomeChainSel] = state.Chains[e.HomeChainSel].Timelock.Address()
	proposerMCMses[e.HomeChainSel] = state.Chains[e.HomeChainSel].ProposerMcm
	contracts[e.HomeChainSel] = append(contracts[e.HomeChainSel],
		state.Chains[e.HomeChainSel].CapabilityRegistry,
		state.Chains[e.HomeChainSel].CCIPHome,
		state.Chains[e.HomeChainSel].RMNHome,
	)

	return commonchangeset.AcceptOwnershipConfig{
		TimelocksPerChain: timelocksPerChain,
		ProposerMCMSes:    proposerMCMses,
		Contracts:         contracts,
		MinDelay:          time.Duration(0),
	}
}

// assertTimelockOwnership asserts that the ownership of the contracts has been transferred
// to the appropriate timelock contract on each chain.
func assertTimelockOwnership(
	t *testing.T,
	e DeployedEnv,
	chains []uint64,
	state CCIPOnChainState,
) {
	ctx := tests.Context(t)
	// check that the ownership has been transferred correctly
	for _, chain := range chains {
		for _, contract := range []commonchangeset.OwnershipTransferrer{
			state.Chains[chain].OnRamp,
			state.Chains[chain].OffRamp,
			state.Chains[chain].FeeQuoter,
			state.Chains[chain].NonceManager,
			state.Chains[chain].RMNRemote,
		} {
			owner, err := contract.Owner(&bind.CallOpts{
				Context: ctx,
			})
			require.NoError(t, err)
			require.Equal(t, state.Chains[chain].Timelock.Address(), owner)
		}
	}

	// check home chain contracts ownership
	homeChainTimelockAddress := state.Chains[e.HomeChainSel].Timelock.Address()
	for _, contract := range []commonchangeset.OwnershipTransferrer{
		state.Chains[e.HomeChainSel].CapabilityRegistry,
		state.Chains[e.HomeChainSel].CCIPHome,
		state.Chains[e.HomeChainSel].RMNHome,
	} {
		owner, err := contract.Owner(&bind.CallOpts{
			Context: ctx,
		})
		require.NoError(t, err)
		require.Equal(t, homeChainTimelockAddress, owner)
	}
}
