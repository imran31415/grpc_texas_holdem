package server_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"grpc_texas_holdem/poker/client"
	"grpc_texas_holdem/poker/deck"
	pb "grpc_texas_holdem/poker/protobufs"
	"grpc_texas_holdem/poker/server"
	"grpc_texas_holdem/poker/server/game_ring"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const dbName = "testDb"

var (
	testClient     pb.PokerClient
	testDatabase   string
	testConnection *grpc.ClientConn
	ops            uint64 = 0
)

const (
	minChips int64 = 10
)

// Useful for generating a unique id each time a test user is generated
func getUniqueName() string {
	atomic.AddUint64(&ops, 1)
	return fmt.Sprintf("testName_%d", ops)
}

// Generates an error message from the server that matches what is returned by the grpc errors .Error() interface
func rpcError(s string) string {
	return fmt.Sprintf("rpc error: code = Unknown desc = %s", s)
}

// Reuse the test database/connections across tests
// Since all tests share the same DB, we use atomic incr to gen names/ids for
// players and games so we can ensure test cases do not conflict.
// this should be kept in mind when writing tests
func init() {
	rand.Seed(time.Now().Unix())
	testDatabase = fmt.Sprintf("test_%s_%d", "Players", rand.Int63())
	go runTestServer(testDatabase)
	connection, clientApp := client.CreateConnectionClient()
	testClient = clientApp
	testConnection = connection
	defer os.Remove(fmt.Sprintf("./%s.db", testDatabase))
}

func TestMain(m *testing.M) {

	s := m.Run()
	// Cleanup connections and remove test database
	testConnection.Close()
	os.Remove(fmt.Sprintf("./%s.db", testDatabase))
	os.Exit(s)

}

