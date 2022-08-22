package keeper_test

import (
	// "fmt"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/stretchr/testify/suite"

	recordtypes "github.com/Stride-Labs/stride/x/records/types"

	// "github.com/Stride-Labs/stride/x/stakeibc/types"
	stakeibc "github.com/Stride-Labs/stride/x/stakeibc/types"
)

type UpdateRedemptionRatesTestCase struct {
	hostZone         stakeibc.HostZone
	redemptionRateT0 sdk.Dec
	stakedBal        uint64
	undelegatedBal   uint64
	justDepositedBal uint64
	stSupply         uint64
	allRecords       []recordtypes.DepositRecord
}

func (s *KeeperTestSuite) SetupUpdateRedemptionRates(
	stakedBal uint64,
	undelegatedBal uint64,
	justDepositedBal uint64,
	stSupply uint64,
	redemptionRateT0 sdk.Dec,
) UpdateRedemptionRatesTestCase {

	// add some deposit records with status STAKE
	//    to comprise the undelegated delegation account balance i.e. "to be staked"
	toBeStakedDepositRecord := recordtypes.DepositRecord{
		HostZoneId: "GAIA",
		Amount:     int64(undelegatedBal),
		Status:     recordtypes.DepositRecord_STAKE,
	}
	s.App.RecordsKeeper.AppendDepositRecord(s.Ctx, toBeStakedDepositRecord)

	// add a balance to the stakeibc module account (via records)
	//    to comprise the stakeibc module account balance i.e. "to be transferred"
	toBeTransferedDepositRecord := recordtypes.DepositRecord{
		HostZoneId: "GAIA",
		Amount:     int64(justDepositedBal),
		Status:     recordtypes.DepositRecord_TRANSFER,
	}
	s.App.RecordsKeeper.AppendDepositRecord(s.Ctx, toBeTransferedDepositRecord)

	// set the stSupply by minting to a random user account
	user := Account{
		acc:           s.TestAccs[0],
		stAtomBalance: sdk.NewInt64Coin(stAtom, int64(stSupply)),
	}
	s.FundAccount(user.acc, user.stAtomBalance)

	// set the staked balance on the host zone
	hostZone := stakeibc.HostZone{
		ChainId:        "GAIA",
		HostDenom:      "uatom",
		StakedBal:      stakedBal,
		RedemptionRate: redemptionRateT0,
	}
	s.App.StakeibcKeeper.SetHostZone(s.Ctx, hostZone)

	return UpdateRedemptionRatesTestCase{
		hostZone:         hostZone,
		redemptionRateT0: redemptionRateT0,
		stakedBal:        stakedBal,
		undelegatedBal:   undelegatedBal,
		justDepositedBal: justDepositedBal,
		stSupply:         stSupply,
		allRecords:       []recordtypes.DepositRecord{toBeStakedDepositRecord, toBeTransferedDepositRecord},
	}
}

func (s *KeeperTestSuite) TestUpdateRedemptionRatesSuccessful() {

	stakedBal := uint64(5)
	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(10)

	redemptionRateT0 := sdk.NewDec(1)
	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	expectedNewRate := sdk.NewDec(5 + 3 + 3).Quo(sdk.NewDec(10))
	s.Require().Equal(rrNew, expectedNewRate)
	s.Require().Equal(rrNew, sdk.NewDec(11).Quo(sdk.NewDec(10)))
}

func (s *KeeperTestSuite) TestUpdateRedemptionRatesRandomized() {
	// run N tests, each with random inputs

	genRandUintBelowMax := func(MAX int) uint64 {
		MIN := int(0)
		n := 0 + rand.Intn(MAX-MIN+1)
		return uint64(n)
	}

	MAX := 1_000_000_000
	stakedBal := genRandUintBelowMax(MAX)
	undelegatedBal := genRandUintBelowMax(MAX)
	justDepositedBal := genRandUintBelowMax(MAX)

	stSupply := genRandUintBelowMax(MAX)

	// s.Require().ElementsMatch([]int{0, 0, 0, 0}, []int{int(stakedBal), int(undelegatedBal), int(justDepositedBal), int(stSupply)}) //
	redemptionRateT0 := sdk.NewDec(1)
	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	numerator := int64(stakedBal) + int64(undelegatedBal) + int64(justDepositedBal)
	denominator := int64(stSupply)
	expectedNewRate := sdk.NewDec(numerator).Quo(sdk.NewDec(denominator))
	s.Require().Equal(rrNew, expectedNewRate, fmt.Sprintf("expectedNewRate: %v, rrNew: %v; inputs: SB: %d, UDB: %d, JDB: %d, STS: %d RRT0: %d", expectedNewRate, rrNew, stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0))
}

