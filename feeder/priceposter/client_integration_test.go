package priceposter

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/NibiruChain/pricefeeder/types"
	e2eTesting "github.com/archway-network/archway/e2e/testing"
	oracletypes "github.com/archway-network/archway/x/oracle/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	network *e2eTesting.TestChain

	client *Client
	logs   *bytes.Buffer
}

func (s *IntegrationTestSuite) SetupSuite() {
	// app.SetPrefixes(app.AccountAddressPrefix)

	s.network = e2eTesting.NewTestChain(s.T(), 1)

	_, err = s.network.WaitForHeight(1)
	require.NoError(s.T(), err)

	val := s.network.Validators[0]
	grpcEndpoint, tmEndpoint := val.AppConfig.GRPC.Address, val.RPCAddress
	url, err := url.Parse(tmEndpoint)
	require.NoError(s.T(), err)

	url.Scheme = "ws"
	url.Path = "/websocket"

	s.logs = new(bytes.Buffer)

	enableTLS := false
	s.client = Dial(
		grpcEndpoint,
		s.cfg.ChainID,
		enableTLS,
		val.ClientCtx.Keyring,
		val.ValAddress,
		val.Address,
		zerolog.New(io.MultiWriter(os.Stderr, s.logs)),
	)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.network.Cleanup()
	s.client.Close()
}

func (s *IntegrationTestSuite) TestClientWorks() {
	s.client.SendPrices(types.VotingPeriod{}, s.randomPrices())

	// assert vote was skipped because no previous prevote
	require.Contains(s.T(), s.logs.String(), "skipping vote preparation as there is no old prevote")
	require.NotContains(s.T(), s.logs.String(), "prepared vote message")

	// wait for next vote period
	s.waitNextVotePeriod()
	s.client.SendPrices(types.VotingPeriod{}, s.randomPrices())
	require.Contains(s.T(), s.logs.String(), "prepared vote message")
}

func (s *IntegrationTestSuite) randomPrices() []types.Price {
	vt, err := s.client.deps.oracleClient.(oracletypes.QueryClient).VoteTargets(context.Background(), &oracletypes.QueryVoteTargetsRequest{})
	require.NoError(s.T(), err)
	prices := make([]types.Price, len(vt.VoteTargets))
	for i, assetPair := range vt.VoteTargets {
		prices[i] = types.Price{
			Pair:       assetPair,
			Price:      float64(i),
			SourceName: "test",
			Valid:      true,
		}
	}
	return prices
}

func (s *IntegrationTestSuite) waitNextVotePeriod() {
	params, err := s.client.deps.oracleClient.(oracletypes.QueryClient).Params(context.Background(), &oracletypes.QueryParamsRequest{})
	require.NoError(s.T(), err)
	height := s.network.GetBlockHeight()
	require.NoError(s.T(), err)
	// err = testutil.WaitForBlocks(s.network.GetContext().Context(), int(uint64(height)%params.Params.VotePeriod), s.network)
	require.NoError(s.T(), err)
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