// runTestServer is the same server is used in all tests
func runTestServer(name string) {
	lis, err := net.Listen("tcp", server.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	serv, err := server.NewServer(name)
	if err != nil {
		log.Fatalf("failed to Start poker server: %v", err)
	}
	pb.RegisterPokerServer(s, serv)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func TestServer_CreatePlayer(t *testing.T) {
	testPlayer := getUniqueName()

	tests := []struct {
		Name     string
		Player   *pb.Player
		ExpError string
	}{
		{
			Name: "Create a player",
			Player: &pb.Player{
				Name:  testPlayer,
				Chips: 0,
			},
			ExpError: "",
		},
		{
			Name: "Create player with empty name",
			Player: &pb.Player{
				Name:  "",
				Chips: 0,
			},
			ExpError: rpcError(server.ErrEmptyPlayerName.Error()),
		},
		{
			Name: "Create player that already exists",
			Player: &pb.Player{
				Name:  testPlayer,
				Chips: 0,
			},
			ExpError: rpcError(server.ErrPlayerNameExists.Error()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			p, err := testClient.CreatePlayer(ctx, tt.Player)

			if tt.ExpError != "" {
				require.Equal(t, tt.ExpError, err.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.Player.GetName(), p.GetName())

		})
	}

}

func TestServer_CreatePlayers(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	var playersSetBOneEmpty = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  "",
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tests := []struct {
		Name     string
		Players  *pb.Players
		ExpError string
	}{
		{
			Name: "Create players",
			Players: &pb.Players{
				Players: playersSetA,
			},
			ExpError: "",
		},
		{
			Name: "Create players with one as empty name",
			Players: &pb.Players{
				Players: playersSetBOneEmpty,
			},
			ExpError: rpcError(server.ErrEmptyPlayerName.Error()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			players, err := testClient.CreatePlayers(ctx, tt.Players)
			if err != nil {
				require.Equal(t, tt.ExpError, err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(tt.Players.GetPlayers()), len(players.GetPlayers()))

		})
	}

}

func TestServer_CreateGame(t *testing.T) {

	testGame := getUniqueName()

	tests := []struct {
		Name     string
		Game     *pb.Game
		ExpError string
	}{
		{
			Name: "Create a game",
			Game: &pb.Game{
				Name: testGame,
			},
			ExpError: "",
		},
		{
			Name: "Create game with empty name",
			Game: &pb.Game{
				Name: "",
			},
			ExpError: rpcError(server.ErrEmptyGameName.Error()),
		},
		{
			Name: "Create game that already exists",
			Game: &pb.Game{
				Name: testGame,
			},
			ExpError: rpcError(server.ErrGameNameExists.Error()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			p, err := testClient.CreateGame(ctx, tt.Game)

			if tt.ExpError != "" {
				require.Equal(t, tt.ExpError, err.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.Game.GetName(), p.GetName())

		})
	}

}

func TestServer_GetGame(t *testing.T) {

	tests := []struct {
		Name     string
		Game     *pb.Game
		ExpError string
	}{
		{
			Name: "Create a game then get it ",
			Game: &pb.Game{
				Name: getUniqueName(),
			},
			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			g, err := testClient.CreateGame(ctx, tt.Game)

			if tt.ExpError != "" {
				require.Equal(t, tt.ExpError, err.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.Game.GetName(), g.GetName())

			// get the game and ensure its the same
			g, err = testClient.GetGame(ctx, g)
			require.NoError(t, err)
			require.Equal(t, tt.Game.GetName(), g.GetName())

			// get a non existent game
			g.Id = 100000
			g, err = testClient.GetGame(ctx, g)
			require.Error(t, err)
			require.Equal(t, rpcError(server.ErrGameDoesntExist.Error()), err.Error())

		})
	}

}

func TestServer_DeleteGames(t *testing.T) {
	g1 := getUniqueName()
	g2 := getUniqueName()
	g3 := getUniqueName()
	tests := []struct {
		Name     string
		Games    *pb.Games
		ExpError string
	}{
		{
			Name: "Create a game then get it ",
			Games: &pb.Games{
				Games: []*pb.Game{
					{Name: g1},
					{Name: g2},
					{Name: g3},
				},
			},
			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			gameIds := []int64{}
			for _, tg := range tt.Games.GetGames() {
				g, err := testClient.CreateGame(ctx, tg)
				require.NoError(t, err)
				require.Equal(t, tg.GetName(), g.GetName())
				// get the game and ensure its the same
				g, err = testClient.GetGame(ctx, g)
				require.NoError(t, err)
				require.Equal(t, tg.GetName(), g.GetName())
				gameIds = append(gameIds, g.GetId())
			}

			// verify we can get all the games we created from DB
			createdGames := &pb.Games{}
			for _, id := range gameIds {
				g, err := testClient.GetGame(ctx, &pb.Game{Id: id})
				require.NoError(t, err)
				require.Equal(t, id, g.GetId())
				createdGames.Games = append(createdGames.GetGames(), g)
			}

			// Delete the first game (out of 3 games)
			_, err := testClient.DeleteGames(ctx,
				&pb.Games{Games: []*pb.Game{
					createdGames.GetGames()[0],
				},
				})
			require.NoError(t, err)

			// it is deleted, so we should get an error trying to get it
			_, err = testClient.GetGame(ctx, &pb.Game{Id: gameIds[0]})
			require.Error(t, err)

			//delete remaining games
			_, err = testClient.DeleteGames(ctx, &pb.Games{Games: createdGames.GetGames()})
			require.NoError(t, err)
			// verify we can get none the games we created from DB
			for _, id := range createdGames.GetGames() {
				_, err := testClient.GetGame(ctx, &pb.Game{Id: id.GetId()})
				require.Error(t, err)
			}

		})
	}

}

func TestServer_SetGamePlayers(t *testing.T) {

	testGame := getUniqueName()

	tests := []struct {
		Name               string
		PlayersToCreate    *pb.Players
		GameToCreate       *pb.Game
		SecondSetOfPlayers *pb.Game
		ThirdSetOfPlayers  *pb.Game
		SecondNumOfPlayers int
		FinalNumOfPlayers  int
		ExpError           string
	}{
		{
			Name: "Create game players",
			// These are all the players that will be referenced  and reused
			// in the test so  don't generate unique ones
			PlayersToCreate: &pb.Players{
				Players: []*pb.Player{
					{
						Name:  "bob",
						Chips: 0,
					},
					{
						Name:  "jim",
						Chips: 0,
					},
					{
						Name:  "fred",
						Chips: 0,
					},
					{
						Name:  "cam",
						Chips: 0,
					},
					{
						Name:  "tim",
						Chips: 0,
					},
					{
						Name:  "mary",
						Chips: 0,
					},
					{
						Name:  "jaimie",
						Chips: 0,
					},
				},
			},
			GameToCreate: &pb.Game{
				Name: testGame,
				Players: &pb.Players{
					Players: []*pb.Player{
						// all are new players
						{
							Name:  "bob",
							Chips: 0,
						},
						{
							Name:  "jim",
							Chips: 0,
						},
						{
							Name:  "fred",
							Chips: 0,
						},
						{
							Name:  "cam",
							Chips: 0,
						},
						{
							Name:  "tim",
							Chips: 0,
						},
					},
				},
			},
			SecondSetOfPlayers: &pb.Game{
				Name: testGame,
				Players: &pb.Players{
					Players: []*pb.Player{
						{
							Name:  "mary", // new player
							Chips: 0,
						},
						{
							Name:  "tim",
							Chips: 0,
						},
						{
							Name:  "jim",
							Chips: 0,
						},
					},
				},
			},
			ThirdSetOfPlayers: &pb.Game{
				Name: testGame,
				Players: &pb.Players{
					Players: []*pb.Player{
						{
							Name:  "mary",
							Chips: 0,
						},
						{
							Name:  "tim",
							Chips: 0,
						},
						{
							Name:  "jaimie", // New
							Chips: 0,
						},
					},
				},
			},
			SecondNumOfPlayers: 6,
			FinalNumOfPlayers:  7,

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))
			// Create the initial game
			game, err := testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)

			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))

			// Set the second set of game players
			_, err = testClient.SetGamePlayers(ctx, &pb.Game{Id: game.GetId(), Players: tt.SecondSetOfPlayers.Players})
			require.NoError(t, err)

			// validate the number of total players after setting second round of players is correct
			players, err = testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, tt.SecondNumOfPlayers, len(players.GetPlayers()))

			// Set the third set of game players
			_, err = testClient.SetGamePlayers(ctx, &pb.Game{Id: game.GetId(), Players: tt.ThirdSetOfPlayers.Players})
			require.NoError(t, err)

			// validate the number of final players is correct
			players, err = testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, tt.FinalNumOfPlayers, len(players.GetPlayers()))

		})
	}
}

func TestServer_SetPlayerSlot(t *testing.T) {
	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
	}{
		{
			Name: "Create game and set player slots",
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))
			// Create the initial game
			game, err := testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)

			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))

			// get a player

			for i, player := range players.GetPlayers() {
				slot := i + 1
				// slot should be empty
				assert.Equal(t, int64(0), player.GetSlot())
				// Set the slot to 1 position
				player.Slot = int64(slot)

				player, err = testClient.SetPlayerSlot(ctx, player)
				require.NoError(t, err)
				// get player
				player, err = testClient.GetPlayer(ctx, &pb.Player{Id: player.GetId()})
				require.NoError(t, err)
				// Slot should now be 1
				assert.Equal(t, int64(slot), player.GetSlot())

			}

		})
	}
}

func TestServer_AllocateGameSlots(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
	}{
		{
			Name: "Create game players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))
			// Create the initial game
			game, err := testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)

			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))

			// get the game
			gameToAllocate, err := testClient.GetGame(ctx, game)
			require.NoError(t, err)
			allocatedGame, err := testClient.AllocateGameSlots(ctx, gameToAllocate)
			require.NoError(t, err)
			// Validate all slots were allocated
			for _, p := range allocatedGame.GetPlayers().GetPlayers() {
				slot := p.GetSlot()
				assert.Greater(t, slot, int64(0))
				assert.Less(t, slot, int64(9))
			}

		})
	}
}