func (s *KeeperTestSuite) TestUpdateRedemptionRatesRandomized_MultipleRuns() {
	for i := 0; i < 100; i++ {
		s.TestUpdateRedemptionRatesRandomized()
		// reset the testing app between runs
		s.Setup()
	}
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateZeroStAssets() {

	stakedBal := uint64(5)
	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(0)

	redemptionRateT0 := sdk.NewDec(1)
	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	// RR should be unchanged
	s.Require().Equal(rrNew, sdk.NewDec(1))
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateZeroNativeAssets() {

	stakedBal := uint64(0)
	undelegatedBal := uint64(0)
	justDepositedBal := uint64(0)
	stSupply := uint64(10)

	redemptionRateT0 := sdk.NewDec(1)
	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	// RR should be 0
	s.Require().Equal(rrNew, sdk.NewDec(0))
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateNoModuleAccountRecords() {

	stakedBal := uint64(5)
	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(10)
	redemptionRateT0 := sdk.NewDec(1)

	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)
	s.App.RecordsKeeper.RemoveDepositRecord(s.Ctx, 0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	expectedNewRate := sdk.NewDec(5 + 3).Quo(sdk.NewDec(10))
	s.Require().Equal(rrNew, expectedNewRate)
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateNoStakeDepositRecords() {

	stakedBal := uint64(5)
	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(10)
	redemptionRateT0 := sdk.NewDec(1)

	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)
	s.App.RecordsKeeper.RemoveDepositRecord(s.Ctx, 1)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	numerator := int64(stakedBal) + int64(justDepositedBal)
	denominator := int64(stSupply)
	expectedNewRate := sdk.NewDec(numerator).Quo(sdk.NewDec(denominator))
	s.Require().Equal(rrNew, expectedNewRate)
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateNoStakedBal() {

	stakedBal := uint64(5)
	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(10)
	redemptionRateT0 := sdk.NewDec(1)

	// SET HZ STAKED BAL TO 0
	tc := s.SetupUpdateRedemptionRates(stakedBal, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, sdk.NewDec(1))

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	expectedNewRate := sdk.NewDec(3 + 3).Quo(sdk.NewDec(10))
	s.Require().Equal(rrNew, expectedNewRate)
}

func (s *KeeperTestSuite) TestUpdateRedemptionRateRandomRedemptionRateT0() {

	undelegatedBal := uint64(3)
	justDepositedBal := uint64(3)
	stSupply := uint64(10)

	genRandUintBelowMax := func(MAX int) int64 {
		MIN := int(1)
		n := 1 + rand.Intn(MAX-MIN+1)
		return int64(n)
	}

	MAX := 1_000_000
	// redemption rate random number, biased to be [1,2)
	redemptionRateT0 := sdk.NewDec(genRandUintBelowMax(MAX)).Quo(sdk.NewDec(genRandUintBelowMax(MAX / 2)))

	// SET HZ STAKED BAL TO 0
	tc := s.SetupUpdateRedemptionRates(0, undelegatedBal, justDepositedBal, stSupply, redemptionRateT0)

	// sanity check on inputs (check redemptionRate at genesis is 1)
	s.Require().Equal(tc.redemptionRateT0, redemptionRateT0)

	records := tc.allRecords
	s.App.StakeibcKeeper.UpdateRedemptionRates(s.Ctx, records)

	hz, found := s.App.StakeibcKeeper.GetHostZone(s.Ctx, tc.hostZone.ChainId)
	s.Require().True(found)
	rrNew := hz.RedemptionRate

	expectedNewRate := sdk.NewDec(3 + 3 + 5).Quo(sdk.NewDec(10))
	s.Require().Equal(rrNew, expectedNewRate)
}
