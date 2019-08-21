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
func getUniqueUser() string {
	atomic.AddUint64(&ops, 1)
	return fmt.Sprintf("testUser_%d", ops)
}

// Reuse the test database/connections across tests
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
	testConnection.Close()
	os.Remove(fmt.Sprintf("./%s.db", testDatabase))
	os.Exit(s)

}

// go test -v poker/server/server_test.go

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
	testPlayer := getUniqueUser()

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
			ExpError: "rpc error: code = Unknown desc = can not create player with empty name",
		},
		{
			Name: "Create player that already exists",
			Player: &pb.Player{
				Name:  testPlayer,
				Chips: 0,
			},
			ExpError: "rpc error: code = Unknown desc = player with that name already exists",
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
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
	}

	var playersSetBOneEmpty = []*pb.Player{
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  "",
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
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
			ExpError: "rpc error: code = Unknown desc = can not create player with empty name",
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

	tests := []struct {
		Name     string
		Game     *pb.Game
		ExpError string
	}{
		{
			Name: "Create a game",
			Game: &pb.Game{
				Name: "testgame0",
			},
			ExpError: "",
		},
		{
			Name: "Create game with empty name",
			Game: &pb.Game{
				Name: "",
			},
			ExpError: "rpc error: code = Unknown desc = can not create game with empty name",
		},
		{
			Name: "Create game that already exists",
			Game: &pb.Game{
				Name: "testgame0",
			},
			ExpError: "rpc error: code = Unknown desc = game with that name already exists",
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
			// These are all the players that will be referenced in the test
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
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:  getUniqueUser(),
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
				Name: "testgame2",
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
			Name:  getUniqueUser(),
			Chips: 0,
		},
		{
			Name:   getUniqueUser(),
			Chips: 0,
		},
		{
			Name:   getUniqueUser(),
			Chips: 0,
		},
		{
			Name:   getUniqueUser(),
			Chips: 0,
		},
		{
			Name:   getUniqueUser(),
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
				Players:playersSetA,
			},
			GameToCreate: &pb.Game{
				Name: "testgame3",
				Players: &pb.Players{
					Players:playersSetA,
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
// and also tests the Game ring logic.
func TestServer_SetButtonPositions(t *testing.T) {

	var playersSetA = []*pb.Player{
		{
			Name:  "bob3333",
			Chips: 0,
		},
		{
			Name:  "jim3333",
			Chips: 0,
		},
		{
			Name:  "fred3333",
			Chips: 0,
		},
		{
			Name:  "cam3333",
			Chips: 0,
		},
		{
			Name:  "tim3333",
			Chips: 0,
		},
	}

	var playersSetB = []*pb.Player{
		{
			Name:  "bob333341",
			Chips: 0,
		},
		{
			Name:  "jim333341",
			Chips: 0,
		},
		{
			Name:  "fred333341",
			Chips: 0,
		},
		{
			Name:  "cam333341",
			Chips: 0,
		},
		{
			Name:  "tim333341",
			Chips: 0,
		},
		{
			Name:  "sam333341",
			Chips: 0,
		},
		{
			Name:  "sarah333341",
			Chips: 0,
		},
		{
			Name:  "joe333341",
			Chips: 0,
		},
	}

	var playersSetC = []*pb.Player{
		{
			Name:  "bob3333412",
			Chips: 0,
		},
		{
			Name:  "jim3333411",
			Chips: 0,
		},
		{
			Name:  "fred3333411",
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
				Name: "testgame4",
				Players: &pb.Players{
					Players: playersSetA,
				},
			},

			ExpError: "",
		},
		{
			Name: "Create game players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetB,
			},
			GameToCreate: &pb.Game{
				Name: "testgame5",
				Players: &pb.Players{
					Players: playersSetB,
				},
			},

			ExpError: "",
		},

		{
			Name: "Create game players",
			// These are all the players that will be referenced in the test
			PlayersToCreate: &pb.Players{
				Players: playersSetC,
			},
			GameToCreate: &pb.Game{
				Name: "testgame6",
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

			game, err = testClient.GetGame(ctx, game)
			fmt.Println("Gamne players c", len(game.GetPlayers().GetPlayers()))

			// validate the number of initial players is correct
			players, err := testClient.GetGamePlayersByGameId(ctx, &pb.Game{Id: game.GetId()})
			fmt.Println("Players count", len(players.GetPlayers()))
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
			allocatedGame.Min = int64(100)

			g, err := testClient.SetButtonPositions(ctx, allocatedGame)
			require.NoError(t, err)
			// Verify game is set.
			assert.Equal(t, int64(100), g.GetMin())

			// assert all positions are set
			assert.NotEqual(t, 0, g.GetDealer())

			// get the game
			readyGame, err := testClient.GetGame(ctx, allocatedGame)
			require.NoError(t, err)
			r, err := game_ring.NewRing(readyGame)
			require.NoError(t, err)

			// Get the dealer according to game ring
			d, err := r.CurrentDealer()
			require.NoError(t, err)
			require.NoError(t, err)

			// Get Small blind
			err = r.CurrentSmallBlind()
			require.NoError(t, err)
			s, err := r.MarshalValue()
			require.NoError(t, err)

			// Get Big blind
			err = r.CurrentBigBlind()
			require.NoError(t, err)
			b, err := r.MarshalValue()
			require.NoError(t, err)

			// ensure the player returned's slot and the games dealer slot match
			require.Equal(t, d.GetSlot(), readyGame.GetDealer())

			// None of the slots should equal each other
			require.NotEqual(t, d.GetSlot(), b.GetSlot())
			require.NotEqual(t, s.GetSlot(), b.GetSlot())
			require.NotEqual(t, b.GetSlot(), s.GetSlot())

			// Check the current dealer according to game ring Equals the games dealer.
			d, err = r.CurrentDealer()
			require.NoError(t, err)
			r.Ring = r.Next()
			player, ok := r.Value.(*pb.Player)
			require.True(t, ok)
			// Player next from the dealer should equal small blind

			//check bigblind
			assert.Equal(t, s.GetSlot(), player.GetSlot())
			r.Ring = r.Next()
			player, ok = r.Value.(*pb.Player)
			require.True(t, ok)
			// Player next from the dealer should equal small blind
			assert.Equal(t, b.GetSlot(), player.GetSlot())

			nextDealerGame, err := testClient.NextDealer(ctx, readyGame)
			// Get the game ring for the game now that dealer has shifted
			r2, err := game_ring.NewRing(nextDealerGame)
			require.NoError(t, err)
			newDealer, err := r2.CurrentDealer()
			//validate the next dealer matches between the one set in the game and in the game ring
			assert.Equal(t, r2.GetDealer(), newDealer.GetSlot())
			assert.Equal(t, newDealer.GetSlot(), s.GetSlot())

		})
	}
}