// TestServer_SetButtonPositions allocates players to slots
// and  tests the Game ring logic.
// In the test we generate a game, allocate players and set positions.
// Next we shift the dealer and validate the game ring logic appropriately
// manages determining the correct small/big blind positions and also
// This process also tests that we are correctly serializing and de-serializing
// game/player info relative to slots
func TestServer_SetButtonPositions(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}
	// test maximum number of players
	var playersSetB = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	// since heads up has different rules,
	// 3 is the minimum we can test for this strategy
	var playersSetC = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
	}{
		{
			Name: "Test a game with 5 players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
		{
			Name: "Test a game with the max number of players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetB,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetB,
				},
			},

			ExpError: "",
		},

		{
			Name: "Test a game with 3 players (min possible for this strat)",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetC,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetC,
				},
			},

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var game *pb.Game
			// First create some players
			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))

			// Create the initial game
			game, err = testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)
			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// Get the game from DB now that players are set
			game, err = testClient.GetGame(ctx, game)

			// validate the number of initial players is correct
			// Check the player-game join table against what is expected
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))
			// check the serialized game has the correct number of players
			require.Equal(t, len(game.GetPlayers().GetPlayers()), len(tt.GameToCreate.GetPlayers().GetPlayers()))

			// allocate players to the game slots
			game, err = testClient.AllocateGameSlots(ctx, game)
			require.NoError(t, err)
			// Validate all slots were allocated
			for _, p := range game.GetPlayers().GetPlayers() {
				slot := p.GetSlot()
				assert.Greater(t, slot, int64(0))
				assert.Less(t, slot, int64(9))
			}

			// Now that players are seated, set dealer position
			game, err = testClient.SetButtonPositions(ctx, game)
			game.Min = minChips
			game, err = testClient.SetMin(ctx, game)
			require.NoError(t, err)
			// assert min is set.
			assert.Equal(t, minChips, game.GetMin())

			// assert dealer is set
			assert.Greater(t, int(game.GetDealer()), 0)

			//--------------------------
			// SECTION 2: Test game ring logic
			// ------------------------

			// Generate a gameRing and get the allocations
			gameRing, err := game_ring.NewRing(game)
			require.NoError(t, err)

			// Get the dealer according to game ring
			d, err := gameRing.CurrentDealer()
			require.NoError(t, err)

			// Get Small blind
			err = gameRing.CurrentSmallBlind()
			require.NoError(t, err)
			s, err := gameRing.MarshalValue()
			require.NoError(t, err)

			// Get Big blind
			err = gameRing.CurrentBigBlind()
			require.NoError(t, err)
			b, err := gameRing.MarshalValue()
			require.NoError(t, err)

			//The dealer's slot according to the gamering should match
			// the same slot as saved in the Game DB
			require.Equal(t, d.GetSlot(), game.GetDealer())

			// None of the slots should equal each other
			// (since there are atl east 3 players in this test)
			require.NotEqual(t, d.GetSlot(), b.GetSlot())
			require.NotEqual(t, s.GetSlot(), b.GetSlot())
			require.NotEqual(t, b.GetSlot(), s.GetSlot())

			// The next sequence tests that the big/small are in the correct position
			// relative to the dealer

			// Shift to current dealer
			_, err = gameRing.CurrentDealer()
			require.NoError(t, err)
			// Get the player in the next position from the dealer
			// Should be the small blind
			gameRing.Ring = gameRing.Next()
			player, ok := gameRing.Value.(*pb.Player)
			require.True(t, ok)
			assert.Equal(t, s.GetSlot(), player.GetSlot())

			// The next one over should be big blind
			gameRing.Ring = gameRing.Next()
			player, ok = gameRing.Value.(*pb.Player)
			require.True(t, ok)
			assert.Equal(t, b.GetSlot(), player.GetSlot())

			//-------------------------------
			// SECTION 3: Simmulate a New round
			//-------------------------------
			// Update the DB that there is a new dealer
			game, err = testClient.NextDealer(ctx, game)

			// Get the game ring for the game now that dealer has shifted
			r2, err := game_ring.NewRing(game)
			require.NoError(t, err)
			newDealer, err := r2.CurrentDealer()
			//validate the next dealer matches between the one set in the game and in the game ring
			assert.Equal(t, r2.GetDealer(), newDealer.GetSlot())

			// The new dealer should equal the small blind from the last round
			assert.Equal(t, newDealer.GetSlot(), s.GetSlot())

			// validate the new big and small blinds are different after dealer has switched.
			// Get Small blind
			err = r2.CurrentSmallBlind()
			require.NoError(t, err)
			s2, err := r2.MarshalValue()
			require.NoError(t, err)
			assert.NotEqual(t, s2.GetId(), s.GetId())

			// validate big blind matches between game ring allocation and game db
			err = r2.CurrentBigBlind()
			require.NoError(t, err)
			b2, err := r2.MarshalValue()
			require.NoError(t, err)
			assert.NotEqual(t, b2.GetId(), b.GetId())

		})
	}
}

func TestServer_SetButtonPositionsErrors(t *testing.T) {
	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToTest      *pb.Game
		ExpError        string
	}{
		{
			Name: "Test a game that doesn't exist",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: nil,
			},
			GameToTest: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: nil,
				},
			},

			ExpError: server.ErrGameDoesntExist.Error(),
		},
		{
			Name: "empty name should return error",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: nil,
			},
			GameToTest: &pb.Game{
				Name: "",
				Players: &pb.Players{
					Players: nil,
				},
			},
			ExpError: server.ErrEmptyGameName.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := testClient.SetButtonPositions(ctx, tt.GameToTest)
			require.Error(t, err)
			require.Equal(t, rpcError(tt.ExpError), err.Error())

		})
	}
}

