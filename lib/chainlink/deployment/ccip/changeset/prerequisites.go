package changeset

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/ccip-owner-contracts/pkg/proposal/timelock"

	"github.com/smartcontractkit/chainlink/deployment"
)

var (
	_ deployment.ChangeSet[DeployPrerequisiteConfig] = DeployPrerequisites
)

// DeployPrerequisites deploys the pre-requisite contracts for CCIP
// pre-requisite contracts are the contracts which can be reused from previous versions of CCIP
// Or the contracts which are already deployed on the chain ( for example, tokens, feeds, etc)
// Caller should update the environment's address book with the returned addresses.
func DeployPrerequisites(env deployment.Environment, cfg DeployPrerequisiteConfig) (deployment.ChangesetOutput, error) {
	err := cfg.Validate()
	if err != nil {
		return deployment.ChangesetOutput{}, errors.Wrapf(deployment.ErrInvalidConfig, "%v", err)
	}
	ab := deployment.NewMemoryAddressBook()
	err = deployPrerequisiteChainContracts(env, ab, cfg.ChainSelectors, cfg.Opts...)
	if err != nil {
		env.Logger.Errorw("Failed to deploy prerequisite contracts", "err", err, "addressBook", ab)
		return deployment.ChangesetOutput{
			AddressBook: ab,
		}, fmt.Errorf("failed to deploy prerequisite contracts: %w", err)
	}
	return deployment.ChangesetOutput{
		Proposals:   []timelock.MCMSWithTimelockProposal{},
		AddressBook: ab,
		JobSpecs:    nil,
	}, nil
}

type DeployPrerequisiteConfig struct {
	ChainSelectors []uint64
	Opts           []PrerequisiteOpt
	// TODO handle tokens and feeds in prerequisite config
	Tokens map[TokenSymbol]common.Address
	Feeds  map[TokenSymbol]common.Address
}

func (c DeployPrerequisiteConfig) Validate() error {
	mapAllChainSelectors := make(map[uint64]struct{})
	for _, cs := range c.ChainSelectors {
		mapAllChainSelectors[cs] = struct{}{}
		if err := deployment.IsValidChainSelector(cs); err != nil {
			return fmt.Errorf("invalid chain selector: %d - %w", cs, err)
		}
	}
	return nil
}
