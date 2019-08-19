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
	testClient pb.PokerClient
	testDatabase string
	testConnection *grpc.ClientConn
)
func init(){
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
				Players:[]*pb.Player{
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
			ExpError:"",

		},
		{
			Name: "Create players with one as empty name",
			Players: &pb.Players{
				Players:[]*pb.Player{
					{
						Name:  "bob2",
						Chips: 0,
					},
					{
						Name:  "jim2",
						Chips: 0,
					},
					{
						Name:  "",  // should cause error
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
		Game   *pb.Game
		ExpError string
	}{
		{
			Name: "Create a game",
			Game: &pb.Game{
				Name:  "testgame0",
			},
			ExpError: "",
		},
		//{
		//	Name: "Create game with empty name",
		//	Game: &pb.Game{
		//		Name:  "",
		//	},
		//	ExpError: "rpc error: code = Unknown desc = can not create game with empty name",
		//},
		//{
		//	Name: "Create game that already exists",
		//	Game: &pb.Game{
		//		Name:  "testgame0",
		//	},
		//	ExpError: "rpc error: code = Unknown desc = game with that name already exists",
		//},
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