// ValidatePreGame returns an error if the game is invalid
// Invalid reasons are
//  1. Not enough, or too many players
//  2. Slots are allocated to players incorrectly
//  3. Button positions and bet is not set.
func TestServer_ValidatePreGame(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}
	var playersSetB = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}
	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		AllocateSlots   bool
		AllocateMinBet  bool
		AllocateDealer  bool
		MinBet          int64
		ExpError        string
	}{
		{
			Name: "Test a validly set game",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},
			AllocateSlots:  true,
			AllocateMinBet: true,
			AllocateDealer: true,
			MinBet:         100,
			ExpError:       "",
		},

		{
			Name: "Test invalid, slots not allocated",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetB,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetB,
				},
			},
			AllocateSlots:  false,
			AllocateMinBet: true,
			AllocateDealer: true,
			MinBet:         100,
			ExpError:       server.ErrInvalidSlotNumber.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var game *pb.Game
			// First create some players
			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))

			// Create the initial game
			game, err = testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)
			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// Get the game from DB now that players are set
			game, err = testClient.GetGame(ctx, game)

			// allocate players to the game slots

			if tt.AllocateSlots {
				game, err = testClient.AllocateGameSlots(ctx, game)
				require.NoError(t, err)
			}

			if tt.AllocateDealer {
				// Now that players are seated, set dealer position and min bet
				game, err = testClient.SetButtonPositions(ctx, game)
				require.NoError(t, err)
			}
			if tt.AllocateMinBet {
				game.Min = tt.MinBet
				game, err = testClient.SetMin(ctx, game)
				require.NoError(t, err)

			}

			game, err = testClient.ValidatePreGame(ctx, game)
			if err != nil {
				require.Equal(t, rpcError(tt.ExpError), err.Error())
			}
		})
	}
}

func TestServer_DeletePlayers(t *testing.T) {

	tests := []struct {
		Name     string
		Players  *pb.Players
		ExpError string
	}{
		{
			Name: "Create a player",
			Players: &pb.Players{
				Players: []*pb.Player{
					{Name: getUniqueName()},
					{Name: getUniqueName()},
					{Name: getUniqueName()},
					{Name: getUniqueName()},
				},
			},
			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers := []*pb.Player{}
			for _, toCreate := range tt.Players.GetPlayers() {
				p, err := testClient.CreatePlayer(ctx, toCreate)
				require.NoError(t, err)
				createdPlayers = append(createdPlayers, p)
			}

			// verify we can get all the pones we created
			players, err := testClient.GetPlayersByName(ctx, &pb.Players{Players: createdPlayers})
			require.NoError(t, err)
			require.Equal(t, len(players.GetPlayers()), len(tt.Players.GetPlayers()))
			// Delete 1 player
			_, err = testClient.DeletePlayers(ctx, &pb.Players{Players: []*pb.Player{
				createdPlayers[0],
			}})
			require.NoError(t, err)

			// returned players should be missing one
			players, err = testClient.GetPlayersByName(ctx, &pb.Players{Players: createdPlayers})
			require.NoError(t, err)
			require.Equal(t, len(players.GetPlayers())+1, len(tt.Players.GetPlayers()))

			// Delete the rest
			_, err = testClient.DeletePlayers(ctx, &pb.Players{Players: createdPlayers})
			require.NoError(t, err)

			// Mo No players should get returned
			players, err = testClient.GetPlayersByName(ctx, &pb.Players{Players: createdPlayers})
			require.NoError(t, err)
			require.Equal(t, 0, len(players.GetPlayers()))
		})
	}

}

// During heads up (1v1) the blinds do not follow the same paradigm (small blind left of dealer)
// We should test the blinds are set correctly in heads up
func TestServer_TestHeadsUp(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
	}{
		{
			Name: "Test a game with 2 players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var game *pb.Game
			// First create some players
			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))

			// Create the initial game
			game, err = testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)
			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// Get the game from DB now that players are set
			game, err = testClient.GetGame(ctx, game)

			// validate the number of initial players is correct
			// Check the player-game join table against what is expected
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))
			// check the serialized game has the correct number of players
			require.Equal(t, len(game.GetPlayers().GetPlayers()), len(tt.GameToCreate.GetPlayers().GetPlayers()))

			// allocate players to the game slots
			game, err = testClient.AllocateGameSlots(ctx, game)
			require.NoError(t, err)
			// Validate all slots were allocated
			for _, p := range game.GetPlayers().GetPlayers() {
				slot := p.GetSlot()
				assert.Greater(t, slot, int64(0))
				assert.Less(t, slot, int64(3)) // only 2 players
			}

			// Now that players are seated, set dealer position
			game, err = testClient.SetMin(ctx, game)

			// Now that players are seated, set dealer position
			game, err = testClient.SetButtonPositions(ctx, game)
			require.NoError(t, err)

			// Set the min bet
			game.Min = minChips
			game, err = testClient.SetMin(ctx, game)
			require.NoError(t, err)
			assert.Equal(t, minChips, game.GetMin())

			// assert dealer is set
			assert.Greater(t, int(game.GetDealer()), 0)

			//--------------------------
			// SECTION 2: Test game ring logic
			// ------------------------

			// Generate a gameRing and get the allocations
			gameRing, err := game_ring.NewRing(game)
			require.NoError(t, err)

			// Get the dealer according to game ring
			d, err := gameRing.CurrentDealer()
			require.NoError(t, err)

			// Get Small blind
			err = gameRing.CurrentSmallBlind()
			require.NoError(t, err)
			s, err := gameRing.MarshalValue()
			require.NoError(t, err)

			// Get Big blind
			err = gameRing.CurrentBigBlind()
			require.NoError(t, err)
			b, err := gameRing.MarshalValue()
			require.NoError(t, err)

			//The dealer's slot according to the gamering should match
			// the same slot as saved in the Game DB
			require.Equal(t, d.GetSlot(), game.GetDealer())

			// in heads up the dealer should be small blind
			require.Equal(t, game.GetDealer(), s.GetSlot())

			// small blind should be the "other person"

			require.NotEqual(t, game.GetDealer(), b.GetSlot())

		})
	}
}

