package server_test

import (
	"context"
	"fmt"
	"imran/poker/client"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	pb "imran/poker/protobufs"
	"imran/poker/server"
)

const dbName = "testDb"

var (
	testClient     pb.PokerClient
	testDatabase   string
	testConnection *grpc.ClientConn
)

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

	tests := []struct {
		Name     string
		Player   *pb.Player
		ExpError string
	}{
		{
			Name: "Create a player",
			Player: &pb.Player{
				Name:  "bob0",
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
				Name:  "bob0",
				Chips: 0,
			},
			ExpError: "rpc error: code = Unknown desc = Player with that name already exists",
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
				Players: []*pb.Player{
					{
						Name:  "bob1",
						Chips: 0,
					},
					{
						Name:  "jim1",
						Chips: 0,
					},
					{
						Name:  "fred1",
						Chips: 0,
					},
					{
						Name:  "cam1",
						Chips: 0,
					},
					{
						Name:  "tim1",
						Chips: 0,
					},
				},
			},
			ExpError: "",
		},
		{
			Name: "Create players with one as empty name",
			Players: &pb.Players{
				Players: []*pb.Player{
					{
						Name:  "bob2",
						Chips: 0,
					},
					{
						Name:  "jim2",
						Chips: 0,
					},
					{
						Name:  "", // should cause error
						Chips: 0,
					},
					{
						Name:  "cam2",
						Chips: 0,
					},
					{
						Name:  "tim2",
						Chips: 0,
					},
				},
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

func TestServer_CreateGamePlayers(t *testing.T) {

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
			PlayersToCreate: &pb.Players{
				Players: []*pb.Player{
					{
						Name:  "bob3",
						Chips: 0,
					},
					{
						Name:  "jim3",
						Chips: 0,
					},
					{
						Name:  "fred3",
						Chips: 0,
					},
					{
						Name:  "cam3",
						Chips: 0,
					},
					{
						Name:  "tim3",
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
						{
							Name:  "bob3",
							Chips: 0,
						},
						{
							Name:  "jim3",
							Chips: 0,
						},
						{
							Name:  "fred3",
							Chips: 0,
						},
						{
							Name:  "cam3",
							Chips: 0,
						},
						{
							Name:  "tim3",
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
							Name:  "tim3",
							Chips: 0,
						},
						{
							Name:  "cam3",
							Chips: 0,
						},
						{
							Name:  "tim3",
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
							Name:  "tim3",
							Chips: 0,
						},
						{
							Name:  "jaimie", //new player
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
			game.Players = tt.GameToCreate.GetPlayers()

			require.NoError(t, err)
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
