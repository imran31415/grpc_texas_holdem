package server_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"imran/poker/client"
	pb "imran/poker/protobufs"
	"imran/poker/server"
	"imran/poker/server/game_ring"
	"log"
	"math/rand"
	"net"
	"os"
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

// Useful for generating a unique id each time a test user is generated
func getUniqueName() string {
	atomic.AddUint64(&ops, 1)
	return fmt.Sprintf("testName_%d", ops)
}

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

func TestServer_SetGamePlayers(t *testing.T) {

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
				Name: "testgame1",
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
				Name: "testgame1",
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
				Name: "testgame1",
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
			fmt.Println("Created players, ", len(createdPlayers.GetPlayers()))
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
			fmt.Println("Players count", len(players.GetPlayers()))
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

			// TODO: create method to set this
			game.Min = int64(100)

			// Now that players are seated, set dealer position
			game, err = testClient.SetButtonPositions(ctx, game)
			require.NoError(t, err)
			// assert min is set.
			assert.Equal(t, int64(100), game.GetMin())

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