func TestServer_RemovePlayerFromGame(t *testing.T) {

	var playerToRemove1 = &pb.Player{
		Name:  getUniqueName(),
		Chips: 0,
	}
	var playerToRemove2 = &pb.Player{
		Name:  getUniqueName(),
		Chips: 0,
	}

	var playerToRemoveInRound = &pb.Player{
		Name:  getUniqueName(),
		Chips: 0,
	}

	var playerToRemoveDoesntExist = &pb.Player{
		Name:  getUniqueName(),
		Chips: 0,
	}

	var playersSetA = []*pb.Player{
		playerToRemove1,
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		playerToRemove2,
		{
			Name:  getUniqueName(),
			Chips: 0,
		},
		playerToRemoveInRound,
	}

	var allPlayersToCreate = append(playersSetA, playerToRemoveDoesntExist)
	allPlayersToCreate = append(allPlayersToCreate)
	tests := []struct {
		Name                  string
		PlayersToCreate       *pb.Players
		GameToCreate          *pb.Game
		PlayerToRemoveFirst   *pb.Player
		PlayerToRemoveSecond  *pb.Player
		PlayerNotInGame       *pb.Player
		PlayerToRemoveInRound *pb.Player
		ExpError              string
	}{
		{
			Name: "Create game players and remove some",
			PlayersToCreate: &pb.Players{
				Players: allPlayersToCreate,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},
			PlayerToRemoveFirst:   playerToRemove1,
			PlayerToRemoveSecond:  playerToRemove2,
			PlayerNotInGame:       playerToRemoveDoesntExist,
			PlayerToRemoveInRound: playerToRemoveInRound,

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))
			// Create the initial game
			game, err := testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)

			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))

			// Remove the first player we want to remove
			playerToRemoveFirst, err := testClient.GetPlayersByName(
				ctx,
				&pb.Players{Players: []*pb.Player{
					tt.PlayerToRemoveFirst,
				},
				})
			require.Equal(t, 1, len(playerToRemoveFirst.GetPlayers()))
			p := playerToRemoveFirst.GetPlayers()[0]
			_, err = testClient.RemovePlayerFromGame(ctx, p)
			require.NoError(t, err)

			// validate the number of  players after removing 1 player is correct
			players, err = testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers())-1, len(players.GetPlayers()))

			// Remove the second player we want to remove
			playerToRemoveSecond, err := testClient.GetPlayersByName(
				ctx,
				&pb.Players{Players: []*pb.Player{
					tt.PlayerToRemoveSecond,
				},
				})
			require.Equal(t, 1, len(playerToRemoveSecond.GetPlayers()))
			p = playerToRemoveSecond.GetPlayers()[0]
			_, err = testClient.RemovePlayerFromGame(ctx, p)
			require.NoError(t, err)

			// validate the number of  players after removing 2 players is correct
			players, err = testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers())-2, len(players.GetPlayers()))

			// Remove a player that doesn't exist in the game
			playerToRemove, err := testClient.GetPlayersByName(
				ctx,
				&pb.Players{Players: []*pb.Player{
					tt.PlayerNotInGame,
				},
				})
			//player exists but is not in game
			require.Equal(t, 1, len(playerToRemove.GetPlayers()))
			p = playerToRemove.GetPlayers()[0]
			// This should return error since player was not in the game
			_, err = testClient.RemovePlayerFromGame(ctx, p)
			require.Equal(t, rpcError(server.ErrPlayerDoesntExist.Error()), err.Error())

			// Set the game in round to true
			game, err = testClient.UpdateGameInRound(ctx, game)
			require.NoError(t, err)
			require.Equal(t, true, game.GetInRound())

			// Try to remove the third playe
			// This should fgail since the game is now InRound=true
			playerToRemoveInRound, err := testClient.GetPlayersByName(
				ctx,
				&pb.Players{Players: []*pb.Player{
					tt.PlayerToRemoveInRound,
				},
				})
			require.Equal(t, 1, len(playerToRemoveInRound.GetPlayers()))
			p = playerToRemoveInRound.GetPlayers()[0]
			_, err = testClient.RemovePlayerFromGame(ctx, p)
			require.Equal(t, rpcError(server.ErrGameInRound.Error()), err.Error())

		})
	}
}

// TestServer_CreateRoundFromGame creates a round from a game, creates a deck,
// shuffles it and then sets the action position
func TestServer_CreateRoundFromGame(t *testing.T) {
	testMin := int64(1000)

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
	}{
		{
			Name: "Create game players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createdPlayers, err := testClient.CreatePlayers(ctx, tt.PlayersToCreate)
			require.NoError(t, err)
			require.Equal(t, len(tt.PlayersToCreate.GetPlayers()), len(createdPlayers.GetPlayers()))
			// Create the initial game
			game, err := testClient.CreateGame(ctx, tt.GameToCreate)
			require.NoError(t, err)

			game.Players = tt.GameToCreate.GetPlayers()

			// Set the initial game players
			_, err = testClient.SetGamePlayers(ctx, game)
			require.NoError(t, err)

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			require.NoError(t, err)
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(players.GetPlayers()))

			// get the game and get it to a ready state
			gameToAllocate, err := testClient.GetGame(ctx, game)
			require.NoError(t, err)
			allocatedGame, err := testClient.AllocateGameSlots(ctx, gameToAllocate)
			require.NoError(t, err)
			buttonSetGame, err := testClient.SetButtonPositions(ctx, allocatedGame)
			require.NoError(t, err)
			buttonSetGame.Min = minChips
			readyGame, err := testClient.SetMin(ctx, buttonSetGame)
			require.NoError(t, err)

			// sanity check the game is set with players correctly before creating round
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(readyGame.GetPlayers().GetPlayers()))

			round, err := testClient.CreateRoundFromGame(ctx, readyGame)
			require.NoError(t, err)
			require.Equal(t, readyGame.GetId(), round.GetGame())

			// num of players in round should equal the game it was created from
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(round.GetPlayers().GetPlayers()))

			roundPlayers, err := testClient.GetRoundPlayersByRoundId(ctx, round)
			require.NoError(t, err)
			// num of players in round should equal the game it was created from
			require.Equal(t, len(tt.GameToCreate.GetPlayers().GetPlayers()), len(roundPlayers.GetPlayers()))

			round, err = testClient.ValidatePreRound(ctx, round)
			require.NoError(t, err)

			round, err = testClient.StartRound(ctx, round)
			require.NoError(t, err)

			for _, p := range round.GetPlayers().GetPlayers() {
				assert.NotEqual(t, "", p.GetCards())
			}

			d := deck.Deck{}
			d = d.Marshal(round.GetDeck())
			// The number of cards left in the deck should = 52 - ((2 * N) + 1), where N is number of players and 1 is for burning the first card
			require.Equal(t, 52-(len(tt.GameToCreate.GetPlayers().GetPlayers())*2+1), len(d))
			require.NotEqual(t, 0, round.GetAction())

			// verify small and big blinds as been deducted
			// Need to re-get the game since the players have been updated
			game, err = testClient.GetGame(ctx, readyGame)
			ring, err := game_ring.NewRing(game)
			require.NoError(t, err)

			err = ring.CurrentSmallBlind()
			require.NoError(t, err)
			small, err := ring.MarshalValue()
			require.Equal(t, testMin-game.GetMin(), small.GetChips())

			err = ring.CurrentBigBlind()
			require.NoError(t, err)
			big, err := ring.MarshalValue()
			require.Equal(t, testMin-(game.GetMin()*2), big.GetChips())
			// at this point the round is ready to go.

		})
	}
}

type betTest struct {
	err string
	bet *pb.Bet
}

// TestServer_MakeBets creates starts a round and tests making bets, progressing through a full round
func TestServer_MakeBets(t *testing.T) {
	testMin := int64(1000)

	var playersSetA = []*pb.Player{
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
		{
			Name:  getUniqueName(),
			Chips: testMin,
		},
	}

	tests := []struct {
		Name            string
		PlayersToCreate *pb.Players
		GameToCreate    *pb.Game
		ExpError        string
		//Pre flop
		bet1 []betTest
		bet2 []betTest
		bet3 []betTest
		bet4 []betTest
		bet5 []betTest
		bet6 []betTest
		bet7 []betTest
		bet8 []betTest
		bet9 []betTest

		posFlop_bet1 []betTest
		posFlop_bet2 []betTest
		posFlop_bet3 []betTest
		posFlop_bet4 []betTest
		posFlop_bet5 []betTest
		posFlop_bet6 []betTest
		posFlop_bet7 []betTest

		river_bet1 []betTest
		river_bet2 []betTest
		river_bet3 []betTest
		river_bet4 []betTest
		river_bet5 []betTest

		turn_bet1 []betTest
		turn_bet2 []betTest
		turn_bet3 []betTest
		turn_bet4 []betTest
		turn_bet5 []betTest

		show_bet1 []betTest
		show_bet2 []betTest
		show_bet3 []betTest
		show_bet4 []betTest
	}{
		{
			Name: "Make bets",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: getUniqueName(),
				Players: &pb.Players{
					Players: playersSetA,
				},
				Min: minChips,
			},

			ExpError: "",
			// First bet, should be a call to the same amount as a big blind
			bet1: []betTest{
				{
					bet: &pb.Bet{
						Chips:  1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrInsufficientBet.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  100000,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrInsufficientChips.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrIncorrectBetForBetType.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  minChips * 2,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			// Second bet, lets do a raise which is 1 more chip than the prev call
			bet2: []betTest{
				{
					bet: &pb.Bet{
						Chips:  1,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrInsufficientBet.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  100000,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrInsufficientChips.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  minChips * 2,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrWrongBetType.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			// Third bet, test a fold will remove player from hand
			bet3: []betTest{
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_FOLD,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			bet4: []betTest{
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrWrongBetType.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			bet5: []betTest{
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			bet6: []betTest{
				{
					bet: &pb.Bet{
						Chips:  minChips*2 + 1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			// there are 7 players left, so this is back to the small blind who owes 11 (big blind-small +raise) (20 -10 +1)
			bet7: []betTest{
				{
					bet: &pb.Bet{
						Chips:  minChips + 1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			// This is the big blind so only need 1 to call since that's what the raise was
			bet8: []betTest{
				{
					bet: &pb.Bet{
						Chips:  1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			// this is the first button position that originally raised, needs 1 to continue
			bet9: []betTest{
				{
					bet: &pb.Bet{
						Chips:  1,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet1: []betTest{
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_FOLD,
						Status: pb.RoundStatus_PRE_FLOP,
					},
					err: rpcError(server.ErrWrongBetStatus.Error()),
				},

				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_FOLD,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet2: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_FLOP,
					},
					err: rpcError(server.ErrIncorrectBetForBetType.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet3: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_FLOP,
					},
					err: rpcError(server.ErrWrongBetType.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet4: []betTest{

				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_FOLD,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet5: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet6: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},
			posFlop_bet7: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_FLOP,
					},
					err: "",
				},
			},

			river_bet1: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_FLOP,
					},
					err: rpcError(server.ErrWrongBetStatus.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_RIVER,
					},
					err: "",
				},
			},
			river_bet2: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_RIVER,
					},
					err: "",
				},
			},
			river_bet3: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_RIVER,
					},
					err: "",
				},
			},
			river_bet4: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_RIVER,
					},
					err: "",
				},
			},
			river_bet5: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_RIVER,
					},
					err: "",
				},
			},
			turn_bet1: []betTest{

				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_RIVER,
					},
					err: rpcError(server.ErrWrongBetStatus.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_FOLD,
						Status: pb.RoundStatus_TURN,
					},
					err: "",
				},
			},
			turn_bet2: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_RAISE,
						Status: pb.RoundStatus_TURN,
					},
					err: "",
				},
			},
			turn_bet3: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_TURN,
					},
					err: "",
				},
			},
			turn_bet4: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_TURN,
					},
					err: "",
				},
			},
			turn_bet5: []betTest{
				{
					bet: &pb.Bet{
						Chips:  20,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_TURN,
					},
					err: "",
				},
			},
			show_bet1: []betTest{

				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_TURN,
					},
					err: rpcError(server.ErrWrongBetStatus.Error()),
				},
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_SHOW,
					},
					err: "",
				},
			},
			show_bet2: []betTest{
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_SHOW,
					},
					err: "",
				},
			},
			show_bet3: []betTest{
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_SHOW,
					},
					err: "",
				},
			},
			show_bet4: []betTest{
				{
					bet: &pb.Bet{
						Chips:  0,
						Type:   pb.Bet_CALL,
						Status: pb.RoundStatus_SHOW,
					},
					err: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			round, bets, readyGame := setupGame(t, tt.PlayersToCreate, tt.GameToCreate)
			require.Equal(t, 2, len(bets.GetBets()))
			b1 := bets.GetBets()[0]
			require.Equal(t, pb.Bet_SMALL, b1.GetType())
			require.Equal(t, readyGame.GetMin(), b1.GetChips())
			b2 := bets.GetBets()[1]
			require.Equal(t, pb.Bet_BIG, b2.GetType())
			require.Equal(t, readyGame.GetMin()*2, b2.GetChips())
			ctx := context.Background()
			b1 = bets.GetBets()[0]
			b2 = bets.GetBets()[1]

			require.Equal(t, pb.Bet_SMALL, b1.GetType())
			require.Equal(t, minChips, b1.GetChips())

			require.Equal(t, pb.Bet_BIG, b2.GetType())
			require.Equal(t, minChips*2, b2.GetChips())

			// Get the player that should be making a bet and try to make one
			p, err := testClient.GetPlayerOnBet(ctx, round)

			require.NoError(t, err)

			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet1)
			///////////////////////////////////////////
			// At this point small and big blinds are set and the first bet has been cast.
			// Check the things we expect
			b3 := bets.GetBets()[2]
			require.Equal(t, pb.Bet_CALL, b3.GetType())
			require.Equal(t, readyGame.GetMin()*2, b3.GetChips())
			bets, err = testClient.GetRoundBetsForStatus(ctx, round)
			require.NoError(t, err)
			require.Equal(t, 3, len(bets.GetBets()))

			b3 = bets.GetBets()[2]
			require.Equal(t, pb.Bet_CALL, b3.GetType())
			require.Equal(t, minChips*2, b3.GetChips())

			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet2)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet3)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet4)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet5)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet6)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet7)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet8)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.bet9)

			///////////////////////////////////////////
			// At this point the betting round is over,
			// The following should be true:
			// - Flop is saved to round
			// - action is set to the next player in hand left of dealer.
			// - the round status is set to the next round

			require.Equal(t, pb.RoundStatus_FLOP, round.GetStatus())
			// 3 cards (2x character per card)
			require.Equal(t, 6, len(round.GetFlop()))
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet1)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet2)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet3)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet4)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet5)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet6)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.posFlop_bet7)

			// First bet of POST FLOP:
			///////////////////////////////////////////
			// At this point the betting round is over,
			// The following should be true:
			// - River is saved to round
			// - action is set to the next player in hand left of dealer.
			// - the round status is set to the next round: Status_River

			require.Equal(t, pb.RoundStatus_RIVER, round.GetStatus())
			// 1 card (2 character)
			require.Equal(t, 2, len(round.GetRiver()))

			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.river_bet1)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.river_bet2)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.river_bet3)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.river_bet4)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.river_bet5)

			///////////////////////////////////////////
			// At this point the betting round is over,
			// The following should be true:
			// - River is saved to round
			// - action is set to the next player in hand left of dealer.
			// - the round status is set to the next round: Status_River

			require.Equal(t, pb.RoundStatus_TURN, round.GetStatus())
			// 1 card (2 character)
			require.Equal(t, 2, len(round.GetTurn()))

			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.turn_bet1)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.turn_bet2)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.turn_bet3)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.turn_bet4)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.turn_bet5)

			require.Equal(t, pb.RoundStatus_SHOW, round.GetStatus())
			// 1 card (2 character)
			require.Equal(t, 2, len(round.GetTurn()))

			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.show_bet1)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.show_bet2)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.show_bet3)
			bets, round, p = makeAndEvaluateBet(t, ctx, round, readyGame, p, tt.show_bet4)

			require.NoError(t, err)
			require.NotZero(t, round.GetWinningPlayer())
			require.NotZero(t, round.GetWinningScore())
			require.Equal(t, 14, len(round.GetWinningHand()))
			g, err := testClient.GetGame(ctx, &pb.Game{Id: round.GetGame()})
			require.NoError(t, err)

			require.False(t, g.GetInRound())

		})

	}
}

func makeAndEvaluateBet(t *testing.T, ctx context.Context, round *pb.Round, readyGame *pb.Game, p *pb.Player, in []betTest) (*pb.Bets, *pb.Round, *pb.Player) {

	// Fifth bet of RIVER:
	for _, bt := range in {
		bt.bet.Player = p.GetId()
		bt.bet.Game = readyGame.GetId()
		bt.bet.Round = round.GetId()
		// Removed status allocation here to test
		_, err := testClient.MakeBet(ctx, bt.bet)
		if err == nil && bt.err != "" {
			log.Println("Returned No Error when expected to receive the error: ", bt.err)
			t.Fail()
		} else if bt.err != "" {
			require.Equal(t, bt.err, err.Error())
		} else {
			require.NoError(t, err)
		}
	}
	prevAction := round.GetAction()
	round, err := testClient.GetRound(ctx, &pb.Round{Id: round.GetId()})
	p, err = testClient.GetPlayerOnBet(ctx, round)
	require.NoError(t, err)
	require.NotEqual(t, prevAction, p.GetSlot())
	bets, err := testClient.GetRoundBets(ctx, round)
	require.NoError(t, err)
	return bets, round, p
}

func setupGame(t *testing.T, players *pb.Players, inGame *pb.Game) (*pb.Round, *pb.Bets, *pb.Game) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	createdPlayers, err := testClient.CreatePlayers(ctx, players)
	require.NoError(t, err)
	require.Equal(t, len(players.GetPlayers()), len(createdPlayers.GetPlayers()))
	// Create the initial game
	game, err := testClient.CreateGame(ctx, inGame)
	require.NoError(t, err)

	game.Players = inGame.GetPlayers()

	// Set the initial game players
	_, err = testClient.SetGamePlayers(ctx, game)
	require.NoError(t, err)

	// validate the number of initial players is correct
	players, err = testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
	require.NoError(t, err)
	require.Equal(t, len(game.GetPlayers().GetPlayers()), len(players.GetPlayers()))

	// get the game and get it to a ready state
	gameToAllocate, err := testClient.GetGame(ctx, game)
	require.NoError(t, err)
	allocatedGame, err := testClient.AllocateGameSlots(ctx, gameToAllocate)
	require.NoError(t, err)
	buttonSetGame, err := testClient.SetButtonPositions(ctx, allocatedGame)
	require.NoError(t, err)
	buttonSetGame.Min = minChips
	readyGame, err := testClient.SetMin(ctx, buttonSetGame)
	require.NoError(t, err)

	// sanity check the game is set with players correctly before creating round
	require.Equal(t, len(game.GetPlayers().GetPlayers()), len(readyGame.GetPlayers().GetPlayers()))

	round, err := testClient.CreateRoundFromGame(ctx, readyGame)
	require.NoError(t, err)
	require.Equal(t, readyGame.GetId(), round.GetGame())

	// num of players in round should equal the game it was created from
	require.Equal(t, len(game.GetPlayers().GetPlayers()), len(round.GetPlayers().GetPlayers()))

	roundPlayers, err := testClient.GetRoundPlayersByRoundId(ctx, round)
	require.NoError(t, err)
	// num of players in round should equal the game it was created from
	require.Equal(t, len(game.GetPlayers().GetPlayers()), len(roundPlayers.GetPlayers()))

	round, err = testClient.ValidatePreRound(ctx, round)
	require.NoError(t, err)

	round, err = testClient.StartRound(ctx, round)
	require.NoError(t, err)

	bets, err := testClient.GetRoundBets(ctx, round)
	require.NoError(t, err)
	// big and small blinds
	require.Equal(t, 2, len(bets.GetBets()))
	bets, err = testClient.GetRoundBetsForStatus(ctx, round)
	require.NoError(t, err)
	// small and big blind
	require.Equal(t, 2, len(bets.GetBets()))
	return round, bets, readyGame
}

// Utility to log the staus of a game.
// TODO: eventually move this into the actual server when needed
func logRoundBets(t *testing.T, ctx context.Context, round *pb.Round) {
	g, err := testClient.GetGame(ctx, &pb.Game{Id: round.GetGame()})
	require.NoError(t, err)

	log.Println("-------")
	log.Println("Dealer Slot: ", g.GetDealer())
	log.Println("Players:")
	for _, player := range round.GetPlayers().GetPlayers() {
		log.Println("\tPlayer: ", player.GetId(), "Slot: ", player.GetSlot(), player.GetInHand())
	}
	log.Println("-------")
	log.Println("Bets")
	bets, err := testClient.GetRoundBets(ctx, round)
	require.NoError(t, err)

	for _, b := range bets.GetBets() {
		log.Println("\t", b.GetPlayer(), b.GetChips(), b.GetStatus(), b.GetType())
	}
	log.Println("-------")
	log.Println("Action on slot: ", round.GetAction())
	log.Println("Round Status: ", round.GetStatus())
	log.Println("Round Showing: ", round.GetFlop()+round.GetRiver()+round.GetTurn())
	log.Println("Round Winner: ", round.GetWinningPlayer())
	log.Println("Round Winner Score: ", round.GetWinningScore())
	log.Println("Round Winner Hand: ", round.GetWinningHand())

}

func TestServer_EvaluateHands(t *testing.T) {

	tests := []struct {
		Name     string
		Players  *pb.Players
		Count    int
		TopScore uint32
	}{
		{
			Name: "1 player with royal flush",

			Players: &pb.Players{
				Players: []*pb.Player{
					{
						Id:    1,
						Name:  getUniqueName(),
						Cards: strings.Join([]string{"As", "Ks", "Qs", "Js", "Ts", "2d", "3c"}, ""),
					},
				},
			},

			Count:    1,
			TopScore: 1,
		},
		{
			Name: "test another hand",

			Players: &pb.Players{
				Players: []*pb.Player{
					{
						Id:    2,
						Name:  getUniqueName(),
						Cards: strings.Join([]string{"4d", "Qs", "Ts", "Ad", "8s", "Js", "Kh"}, ""),
					},
				},
			},

			Count:    1,
			TopScore: 1600,
		},
		{
			Name: "Multiple players",
			Players: &pb.Players{
				Players: []*pb.Player{
					{
						Id:    2,
						Name:  getUniqueName(),
						Cards: strings.Join([]string{"5c", "Ks", "Qs", "3d", "Ts", "2d", "3c"}, ""),
					},
					{
						Id:    3,
						Name:  getUniqueName(),
						Cards: strings.Join([]string{"Js", "Jh", "Jc", "Js", "Ts", "2d", "3c"}, ""),
					},
					{
						Id:    4,
						Name:  getUniqueName(),
						Cards: strings.Join([]string{"4s", "8s", "9s", "Js", "Ts", "2d", "3c"}, ""),
					},
				},
			},
			Count:    3,
			TopScore: 50, // 4 jacks = 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			r := &pb.Round{
				Status:  pb.RoundStatus_OVER,
				Players: tt.Players,
			}

			players, err := testClient.EvaluateHands(ctx, r)
			require.NoError(t, err)
			require.Equal(t, tt.Count, len(players.GetPlayers().GetPlayers()))

			topPlayer := players.GetPlayers().GetPlayers()[0]
			require.Equal(t, int(tt.TopScore), int(topPlayer.GetScore()))

		})
	}

}
